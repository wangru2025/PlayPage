package http

import (
	"context"
	"encoding/json"
	"net/http"
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

	writeJSON(w, http.StatusForbidden, map[string]string{"error": "\u516c\u5f00\u63a5\u53e3\u4e0d\u5141\u8bb8\u5220\u9664\u4f5c\u54c1\u6570\u636e\u8868\u3002"})
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
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "\u8fd9\u4e2a\u4f5c\u54c1\u6570\u636e\u8868\u4e0d\u5141\u8bb8\u516c\u5f00\u8bfb\u53d6"})
		return
	}

	items, err := rt.store.ListRecords(r.Context(), access.Project.ID, collectionName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取记录失败"})
		return
	}

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
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "\u8fd9\u4e2a\u4f5c\u54c1\u6570\u636e\u8868\u4e0d\u5141\u8bb8\u516c\u5f00\u5199\u5165"})
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
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "\u8fd9\u4e2a\u4f5c\u54c1\u6570\u636e\u8868\u4e0d\u5141\u8bb8\u516c\u5f00\u8bfb\u53d6"})
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
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "\u8fd9\u4e2a\u4f5c\u54c1\u6570\u636e\u8868\u4e0d\u5141\u8bb8\u516c\u5f00\u5199\u5165"})
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
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "\u8fd9\u4e2a\u4f5c\u54c1\u6570\u636e\u8868\u4e0d\u5141\u8bb8\u516c\u5f00\u5199\u5165"})
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
