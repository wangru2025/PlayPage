package http

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-static-host/api/internal/domain"
	"ai-static-host/api/internal/service"
	"github.com/gorilla/websocket"
)

const maxRepairHTMLBytes = 2 << 20
const maxRepairPromptHTMLChars = 180000

type repairAIHub struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]bool
}

func newRepairAIHub() *repairAIHub {
	return &repairAIHub{clients: map[string]map[*websocket.Conn]bool{}}
}
func (h *repairAIHub) add(jobID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[jobID] == nil {
		h.clients[jobID] = map[*websocket.Conn]bool{}
	}
	h.clients[jobID][c] = true
}
func (h *repairAIHub) remove(jobID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[jobID] != nil {
		delete(h.clients[jobID], c)
		if len(h.clients[jobID]) == 0 {
			delete(h.clients, jobID)
		}
	}
}
func (h *repairAIHub) broadcast(jobID string, payload any) {
	h.mu.Lock()
	list := make([]*websocket.Conn, 0, len(h.clients[jobID]))
	for c := range h.clients[jobID] {
		list = append(list, c)
	}
	h.mu.Unlock()
	for _, c := range list {
		_ = c.WriteJSON(payload)
	}
}

type repairAIRunRegistry struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func newRepairAIRunRegistry() *repairAIRunRegistry {
	return &repairAIRunRegistry{cancels: map[string]context.CancelFunc{}}
}

func (r *repairAIRunRegistry) add(jobID string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cancels[jobID] = cancel
}

func (r *repairAIRunRegistry) remove(jobID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cancels, jobID)
}

func (r *repairAIRunRegistry) stop(jobID string) bool {
	r.mu.Lock()
	cancel := r.cancels[jobID]
	r.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

type repairAIStartResponse struct {
	Job      domain.RepairAIJob       `json:"job"`
	Messages []domain.RepairAIMessage `json:"messages,omitempty"`
}
type repairAIPreviewResponse struct {
	URL string             `json:"url"`
	Job domain.RepairAIJob `json:"job"`
}

func (rt *Router) handleStartRepairAI(w http.ResponseWriter, r *http.Request, projectID, requestID, feedback string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	req, found, err := rt.store.GetRepairRequest(r.Context(), projectID, requestID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "读取修复申请失败"})
		return
	}
	if !found || req.OwnerUserID != user.ID {
		writeJSON(w, 404, map[string]string{"error": "没有找到这条修复申请"})
		return
	}
	if !req.AllowAdminEdit {
		writeJSON(w, 400, map[string]string{"error": "这条申请没有授权修复作品，请重新提交并勾选授权"})
		return
	}
	if active, ok, _ := rt.store.GetActiveRepairAIJobByUser(r.Context(), user.ID); ok && active.ID != "" {
		writeJSON(w, 409, map[string]string{"error": "你已经有一个 AI 圆桌正在运行，请等它结束后再开始新的修复。"})
		return
	}
	if !domain.IsAdminRole(user.Role) {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Now().In(loc)
		since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC()
		count, err := rt.store.CountRepairAIJobsForUserSince(r.Context(), user.ID, since)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "检查今日次数失败"})
			return
		}
		if count >= 2 {
			writeJSON(w, 429, map[string]string{"error": "今天的 AI 圆桌次数已经用完。你可以明天再试，或者等待管理员处理。"})
			return
		}
	}
	round := 1
	if latest, ok, _ := rt.store.GetLatestRepairAIJob(r.Context(), requestID); ok {
		round = latest.Round + 1
	}
	if round > 2 {
		writeJSON(w, 400, map[string]string{"error": "这条申请已经交给管理员处理，不能再启动 AI 圆桌。"})
		return
	}
	job, err := rt.store.CreateRepairAIJob(r.Context(), domain.RepairAIJob{RepairRequestID: requestID, ProjectID: projectID, OwnerUserID: user.ID, Status: "running", Round: round, Feedback: strings.TrimSpace(feedback)})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "创建 AI 圆桌失败"})
		return
	}
	rt.broadcastJob(job, "job")
	runCtx, cancel := context.WithCancel(context.Background())
	rt.aiRuns.add(job.ID, cancel)
	go func() {
		defer rt.aiRuns.remove(job.ID)
		defer cancel()
		rt.runRepairAIRoundtable(runCtx, user, project, req, job)
	}()
	writeJSON(w, http.StatusCreated, repairAIStartResponse{Job: job})
}

func (rt *Router) handleStopRepairAI(w http.ResponseWriter, r *http.Request, projectID, requestID string) {
	user, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	job, found, err := rt.store.GetLatestRepairAIJob(r.Context(), requestID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "读取 AI 圆桌失败"})
		return
	}
	if !found || job.ProjectID != projectID || job.OwnerUserID != user.ID {
		writeJSON(w, 404, map[string]string{"error": "没有找到这个 AI 圆桌"})
		return
	}
	if job.Status != "running" {
		messages, _ := rt.store.ListRepairAIMessages(r.Context(), job.ID)
		writeJSON(w, 200, repairAIStartResponse{Job: job, Messages: messages})
		return
	}
	rt.aiRuns.stop(job.ID)
	job.Status = "canceled"
	job.ErrorMessage = "用户已叫停 AI 圆桌。"
	job.FinishedAt = time.Now().UTC()
	updated, err := rt.store.UpdateRepairAIJob(r.Context(), job)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "叫停 AI 圆桌失败"})
		return
	}
	rt.addAIMessage(r.Context(), &updated, "system", "系统记录员", "assistant", "error", "用户已叫停 AI 圆桌。")
	rt.broadcastJob(updated, "job")
	messages, _ := rt.store.ListRepairAIMessages(r.Context(), updated.ID)
	writeJSON(w, 200, repairAIStartResponse{Job: updated, Messages: messages})
}

func (rt *Router) handleRepairAIFeedback(w http.ResponseWriter, r *http.Request, projectID, requestID string) {
	_, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	latest, found, err := rt.store.GetLatestRepairAIJob(r.Context(), requestID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "读取 AI 圆桌失败"})
		return
	}
	if !found {
		writeJSON(w, 404, map[string]string{"error": "还没有 AI 圆桌记录"})
		return
	}
	if latest.Round >= 2 {
		latest.Status = "needs_admin"
		latest.ErrorMessage = "用户二次反馈仍有问题，已转人工处理。"
		latest.FinishedAt = time.Now().UTC()
		latest, _ = rt.store.UpdateRepairAIJob(r.Context(), latest)
		rt.broadcastJob(latest, "job")
		writeJSON(w, http.StatusOK, map[string]any{"needsAdmin": true, "message": "真不好意思，这几个牛马太傻了，已经把此申请交给管理员处理。", "job": latest})
		return
	}
	var input domain.RepairAIFeedbackInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	feedback := trimLimit(input.Feedback, 2000)
	if feedback == "" {
		writeJSON(w, 400, map[string]string{"error": "请填写问题还存在什么情况"})
		return
	}
	feedback = rt.feedbackWithPreviousRoundContext(r.Context(), latest, feedback)
	rt.handleStartRepairAI(w, r, projectID, requestID, feedback)
}

func (rt *Router) handleLatestRepairAI(w http.ResponseWriter, r *http.Request, projectID, requestID string) {
	_, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	job, found, err := rt.store.GetLatestRepairAIJob(r.Context(), requestID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "读取 AI 圆桌失败"})
		return
	}
	if !found {
		writeJSON(w, 404, map[string]string{"error": "还没有 AI 圆桌记录"})
		return
	}
	messages, _ := rt.store.ListRepairAIMessages(r.Context(), job.ID)
	writeJSON(w, 200, repairAIStartResponse{Job: job, Messages: messages})
}

func (rt *Router) handleRepairAIWebSocket(w http.ResponseWriter, r *http.Request, projectID, requestID string) {
	_, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
	if jobID == "" {
		latest, found, _ := rt.store.GetLatestRepairAIJob(r.Context(), requestID)
		if found {
			jobID = latest.ID
		}
	}
	if jobID == "" {
		writeJSON(w, 404, map[string]string{"error": "还没有 AI 圆桌记录"})
		return
	}
	job, found, err := rt.store.GetRepairAIJob(r.Context(), jobID)
	if err != nil || !found || job.ProjectID != projectID || job.RepairRequestID != requestID {
		writeJSON(w, 404, map[string]string{"error": "没有找到这个 AI 圆桌"})
		return
	}
	up := websocket.Upgrader{CheckOrigin: sameHostOrigin}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	rt.aiHub.add(jobID, conn)
	defer func() { rt.aiHub.remove(jobID, conn); _ = conn.Close() }()
	messages, _ := rt.store.ListRepairAIMessages(r.Context(), jobID)
	_ = conn.WriteJSON(map[string]any{"type": "snapshot", "job": job, "messages": messages})
	for {
		if _, _, err := conn.NextReader(); err != nil {
			return
		}
	}
}

func (rt *Router) handleRepairAIPreview(w http.ResponseWriter, r *http.Request, projectID, requestID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	job, found, err := rt.store.GetLatestRepairAIJob(r.Context(), requestID)
	if err != nil || !found {
		writeJSON(w, 404, map[string]string{"error": "还没有可预览的 AI 修复结果"})
		return
	}
	if job.GeneratedHTML == "" {
		writeJSON(w, 400, map[string]string{"error": "AI 还没有生成可预览的修复结果"})
		return
	}
	rel, found, err := rt.currentRelease(r.Context(), project)
	if err != nil || !found {
		writeJSON(w, 400, map[string]string{"error": "这个作品还没有可用于预览的已发布版本"})
		return
	}
	relPath, err := rt.pub.CreatePreview(project, rel.PublicPath, rel.EntryFile, job.ID, []byte(job.GeneratedHTML))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "生成预览失败：" + err.Error()})
		return
	}
	job.PreviewURL = strings.TrimRight(rt.cfg.PublicBase, "/") + "/" + strings.TrimLeft(relPath, "/")
	job, _ = rt.store.UpdateRepairAIJob(r.Context(), job)
	rt.broadcastJob(job, "job")
	writeJSON(w, 200, repairAIPreviewResponse{URL: job.PreviewURL, Job: job})
}

func (rt *Router) handleRepairAIPublish(w http.ResponseWriter, r *http.Request, projectID, requestID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	job, found, err := rt.store.GetLatestRepairAIJob(r.Context(), requestID)
	if err != nil || !found {
		writeJSON(w, 404, map[string]string{"error": "还没有可发布的 AI 修复结果"})
		return
	}
	if job.GeneratedHTML == "" {
		writeJSON(w, 400, map[string]string{"error": "AI 还没有生成可发布的修复结果"})
		return
	}
	rel, found, err := rt.currentRelease(r.Context(), project)
	if err != nil || !found {
		writeJSON(w, 400, map[string]string{"error": "这个作品还没有可发布的旧版本"})
		return
	}
	if _, err := rt.pub.PublishPatchedHTML(r.Context(), project, rel.PublicPath, rel.EntryFile, []byte(job.GeneratedHTML)); err != nil {
		writeJSON(w, 500, map[string]string{"error": "发布修复版本失败：" + err.Error()})
		return
	}
	job.Status = "published"
	job.FinishedAt = time.Now().UTC()
	job, _ = rt.store.UpdateRepairAIJob(r.Context(), job)
	rt.broadcastJob(job, "job")
	writeJSON(w, 200, map[string]any{"job": job, "message": "修复版本已发布。"})
}

func (rt *Router) runRepairAIRoundtable(ctx context.Context, user domain.User, project domain.Project, req domain.RepairRequest, job domain.RepairAIJob) {
	fail := func(msg string) {
		if ctx.Err() != nil {
			rt.finishRepairAICanceled(context.Background(), job)
			return
		}
		job.Status = "failed"
		job.ErrorMessage = msg
		job.FinishedAt = time.Now().UTC()
		updated, _ := rt.store.UpdateRepairAIJob(ctx, job)
		rt.addAIMessage(ctx, &updated, "system", "系统记录员", "assistant", "error", msg)
		rt.broadcastJob(updated, "job")
	}
	rt.addAIMessage(ctx, &job, "host", "圆桌主持", "assistant", "status", fmt.Sprintf("第 %d 轮 AI 急救圆桌已经开始。今天值班的几个牛马会先吵一小会儿，再给出可预览的修复版。", job.Round))
	rel, found, err := rt.currentRelease(ctx, project)
	if err != nil || !found {
		fail("这个作品还没有上传内容，不能启动 AI 急救圆桌。")
		return
	}
	rt.addAIMessage(ctx, &job, "reader", "作品读取员", "assistant", "status", "已经找到当前发布版本，正在读取入口 HTML。")
	source, err := readReleaseHTML(rel.PublicPath, rel.EntryFile)
	if err != nil {
		fail(err.Error())
		return
	}
	sourceForPrompt := trimMiddle(source, maxRepairPromptHTMLChars)
	if sourceForPrompt != source {
		rt.addAIMessage(ctx, &job, "reader", "作品读取员", "assistant", "status", "入口 HTML 较大，已保留开头和结尾的关键代码给 AI 分析。")
	}
	apiDoc := "这个作品没有启用互动 API。"
	if project.Interactive {
		apiDoc = rt.buildInteractiveDocForPrompt(ctx, project)
	}
	previous := rt.previousRepairAIMessages(ctx, req.ID, job.ID)
	if previous == "" && strings.Contains(job.Feedback, "上一轮圆桌记录：") {
		previous = job.Feedback
	}
	basePrompt := fmt.Sprintf("作品名：%s\n作品地址：%s\n用户问题：%s\n期望效果：%s\n本轮用户补充：%s\n上一轮圆桌上下文：\n%s\n\n互动 API 文档：\n%s\n\n当前入口 HTML：\n%s", project.Name, project.PublicURL, req.Description, req.Expected, job.Feedback, previous, trimMiddle(apiDoc, 60000), sourceForPrompt)
	transcript := ""
	roundtableAgents := []repairAIAgent{
		{Key: "reader", Name: "作品读取员", Persona: "你负责快速读懂作品。指出作品结构、入口文件、用户想要什么，以及最可能坏在哪里。语气可以像打工人吐槽，但不要攻击用户。"},
		{Key: "frontend", Name: "前端急救员", Persona: "你是资深前端。重点检查 HTML、CSS、JavaScript、事件绑定、运行时报错、编码和可访问性。请明确列出要改的点。语气直接一点，可以幽默。"},
		{Key: "api", Name: "互动 API 守门员", Persona: "你负责检查互动 API 和云数据读写。重点看 API_BASE、X-Project-Key、数据表、fetch 错误处理、不要跨域乱写、不要删除用户数据。没有互动 API 问题也要说明。"},
		{Key: "critic", Name: "挑刺检查员", Persona: "你负责挑刺。找前面分析里的漏洞，指出还可能漏掉的风险，并给修复员下最后指令。语气像代码评审，短句，清楚。"},
	}
	for _, agent := range roundtableAgents {
		rt.addAIMessage(ctx, &job, agent.Key, agent.Name, "assistant", "status", agent.Name+"正在发言。")
		speech, err := rt.generateRepairAISpeech(ctx, agent, basePrompt, transcript)
		if err != nil {
			fail(err.Error())
			return
		}
		speech = trimLimit(speech, 1600)
		rt.addAIMessage(ctx, &job, agent.Key, agent.Name, "assistant", "text", speech)
		transcript += agent.Name + "：" + speech + "\n"
	}
	rt.addAIMessage(ctx, &job, "fixer", "修复员", "assistant", "status", "前面几个牛马吵完了，修复员正在生成局部补丁。")
	system := `你是 PlayPage 网页急救圆桌的最终修复员。你必须根据用户问题、当前 HTML、互动 API 文档和圆桌讨论，输出一个 JSON 对象，不要 Markdown。
JSON 字段：
- visible_message：中文，给用户看的简短圆桌总结。
- patch：apply_patch 风格补丁文本。

patch 必须严格使用这种格式：
*** Begin Patch
*** Update File: index.html
@@
 原文上下文行
-需要删除的原文
+需要新增的新文
 原文上下文行
*** End Patch

硬性要求：
- 必须局部修改，不要重写完整 HTML。
- patch 里的旧文本必须来自“当前入口 HTML”里的原文，保证后端可以精确替换。
- 每个 hunk 至少保留 1 到 3 行原文上下文。
- 可以有多个 @@ hunk。
- 不要删除用户数据，不要删除互动 API key，不要把 API 地址改成其他域名。
- 如果使用互动 API，严格按文档使用相对 API_BASE 和 X-Project-Key。
- 不要引入需要构建的框架。
- 不要输出解释文本，只输出 JSON。`
	prompt := basePrompt + "\n\n圆桌讨论记录：\n" + transcript + "\n请现在给出最终局部补丁。"
	visible, fixed, err := rt.generateAndApplyRepairPatch(ctx, &job, system, prompt, source)
	if err != nil {
		fail(err.Error())
		return
	}
	fixed = ensureRepairHTML(fixed)
	if visible == "" {
		visible = "圆桌已经打好局部补丁，请先预览效果。"
	}
	review := rt.generateRepairAIReview(ctx, basePrompt, transcript, visible, fixed)
	if review != "" {
		rt.addAIMessage(ctx, &job, "critic", "挑刺检查员", "assistant", "text", review)
	}
	rt.addAIMessage(ctx, &job, "fixer", "修复员", "assistant", "result", visible)
	job.GeneratedHTML = fixed
	job.Status = "ready_preview"
	job.FinishedAt = time.Now().UTC()
	updated, err := rt.store.UpdateRepairAIJob(ctx, job)
	if err != nil {
		return
	}
	rt.broadcastJob(updated, "job")
}

func (rt *Router) generateAndApplyRepairPatch(ctx context.Context, job *domain.RepairAIJob, system, prompt, source string) (string, string, error) {
	var lastPatch string
	var lastErr error
	var visible string
	for attempt := 1; ; attempt++ {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		if attempt > 1 {
			rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", fmt.Sprintf("第 %d 次重写补丁。上一次失败原因：%s", attempt, lastErr.Error()))
		}
		nextPrompt := prompt
		if lastErr != nil {
			nextPrompt = prompt + "\n\n上一份补丁：\n" + lastPatch + "\n\n后端应用补丁失败，错误是：\n" + lastErr.Error() + "\n\n请重新输出 JSON，只输出 visible_message 和 patch。新 patch 必须使用当前入口 HTML 中可以精确匹配的原文；每一行必须是 apply_patch 兼容格式。"
		}
		content, err := rt.aiClient.CompletePlain(ctx, []service.AIMessage{{Role: "system", Content: system}, {Role: "user", Content: nextPrompt}}, 6000)
		if err != nil {
			if ctx.Err() != nil {
				return "", "", ctx.Err()
			}
			lastPatch = ""
			lastErr = err
			rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", fmt.Sprintf("第 %d 次请求 AI 失败：%s。系统会继续重试，用户可以随时叫停。", attempt, err.Error()))
			select {
			case <-ctx.Done():
				return "", "", ctx.Err()
			case <-time.After(5 * time.Second):
			}
			continue
		}
		rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", fmt.Sprintf("修复员第 %d 次交卷，正在应用局部补丁。", attempt))
		patchVisible, patchText, err := parseRepairAIPatchResult(content)
		if err != nil {
			lastPatch = content
			lastErr = fmt.Errorf("AI 返回的补丁格式不正确：%w", err)
			continue
		}
		if patchVisible != "" {
			visible = patchVisible
		}
		lastPatch = patchText
		fixed, err := applyRepairAIPatch(source, patchText)
		if err != nil {
			lastErr = err
			continue
		}
		return visible, fixed, nil
	}
}

func (rt *Router) finishRepairAICanceled(ctx context.Context, job domain.RepairAIJob) {
	latest, found, err := rt.store.GetRepairAIJob(ctx, job.ID)
	if err == nil && found {
		job = latest
	}
	if job.Status != "running" {
		return
	}
	job.Status = "canceled"
	job.ErrorMessage = "用户已叫停 AI 圆桌。"
	job.FinishedAt = time.Now().UTC()
	updated, err := rt.store.UpdateRepairAIJob(ctx, job)
	if err != nil {
		return
	}
	rt.addAIMessage(ctx, &updated, "system", "系统记录员", "assistant", "error", "用户已叫停 AI 圆桌。")
	rt.broadcastJob(updated, "job")
}

type repairAIAgent struct {
	Key     string
	Name    string
	Persona string
}

func (rt *Router) generateRepairAISpeech(ctx context.Context, agent repairAIAgent, basePrompt, transcript string) (string, error) {
	system := fmt.Sprintf(`你是 PlayPage 网页急救圆桌里的「%s」。%s
请只输出你这一位角色的一段中文发言，不要 JSON，不要 Markdown 表格，不要代码块。
要求：
- 80 到 220 字。
- 说人话，用户能看懂。
- 可以有一点打工人式幽默，但不要低俗，不要嘲笑用户。
- 必须围绕这次作品修复，说清楚你发现了什么、建议怎么改。
- 不要声称已经修改文件，你现在只是圆桌发言。`, agent.Name, agent.Persona)
	prompt := basePrompt + "\n\n前面角色发言：\n" + transcript + "\n请轮到你发言。"
	return rt.aiClient.CompletePlain(ctx, []service.AIMessage{{Role: "system", Content: system}, {Role: "user", Content: prompt}}, 1200)
}

func (rt *Router) generateRepairAIReview(ctx context.Context, basePrompt, transcript, visible, fixed string) string {
	system := `你是 PlayPage 网页急救圆桌里的挑刺检查员。请对最终修复结果做一段给用户看的中文检查结论。
只输出纯文本，不要 JSON，不要 Markdown 表格，不要代码块。80 到 180 字。语气可以轻微吐槽，但重点是说明已经检查了什么、用户应该预览什么。`
	prompt := basePrompt + "\n\n圆桌讨论记录：\n" + transcript + "\n\n修复员总结：\n" + visible + "\n\n修复后 HTML 片段：\n" + trimMiddle(fixed, 12000)
	content, err := rt.aiClient.CompletePlain(ctx, []service.AIMessage{{Role: "system", Content: system}, {Role: "user", Content: prompt}}, 1000)
	if err != nil {
		log.Printf("repair ai review failed: %v", err)
		return ""
	}
	return trimLimit(content, 1200)
}

func (rt *Router) addAIMessage(ctx context.Context, job *domain.RepairAIJob, agentKey, agentName, role, messageType, content string) {
	messages, _ := rt.store.ListRepairAIMessages(ctx, job.ID)
	msg, err := rt.store.CreateRepairAIMessage(ctx, domain.RepairAIMessage{JobID: job.ID, RepairRequestID: job.RepairRequestID, ProjectID: job.ProjectID, OwnerUserID: job.OwnerUserID, AgentKey: agentKey, AgentName: agentName, Role: role, Visibility: "public", MessageType: messageType, Content: content, MessageSeq: len(messages) + 1})
	if err == nil {
		rt.aiHub.broadcast(job.ID, map[string]any{"type": "message", "message": msg})
		return
	}
	log.Printf("create repair ai message failed: job_id=%s agent=%s err=%v", job.ID, agentKey, err)
}
func (rt *Router) broadcastJob(job domain.RepairAIJob, typ string) {
	if rt.aiHub != nil {
		rt.aiHub.broadcast(job.ID, map[string]any{"type": typ, "job": job})
	}
}

func (rt *Router) currentRelease(ctx context.Context, project domain.Project) (domain.Release, bool, error) {
	releases, err := rt.store.ListReleases(ctx, project.ID)
	if err != nil {
		return domain.Release{}, false, err
	}
	if len(releases) == 0 {
		return domain.Release{}, false, nil
	}
	for _, rel := range releases {
		if rel.ID == project.CurrentRelease {
			return rel, true, nil
		}
	}
	sort.Slice(releases, func(i, j int) bool { return releases[i].CreatedAt.After(releases[j].CreatedAt) })
	return releases[0], true, nil
}
func readReleaseHTML(publicPath, entryFile string) (string, error) {
	if entryFile == "" {
		entryFile = "index.html"
	}
	path := filepath.Join(publicPath, entryFile)
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("读取作品 HTML 失败")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRepairHTMLBytes+1))
	if err != nil {
		return "", fmt.Errorf("读取作品 HTML 失败")
	}
	if len(data) > maxRepairHTMLBytes {
		return "", fmt.Errorf("作品入口 HTML 太大，暂时不能使用 AI 圆桌自动修复")
	}
	return string(data), nil
}
func (rt *Router) buildInteractiveDocForPrompt(ctx context.Context, project domain.Project) string {
	access, found, err := rt.store.GetProjectPublicAccess(ctx, project.ID)
	if err != nil || !found {
		return "互动 API 已开启，但读取密钥失败。"
	}
	cols, _ := rt.store.ListCollections(ctx, project.ID)
	return buildInteractiveAPIDoc(project, access.PublicKey, cols)
}
func (rt *Router) feedbackWithPreviousRoundContext(ctx context.Context, previousJob domain.RepairAIJob, feedback string) string {
	messages, _ := rt.store.ListRepairAIMessages(ctx, previousJob.ID)
	var b strings.Builder
	b.WriteString(feedback)
	b.WriteString("\n\n上一轮圆桌记录：\n")
	for _, m := range messages {
		b.WriteString(m.AgentName + ": " + trimLimit(m.Content, 1200) + "\n")
	}
	if previousJob.GeneratedHTML != "" {
		b.WriteString("上一轮已经生成过 HTML。请在上一轮修复结果基础上，按照用户最新反馈继续修复。\n")
		b.WriteString("上一轮生成 HTML 摘要/片段：\n")
		b.WriteString(trimLimit(previousJob.GeneratedHTML, 12000))
		b.WriteString("\n")
	}
	return trimLimit(b.String(), 20000)
}

func (rt *Router) previousRepairAIMessages(ctx context.Context, requestID, excludeJobID string) string {
	latest, found, _ := rt.store.GetLatestRepairAIJob(ctx, requestID)
	if !found || latest.ID == excludeJobID {
		return ""
	}
	messages, _ := rt.store.ListRepairAIMessages(ctx, latest.ID)
	var b strings.Builder
	for _, m := range messages {
		b.WriteString(m.AgentName + ": " + m.Content + "\n")
	}
	if latest.GeneratedHTML != "" {
		b.WriteString("上一轮已经生成过 HTML，用户反馈后需要在此基础上继续修复。\n")
	}
	return b.String()
}

type repairAIJSONResult struct {
	VisibleMessage string `json:"visible_message"`
	FixedHTML      string `json:"fixed_html"`
}

type repairAIPatchJSONResult struct {
	VisibleMessage string `json:"visible_message"`
	Patch          string `json:"patch"`
}

func parseRepairAIPatchResult(content string) (string, string, error) {
	raw := strings.TrimSpace(content)
	raw = stripCodeFence(raw, "json")
	var out repairAIPatchJSONResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		patch := extractApplyPatch(content)
		if patch == "" {
			return "", "", err
		}
		return "AI 没有按标准 JSON 返回，但我提取到了局部补丁。", patch, nil
	}
	patch := strings.TrimSpace(out.Patch)
	if patch == "" {
		patch = extractApplyPatch(content)
	}
	if patch == "" {
		return "", "", fmt.Errorf("缺少 patch")
	}
	return strings.TrimSpace(out.VisibleMessage), patch, nil
}

func extractApplyPatch(s string) string {
	raw := strings.TrimSpace(s)
	raw = stripCodeFence(raw, "diff")
	raw = stripCodeFence(raw, "patch")
	start := strings.Index(raw, "*** Begin Patch")
	end := strings.LastIndex(raw, "*** End Patch")
	if start < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(raw[start : end+len("*** End Patch")])
}

func ensureRepairHTML(fixed string) string {
	fixed = strings.TrimSpace(fixed)
	if !strings.Contains(strings.ToLower(fixed), "<html") {
		fixed = "<!doctype html>\n<html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>修复后的作品</title></head><body>" + html.EscapeString(fixed) + "</body></html>"
	}
	if !strings.Contains(strings.ToLower(fixed), "charset=") {
		fixed = strings.Replace(fixed, "<head>", "<head><meta charset=\"utf-8\">", 1)
	}
	return fixed
}

func parseRepairAIResult(content string) (string, string, error) {
	raw := strings.TrimSpace(content)
	raw = stripCodeFence(raw, "json")
	var out repairAIJSONResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		htmlText := extractHTML(content)
		if htmlText == "" {
			return "", "", err
		}
		return "AI 没有按标准格式返回，但我提取到了修复后的 HTML。", htmlText, nil
	}
	fixed := strings.TrimSpace(out.FixedHTML)
	if fixed == "" {
		fixed = extractHTML(content)
	}
	if fixed == "" {
		return "", "", fmt.Errorf("缺少 fixed_html")
	}
	return strings.TrimSpace(out.VisibleMessage), ensureRepairHTML(fixed), nil
}
func stripCodeFence(s, lang string) string {
	patterns := []string{
		"(?is)^\\s*```" + lang + "\\s*(.*?)\\s*```\\s*$",
		"(?is)```" + lang + "\\s*(.*?)\\s*```",
		"(?is)^\\s*```\\s*(.*?)\\s*```\\s*$",
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if m := re.FindStringSubmatch(s); len(m) == 2 {
			return strings.TrimSpace(m[1])
		}
	}
	if start := strings.Index(s, "{"); start >= 0 {
		if end := strings.LastIndex(s, "}"); end > start {
			return strings.TrimSpace(s[start : end+1])
		}
	}
	return s
}

func extractHTML(s string) string {
	re := regexp.MustCompile("(?is)```html\\s*(.*?)\\s*```")
	if m := re.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	idx := strings.Index(strings.ToLower(s), "<!doctype html")
	if idx < 0 {
		idx = strings.Index(strings.ToLower(s), "<html")
	}
	if idx >= 0 {
		return strings.TrimSpace(s[idx:])
	}
	return ""
}

type repairPatchLine struct {
	kind string
	text string
}

func applyRepairAIPatch(source, patch string) (string, error) {
	lines := splitPatchLines(patch)
	if len(lines) == 0 {
		return "", fmt.Errorf("补丁为空")
	}
	if strings.TrimSpace(lines[0]) != "*** Begin Patch" {
		return "", fmt.Errorf("补丁必须以 *** Begin Patch 开头")
	}
	if strings.TrimSpace(lines[len(lines)-1]) != "*** End Patch" {
		return "", fmt.Errorf("补丁必须以 *** End Patch 结尾")
	}
	i := 1
	applied := source
	updatedFiles := 0
	for i < len(lines)-1 {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}
		if !strings.HasPrefix(line, "*** Update File:") {
			return "", fmt.Errorf("只支持 *** Update File 补丁")
		}
		fileName := strings.TrimSpace(strings.TrimPrefix(line, "*** Update File:"))
		if fileName == "" {
			return "", fmt.Errorf("Update File 缺少文件名")
		}
		if fileName != "index.html" && fileName != "./index.html" {
			return "", fmt.Errorf("只允许修改 index.html")
		}
		updatedFiles++
		i++
		for i < len(lines)-1 {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), "*** Update File:") {
				break
			}
			if strings.TrimSpace(lines[i]) == "" {
				i++
				continue
			}
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), "@@") {
				return "", fmt.Errorf("每个修改块必须以 @@ 开头")
			}
			i++
			var hunk []repairPatchLine
			for i < len(lines)-1 {
				trimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(trimmed, "@@") || strings.HasPrefix(trimmed, "*** Update File:") {
					break
				}
				raw := lines[i]
				if raw == "" {
					hunk = append(hunk, repairPatchLine{kind: " ", text: ""})
					i++
					continue
				}
				switch {
				case strings.HasPrefix(raw, "+"):
					hunk = append(hunk, repairPatchLine{kind: "+", text: raw[1:]})
				case strings.HasPrefix(raw, "-"):
					hunk = append(hunk, repairPatchLine{kind: "-", text: raw[1:]})
				case strings.HasPrefix(raw, " "):
					hunk = append(hunk, repairPatchLine{kind: " ", text: raw[1:]})
				default:
					return "", fmt.Errorf("补丁行必须以空格、+ 或 - 开头")
				}
				i++
			}
			var err error
			applied, err = applyRepairPatchHunk(applied, hunk)
			if err != nil {
				return "", err
			}
		}
	}
	if updatedFiles == 0 {
		return "", fmt.Errorf("补丁没有包含 Update File")
	}
	if applied == source {
		return "", fmt.Errorf("补丁没有产生任何变化")
	}
	return applied, nil
}

func applyRepairPatchHunk(source string, hunk []repairPatchLine) (string, error) {
	if len(hunk) == 0 {
		return "", fmt.Errorf("修改块为空")
	}
	var oldLines []string
	var newLines []string
	removed := 0
	added := 0
	for _, line := range hunk {
		switch line.kind {
		case " ":
			oldLines = append(oldLines, line.text)
			newLines = append(newLines, line.text)
		case "-":
			oldLines = append(oldLines, line.text)
			removed++
		case "+":
			newLines = append(newLines, line.text)
			added++
		}
	}
	if removed == 0 && added == 0 {
		return "", fmt.Errorf("修改块没有新增或删除内容")
	}
	oldText := strings.Join(oldLines, "\n")
	newText := strings.Join(newLines, "\n")
	if oldText == "" {
		return "", fmt.Errorf("修改块缺少可匹配的原文")
	}
	if strings.Count(source, oldText) != 1 {
		return "", fmt.Errorf("补丁原文匹配次数不是 1，实际为 %d；请增加上下文或改用更精确的原文", strings.Count(source, oldText))
	}
	return strings.Replace(source, oldText, newText, 1), nil
}

func splitPatchLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func sameHostOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}
