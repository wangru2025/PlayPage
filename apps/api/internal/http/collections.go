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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "\u8bfb\u53d6\u4f5c\u54c1\u6570\u636e\u8868\u5931\u8d25"})
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u8bf7\u6c42\u5185\u5bb9\u683c\u5f0f\u4e0d\u6b63\u786e"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Permissions = normalizePublicPermissions(input.Permissions)
	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u8bf7\u6c42\u5185\u5bb9\u683c\u5f0f\u4e0d\u6b63\u786e"})
		return
	}

	collection, err := rt.store.CreateCollection(r.Context(), project.ID, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "\u521b\u5efa\u4f5c\u54c1\u6570\u636e\u8868\u5931\u8d25"})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "\u8bfb\u53d6\u4f5c\u54c1\u6570\u636e\u8868\u5931\u8d25"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "\u627e\u4e0d\u5230\u8fd9\u4e2a\u4f5c\u54c1\u6570\u636e\u8868"})
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u8bf7\u6c42\u5185\u5bb9\u683c\u5f0f\u4e0d\u6b63\u786e"})
		return
	}
	input.Permissions = normalizePublicPermissions(input.Permissions)

	collection, found, err := rt.store.UpdateCollection(r.Context(), project.ID, collectionName, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "\u627e\u4e0d\u5230\u8fd9\u4e2a\u4f5c\u54c1\u6570\u636e\u8868"})
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
