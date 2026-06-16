package http

import (
	"net/http"
	"time"
)

func (rt *Router) handleProjectStats(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	if !project.AnalyticsEnabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品没有启用访问量统计"})
		return
	}

	now := time.Now().UTC()
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if to == "" {
		to = now.Format("2006-01-02")
	}
	if from == "" {
		from = now.AddDate(0, 0, -29).Format("2006-01-02")
	}
	fromDay, err := time.Parse("2006-01-02", from)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "开始日期格式不正确"})
		return
	}
	toDay, err := time.Parse("2006-01-02", to)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "结束日期格式不正确"})
		return
	}
	if fromDay.After(toDay) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "开始日期不能晚于结束日期"})
		return
	}
	if toDay.Sub(fromDay) > 370*24*time.Hour {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "单次最多查看 370 天统计数据"})
		return
	}

	stats, err := rt.store.ProjectStats(r.Context(), projectID, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取统计数据失败"})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
