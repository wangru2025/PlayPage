package http

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
)

const dailyAppBuildLimitPerPlatform = 1

var androidPackagePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

func (rt *Router) handleGetAppBuildSettings(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	settings, found, err := rt.store.GetAppBuildSettings(r.Context(), user.ID, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取安装包设置失败"})
		return
	}
	if !found {
		settings = defaultAppBuildSettings(user, project)
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (rt *Router) handleSaveAppBuildSettings(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	var input domain.AppBuildSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input = normalizeAppBuildSettingsInput(input, user, project)
	if err := validateAppBuildSettingsInput(input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	settings, err := rt.store.UpsertAppBuildSettings(r.Context(), user.ID, projectID, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存安装包设置失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (rt *Router) handleListAppBuilds(w http.ResponseWriter, r *http.Request, projectID string) {
	user, _, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	items, err := rt.store.ListAppBuildJobs(r.Context(), user.ID, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取安装包构建记录失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleCreateAppBuild(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	var input domain.AppBuildCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	platform := normalizeAppBuildPlatform(input.Platform)
	if platform == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "安装包平台只能是 Android 或 Windows"})
		return
	}
	if project.CurrentRelease == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先上传作品内容，再导出安装包"})
		return
	}
	job, err := rt.createAppBuildJob(r.Context(), user, project, project.CurrentRelease, platform, input.AutoUpdate, false)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	rt.broadcastAppBuildUpdate(projectID)
	writeJSON(w, http.StatusCreated, map[string]any{"job": job})
}

func (rt *Router) handleCheckAppUpdate(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
	platform := normalizeAppBuildPlatform(r.URL.Query().Get("platform"))
	versionCode, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("versionCode")))
	if projectID == "" || platform == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少作品或平台参数"})
		return
	}
	info, err := rt.store.GetAppUpdateInfo(r.Context(), projectID, platform, versionCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "检查更新失败"})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (rt *Router) handleInternalAppBuildRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/internal/app-builds/")
	parts := strings.Split(path, "/")
	if rt.cfg.AppBuildCallbackToken == "" || r.Header.Get("Authorization") != "Bearer "+rt.cfg.AppBuildCallbackToken {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "内部回调凭证不正确"})
		return
	}
	if len(parts) == 2 && parts[1] == "source" && r.Method == http.MethodGet {
		rt.handleInternalDownloadAppBuildSource(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "metadata" && r.Method == http.MethodGet {
		rt.handleInternalGetAppBuildMetadata(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "artifact" && r.Method == http.MethodPost {
		rt.handleInternalUploadAppBuildArtifact(w, r, parts[0])
		return
	}
	if len(parts) != 2 || parts[1] != "complete" || r.Method != http.MethodPost {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
		return
	}
	var input domain.AppBuildCompleteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if input.Status != "succeeded" && input.Status != "failed" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "构建状态不正确"})
		return
	}
	job, found, err := rt.store.CompleteAppBuildJob(r.Context(), parts[0], input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存构建结果失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到构建任务"})
		return
	}
	rt.broadcastAppBuildUpdate(job.ProjectID)
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (rt *Router) handleInternalGetAppBuildMetadata(w http.ResponseWriter, r *http.Request, jobID string) {
	job, found, err := rt.store.GetAppBuildJob(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取构建任务失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到构建任务"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (rt *Router) handlePublicAppBuildRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/app-builds/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "download" && r.Method == http.MethodGet {
		rt.handleDownloadAppBuildArtifact(w, r, parts[0])
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
}

func (rt *Router) handleInternalDownloadAppBuildSource(w http.ResponseWriter, r *http.Request, jobID string) {
	job, found, err := rt.store.GetAppBuildJob(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取构建任务失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到构建任务"})
		return
	}
	releases, err := rt.store.ListReleases(r.Context(), job.ProjectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取发布版本失败"})
		return
	}
	var release domain.Release
	for _, item := range releases {
		if item.ID == job.ReleaseID {
			release = item
			break
		}
	}
	if release.ID == "" || release.PublicPath == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到构建用发布内容"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="playpage-app-source.zip"`)
	if err := writeZipDir(w, release.PublicPath); err != nil {
		log.Printf("write_app_build_source_failed job=%s dir=%s err=%v", jobID, release.PublicPath, err)
	}
}

func (rt *Router) handleInternalUploadAppBuildArtifact(w http.ResponseWriter, r *http.Request, jobID string) {
	job, found, err := rt.store.GetAppBuildJob(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取构建任务失败"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到构建任务"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 600<<20)
	if err := r.ParseMultipartForm(600 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "上传安装包表单无效"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少安装包文件"})
		return
	}
	defer file.Close()
	name := sanitizeArtifactName(header.Filename, job.Platform)
	dir := filepath.Join(rt.cfg.DataRoot, "app-build-artifacts", jobID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "创建安装包目录失败"})
		return
	}
	target := filepath.Join(dir, name)
	dst, err := os.Create(target)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存安装包失败"})
		return
	}
	hasher := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(dst, hasher), file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "写入安装包失败"})
		return
	}
	artifactSHA := fmt.Sprintf("%x", hasher.Sum(nil))
	if job.Platform == "windows" && strings.EqualFold(filepath.Ext(name), ".zip") {
		_, updateSHA, updateSize, err := extractWindowsAppBundle(target, dir)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		artifactSHA = updateSHA
		size = updateSize
	}
	publicURL := strings.TrimRight(rt.cfg.PublicBase, "/") + "/api/v1/app-builds/" + jobID + "/download"
	completed, _, err := rt.store.CompleteAppBuildJob(r.Context(), jobID, domain.AppBuildCompleteInput{
		Status:         "succeeded",
		ArtifactPath:   publicURL,
		ArtifactSHA256: artifactSHA,
		ArtifactSize:   size,
		GitHubRunID:    strings.TrimSpace(r.FormValue("githubRunId")),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存构建结果失败"})
		return
	}
	rt.broadcastAppBuildUpdate(completed.ProjectID)
	writeJSON(w, http.StatusOK, map[string]any{"job": completed})
}

func (rt *Router) handleDownloadAppBuildArtifact(w http.ResponseWriter, r *http.Request, jobID string) {
	job, found, err := rt.store.GetAppBuildJob(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取构建任务失败"})
		return
	}
	if !found || job.Status != "succeeded" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到安装包"})
		return
	}
	dir := filepath.Join(rt.cfg.DataRoot, "app-build-artifacts", jobID)
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	filename, err := selectAppBuildDownloadFile(dir, job.Platform, kind)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "安装包文件不存在"})
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(filename)))
	http.ServeFile(w, r, filename)
}

func (rt *Router) createAppBuildJob(ctx context.Context, user domain.User, project domain.Project, releaseID, platform string, autoUpdate, automatic bool) (domain.AppBuildJob, error) {
	count := 0
	if !domain.IsAdminRole(user.Role) {
		now := time.Now().In(time.Local)
		since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var err error
		count, err = rt.store.CountAppBuildJobsForUserSince(ctx, user.ID, platform, since)
		if err != nil {
			return domain.AppBuildJob{}, fmt.Errorf("检查今日构建额度失败")
		}
	}
	settings, found, err := rt.store.GetAppBuildSettings(ctx, user.ID, project.ID)
	if err != nil {
		return domain.AppBuildJob{}, fmt.Errorf("读取安装包设置失败")
	}
	if !found {
		settings = defaultAppBuildSettings(user, project)
	}
	packageName := settings.WindowsPackageName
	if platform == "android" {
		packageName = settings.AndroidPackageName
	}
	versionCode, err := rt.nextAppBuildVersionCode(ctx, user.ID, project.ID, platform)
	if err != nil {
		return domain.AppBuildJob{}, fmt.Errorf("计算安装包版本号失败")
	}
	job := domain.AppBuildJob{
		UserID:       user.ID,
		ProjectID:    project.ID,
		ReleaseID:    releaseID,
		Platform:     platform,
		AppName:      fallbackString(settings.AppName, project.Name),
		PackageName:  packageName,
		VersionCode:  versionCode,
		VersionName:  fmt.Sprintf("1.0.%d", versionCode),
		AutoUpdate:   autoUpdate,
		Status:       "pending",
		ErrorMessage: "",
	}
	if !domain.IsAdminRole(user.Role) && count >= dailyAppBuildLimitPerPlatform {
		job.Status = "skipped"
		job.ErrorMessage = "今天这个平台的安装包构建额度已经用完"
		saved, saveErr := rt.store.CreateAppBuildJob(ctx, job)
		if saveErr != nil {
			return domain.AppBuildJob{}, fmt.Errorf("保存构建任务失败")
		}
		rt.notifyAppBuildLimitExceeded(user, project, platform)
		rt.broadcastAppBuildUpdate(project.ID)
		if automatic {
			return saved, nil
		}
		return saved, fmt.Errorf("今天%s安装包构建额度已经用完，明天可以继续构建", appBuildPlatformLabel(platform))
	}
	saved, err := rt.store.CreateAppBuildJob(ctx, job)
	if err != nil {
		return domain.AppBuildJob{}, fmt.Errorf("保存构建任务失败")
	}
	if err := rt.appBuild.Dispatch(ctx, saved, rt.cfg.PublicBase, rt.cfg.AppBuildCallbackToken); err != nil {
		failed, _, _ := rt.store.CompleteAppBuildJob(ctx, saved.ID, domain.AppBuildCompleteInput{Status: "failed", ErrorMessage: err.Error()})
		if failed.ProjectID != "" {
			rt.broadcastAppBuildUpdate(failed.ProjectID)
		} else {
			rt.broadcastAppBuildUpdate(project.ID)
		}
		return saved, fmt.Errorf("触发安装包构建失败：%w", err)
	}
	rt.broadcastAppBuildUpdate(project.ID)
	return saved, nil
}

func (rt *Router) scheduleAppBuildsAfterRelease(user domain.User, project domain.Project, release domain.Release) {
	ctx := context.Background()
	settings, found, err := rt.store.GetAppBuildSettings(ctx, user.ID, project.ID)
	if err != nil || !found || !settings.AutoUpdate {
		return
	}
	if settings.AndroidEnabled && rt.hasSuccessfulAppBuild(ctx, user.ID, project.ID, "android") {
		if _, err := rt.createAppBuildJob(ctx, user, project, release.ID, "android", true, true); err != nil {
			log.Printf("schedule_android_app_build_failed project=%s err=%v", project.ID, err)
		}
	}
	if settings.WindowsEnabled && rt.hasSuccessfulAppBuild(ctx, user.ID, project.ID, "windows") {
		if _, err := rt.createAppBuildJob(ctx, user, project, release.ID, "windows", true, true); err != nil {
			log.Printf("schedule_windows_app_build_failed project=%s err=%v", project.ID, err)
		}
	}
}

func (rt *Router) hasSuccessfulAppBuild(ctx context.Context, userID, projectID, platform string) bool {
	items, err := rt.store.ListAppBuildJobs(ctx, userID, projectID)
	if err != nil {
		log.Printf("check_successful_app_build_failed project=%s platform=%s err=%v", projectID, platform, err)
		return false
	}
	for _, item := range items {
		if item.Platform == platform && item.Status == "succeeded" {
			return true
		}
	}
	return false
}

func (rt *Router) nextAppBuildVersionCode(ctx context.Context, userID, projectID, platform string) (int, error) {
	items, err := rt.store.ListAppBuildJobs(ctx, userID, projectID)
	if err != nil {
		return 0, err
	}
	maxVersion := 0
	for _, item := range items {
		if item.Platform == platform && item.Status == "succeeded" && item.VersionCode > maxVersion {
			maxVersion = item.VersionCode
		}
	}
	return maxVersion + 1, nil
}

func (rt *Router) notifyAppBuildLimitExceeded(user domain.User, project domain.Project, platform string) {
	if rt.mailer == nil || user.Email == "" {
		return
	}
	subject := "PlayPage 安装包自动更新已跳过"
	body := fmt.Sprintf("您好，%s：\n\n你的作品《%s》刚刚更新了网页内容，但今天账号级的 %s 安装包构建额度已经用完，所以这次没有自动生成新的安装包。\n\n当前规则：每个账号每天最多构建 1 次 Android 安装包、1 次 Windows 安装包。明天可以继续构建。\n\nPlayPage", fallbackString(user.Username, user.Email), project.Name, appBuildPlatformLabel(platform))
	go func() {
		if err := rt.mailer.SendText(user.Email, subject, body); err != nil {
			log.Printf("send_app_build_limit_mail_failed email=%s err=%v", user.Email, err)
		}
	}()
}

func defaultAppBuildSettings(user domain.User, project domain.Project) domain.AppBuildSettings {
	base := "u" + shortStableHex(user.ID, 10)
	slug := "p" + shortStableHex(project.ID, 10)
	return domain.AppBuildSettings{
		UserID:             user.ID,
		ProjectID:          project.ID,
		AppName:            project.Name,
		AndroidPackageName: "net.wangru.playpage." + base + "." + slug,
		WindowsPackageName: "PlayPage-" + project.Slug,
	}
}

func normalizeAppBuildSettingsInput(input domain.AppBuildSettingsInput, user domain.User, project domain.Project) domain.AppBuildSettingsInput {
	defaults := defaultAppBuildSettings(user, project)
	input.AppName = trimLimit(strings.TrimSpace(input.AppName), 60)
	if input.AppName == "" {
		input.AppName = defaults.AppName
	}
	input.AndroidPackageName = strings.ToLower(strings.TrimSpace(input.AndroidPackageName))
	if input.AndroidPackageName == "" {
		input.AndroidPackageName = defaults.AndroidPackageName
	}
	input.WindowsPackageName = trimLimit(strings.TrimSpace(input.WindowsPackageName), 80)
	if input.WindowsPackageName == "" {
		input.WindowsPackageName = defaults.WindowsPackageName
	}
	return input
}

func validateAppBuildSettingsInput(input domain.AppBuildSettingsInput) error {
	if input.AppName == "" {
		return fmt.Errorf("请填写应用名称")
	}
	if input.AndroidEnabled && !androidPackagePattern.MatchString(input.AndroidPackageName) {
		return fmt.Errorf("Android 包名格式不正确，例如 net.wangru.playpage.myapp")
	}
	if input.AutoUpdate && !input.AndroidEnabled && !input.WindowsEnabled {
		return fmt.Errorf("启用自动更新前，请至少选择一个安装包平台")
	}
	return nil
}

func normalizeAppBuildPlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "android":
		return "android"
	case "windows":
		return "windows"
	default:
		return ""
	}
}

func appBuildPlatformLabel(platform string) string {
	if platform == "android" {
		return "Android"
	}
	return "Windows"
}

func normalizePackagePart(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDot := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDot = false
		} else if !lastDot && b.Len() > 0 {
			b.WriteByte('.')
			lastDot = true
		}
	}
	out := strings.Trim(b.String(), ".")
	if out == "" || out[0] < 'a' || out[0] > 'z' {
		out = "u" + out
	}
	return strings.ReplaceAll(out, "..", ".")
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func shortStableHex(value string, length int) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	out := fmt.Sprintf("%x", sum[:])
	if length <= 0 || length > len(out) {
		return out
	}
	return out[:length]
}

func sanitizeArtifactName(filename, platform string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if platform == "android" && ext != ".apk" {
		ext = ".apk"
	}
	if platform == "windows" && ext != ".zip" && ext != ".exe" && ext != ".msi" {
		ext = ".zip"
	}
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = normalizePackagePart(base)
	if base == "" || base == "u" {
		base = "playpage"
	}
	return base + ext
}

func extractWindowsAppBundle(bundlePath, dir string) (string, string, int64, error) {
	zr, err := zip.OpenReader(bundlePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("Windows 安装包格式不正确")
	}
	defer zr.Close()
	var updatePath string
	for _, item := range zr.File {
		if item.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(item.Name)
		if !strings.HasPrefix(name, "install/") && !strings.HasPrefix(name, "update/") {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(name))
		target := filepath.Join(dir, clean)
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(os.PathSeparator)) {
			return "", "", 0, fmt.Errorf("Windows 安装包路径不安全")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", "", 0, fmt.Errorf("创建 Windows 安装包目录失败")
		}
		src, err := item.Open()
		if err != nil {
			return "", "", 0, fmt.Errorf("读取 Windows 安装包失败")
		}
		dst, err := os.Create(target)
		if err != nil {
			src.Close()
			return "", "", 0, fmt.Errorf("保存 Windows 安装包失败")
		}
		_, copyErr := io.Copy(dst, src)
		closeErr := dst.Close()
		src.Close()
		if copyErr != nil || closeErr != nil {
			return "", "", 0, fmt.Errorf("写入 Windows 安装包失败")
		}
		if strings.HasPrefix(name, "update/") && strings.EqualFold(filepath.Ext(target), ".zip") {
			updatePath = target
		}
	}
	if updatePath == "" {
		return "", "", 0, fmt.Errorf("Windows 自动更新包不存在")
	}
	sha, size, err := fileSHA256AndSize(updatePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("读取 Windows 自动更新包失败")
	}
	return updatePath, sha, size, nil
}

func selectAppBuildDownloadFile(dir, platform, kind string) (string, error) {
	if platform == "windows" {
		subdir := "install"
		ext := ".exe"
		if kind == "update" {
			subdir = "update"
			ext = ".zip"
		}
		if file, err := firstFileWithExt(filepath.Join(dir, subdir), ext); err == nil {
			return file, nil
		}
	}
	if platform == "android" {
		return firstFileWithExt(dir, ".apk")
	}
	if platform == "windows" && kind == "update" {
		return firstFileWithExt(dir, ".zip")
	}
	if platform == "windows" {
		if file, err := firstFileWithExt(dir, ".exe"); err == nil {
			return file, nil
		}
	}
	return firstRegularFile(dir)
}

func firstFileWithExt(dir, ext string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ext) {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", os.ErrNotExist
}

func firstRegularFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", os.ErrNotExist
}

func fileSHA256AndSize(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), size, nil
}

func writeZipDir(w io.Writer, root string) error {
	zw := zip.NewWriter(w)
	defer zw.Close()
	root = filepath.Clean(root)
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		dst, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(dst, src)
		return err
	})
}
