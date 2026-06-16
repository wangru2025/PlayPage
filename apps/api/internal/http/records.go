package http

import (
	"encoding/json"
	"net/http"

	"ai-static-host/api/internal/domain"
)

func (rt *Router) handleListRecords(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	items, err := rt.store.ListRecords(r.Context(), project.ID, collectionName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "\u8bfb\u53d6\u8bb0\u5f55\u5931\u8d25"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projectId":      projectID,
		"collectionName": collectionName,
		"items":          items,
	})
}

func (rt *Router) handleCreateRecord(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	var input domain.RecordCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u8bf7\u6c42\u5185\u5bb9\u683c\u5f0f\u4e0d\u6b63\u786e"})
		return
	}
	if len(input.Data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u8bb0\u5f55\u5185\u5bb9\u4e0d\u80fd\u4e3a\u7a7a"})
		return
	}

	record, err := rt.store.CreateRecord(r.Context(), project.ID, collectionName, "", input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, record)
}
