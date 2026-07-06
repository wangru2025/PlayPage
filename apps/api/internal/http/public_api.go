package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
)

func (rt *Router) handlePublicProjectRoutes(w http.ResponseWriter, r *http.Request) {
	applyPublicAPIHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/public/projects/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}

	projectID := parts[0]
	if len(parts) == 2 && parts[1] == "visit" && r.Method == http.MethodPost {
		rt.handlePublicTrackVisit(w, r, projectID)
		return
	}
	if len(parts) == 2 && parts[1] == "open" && r.Method == http.MethodGet {
		rt.handlePublicOpenProject(w, r, projectID)
		return
	}
	if len(parts) == 2 && parts[1] == "profile" && r.Method == http.MethodGet {
		rt.handlePublicProjectProfile(w, r, projectID)
		return
	}

	recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	defer func() {
		rt.trackPublicAPIRequest(r.Context(), projectID, recorder.status)
	}()
	w = recorder

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		rt.handlePublicProjectInfo(w, r, projectID)
	case len(parts) == 2 && parts[1] == "collections" && r.Method == http.MethodGet:
		rt.handlePublicListCollections(w, r, projectID)
	case len(parts) == 2 && parts[1] == "collections" && r.Method == http.MethodPost:
		rt.handlePublicCreateCollection(w, r, projectID)
	case len(parts) == 3 && parts[1] == "auth" && (parts[2] == "email-code" || parts[2] == "send-code") && r.Method == http.MethodPost:
		rt.handlePublicSendEmailCode(w, r, projectID)
	case len(parts) == 4 && parts[1] == "auth" && parts[2] == "email-code" && parts[3] == "send" && r.Method == http.MethodPost:
		rt.handlePublicSendEmailCode(w, r, projectID)
	case len(parts) == 3 && parts[1] == "auth" && parts[2] == "verify-code" && r.Method == http.MethodPost:
		rt.handlePublicVerifyEmailCode(w, r, projectID)
	case len(parts) == 4 && parts[1] == "auth" && parts[2] == "email-code" && parts[3] == "verify" && r.Method == http.MethodPost:
		rt.handlePublicVerifyEmailCode(w, r, projectID)
	case len(parts) == 3 && parts[1] == "collections" && r.Method == http.MethodGet:
		rt.handlePublicGetCollection(w, r, projectID, parts[2])
	case len(parts) == 3 && parts[1] == "collections" && r.Method == http.MethodDelete:
		rt.handlePublicDeleteCollection(w, r, projectID, parts[2])
	case len(parts) == 4 && parts[1] == "collections" && parts[3] == "records" && r.Method == http.MethodGet:
		rt.handlePublicListRecords(w, r, projectID, parts[2])
	case len(parts) == 4 && parts[1] == "collections" && parts[3] == "records" && r.Method == http.MethodPost:
		rt.handlePublicCreateRecord(w, r, projectID, parts[2])
	case len(parts) == 5 && parts[1] == "collections" && parts[3] == "records" && r.Method == http.MethodGet:
		rt.handlePublicGetRecord(w, r, projectID, parts[2], parts[4])
	case len(parts) == 5 && parts[1] == "collections" && parts[3] == "records" && (r.Method == http.MethodPut || r.Method == http.MethodPatch):
		rt.handlePublicUpdateRecord(w, r, projectID, parts[2], parts[4])
	case len(parts) == 5 && parts[1] == "collections" && parts[3] == "records" && r.Method == http.MethodDelete:
		rt.handlePublicDeleteRecord(w, r, projectID, parts[2], parts[4])
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
	}
}

func (rt *Router) handlePublicProjectProfile(w http.ResponseWriter, r *http.Request, projectID string) {
	project, found, err := rt.store.GetPublicProject(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "作品不存在或没有公开"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project})
}

func (rt *Router) handlePublicOpenProject(w http.ResponseWriter, r *http.Request, projectID string) {
	project, found, err := rt.store.GetPublicProject(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "原作品不存在或没有公开"})
		return
	}
	http.Redirect(w, r, project.PublicURL, http.StatusFound)
}

func (rt *Router) handlePublicProjectInfo(w http.ResponseWriter, r *http.Request, projectID string) {
	access, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"project": map[string]any{
			"id":          access.Project.ID,
			"name":        access.Project.Name,
			"username":    access.Project.Username,
			"slug":        access.Project.Slug,
			"interactive": access.Project.Interactive,
			"publicUrl":   access.Project.PublicURL,
		},
	})
}

func (rt *Router) handlePublicSendEmailCode(w http.ResponseWriter, r *http.Request, projectID string) {
	access, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return
	}
	if !access.Project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品没有启用互动功能"})
		return
	}

	var input domain.ProjectEmailCodeSendInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	email := normalizeProjectEmail(input.Email)
	purpose := normalizeProjectEmailPurpose(input.Purpose)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先填写邮箱"})
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱地址格式不对"})
		return
	}
	if purpose == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "验证码用途不正确"})
		return
	}

	code, err := generateDigits(6)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "生成验证码失败"})
		return
	}
	now := time.Now().UTC()
	limit := domain.DailyEmailCodeLimit(access.OwnerPlan, access.OwnerRole)
	used, err := rt.store.CreateProjectEmailCode(r.Context(), access.Project.ID, access.OwnerUserID, email, purpose, code, now.Add(10*time.Minute), now, limit)
	if err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if err := rt.mailer.SendProjectCode(email, access.Project.Name, projectEmailPurposeLabel(purpose), code); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "发送验证邮件失败"})
		return
	}

	response := map[string]any{
		"ok":        true,
		"status":    "code-sent",
		"message":   "验证码已发送，请查看邮箱。",
		"email":     email,
		"purpose":   purpose,
		"expiresIn": 600,
		"quota": map[string]any{
			"used":  used,
			"limit": limit,
		},
	}
	if rt.mailer.IsNoop() {
		response["devCode"] = code
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (rt *Router) handlePublicVerifyEmailCode(w http.ResponseWriter, r *http.Request, projectID string) {
	access, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return
	}
	if !access.Project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品没有启用互动功能"})
		return
	}

	var input domain.ProjectEmailCodeVerifyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	email := normalizeProjectEmail(input.Email)
	purpose := normalizeProjectEmailPurpose(input.Purpose)
	code := strings.TrimSpace(input.Code)
	if email == "" || code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱和验证码都要填"})
		return
	}
	if purpose == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "验证码用途不正确"})
		return
	}

	verified, err := rt.store.ConsumeProjectEmailCode(r.Context(), access.Project.ID, email, purpose, code, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "校验验证码失败"})
		return
	}
	if !verified {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "验证码不对，或者已经过期"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"verified": true,
		"email":    email,
		"purpose":  purpose,
	})
}

func normalizeProjectEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func normalizeProjectEmailPurpose(purpose string) string {
	purpose = strings.TrimSpace(strings.ToLower(purpose))
	switch purpose {
	case "", "login":
		return "login"
	case "register", "reset", "bind", "custom":
		return purpose
	default:
		return ""
	}
}

func projectEmailPurposeLabel(purpose string) string {
	switch purpose {
	case "register":
		return "注册"
	case "reset":
		return "重置密码"
	case "bind":
		return "绑定或更换邮箱"
	case "custom":
		return "邮箱验证"
	default:
		return "登录"
	}
}

func (rt *Router) handlePublicListCollections(w http.ResponseWriter, r *http.Request, projectID string) {
	access, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 1, 0); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !access.Project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品没有启用互动功能"})
		return
	}

	items, err := rt.store.ListCollections(r.Context(), access.Project.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品数据表失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projectId": access.Project.ID,
		"items":     items,
	})
	rt.addProjectUsage(access.Project.ID, 1, 0)
}

func (rt *Router) handlePublicGetCollection(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	access, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 1, 0); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !access.Project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品没有启用互动功能"})
		return
	}

	collection, found, err := rt.store.GetCollectionByName(r.Context(), access.Project.ID, collectionName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品数据表失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品数据表"})
		return
	}

	writeJSON(w, http.StatusOK, collection)
	rt.addProjectUsage(access.Project.ID, 1, 0)
}

func (rt *Router) handlePublicCreateCollection(w http.ResponseWriter, r *http.Request, projectID string) {
	access, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 0, 1); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !access.Project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品没有启用互动功能"})
		return
	}

	var input domain.CollectionCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Permissions = normalizePublicPermissions(input.Permissions)
	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "作品数据表名字不能为空"})
		return
	}

	collection, err := rt.store.CreateCollection(r.Context(), access.Project.ID, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, collection)
	rt.addProjectUsage(access.Project.ID, 0, 1)
}

func (rt *Router) handlePublicDeleteCollection(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	_, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return
	}

	writeJSON(w, http.StatusForbidden, map[string]string{"error": "公开接口不允许删除作品数据表。"})
}

func (rt *Router) handlePublicListRecords(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	access, collection, ok := rt.requirePublicCollection(w, r, projectID, collectionName)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 1, 0); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !collection.Permissions.PublicRead {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "这个作品数据表不允许公开读取"})
		return
	}

	items, err := rt.store.ListRecords(r.Context(), access.Project.ID, collectionName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取记录失败"})
		return
	}
	query, err := parseRecordQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	items = applyRecordQuery(items, query)

	writeJSON(w, http.StatusOK, map[string]any{
		"projectId":      access.Project.ID,
		"collectionName": collection.Name,
		"items":          items,
	})
	rt.addProjectUsage(access.Project.ID, 1, 0)
}

func (rt *Router) handlePublicCreateRecord(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	access, collection, ok := rt.requirePublicCollection(w, r, projectID, collectionName)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 0, 1); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !collection.Permissions.PublicWrite {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "这个作品数据表不允许公开写入"})
		return
	}

	var input domain.RecordCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if len(input.Data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "记录内容不能为空"})
		return
	}

	record, err := rt.store.CreateRecord(r.Context(), access.Project.ID, collectionName, "", input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, record)
	rt.addProjectUsage(access.Project.ID, 0, 1)
}

func (rt *Router) handlePublicGetRecord(w http.ResponseWriter, r *http.Request, projectID, collectionName, recordID string) {
	access, collection, ok := rt.requirePublicCollection(w, r, projectID, collectionName)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 1, 0); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !collection.Permissions.PublicRead {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "这个作品数据表不允许公开读取"})
		return
	}

	record, found, err := rt.store.GetRecord(r.Context(), access.Project.ID, collectionName, recordID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取记录失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这条记录"})
		return
	}

	writeJSON(w, http.StatusOK, record)
	rt.addProjectUsage(access.Project.ID, 1, 0)
}

func (rt *Router) handlePublicUpdateRecord(w http.ResponseWriter, r *http.Request, projectID, collectionName, recordID string) {
	access, collection, ok := rt.requirePublicCollection(w, r, projectID, collectionName)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 0, 1); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !collection.Permissions.PublicWrite {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "这个作品数据表不允许公开写入"})
		return
	}

	var input domain.RecordUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if len(input.Data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "记录内容不能为空"})
		return
	}

	record, found, err := rt.store.UpdateRecord(r.Context(), access.Project.ID, collectionName, recordID, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这条记录"})
		return
	}

	writeJSON(w, http.StatusOK, record)
	rt.addProjectUsage(access.Project.ID, 0, 1)
}

func (rt *Router) handlePublicDeleteRecord(w http.ResponseWriter, r *http.Request, projectID, collectionName, recordID string) {
	access, collection, ok := rt.requirePublicCollection(w, r, projectID, collectionName)
	if !ok {
		return
	}
	if err := rt.checkProjectUsageAllowed(access, 0, 1); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	if !collection.Permissions.PublicWrite {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "这个作品数据表不允许公开写入"})
		return
	}

	found, err := rt.store.DeleteRecord(r.Context(), access.Project.ID, collectionName, recordID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这条记录"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	rt.addProjectUsage(access.Project.ID, 0, 1)
}

func (rt *Router) requireProjectKey(w http.ResponseWriter, r *http.Request, projectID string) (domain.PublicProjectAccess, bool) {
	access, found, err := rt.store.GetProjectPublicAccess(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品失败"})
		return domain.PublicProjectAccess{}, false
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return domain.PublicProjectAccess{}, false
	}

	projectKey := strings.TrimSpace(r.Header.Get("X-Project-Key"))
	if projectKey == "" {
		projectKey = strings.TrimSpace(r.URL.Query().Get("key"))
	}
	if projectKey == "" || projectKey != access.PublicKey {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "作品数据密钥不正确"})
		return domain.PublicProjectAccess{}, false
	}

	return access, true
}

func (rt *Router) requirePublicCollection(w http.ResponseWriter, r *http.Request, projectID, collectionName string) (domain.PublicProjectAccess, domain.Collection, bool) {
	access, ok := rt.requireProjectKey(w, r, projectID)
	if !ok {
		return domain.PublicProjectAccess{}, domain.Collection{}, false
	}
	if !access.Project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品没有启用互动功能"})
		return domain.PublicProjectAccess{}, domain.Collection{}, false
	}

	collection, found, err := rt.store.GetCollectionByName(r.Context(), access.Project.ID, collectionName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品数据表失败"})
		return domain.PublicProjectAccess{}, domain.Collection{}, false
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品数据表"})
		return domain.PublicProjectAccess{}, domain.Collection{}, false
	}

	return access, collection, true
}

func normalizePublicPermissions(permissions domain.PermissionSet) domain.PermissionSet {
	if !permissions.PublicWrite && (permissions.AuthWrite || permissions.OwnerWrite || permissions.OwnerDelete) {
		permissions.PublicWrite = true
	}
	permissions.AuthWrite = false
	permissions.OwnerWrite = false
	permissions.OwnerDelete = false
	return permissions
}

func applyPublicAPIHeaders(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Project-Key")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (rt *Router) handlePublicTrackVisit(w http.ResponseWriter, r *http.Request, projectID string) {
	access, found, err := rt.store.GetProjectPublicAccess(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "记录访问量失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}
	if !access.Project.AnalyticsEnabled || access.Project.CurrentRelease == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	day := time.Now().UTC().Format("2006-01-02")
	if err := rt.store.IncrementProjectDailyStats(r.Context(), projectID, day, 1, 0, 0, 0); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "记录访问量失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (rt *Router) trackPublicAPIRequest(ctx context.Context, projectID string, status int) {
	access, found, err := rt.store.GetProjectPublicAccess(ctx, projectID)
	if err != nil || !found || !access.Project.AnalyticsEnabled {
		return
	}
	success := int64(0)
	failure := int64(0)
	if status >= 200 && status < 400 {
		success = 1
	} else {
		failure = 1
	}
	day := time.Now().UTC().Format("2006-01-02")
	_ = rt.store.IncrementProjectDailyStats(context.WithoutCancel(ctx), projectID, day, 0, 1, success, failure)
}
