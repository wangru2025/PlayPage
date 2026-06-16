package http

import (
	"net/http"

	"ai-static-host/api/internal/domain"
)

func (rt *Router) requireOwnedProject(w http.ResponseWriter, r *http.Request, projectID string) (domain.User, domain.Project, bool) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return domain.User{}, domain.Project{}, false
	}

	project, found, err := rt.store.GetProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get project failed"})
		return domain.User{}, domain.Project{}, false
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return domain.User{}, domain.Project{}, false
	}

	return user, project, true
}
