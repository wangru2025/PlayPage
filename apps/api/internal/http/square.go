package http

import "net/http"

func (rt *Router) handleListSquare(w http.ResponseWriter, r *http.Request) {
	items, err := rt.store.ListPublicProjects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取广场作品失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}
