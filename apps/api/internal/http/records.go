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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if len(input.Data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "记录内容不能为空"})
		return
	}

	record, err := rt.store.CreateRecord(r.Context(), project.ID, collectionName, "", input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, record)
}
