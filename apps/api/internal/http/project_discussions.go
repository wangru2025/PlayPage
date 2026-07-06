package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
)

type projectDiscussionDetailResponse struct {
	Discussion domain.ProjectDiscussion          `json:"discussion"`
	Comments   []domain.ProjectDiscussionComment `json:"comments"`
}

func (rt *Router) ensureDiscussionProjectReadable(w http.ResponseWriter, r *http.Request, projectID string) bool {
	access, ok, err := rt.store.GetProjectPublicAccess(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品信息失败"})
		return false
	}
	if !ok || access.Project.CurrentRelease == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return false
	}
	return true
}

func (rt *Router) handleListProjectDiscussions(w http.ResponseWriter, r *http.Request, projectID string) {
	if !rt.ensureDiscussionProjectReadable(w, r, projectID) {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if status != "" && status != "open" && status != "closed" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "讨论状态不正确"})
		return
	}
	if len([]rune(query)) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "搜索关键词不能超过 100 个字"})
		return
	}
	items, err := rt.store.ListProjectDiscussions(r.Context(), projectID, status, query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取讨论列表失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleCreateProjectDiscussion(w http.ResponseWriter, r *http.Request, projectID string) {
	if !rt.ensureDiscussionProjectReadable(w, r, projectID) {
		return
	}
	user, ok, err := rt.currentUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取登录状态失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录后再发起讨论"})
		return
	}
	var input domain.ProjectDiscussionCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	if input.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写讨论标题"})
		return
	}
	if input.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写讨论内容"})
		return
	}
	if len([]rune(input.Title)) > 120 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "讨论标题不能超过 120 个字"})
		return
	}
	if len([]rune(input.Body)) > 20000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "讨论内容不能超过 20000 个字"})
		return
	}
	item, err := rt.store.CreateProjectDiscussion(r.Context(), projectID, user.ID, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "创建讨论失败"})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (rt *Router) handleGetProjectDiscussion(w http.ResponseWriter, r *http.Request, projectID, discussionID string) {
	if !rt.ensureDiscussionProjectReadable(w, r, projectID) {
		return
	}
	discussion, ok, err := rt.store.GetProjectDiscussion(r.Context(), discussionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取讨论失败"})
		return
	}
	if !ok || discussion.ProjectID != projectID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个讨论"})
		return
	}
	comments, err := rt.store.ListProjectDiscussionComments(r.Context(), discussionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取回复失败"})
		return
	}
	writeJSON(w, http.StatusOK, projectDiscussionDetailResponse{Discussion: discussion, Comments: comments})
}

func (rt *Router) handleCreateProjectDiscussionComment(w http.ResponseWriter, r *http.Request, projectID, discussionID string) {
	user, ok, err := rt.currentUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取登录状态失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录后再回复"})
		return
	}
	discussion, found, err := rt.store.GetProjectDiscussion(r.Context(), discussionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取讨论失败"})
		return
	}
	if !found || discussion.ProjectID != projectID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个讨论"})
		return
	}
	if discussion.Status == "closed" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "这个讨论已经关闭，不能继续回复"})
		return
	}
	var input domain.ProjectDiscussionCommentCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Body = strings.TrimSpace(input.Body)
	if input.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写回复内容"})
		return
	}
	if len([]rune(input.Body)) > 20000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "回复内容不能超过 20000 个字"})
		return
	}
	comment, err := rt.store.CreateProjectDiscussionComment(r.Context(), discussionID, user.ID, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "发送回复失败"})
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (rt *Router) handleUpdateProjectDiscussionStatus(w http.ResponseWriter, r *http.Request, projectID, discussionID string) {
	user, ok, err := rt.currentUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取登录状态失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
		return
	}
	if user.Role != "admin" {
		if _, owned, err := rt.store.GetProject(r.Context(), user.ID, projectID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查作品权限失败"})
			return
		} else if !owned {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "只有作品作者可以修改讨论状态"})
			return
		}
	}
	discussion, found, err := rt.store.GetProjectDiscussion(r.Context(), discussionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取讨论失败"})
		return
	}
	if !found || discussion.ProjectID != projectID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个讨论"})
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Status = strings.TrimSpace(input.Status)
	if input.Status != "open" && input.Status != "closed" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "讨论状态不正确"})
		return
	}
	updated, ok, err := rt.store.UpdateProjectDiscussionStatus(r.Context(), discussionID, input.Status, user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "更新讨论状态失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个讨论"})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
