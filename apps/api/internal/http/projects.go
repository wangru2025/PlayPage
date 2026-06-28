package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

const maxReleaseRequestBytes = 12 << 20

func (rt *Router) handleCreateUpgradeRequest(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if domain.IsAdminRole(user.Role) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "管理员账号不需要提交升级申请"})
		return
	}

	var input domain.UpgradeRequestCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	switch input.TargetPlan {
	case domain.PlanLight, domain.PlanSupport:
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "只能申请轻享版或支持版"})
		return
	}
	switch input.PaymentMethod {
	case "wechat", "alipay":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "支付方式只能是 wechat 或 alipay"})
		return
	}

	existing, err := rt.store.ListUserUpgradeRequests(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查已有升级申请失败"})
		return
	}
	for _, item := range existing {
		if item.Status == "pending" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "你已经有一个待审核的升级申请，请不要重复提交"})
			return
		}
	}

	request, err := rt.store.CreateUpgradeRequest(r.Context(), domain.UpgradeRequest{
		UserID:        user.ID,
		UserEmail:     user.Email,
		Username:      user.Username,
		CurrentPlan:   user.PlanCode,
		TargetPlan:    input.TargetPlan,
		PaymentMethod: input.PaymentMethod,
		PayerNote:     strings.TrimSpace(input.PayerNote),
		SystemNote:    fmt.Sprintf("用户名：%s；邮箱：%s；提交时间：%s", user.Username, user.Email, time.Now().Format("2006-01-02 15:04:05")),
		Status:        "pending",
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "你已经有一个待审核的升级申请，请不要重复提交"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "提交升级申请失败"})
		return
	}

	writeJSON(w, http.StatusCreated, request)
}

func (rt *Router) handleListMyUpgradeRequests(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}

	items, err := rt.store.ListUserUpgradeRequests(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取升级申请失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleListProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}

	items, err := rt.store.ListProjects(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品列表失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (rt *Router) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if err := rt.checkProjectCreationAllowed(user); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var input domain.ProjectCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Username = normalizePathSegment(user.Username)
	if input.Username == "" {
		input.Username = inferUsernameFromEmail(user.Email)
	}
	input.Slug = normalizePathSegment(input.Slug)
	if input.Slug == "" && input.Name != "" {
		input.Slug = normalizePathSegment(input.Name)
	}
	if input.Name == "" || input.Username == "" || input.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "作品名字和作品地址都要填"})
		return
	}

	project, err := rt.store.CreateProject(r.Context(), user.ID, input)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个作品地址你已经用过了，请换一个作品地址"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "创建作品失败"})
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (rt *Router) handleGetProject(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	_ = rt.ensureProjectLiveLink(r.Context(), project)

	writeJSON(w, http.StatusOK, map[string]any{
		"project":   project,
		"publicKey": rt.lookupProjectPublicKey(r.Context(), projectID),
		"releaseLayout": map[string]string{
			"uploads":  rt.releases.UploadsDir(projectID),
			"runtime":  rt.releases.RuntimeDir(projectID),
			"releases": rt.releases.ReleasesDir(projectID),
			"live":     rt.releases.LivePublicDir(project.Username, project.Slug),
		},
	})
}

func (rt *Router) handleListReleases(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	_ = rt.ensureProjectLiveLink(r.Context(), project)

	items, err := rt.store.ListReleases(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取发布版本失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projectId": projectID,
		"items":     items,
	})
}

func (rt *Router) handleCreateRelease(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	switch mode {
	case "", "zip":
		rt.handleCreateReleaseFromZip(w, r, project)
	case "html":
		rt.handleCreateReleaseFromHTMLFile(w, r, project)
	case "text":
		rt.handleCreateReleaseFromHTMLText(w, r, project)
	case "template":
		rt.handleCreateReleaseFromTemplate(w, r, project)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "上传类型不正确"})
	}
}

func (rt *Router) handleCreateReleaseFromZip(w http.ResponseWriter, r *http.Request, project domain.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReleaseRequestBytes)
	if err := r.ParseMultipartForm(1200 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "上传表单无效"})
		return
	}

	upload, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少上传文件"})
		return
	}
	defer upload.Close()

	changeNote := trimLimit(r.FormValue("changeNote"), 500)
	release, err := rt.pub.PublishZip(r.Context(), project, upload, header.Filename, changeNote)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, release)
}

func (rt *Router) handleCreateReleaseFromHTMLFile(w http.ResponseWriter, r *http.Request, project domain.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReleaseRequestBytes)
	if err := r.ParseMultipartForm(1200 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "上传表单无效"})
		return
	}

	upload, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少上传文件"})
		return
	}
	defer upload.Close()

	body, err := io.ReadAll(upload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "读取上传文件失败"})
		return
	}

	changeNote := trimLimit(r.FormValue("changeNote"), 500)
	release, err := rt.pub.PublishSingleHTML(r.Context(), project, header.Filename, body, changeNote)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, release)
}

func (rt *Router) handleCreateReleaseFromHTMLText(w http.ResponseWriter, r *http.Request, project domain.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReleaseRequestBytes)
	var input struct {
		HTML       string `json:"html"`
		ChangeNote string `json:"changeNote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if strings.TrimSpace(input.HTML) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "HTML 内容不能为空"})
		return
	}

	release, err := rt.pub.PublishHTMLText(r.Context(), project, input.HTML, trimLimit(input.ChangeNote, 500))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, release)
}

func (rt *Router) handleCreateReleaseFromTemplate(w http.ResponseWriter, r *http.Request, project domain.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReleaseRequestBytes)
	var input domain.TemplateCreateReleaseInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.TemplateID = strings.TrimSpace(input.TemplateID)
	if input.TemplateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先选择模板"})
		return
	}
	tpl, ok, err := rt.findTemplate(r.Context(), input.TemplateID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取模板失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个模板"})
		return
	}
	if tpl.InteractiveRequired && !project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个模板需要先开启互动功能"})
		return
	}
	if input.Params == nil {
		input.Params = map[string]string{}
	}
	for _, field := range tpl.ConfigFields {
		if field.Required && strings.TrimSpace(input.Params[field.Name]) == "" && strings.TrimSpace(field.Default) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写模板参数：" + field.Label})
			return
		}
	}

	for _, collection := range tpl.Collections {
		if _, found, err := rt.store.GetCollectionByName(r.Context(), project.ID, collection.Name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查模板数据表失败"})
			return
		} else if !found {
			if _, err := rt.store.CreateCollection(r.Context(), project.ID, domain.CollectionCreateInput{
				Name:        collection.Name,
				Permissions: collection.Permissions,
				Fields:      collection.Fields,
			}); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "创建模板数据表失败：" + err.Error()})
				return
			}
		}
	}

	htmlBody := renderTemplateHTML(tpl, project, rt.lookupProjectPublicKey(r.Context(), project.ID), input.Params, false)
	changeNote := trimLimit(input.ChangeNote, 500)
	if changeNote == "" {
		changeNote = "从模板创建：" + tpl.Name
	}
	release, err := rt.pub.PublishHTMLText(r.Context(), project, htmlBody, changeNote)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, release)
}

func (rt *Router) handleRollbackRelease(w http.ResponseWriter, r *http.Request, projectID, releaseID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	if releaseID == "" || releaseID == project.CurrentRelease {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请选择一个历史版本"})
		return
	}

	releases, err := rt.store.ListReleases(r.Context(), project.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取历史版本失败"})
		return
	}
	var target domain.Release
	found := false
	for _, release := range releases {
		if release.ID == releaseID {
			target = release
			found = true
			break
		}
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个历史版本"})
		return
	}
	if err := rt.pub.EnsureLivePublicLink(project, target.PublicPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "切换历史版本失败"})
		return
	}
	if err := rt.store.SetCurrentRelease(r.Context(), project.ID, target.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存当前版本失败"})
		return
	}
	project.CurrentRelease = target.ID
	writeJSON(w, http.StatusOK, map[string]any{"project": project, "release": target})
}

func (rt *Router) handleUpdateProjectVisibility(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	var input domain.ProjectVisibilityUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	switch input.Visibility {
	case "public", "unlisted":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "可见范围只能是 public 或 unlisted"})
		return
	}

	if input.Visibility == "public" && project.CurrentRelease == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先上传作品内容，再公开到广场"})
		return
	}

	updated, found, err := rt.store.UpdateProjectVisibility(r.Context(), user.ID, projectID, input.Visibility)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "更新公开状态失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (rt *Router) handleUpdateProjectPath(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	var input domain.ProjectPathUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Slug = normalizePathSegment(input.Slug)
	if input.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "作品地址不能为空"})
		return
	}

	updated, found, err := rt.store.UpdateProjectPath(r.Context(), user.ID, projectID, input.Slug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存作品地址失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}

	if project.CurrentRelease != "" {
		oldLive := rt.releases.LivePublicDir(project.Username, project.Slug)
		newLive := rt.releases.LivePublicDir(updated.Username, updated.Slug)
		if oldLive != newLive {
			_ = os.Remove(oldLive)
			_ = os.RemoveAll(oldLive)
		}

		releases, err := rt.store.ListReleases(r.Context(), projectID)
		if err == nil {
			for _, release := range releases {
				if release.ID == updated.CurrentRelease {
					_ = rt.pub.EnsureLivePublicLink(updated, release.PublicPath)
					break
				}
			}
		}

		oldParent := filepath.Dir(oldLive)
		entries, err := os.ReadDir(oldParent)
		if err == nil && len(entries) == 0 {
			_ = os.Remove(oldParent)
		}
	}

	writeJSON(w, http.StatusOK, updated)
}

func (rt *Router) handleUpdateProjectSettings(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	var input domain.ProjectSettingsUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = normalizePathSegment(input.Slug)
	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "作品名字不能为空"})
		return
	}
	if input.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "作品链接名不能为空"})
		return
	}

	updated, found, err := rt.store.UpdateProjectSettings(r.Context(), user.ID, projectID, input)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个作品链接名你已经用过了，请换一个"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存作品设置失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}

	if project.CurrentRelease != "" && project.Slug != updated.Slug {
		oldLive := rt.releases.LivePublicDir(project.Username, project.Slug)
		newLive := rt.releases.LivePublicDir(updated.Username, updated.Slug)
		if oldLive != newLive {
			_ = os.Remove(oldLive)
			_ = os.RemoveAll(oldLive)
		}

		releases, err := rt.store.ListReleases(r.Context(), projectID)
		if err == nil {
			for _, release := range releases {
				if release.ID == updated.CurrentRelease {
					_ = rt.pub.EnsureLivePublicLink(updated, release.PublicPath)
					break
				}
			}
		}

		oldParent := filepath.Dir(oldLive)
		entries, err := os.ReadDir(oldParent)
		if err == nil && len(entries) == 0 {
			_ = os.Remove(oldParent)
		}
	}

	writeJSON(w, http.StatusOK, updated)
}

func (rt *Router) ensureProjectLiveLink(ctx context.Context, project domain.Project) error {
	if project.CurrentRelease == "" {
		return nil
	}

	liveDir := rt.releases.LivePublicDir(project.Username, project.Slug)
	if _, err := os.Lstat(liveDir); err == nil {
		return nil
	}

	releases, err := rt.store.ListReleases(ctx, project.ID)
	if err != nil {
		return err
	}
	for _, release := range releases {
		if release.ID == project.CurrentRelease {
			return rt.pub.EnsureLivePublicLink(project, release.PublicPath)
		}
	}

	return nil
}

func (rt *Router) lookupProjectPublicKey(ctx context.Context, projectID string) string {
	access, ok, err := rt.store.GetProjectPublicAccess(ctx, projectID)
	if err != nil || !ok {
		return ""
	}
	return access.PublicKey
}
