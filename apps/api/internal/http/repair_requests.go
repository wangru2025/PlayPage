package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

var repairIssueTypes = map[string]bool{
	"page_broken":       true,
	"button_broken":     true,
	"interactive_error": true,
	"data_error":        true,
	"encoding_error":    true,
	"style_error":       true,
	"ai_code_error":     true,
	"other":             true,
}

func isOpenRepairStatus(status string) bool {
	return status == "pending" || status == "processing" || status == "need_info"
}

func isValidRepairStatus(status string) bool {
	switch status {
	case "pending", "processing", "need_info", "fixed", "rejected", "closed":
		return true
	default:
		return false
	}
}

func trimLimit(value string, max int) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= max {
		return value
	}
	return string([]rune(value)[:max])
}

func trimMiddle(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if max <= 0 || len(runes) <= max {
		return value
	}
	if max < 100 {
		return string(runes[:max])
	}
	head := max / 2
	tail := max - head
	return string(runes[:head]) + "\n\n<!-- 中间内容过长，已省略 -->\n\n" + string(runes[len(runes)-tail:])
}

func (rt *Router) handleListProjectRepairRequests(w http.ResponseWriter, r *http.Request, projectID string) {
	_, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	items, err := rt.store.ListProjectRepairRequests(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取修复申请失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleCreateRepairRequest(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	var input domain.RepairRequestCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	issueType := strings.TrimSpace(input.IssueType)
	if !repairIssueTypes[issueType] {
		issueType = "other"
	}
	description := trimLimit(input.Description, 2000)
	expected := trimLimit(input.Expected, 1000)
	contact := trimLimit(input.Contact, 200)
	if description == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写遇到的问题"})
		return
	}
	if !input.AllowAdminEdit {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先确认允许管理员查看并修改这个作品，用于处理本次修复申请"})
		return
	}
	items, err := rt.store.ListProjectRepairRequests(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查已有修复申请失败"})
		return
	}
	for _, item := range items {
		if isOpenRepairStatus(item.Status) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个作品已经有待处理的修复申请，请不要重复提交"})
			return
		}
	}
	created, err := rt.store.CreateRepairRequest(r.Context(), domain.RepairRequest{
		ProjectID:      project.ID,
		OwnerUserID:    user.ID,
		IssueType:      issueType,
		Description:    description,
		Expected:       expected,
		AllowAdminEdit: input.AllowAdminEdit,
		Contact:        contact,
		Status:         "pending",
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个作品已经有待处理的修复申请，请不要重复提交"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "提交修复申请失败"})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (rt *Router) handleReplyRepairRequest(w http.ResponseWriter, r *http.Request, projectID string, requestID string) {
	_, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	var input domain.RepairRequestUserReplyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式不正确"})
		return
	}
	reply := trimLimit(input.Reply, 2000)
	if reply == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写补充信息"})
		return
	}
	item, err := rt.store.ReplyRepairRequest(r.Context(), projectID, requestID, reply)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "只有管理员要求补充信息，并且你还没有回复过时，才能提交补充信息"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (rt *Router) handleAdminListRepairRequests(w http.ResponseWriter, r *http.Request) {
	if _, ok := rt.requireAdminUser(w, r); !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !isValidRepairStatus(status) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "状态筛选不正确"})
		return
	}
	items, err := rt.store.ListAdminRepairRequests(r.Context(), status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取修复申请失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleAdminRepairRequestRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/repair-requests/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "review" && r.Method == http.MethodPost {
		rt.handleAdminUpdateRepairRequest(w, r, parts[0])
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
}

func (rt *Router) handleAdminUpdateRepairRequest(w http.ResponseWriter, r *http.Request, requestID string) {
	admin, ok := rt.requireAdminUser(w, r)
	if !ok {
		return
	}
	var input domain.RepairRequestUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	status := strings.TrimSpace(input.Status)
	if !isValidRepairStatus(status) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "修复申请状态不正确"})
		return
	}
	adminReply := trimLimit(input.AdminReply, 2000)
	if (status == "fixed" || status == "rejected" || status == "need_info") && adminReply == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写给用户的回复"})
		return
	}
	item, err := rt.store.UpdateRepairRequest(r.Context(), requestID, status, adminReply, admin.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存修复申请处理结果失败"})
		return
	}
	rt.notifyRepairRequestStatus(item)
	writeJSON(w, http.StatusOK, item)
}
