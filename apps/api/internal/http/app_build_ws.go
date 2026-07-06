package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type appBuildHub struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]bool
}

func newAppBuildHub() *appBuildHub {
	return &appBuildHub{clients: map[string]map[*websocket.Conn]bool{}}
}

func (h *appBuildHub) add(projectID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[projectID] == nil {
		h.clients[projectID] = map[*websocket.Conn]bool{}
	}
	h.clients[projectID][c] = true
}

func (h *appBuildHub) remove(projectID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[projectID] != nil {
		delete(h.clients[projectID], c)
		if len(h.clients[projectID]) == 0 {
			delete(h.clients, projectID)
		}
	}
}

func (h *appBuildHub) broadcast(projectID string, payload any) {
	h.mu.Lock()
	list := make([]*websocket.Conn, 0, len(h.clients[projectID]))
	for c := range h.clients[projectID] {
		list = append(list, c)
	}
	h.mu.Unlock()
	for _, c := range list {
		_ = c.WriteJSON(payload)
	}
}

func (rt *Router) handleAppBuildWebSocket(w http.ResponseWriter, r *http.Request, projectID string) {
	if _, _, ok := rt.requireOwnedProject(w, r, projectID); !ok {
		return
	}
	up := websocket.Upgrader{CheckOrigin: sameHostOrigin}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	rt.appHub.add(projectID, conn)
	defer func() {
		rt.appHub.remove(projectID, conn)
		_ = conn.Close()
	}()
	_ = conn.WriteJSON(map[string]any{"type": "snapshot", "projectId": projectID, "time": time.Now().UTC()})
	for {
		if _, _, err := conn.NextReader(); err != nil {
			return
		}
	}
}

func (rt *Router) broadcastAppBuildUpdate(projectID string) {
	if rt.appHub == nil || projectID == "" {
		return
	}
	rt.appHub.broadcast(projectID, map[string]any{
		"type":      "app_builds_updated",
		"projectId": projectID,
		"time":      time.Now().UTC(),
	})
}
