package http

import (
	"encoding/json"
	"net/http"

	"ai-static-host/api/internal/domain"
)

func (rt *Router) requireAdminUser(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return domain.User{}, false
	}
	if !domain.IsAdminRole(user.Role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "只有管理员才能访问这里"})
		return domain.User{}, false
	}
	return user, true
}

func (rt *Router) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := rt.requireAdminUser(w, r); !ok {
		return
	}

	items, err := rt.store.ListAdminUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取用户列表失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request, userID string) {
	admin, ok := rt.requireAdminUser(w, r)
	if !ok {
		return
	}
	if !domain.IsSuperAdminRole(admin.Role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "只有最高管理员才能修改用户身份和套餐"})
		return
	}

	var input domain.AdminUserUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	switch input.Role {
	case domain.RoleUser, domain.RoleAdmin, domain.RoleSuperAdmin:
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "用户身份不正确"})
		return
	}
	switch input.PlanCode {
	case domain.PlanFree, domain.PlanLight, domain.PlanSupport, domain.PlanAdmin:
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "套餐类型不正确"})
		return
	}

	updated, err := rt.store.UpdateUserRoleAndPlan(r.Context(), userID, input.Role, input.PlanCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存用户设置失败"})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (rt *Router) handleAdminListProjects(w http.ResponseWriter, r *http.Request) {
	if _, ok := rt.requireAdminUser(w, r); !ok {
		return
	}

	items, err := rt.store.ListAdminProjects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品列表失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleAdminListUpgradeRequests(w http.ResponseWriter, r *http.Request) {
	if _, ok := rt.requireAdminUser(w, r); !ok {
		return
	}

	items, err := rt.store.ListUpgradeRequests(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取升级申请失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleAdminReviewUpgradeRequest(w http.ResponseWriter, r *http.Request, requestID string) {
	admin, ok := rt.requireAdminUser(w, r)
	if !ok {
		return
	}

	var input domain.UpgradeRequestReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	if input.Status != "approved" && input.Status != "rejected" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "审核状态只能是 approved 或 rejected"})
		return
	}

	item, err := rt.store.UpdateUpgradeRequest(r.Context(), requestID, input.Status, input.AdminNote, admin.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存审核结果失败"})
		return
	}

	if input.Status == "approved" {
		targetPlan := input.TargetPlan
		if targetPlan == "" {
			targetPlan = item.TargetPlan
		}
		user, found, err := rt.store.GetUserByID(r.Context(), item.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取申请用户失败"})
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到申请用户"})
			return
		}
		updated, err := rt.store.UpdateUserRoleAndPlan(r.Context(), user.ID, user.Role, targetPlan)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "开通套餐失败"})
			return
		}
		item.TargetPlan = updated.PlanCode
	}

	writeJSON(w, http.StatusOK, item)
}
