package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
)

type projectProposalDetailResponse struct {
	Proposal domain.ProjectProposal `json:"proposal"`
	CanReview bool                   `json:"canReview"`
}

func proposalStatusText(status string) string {
	switch status {
	case "open":
		return "等待作者处理"
	case "accepted":
		return "已采纳"
	case "rejected":
		return "已拒绝"
	case "closed":
		return "已关闭"
	default:
		return status
	}
}

func (rt *Router) handleListMyProjectProposals(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	items, err := rt.store.ListUserProjectProposals(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取我的提案失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleListProjectProposals(w http.ResponseWriter, r *http.Request, projectID string) {
	access, found, err := rt.store.GetProjectPublicAccess(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品失败"})
		return
	}
	if !found || access.Project.CurrentRelease == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "open" && status != "accepted" && status != "rejected" && status != "closed" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "提案状态不正确"})
		return
	}
	items, err := rt.store.ListProjectProposalsForTarget(r.Context(), projectID, status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取提案列表失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": access.Project, "items": items})
}

func (rt *Router) handleCreateProjectProposal(w http.ResponseWriter, r *http.Request, sourceProjectID string) {
	user, source, ok := rt.requireOwnedProject(w, r, sourceProjectID)
	if !ok {
		return
	}
	if source.ForkedFromProjectID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "只有改编作品才能向原作者提交改进提案"})
		return
	}
	if source.CurrentRelease == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先发布改编作品，再提交提案"})
		return
	}
	targetAccess, found, err := rt.store.GetProjectPublicAccess(r.Context(), source.ForkedFromProjectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取原作品失败"})
		return
	}
	if !found || targetAccess.Project.CurrentRelease == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "原作品不存在或还没有发布，不能提交提案"})
		return
	}
	if !targetAccess.Project.AllowForks {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "原作者已经关闭改编，不再接收提案"})
		return
	}
	if targetAccess.OwnerUserID == user.ID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "不能给自己的原作品提交提案"})
		return
	}

	var input domain.ProjectProposalCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	if input.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写提案标题"})
		return
	}
	if input.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请说明你改进了什么"})
		return
	}
	if len([]rune(input.Title)) > 120 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "提案标题不能超过 120 个字"})
		return
	}
	if len([]rune(input.Body)) > 20000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "提案说明不能超过 20000 个字"})
		return
	}
	item, err := rt.store.CreateProjectProposal(r.Context(), domain.ProjectProposal{
		SourceProjectID:   source.ID,
		TargetProjectID:   targetAccess.Project.ID,
		AuthorUserID:      user.ID,
		TargetOwnerUserID: targetAccess.OwnerUserID,
		Title:             input.Title,
		Body:              input.Body,
		SourceReleaseID:   source.CurrentRelease,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "提交提案失败"})
		return
	}
	rt.notifyProjectProposalCreated(item)
	writeJSON(w, http.StatusCreated, item)
}

func (rt *Router) handleGetProjectProposal(w http.ResponseWriter, r *http.Request, projectID, proposalID string) {
	proposal, found, err := rt.store.GetProjectProposal(r.Context(), proposalID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取提案失败"})
		return
	}
	if !found || (proposal.SourceProjectID != projectID && proposal.TargetProjectID != projectID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个提案"})
		return
	}
	canReview := false
	if user, ok, err := rt.currentUser(r); err == nil && ok {
		canReview = user.Role == "admin" || user.ID == proposal.TargetOwnerUserID
	}
	writeJSON(w, http.StatusOK, projectProposalDetailResponse{Proposal: proposal, CanReview: canReview})
}

func (rt *Router) handleReviewProjectProposal(w http.ResponseWriter, r *http.Request, projectID, proposalID string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	proposal, found, err := rt.store.GetProjectProposal(r.Context(), proposalID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取提案失败"})
		return
	}
	if !found || proposal.TargetProjectID != projectID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个提案"})
		return
	}
	if user.Role != "admin" && user.ID != proposal.TargetOwnerUserID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "只有原作品作者可以处理提案"})
		return
	}
	if proposal.Status != "open" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "这个提案已经处理过了"})
		return
	}
	var input domain.ProjectProposalReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Status = strings.TrimSpace(input.Status)
	input.Note = trimLimit(input.Note, 1000)
	if input.Status != "accepted" && input.Status != "rejected" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "只能选择采纳或拒绝"})
		return
	}

	mergedReleaseID := ""
	if input.Status == "accepted" {
		targetAccess, found, err := rt.store.GetProjectPublicAccess(r.Context(), proposal.TargetProjectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取原作品失败"})
			return
		}
		if !found || targetAccess.Project.CurrentRelease == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "原作品不存在或还没有发布"})
			return
		}
		sourceRelease, ok := rt.findRelease(r.Context(), proposal.SourceProjectID, proposal.SourceReleaseID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到提案提交时的改编版本"})
			return
		}
		note := fmt.Sprintf("合并来自 @%s 的改进提案：%s", proposal.AuthorUsername, proposal.Title)
		release, err := rt.pub.PublishForkFromRelease(r.Context(), targetAccess.Project, sourceRelease, note)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "采纳提案并发布失败：" + err.Error()})
			return
		}
		mergedReleaseID = release.ID
		if targetAccess.Project.Interactive {
			_ = rt.copyCollectionSchemas(r.Context(), proposal.SourceProjectID, proposal.TargetProjectID)
		}
		rt.notifyProjectReleaseIfNeeded(targetAccess.Project, release)
	}

	updated, found, err := rt.store.ReviewProjectProposal(r.Context(), proposalID, input.Status, input.Note, user.ID, mergedReleaseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存提案处理结果失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个提案"})
		return
	}
	rt.notifyProjectProposalReviewed(updated)
	writeJSON(w, http.StatusOK, updated)
}

func (rt *Router) findRelease(ctx context.Context, projectID, releaseID string) (domain.Release, bool) {
	releases, err := rt.store.ListReleases(ctx, projectID)
	if err != nil {
		return domain.Release{}, false
	}
	for _, release := range releases {
		if release.ID == releaseID {
			return release, true
		}
	}
	return domain.Release{}, false
}
