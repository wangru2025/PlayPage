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
	allRoundsContext := rt.allRepairAIRoundContext(ctx, req.ID, job.ID)
	roundGoal := "这是第 1 轮圆桌：先根据用户原始问题诊断并修复。"
	if job.Round > 1 {
		roundGoal = fmt.Sprintf("这是第 %d 轮圆桌：必须先理解当前这条修复申请内的所有历史轮次、失败原因和用户最新反馈，然后在现有发布版基础上继续迭代，不能重复上一轮已经失败的方向。不要读取或引用同一作品其他修复申请的问题。", job.Round)
	}
	basePrompt := fmt.Sprintf("作品名：%s\n作品地址：%s\n修复申请 ID：%s\n用户原始问题：%s\n期望效果：%s\n当前轮次：第 %d 轮\n本轮目标：%s\n本轮用户补充：%s\n当前修复申请内的历史圆桌上下文：\n%s\n\n互动 API 文档：\n%s\n\n当前入口 HTML：\n%s", project.Name, project.PublicURL, req.ID, req.Description, req.Expected, job.Round, roundGoal, job.Feedback, allRoundsContext, trimMiddle(apiDoc, 60000), sourceForPrompt)
	transcript := ""
	roundtableAgents := []repairAIAgent{
		{Key: "reader", Name: "作品读取员", Persona: "你负责快速读懂作品。指出作品结构、入口文件、用户想要什么，以及最可能坏在哪里。语气可以像打工人吐槽，但不要攻击用户。"},
		{Key: "frontend", Name: "前端急救员", Persona: "你是资深前端。重点检查 HTML、CSS、JavaScript、事件绑定、运行时报错、编码和可访问性。请明确列出要改的点。语气直接一点，可以幽默。"},
		{Key: "api", Name: "互动 API 守门员", Persona: "你负责检查互动 API 和云数据读写。重点看 API_BASE、X-Project-Key、数据表、fetch 错误处理、不要跨域乱写、不要删除用户数据。没有互动 API 问题也要说明。"},
		{Key: "critic", Name: "挑刺检查员", Persona: "你负责挑刺。找前面分析里的漏洞，指出还可能漏掉的风险，并给修复员下最后指令。语气像代码评审，短句，清楚。"},
	}
	for _, agent := range roundtableAgents {
		rt.addAIMessage(ctx, &job, agent.Key, agent.Name, "assistant", "status", agent.Name+"正在发言。")
		speech, err := rt.generateRepairAISpeechWithTools(ctx, agent, basePrompt, source, apiDoc, allRoundsContext, transcript, job.Round)
		if err != nil {
			speech, err = rt.generateRepairAISpeech(ctx, agent, basePrompt, transcript)
			if err != nil {
				fail(err.Error())
				return
			}
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
	visible, fixed, err := rt.generateAndApplyRepairPatch(ctx, &job, system, prompt, source, apiDoc, allRoundsContext, transcript)
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

func (rt *Router) generateAndApplyRepairPatch(ctx context.Context, job *domain.RepairAIJob, system, prompt, source, apiDoc, allRoundsContext, transcript string) (string, string, error) {
	if fixedVisible, fixedHTML, err := rt.generateRepairWithToolLoop(ctx, job, system, prompt, source, apiDoc, allRoundsContext, transcript); err == nil {
		return fixedVisible, fixedHTML, nil
	} else {
		rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", "工具循环没有完成修复，降级为兼容补丁模式："+err.Error())
	}
	var lastPatch string
	var lastErr error
	var visible string
	const maxPatchAttempts = 8
	for attempt := 1; ; attempt++ {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		if attempt > 1 {
			rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", fmt.Sprintf("第 %d 次重写补丁。上一次失败原因：%s", attempt, lastErr.Error()))
		}
		nextPrompt := prompt
		if lastErr != nil {
			nextPrompt = prompt + "\n\n上一份补丁摘要：\n" + trimLimit(lastPatch, 5000) + "\n\n后端应用补丁失败，错误是：\n" + lastErr.Error() + "\n\n请重新输出 JSON，只输出 visible_message 和 patch。新 patch 必须使用当前入口 HTML 中可以精确匹配的原文；每一行必须是 apply_patch 兼容格式；每个修改块至少保留 2 行上下文。"
			if attempt >= maxPatchAttempts {
				nextPrompt = prompt + "\n\n前面多次补丁都没能成功应用。请改为直接输出完整修复后的 HTML，只输出 JSON，字段为 visible_message 和 fixed_html。不要再输出 patch。"
			}
		}
		activeSystem := system
		if attempt >= maxPatchAttempts {
			activeSystem = `你是 PlayPage 网页急救圆桌的最终修复员。前面多次局部补丁都无法应用。
现在请直接输出一个 JSON 对象，不要 Markdown，不要代码块。
JSON 字段：
- visible_message：中文，给用户看的简短圆桌总结。
- fixed_html：完整修复后的 HTML。

硬性要求：
- fixed_html 必须是完整 HTML 页面。
- 保留原作品的主要功能、样式、互动 API key 和相对 API 地址。
- 不要删除用户数据，不要把 API 地址改成其他域名。
- 不要引入需要构建的框架。`
		}
		content, err := rt.aiClient.CompletePlain(ctx, []service.AIMessage{{Role: "system", Content: activeSystem}, {Role: "user", Content: nextPrompt}}, 6000)
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
		if attempt >= maxPatchAttempts {
			patchVisible, fixedHTML, err := parseRepairAIResult(content)
			if err == nil {
				if patchVisible != "" {
					visible = patchVisible
				}
				return visible, ensureRepairHTML(fixedHTML), nil
			}
		}
		patchVisible, patchText, err := parseRepairAIPatchResult(content)
		if err != nil {
			lastPatch = content
			lastErr = fmt.Errorf("AI 返回的补丁格式不正确：%w", err)
			rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", fmt.Sprintf("第 %d 次补丁解析失败：%s", attempt, summarizePatchFailure(content, lastErr.Error())))
			continue
		}
		if patchVisible != "" {
			visible = patchVisible
		}
		lastPatch = patchText
		fixed, err := applyRepairAIPatch(source, patchText)
		if err != nil {
			lastErr = err
			rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", fmt.Sprintf("第 %d 次补丁应用失败：%s", attempt, summarizePatchFailure(patchText, err.Error())))
			continue
		}
		return visible, fixed, nil
	}
}

func (rt *Router) generateRepairWithToolLoop(ctx context.Context, job *domain.RepairAIJob, system, prompt, source, apiDoc, allRoundsContext, transcript string) (string, string, error) {
	const maxToolTurns = 10
	current := source
	visible := ""
	tools := append(repairReadTools(), repairApplyPatchTool())
	messages := []service.AIMessage{
		{Role: "system", Content: system + "\n\n你现在拥有工具。硬性流程：\n1. 必须先调用 read_current_html，至少读取 summary；涉及具体函数时必须用 search 查目标函数或关键词。\n2. 如果作品有互动功能，必须调用 read_interactive_api_doc。\n3. 如果当前不是第 1 轮，必须调用 read_all_rounds_context。这个工具只返回当前修复申请内的历史轮次，不包含同一作品其他申请。你要理解此前每一轮做过什么、失败在哪里、用户最新反馈是什么，然后继续迭代，不能从头乱改。\n4. 看完工具返回后，才能调用 apply_patch 修改当前 HTML。\n5. apply_patch 返回失败时，必须根据工具返回的错误重新读取相关 HTML 片段，再重新生成补丁。\n6. 补丁成功后，再输出 JSON：{\"visible_message\":\"...\",\"fixed_html\":\"...\"}。\n7. 不要凭记忆改代码，不要假装工具成功。"},
		{Role: "user", Content: basePromptWithoutLargeContext(prompt) + "\n\n请按工具流程修复。"},
	}
	readHTML := false
	readHistory := false
	for turn := 1; turn <= maxToolTurns; turn++ {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		msg, finish, err := rt.aiClient.Chat(ctx, messages, tools, "auto", 8000, true)
		if err != nil {
			return "", "", err
		}
		messages = append(messages, msg)
		if len(msg.ToolCalls) == 0 {
			if !readHTML {
				messages = append(messages, service.AIMessage{Role: "user", Content: "你还没有读取当前 HTML。必须先调用 read_current_html 工具，必要时用 search 读取相关代码片段，然后再修复。"})
				continue
			}
			if job.Round > 1 && !readHistory {
				messages = append(messages, service.AIMessage{Role: "user", Content: "这是多轮圆桌。你还没有读取当前修复申请内的历史轮次。必须调用 read_all_rounds_context，理解历史失败原因和用户反馈后再继续。"})
				continue
			}
			if strings.TrimSpace(msg.Content) == "" && finish == "tool_calls" {
				return "", "", fmt.Errorf("AI 要求调用工具，但没有返回工具参数")
			}
			v, htmlText, err := parseRepairAIResult(msg.Content)
			if err == nil {
				if v != "" {
					visible = v
				}
				return visible, ensureRepairHTML(htmlText), nil
			}
			return "", "", fmt.Errorf("AI 没有调用 apply_patch，也没有返回完整 HTML：%w", err)
		}
		for _, call := range msg.ToolCalls {
			switch call.Function.Name {
			case "read_current_html":
				readHTML = true
			case "read_all_rounds_context", "read_previous_context":
				readHistory = true
			}
			if call.Function.Name != "apply_patch" {
				result := executeRepairReadTool(call.Function.Name, call.Function.Arguments, current, apiDoc, allRoundsContext, transcript)
				messages = append(messages, service.AIMessage{Role: "tool", ToolCallID: call.ID, Content: result})
				continue
			}
			if !readHTML {
				messages = append(messages, service.AIMessage{Role: "tool", ToolCallID: call.ID, Content: `{"ok":false,"error":"调用 apply_patch 前必须先调用 read_current_html 读取当前 HTML。"}`})
				continue
			}
			if job.Round > 1 && !readHistory {
				messages = append(messages, service.AIMessage{Role: "tool", ToolCallID: call.ID, Content: `{"ok":false,"error":"这是多轮圆桌，调用 apply_patch 前必须先调用 read_all_rounds_context 读取当前修复申请内的历史轮次。"}`})
				continue
			}
			var args struct {
				Patch string `json:"patch"`
			}
			if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil || strings.TrimSpace(args.Patch) == "" {
				messages = append(messages, service.AIMessage{Role: "tool", ToolCallID: call.ID, Content: `{"ok":false,"error":"apply_patch 参数必须是 JSON，且包含非空 patch 字符串"}`})
				continue
			}
			rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", fmt.Sprintf("修复员第 %d 轮调用 apply_patch 工具。", turn))
			next, err := applyRepairAIPatch(current, args.Patch)
			if err != nil {
				result := map[string]any{
					"ok":              false,
					"error":           err.Error(),
					"patch_summary":   trimLimit(args.Patch, 1200),
					"current_hint":    "请基于当前入口 HTML 中真实存在的原文重新生成补丁。不要复用失败补丁。",
					"supported_files": []string{"index.html"},
				}
				data, _ := json.Marshal(result)
				messages = append(messages, service.AIMessage{Role: "tool", ToolCallID: call.ID, Content: string(data)})
				rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", "apply_patch 工具返回失败："+err.Error())
				continue
			}
			current = next
			result := map[string]any{
				"ok":           true,
				"message":      "补丁已应用到当前 HTML。可以继续调用 apply_patch，或输出最终 JSON。",
				"current_html": trimMiddle(current, 30000),
			}
			data, _ := json.Marshal(result)
			messages = append(messages, service.AIMessage{Role: "tool", ToolCallID: call.ID, Content: string(data)})
			rt.addAIMessage(ctx, job, "fixer", "修复员", "assistant", "status", "apply_patch 工具已成功应用补丁。")
		}
	}
	if current != source {
		return "AI 已通过 apply_patch 工具完成局部修复，请预览效果。", current, nil
	}
	return "", "", fmt.Errorf("AI 多轮工具调用后仍未产生可用修改")
}

func repairApplyPatchTool() service.AITool {
	return service.AITool{
		Type: "function",
		Function: service.AIToolFunction{
			Name:        "apply_patch",
			Description: "Apply an apply_patch style patch to the current index.html. Use this for every code change instead of describing changes in prose.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"patch": map[string]any{
						"type":        "string",
						"description": "Patch text including *** Begin Patch and *** End Patch. Only update index.html.",
					},
				},
				"required": []string{"patch"},
			},
		},
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

func (rt *Router) generateRepairAISpeechWithTools(ctx context.Context, agent repairAIAgent, basePrompt, source, apiDoc, allRoundsContext, transcript string, round int) (string, error) {
	tools := repairReadTools()
	system := fmt.Sprintf(`你是 PlayPage 网页急救圆桌里的「%s」。%s
你可以调用阅读工具获取当前 HTML、互动 API 文档、当前修复申请内的所有历史轮次上下文和前面角色发言。
硬性流程：
1. 必须先调用 read_current_html，至少读取 summary。
2. 如果你要评价某个函数、API 调用或按钮事件，必须用 read_current_html 的 search 模式读取相关片段。
3. 互动 API 守门员必须调用 read_interactive_api_doc；其他角色发现涉及云存档/评论/排行榜也必须调用。
4. 挑刺检查员必须调用 read_roundtable_transcript。
5. 有历史轮次时必须调用 read_all_rounds_context；它只返回当前修复申请内的历史圆桌，不包含同一作品其他修复申请。read_previous_context 只是兼容别名，也返回当前修复申请内全部历史轮次。
6. 工具读完后，只输出你这一位角色的一段中文发言，不要 JSON，不要 Markdown 表格，不要代码块。
不要假装看过没读的内容，不要编造代码细节，不要声称已经修改文件。
要求：
- 80 到 220 字。
- 说人话，用户能看懂。
- 可以有一点打工人式幽默，但不要低俗，不要嘲笑用户。
- 必须围绕这次作品修复，说清楚你发现了什么、建议怎么改。
- 你现在只是圆桌发言。`, agent.Name, agent.Persona)
	messages := []service.AIMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: basePromptWithoutLargeContext(basePrompt) + fmt.Sprintf("\n\n你现在是第 %d 轮圆桌角色。请先调用工具读取必要上下文，然后轮到你发言。", round)},
	}
	readHTML := false
	readHistory := false
	readAPIDoc := false
	readTranscript := false
	for turn := 1; turn <= 6; turn++ {
		msg, _, err := rt.aiClient.Chat(ctx, messages, tools, "auto", 1200, true)
		if err != nil {
			return "", err
		}
		messages = append(messages, msg)
		if len(msg.ToolCalls) == 0 {
			if !readHTML {
				messages = append(messages, service.AIMessage{Role: "user", Content: "你还没有调用任何阅读工具。请先调用 read_current_html，必要时再调用其他阅读工具，然后再发言。"})
				continue
			}
			if round > 1 && !readHistory {
				messages = append(messages, service.AIMessage{Role: "user", Content: "这是多轮圆桌。请先调用 read_all_rounds_context，读取当前修复申请内的历史轮次和用户反馈，然后再发言。"})
				continue
			}
			if agent.Key == "api" && !readAPIDoc {
				messages = append(messages, service.AIMessage{Role: "user", Content: "你是互动 API 守门员。必须先调用 read_interactive_api_doc，再判断互动 API 有没有问题。"})
				continue
			}
			if agent.Key == "critic" && !readTranscript {
				messages = append(messages, service.AIMessage{Role: "user", Content: "你是挑刺检查员。必须先调用 read_roundtable_transcript，读取前面角色发言后再挑刺。"})
				continue
			}
			content := strings.TrimSpace(msg.Content)
			if content == "" {
				return "", fmt.Errorf("AI 没有返回发言")
			}
			return content, nil
		}
		for _, call := range msg.ToolCalls {
			switch call.Function.Name {
			case "read_current_html":
				readHTML = true
			case "read_all_rounds_context", "read_previous_context":
				readHistory = true
			case "read_interactive_api_doc":
				readAPIDoc = true
			case "read_roundtable_transcript":
				readTranscript = true
			}
			result := executeRepairReadTool(call.Function.Name, call.Function.Arguments, source, apiDoc, allRoundsContext, transcript)
			messages = append(messages, service.AIMessage{Role: "tool", ToolCallID: call.ID, Content: result})
		}
	}
	return "", fmt.Errorf("AI 多次调用阅读工具后没有发言")
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

func repairReadTools() []service.AITool {
	stringParam := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	return []service.AITool{
		{
			Type: "function",
			Function: service.AIToolFunction{
				Name:        "read_current_html",
				Description: "Read the current entry HTML. Use mode=summary for overview, mode=search with query to locate code, or mode=full when the file is small enough.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"mode":  stringParam("summary, search, or full"),
						"query": stringParam("Keyword used when mode=search."),
					},
					"required": []string{"mode"},
				},
			},
		},
		{
			Type: "function",
			Function: service.AIToolFunction{
				Name:        "read_interactive_api_doc",
				Description: "Read the PlayPage interactive API document for this project.",
				Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
		{
			Type: "function",
			Function: service.AIToolFunction{
				Name:        "read_all_rounds_context",
				Description: "Read all historical repair round contexts and user feedback for this repair request only. It does not include other repair requests of the same project.",
				Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
		{
			Type: "function",
			Function: service.AIToolFunction{
				Name:        "read_previous_context",
				Description: "Compatibility alias for read_all_rounds_context.",
				Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
		{
			Type: "function",
			Function: service.AIToolFunction{
				Name:        "read_roundtable_transcript",
				Description: "Read previous agents' discussion in this round.",
				Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
	}
}

func executeRepairReadTool(name, rawArgs, source, apiDoc, allRoundsContext, transcript string) string {
	switch name {
	case "read_current_html":
		var args struct {
			Mode  string `json:"mode"`
			Query string `json:"query"`
		}
		_ = json.Unmarshal([]byte(rawArgs), &args)
		mode := strings.ToLower(strings.TrimSpace(args.Mode))
		switch mode {
		case "full":
			return repairToolJSON(map[string]any{"ok": true, "content": trimMiddle(source, maxRepairPromptHTMLChars)})
		case "search":
			return repairToolJSON(map[string]any{"ok": true, "content": searchRepairHTML(source, args.Query)})
		default:
			return repairToolJSON(map[string]any{"ok": true, "content": summarizeRepairHTML(source)})
		}
	case "read_interactive_api_doc":
		return repairToolJSON(map[string]any{"ok": true, "content": trimMiddle(apiDoc, 60000)})
	case "read_all_rounds_context":
		return repairToolJSON(map[string]any{"ok": true, "content": trimLimit(allRoundsContext, 40000)})
	case "read_previous_context":
		return repairToolJSON(map[string]any{"ok": true, "content": trimLimit(allRoundsContext, 40000)})
	case "read_roundtable_transcript":
		return repairToolJSON(map[string]any{"ok": true, "content": trimLimit(transcript, 12000)})
	default:
		return repairToolJSON(map[string]any{"ok": false, "error": "未知阅读工具"})
	}
}

func repairToolJSON(value map[string]any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func summarizeRepairHTML(source string) string {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	var hits []string
	keywords := []string{"function ", "const ", "let ", "var ", "fetch(", "addEventListener", "onclick", "alert(", "API_BASE", "X-Project-Key"}
	for i, line := range lines {
		for _, keyword := range keywords {
			if strings.Contains(line, keyword) {
				hits = append(hits, fmt.Sprintf("%d: %s", i+1, strings.TrimSpace(line)))
				break
			}
		}
		if len(hits) >= 120 {
			break
		}
	}
	return trimLimit("HTML 总长度："+fmt.Sprint(len(source))+" 字符\n关键行：\n"+strings.Join(hits, "\n"), 20000)
}

func searchRepairHTML(source, query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return summarizeRepairHTML(source)
	}
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	var chunks []string
	lowerQuery := strings.ToLower(query)
	for i, line := range lines {
		if !strings.Contains(strings.ToLower(line), lowerQuery) {
			continue
		}
		start := i - 8
		if start < 0 {
			start = 0
		}
		end := i + 9
		if end > len(lines) {
			end = len(lines)
		}
		var chunk []string
		for j := start; j < end; j++ {
			chunk = append(chunk, fmt.Sprintf("%d: %s", j+1, lines[j]))
		}
		chunks = append(chunks, strings.Join(chunk, "\n"))
		if len(chunks) >= 5 {
			break
		}
	}
	if len(chunks) == 0 {
		return "没有找到关键词：" + query
	}
	return trimLimit(strings.Join(chunks, "\n\n---\n\n"), 30000)
}

func basePromptWithoutLargeContext(basePrompt string) string {
	if idx := strings.Index(basePrompt, "\n\n互动 API 文档："); idx >= 0 {
		return basePrompt[:idx] + "\n\n大段 HTML、API 文档和当前修复申请内所有历史轮次上下文请通过工具读取。"
	}
	return trimLimit(basePrompt, 4000)
}

func (rt *Router) buildInteractiveDocForPrompt(ctx context.Context, project domain.Project) string {
	access, found, err := rt.store.GetProjectPublicAccess(ctx, project.ID)
	if err != nil || !found {
		return "互动 API 已开启，但读取密钥失败。"
	}
	cols, _ := rt.store.ListCollections(ctx, project.ID)
	return buildInteractiveAPIDoc(project, access.PublicKey, cols)
}

func (rt *Router) allRepairAIRoundContext(ctx context.Context, requestID, excludeJobID string) string {
	jobs, err := rt.store.ListRepairAIJobs(ctx, requestID)
	if err != nil {
		log.Printf("list repair ai jobs failed: request_id=%s err=%v", requestID, err)
		return "历史圆桌上下文读取失败。"
	}
	var b strings.Builder
	b.WriteString("以下内容只来自当前修复申请 ID=" + requestID + " 的历史圆桌，不包含同一作品的其他修复申请。\n\n")
	written := 0
	for _, job := range jobs {
		if job.ID == excludeJobID {
			continue
		}
		written++
		b.WriteString(fmt.Sprintf("## 历史第 %d 轮\n", job.Round))
		b.WriteString("状态：" + job.Status + "\n")
		if job.Feedback != "" {
			b.WriteString("该轮用户反馈/补充：" + trimLimit(job.Feedback, 2500) + "\n")
		}
		if job.ErrorMessage != "" {
			b.WriteString("该轮错误信息：" + trimLimit(job.ErrorMessage, 1200) + "\n")
		}
		if job.PreviewURL != "" {
			b.WriteString("该轮预览地址：" + job.PreviewURL + "\n")
		}
		messages, err := rt.store.ListRepairAIMessages(ctx, job.ID)
		if err != nil {
			b.WriteString("该轮消息读取失败。\n\n")
			continue
		}
		b.WriteString("该轮圆桌消息：\n")
		for _, m := range messages {
			if strings.TrimSpace(m.Content) == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("- %s/%s：%s\n", m.AgentName, m.MessageType, trimLimit(m.Content, 1200)))
		}
		if job.GeneratedHTML != "" {
			b.WriteString("该轮曾生成修复 HTML。后续轮次应理解其方向，但以当前发布版 HTML 为准继续修改。\n")
			b.WriteString("该轮生成 HTML 摘要：\n")
			b.WriteString(trimLimit(summarizeRepairHTML(job.GeneratedHTML), 6000))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if written == 0 {
		return "暂无历史圆桌轮次。"
	}
	return trimLimit(b.String(), 60000)
}

func (rt *Router) previousRepairAIMessages(ctx context.Context, requestID, excludeJobID string) string {
	return rt.allRepairAIRoundContext(ctx, requestID, excludeJobID)
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
		fixed, ok := applyRepairPatchHunkByAnchor(source, oldLines, newLines)
		if ok {
			return fixed, nil
		}
		return "", fmt.Errorf("补丁原文匹配次数不是 1，实际为 %d；请增加上下文或改用更精确的原文", strings.Count(source, oldText))
	}
	return strings.Replace(source, oldText, newText, 1), nil
}

func applyRepairPatchHunkByAnchor(source string, oldLines, newLines []string) (string, bool) {
	if len(oldLines) == 0 {
		return "", false
	}
	sourceLines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	if len(oldLines) > len(sourceLines) {
		return "", false
	}
	oldBody := normalizePatchBlockWithBlankLines(oldLines)
	matchStart := -1
	matches := 0
	for i := 0; i+len(oldLines) <= len(sourceLines); i++ {
		window := sourceLines[i : i+len(oldLines)]
		if normalizePatchBlockWithBlankLines(window) != oldBody {
			continue
		}
		matchStart = i
		matches++
		if matches > 1 {
			return "", false
		}
	}
	if matches == 1 {
		anchorStart := matchStart
		anchorEnd := matchStart + len(oldLines) - 1
		updated := append([]string{}, sourceLines[:anchorStart]...)
		updated = append(updated, newLines...)
		updated = append(updated, sourceLines[anchorEnd+1:]...)
		return strings.Join(updated, "\n"), true
	}
	return "", false
}

func normalizePatchLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, "\t", " ")
	line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
	return line
}

func normalizePatchBlock(lines []string) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts = append(parts, normalizePatchLine(line))
	}
	return strings.Join(parts, "\n")
}

func normalizePatchBlockWithBlankLines(lines []string) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, normalizePatchLine(line))
	}
	return strings.Join(parts, "\n")
}

func summarizePatchFailure(value, errText string) string {
	value = strings.TrimSpace(value)
	if len(value) > 800 {
		value = value[:800] + "…"
	}
	errText = strings.TrimSpace(errText)
	if len(errText) > 300 {
		errText = errText[:300] + "…"
	}
	return errText + "\n" + value
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
