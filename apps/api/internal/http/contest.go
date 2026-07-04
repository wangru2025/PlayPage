package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
)

func (rt *Router) handleGetProjectContestSubmission(w http.ResponseWriter, r *http.Request, projectID string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if _, found, err := rt.store.GetProject(r.Context(), user.ID, projectID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品失败"})
		return
	} else if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}
	item, found, err := rt.store.GetContestSubmissionByUserProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取参赛状态失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"submitted": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submitted": true, "submission": item})
}

func (rt *Router) handleCreateContestSubmission(w http.ResponseWriter, r *http.Request, projectID string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	project, found, err := rt.store.GetProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}
	if project.CurrentRelease == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品还没有上传网页，暂时不能参赛"})
		return
	}
	var input domain.ContestSubmissionCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	track := normalizeContestTrack(input.Track)
	if track == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请选择参赛赛道"})
		return
	}
	intro := trimLimit(input.Intro, 600)
	if intro == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写作品介绍"})
		return
	}
	story := trimLimit(input.Story, 1000)
	username := user.Username
	if username == "" {
		username = user.Email
	}
	item, err := rt.store.CreateContestSubmission(r.Context(), domain.ContestSubmission{
		UserID:        user.ID,
		UserEmail:     user.Email,
		Username:      username,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		ProjectURL:    project.PublicURL,
		Track:         track,
		Intro:         intro,
		Story:         story,
		AllowShowcase: input.AllowShowcase,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "已经提交") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个作品已经提交过比赛"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "提交参赛作品失败"})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func normalizeContestTrack(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "creative", "fun", "useful", "interactive", "newcomer":
		return value
	default:
		return ""
	}
}
