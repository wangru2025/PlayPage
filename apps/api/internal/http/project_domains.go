package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
)

const platformDomainSuffix = ".wangru.net"

func normalizeSubdomain(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func validateSubdomain(value string) error {
	if len(value) < 3 || len(value) > 32 {
		return fmt.Errorf("子域名长度需要在 3 到 32 个字符之间")
	}
	if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return fmt.Errorf("子域名不能以短横线开头或结尾")
	}
	if strings.Contains(value, "--") {
		return fmt.Errorf("子域名不能包含连续短横线")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return fmt.Errorf("子域名只能包含小写字母、数字和短横线")
	}
	return nil
}

func isOpenProjectDomainStatus(status string) bool {
	return status == "pending" || status == "active"
}

func (rt *Router) handleListProjectDomains(w http.ResponseWriter, r *http.Request, projectID string) {
	_, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	items, err := rt.store.ListProjectDomains(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取独立网址申请失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleCreateProjectDomain(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	var input domain.ProjectDomainCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	subdomain := normalizeSubdomain(input.Subdomain)
	if err := validateSubdomain(subdomain); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	reserved, err := rt.store.IsReservedSubdomain(r.Context(), subdomain)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查独立网址失败"})
		return
	}
	if reserved {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "这个独立网址是平台保留或已有站点，请换一个"})
		return
	}
	items, err := rt.store.ListProjectDomains(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查已有申请失败"})
		return
	}
	for _, item := range items {
		if isOpenProjectDomainStatus(item.Status) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个作品已经有待审核或已通过的独立网址申请，请不要重复提交"})
			return
		}
	}
	created, err := rt.store.CreateProjectDomain(r.Context(), domain.ProjectDomain{
		ProjectID:   project.ID,
		OwnerUserID: user.ID,
		Subdomain:   subdomain,
		Domain:      subdomain + platformDomainSuffix,
		Status:      "pending",
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个独立网址已经被申请或使用，请换一个"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "提交独立网址申请失败"})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (rt *Router) handleAdminListProjectDomains(w http.ResponseWriter, r *http.Request) {
	if _, ok := rt.requireAdminUser(w, r); !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "pending" && status != "active" && status != "rejected" && status != "disabled" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "状态筛选不正确"})
		return
	}
	items, err := rt.store.ListAdminProjectDomains(r.Context(), status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取独立网址申请失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleAdminProjectDomainRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/project-domains/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "review" && r.Method == http.MethodPost {
		rt.handleAdminReviewProjectDomain(w, r, parts[0])
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
}

func (rt *Router) handleAdminReviewProjectDomain(w http.ResponseWriter, r *http.Request, domainID string) {
	admin, ok := rt.requireAdminUser(w, r)
	if !ok {
		return
	}
	var input domain.ProjectDomainReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if input.Status != "active" && input.Status != "rejected" && input.Status != "disabled" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "审核状态只能是 active、rejected 或 disabled"})
		return
	}
	if input.Status == "rejected" && strings.TrimSpace(input.RejectReason) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "拒绝申请时需要填写原因"})
		return
	}
	item, err := rt.store.UpdateProjectDomainReview(r.Context(), domainID, input.Status, strings.TrimSpace(input.RejectReason), strings.TrimSpace(input.AdminNote), admin.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存独立网址审核结果失败"})
		return
	}
	rt.notifyProjectDomainReview(item)
	writeJSON(w, http.StatusOK, item)
}
