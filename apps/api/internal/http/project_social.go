package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

func (rt *Router) handleAuthorRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/authors/"), "/")
	parts := strings.Split(path, "/")
	username := ""
	if len(parts) > 0 {
		username = strings.TrimSpace(parts[0])
	}
	if username == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作者"})
		return
	}
	if len(parts) == 2 && parts[1] == "follow" && r.Method == http.MethodPost {
		rt.handleFollowAuthor(w, r, username)
		return
	}
	if len(parts) == 2 && parts[1] == "follow" && r.Method == http.MethodDelete {
		rt.handleUnfollowAuthor(w, r, username)
		return
	}
	if len(parts) != 1 || r.Method != http.MethodGet {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
		return
	}
	profile, found, err := rt.store.GetAuthorProfile(r.Context(), username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作者主页失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作者"})
		return
	}
	if user, ok, err := rt.currentUser(r); err == nil && ok {
		following, err := rt.store.IsFollowingAuthor(r.Context(), user.ID, username)
		if err == nil {
			profile.FollowingByMe = following
		}
	}
	writeJSON(w, http.StatusOK, profile)
}

func (rt *Router) handleFollowAuthor(w http.ResponseWriter, r *http.Request, username string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if user.Username == username {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "不能关注自己"})
		return
	}
	if _, found, err := rt.store.GetAuthorProfile(r.Context(), username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作者失败"})
		return
	} else if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作者"})
		return
	}
	if following, err := rt.store.IsFollowingAuthor(r.Context(), user.ID, username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查关注状态失败"})
		return
	} else if following {
		writeJSON(w, http.StatusOK, map[string]string{"status": "following"})
		return
	}
	if err := rt.store.FollowAuthor(r.Context(), user.ID, username); err != nil {
		log.Printf("follow_author_failed follower_user_id=%s target_username=%q err=%v", user.ID, username, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "关注作者失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "following"})
}

func (rt *Router) handleUnfollowAuthor(w http.ResponseWriter, r *http.Request, username string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if err := rt.store.UnfollowAuthor(r.Context(), user.ID, username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "取消关注失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unfollowed"})
}

func (rt *Router) handleListMyFollowedAuthors(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	items, err := rt.store.ListFollowedAuthors(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取关注列表失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) getProfileVisibleProjectForFavorite(w http.ResponseWriter, r *http.Request, projectID string) (domain.Project, bool) {
	access, found, err := rt.store.GetProjectPublicAccess(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品失败"})
		return domain.Project{}, false
	}
	if !found || access.Project.CurrentRelease == "" || !access.Project.ShowOnProfile {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "只能收藏作者主页可见的作品"})
		return domain.Project{}, false
	}
	return access.Project, true
}

func (rt *Router) handleListMyFavoriteProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	items, err := rt.store.ListUserFavoriteProjects(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取收藏作品失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleGetProjectFavorite(w http.ResponseWriter, r *http.Request, projectID string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	project, ok := rt.getProfileVisibleProjectForFavorite(w, r, projectID)
	if !ok {
		return
	}
	favorited, err := rt.store.IsProjectFavorited(r.Context(), user.ID, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取收藏状态失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"favorited":      favorited,
		"favoritesCount": project.FavoritesCount,
	})
}

func (rt *Router) handleAddProjectFavorite(w http.ResponseWriter, r *http.Request, projectID string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if _, ok := rt.getProfileVisibleProjectForFavorite(w, r, projectID); !ok {
		return
	}
	if err := rt.store.AddProjectFavorite(r.Context(), user.ID, projectID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "收藏作品失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "favorited"})
}

func (rt *Router) handleRemoveProjectFavorite(w http.ResponseWriter, r *http.Request, projectID string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if err := rt.store.RemoveProjectFavorite(r.Context(), user.ID, projectID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "取消收藏失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unfavorited"})
}

func (rt *Router) handleForkProject(w http.ResponseWriter, r *http.Request, sourceProjectID string) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	if err := rt.checkProjectCreationAllowed(user); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	source, found, err := rt.store.GetPublicProject(r.Context(), sourceProjectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取原作品失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "只能改编公开作品"})
		return
	}
	if !source.AllowForks {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "这个作品的作者没有开放改编"})
		return
	}
	releases, err := rt.store.ListReleases(r.Context(), source.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取原作品版本失败"})
		return
	}
	var sourceRelease domain.Release
	for _, release := range releases {
		if release.ID == source.CurrentRelease {
			sourceRelease = release
			break
		}
	}
	if sourceRelease.ID == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到原作品当前版本"})
		return
	}

	var input domain.ProjectForkInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = source.Name + " 的改编版"
	}
	username := normalizePathSegment(user.Username)
	if username == "" {
		username = inferUsernameFromEmail(user.Email)
	}
	slug := normalizePathSegment(input.Slug)
	if slug == "" {
		slug = normalizePathSegment(name)
	}
	if username == "" || slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先设置公开名字，并填写改编作品地址"})
		return
	}

	project, err := rt.store.CreateForkProject(r.Context(), user.ID, domain.ProjectCreateInput{
		Name:             name,
		Username:         username,
		Slug:             slug,
		Interactive:      input.Interactive,
		AnalyticsEnabled: input.AnalyticsEnabled,
		AllowForks:       input.AllowForks,
	}, source)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个作品地址你已经用过了，请换一个作品地址"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "创建改编作品失败"})
		return
	}

	if project.Interactive {
		_ = rt.copyCollectionSchemas(r.Context(), source.ID, project.ID)
	}

	release, err := rt.pub.PublishForkFromRelease(r.Context(), project, sourceRelease, "改编自 "+source.Username+" 的《"+source.Name+"》")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "复制原作品内容失败：" + err.Error()})
		return
	}
	project.CurrentRelease = release.ID
	writeJSON(w, http.StatusCreated, map[string]any{"project": project, "release": release})
}

func (rt *Router) copyCollectionSchemas(ctx context.Context, sourceProjectID, targetProjectID string) error {
	items, err := rt.store.ListCollections(ctx, sourceProjectID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, found, err := rt.store.GetCollectionByName(ctx, targetProjectID, item.Name); err != nil {
			return err
		} else if found {
			continue
		}
		if _, err := rt.store.CreateCollection(ctx, targetProjectID, domain.CollectionCreateInput{
			Name:        item.Name,
			Permissions: item.Permissions,
			Fields:      item.Fields,
		}); err != nil {
			return err
		}
	}
	return nil
}
