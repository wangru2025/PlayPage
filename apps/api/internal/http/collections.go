package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
)

func (rt *Router) handleListCollections(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	items, err := rt.store.ListCollections(r.Context(), project.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品数据表失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projectId": projectID,
		"items":     items,
	})
}

func (rt *Router) handleCreateCollection(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	collection, err := rt.store.CreateCollection(r.Context(), project.ID, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "创建作品数据表失败"})
		return
	}

	writeJSON(w, http.StatusCreated, collection)
}

func (rt *Router) handleGetCollection(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	collection, found, err := rt.store.GetCollectionByName(r.Context(), project.ID, collectionName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品数据表失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品数据表"})
		return
	}

	writeJSON(w, http.StatusOK, collection)
}

func (rt *Router) handleUpdateCollection(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	var input domain.CollectionUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Permissions = normalizePublicPermissions(input.Permissions)

	collection, found, err := rt.store.UpdateCollection(r.Context(), project.ID, collectionName, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品数据表"})
		return
	}

	writeJSON(w, http.StatusOK, collection)
}

func (rt *Router) handleDeleteCollection(w http.ResponseWriter, r *http.Request, projectID, collectionName string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	if err := rt.store.DeleteCollection(r.Context(), project.ID, collectionName); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
