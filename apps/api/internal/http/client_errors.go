package http

import (
	"encoding/json"
	"log"
	"net/http"
)

func (rt *Router) handleClientErrorReport(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	log.Printf("client_error_report payload=%v", payload)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
