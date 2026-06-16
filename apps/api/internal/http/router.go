package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode"

	"ai-static-host/api/internal/config"
	"ai-static-host/api/internal/domain"
	"ai-static-host/api/internal/service"
	"ai-static-host/api/internal/storage"
)

const sessionCookieName = "web_wangru_session"
const csrfCookieName = "web_wangru_csrf"
const csrfHeaderName = "X-CSRF-Token"
const csrfErrorCode = "csrf_token_invalid"

type Router struct {
	cfg      config.Config
	store    storage.Store
	releases *storage.ReleaseLayout
	pub      *service.Publisher
	mailer   *service.Mailer
	limiter  *authRateLimiter
	aiClient *service.AIClient
	aiHub    *repairAIHub
	aiRuns   *repairAIRunRegistry
}

func NewRouter(cfg config.Config) http.Handler {
	router := &Router{
		cfg:      cfg,
		store:    buildStore(cfg),
		releases: storage.NewReleaseLayout(cfg.DataRoot),
		limiter:  newAuthRateLimiter(),
		mailer:   service.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFromEmail),
		aiClient: service.NewAIClient(cfg.AIProviderBaseURL, cfg.AIProviderAPIKey, cfg.AIProviderModel),
		aiHub:    newRepairAIHub(),
		aiRuns:   newRepairAIRunRegistry(),
	}
	router.pub = service.NewPublisher(router.store, router.releases)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", router.handleHealth)
	mux.HandleFunc("GET /api/v1/me", router.handleMe)
	mux.HandleFunc("POST /api/v1/me/profile", router.handleUpdateProfile)
	mux.HandleFunc("POST /api/v1/me/upgrade-requests", router.handleCreateUpgradeRequest)
	mux.HandleFunc("GET /api/v1/me/upgrade-requests", router.handleListMyUpgradeRequests)
	mux.HandleFunc("POST /api/v1/client-errors", router.handleClientErrorReport)
	mux.HandleFunc("GET /api/v1/projects", router.handleListProjects)
	mux.HandleFunc("POST /api/v1/projects", router.handleCreateProject)
	mux.HandleFunc("GET /api/v1/admin/users", router.handleAdminListUsers)
	mux.HandleFunc("/api/v1/admin/users/", router.handleAdminUserRoutes)
	mux.HandleFunc("GET /api/v1/admin/projects", router.handleAdminListProjects)
	mux.HandleFunc("GET /api/v1/admin/upgrade-requests", router.handleAdminListUpgradeRequests)
	mux.HandleFunc("/api/v1/admin/upgrade-requests/", router.handleAdminRoutes)
	mux.HandleFunc("GET /api/v1/admin/project-domains", router.handleAdminListProjectDomains)
	mux.HandleFunc("/api/v1/admin/project-domains/", router.handleAdminProjectDomainRoutes)
	mux.HandleFunc("GET /api/v1/admin/repair-requests", router.handleAdminListRepairRequests)
	mux.HandleFunc("/api/v1/admin/repair-requests/", router.handleAdminRepairRequestRoutes)
	mux.HandleFunc("GET /api/v1/square", router.handleListSquare)
	mux.HandleFunc("/api/v1/public/projects/", router.handlePublicProjectRoutes)
	mux.HandleFunc("/api/v1/projects/", router.handleProjectRoutes)
	mux.HandleFunc("POST /api/v1/auth/request-code", router.handleRequestCode)
	mux.HandleFunc("POST /api/v1/auth/verify-code", router.handleVerifyCode)
	mux.HandleFunc("POST /api/v1/auth/logout", router.handleLogout)

	return withLogging(router.withCSRF(mux))
}

func (rt *Router) handleAdminRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/upgrade-requests/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "review" && r.Method == http.MethodPost {
		rt.handleAdminReviewUpgradeRequest(w, r, parts[0])
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
}

func (rt *Router) handleAdminUserRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 && r.Method == http.MethodPost {
		rt.handleAdminUpdateUser(w, r, parts[0])
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
}

func buildStore(cfg config.Config) storage.Store {
	if cfg.DatabaseURL == "" {
		return storage.NewMemoryStore(cfg.PublicBase)
	}

	store, err := storage.NewPostgresStore(context.Background(), cfg.DatabaseURL, cfg.PublicBase)
	if err != nil {
		log.Printf("postgres unavailable, falling back to memory store: %v", err)
		return storage.NewMemoryStore(cfg.PublicBase)
	}

	return store
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (rt *Router) handleProjectRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}

	projectID := parts[0]
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		rt.handleGetProject(w, r, projectID)
	case len(parts) == 1 && r.Method == http.MethodDelete:
		rt.handleDeleteProject(w, r, projectID)
	case len(parts) == 2 && parts[1] == "interactive-doc" && r.Method == http.MethodGet:
		rt.handleProjectInteractiveDoc(w, r, projectID)
	case len(parts) == 2 && parts[1] == "stats" && r.Method == http.MethodGet:
		rt.handleProjectStats(w, r, projectID)
	case len(parts) == 2 && parts[1] == "collections" && r.Method == http.MethodGet:
		rt.handleListCollections(w, r, projectID)
	case len(parts) == 2 && parts[1] == "collections" && r.Method == http.MethodPost:
		rt.handleCreateCollection(w, r, projectID)
	case len(parts) == 3 && parts[1] == "collections" && r.Method == http.MethodGet:
		rt.handleGetCollection(w, r, projectID, parts[2])
	case len(parts) == 3 && parts[1] == "collections" && (r.Method == http.MethodPut || r.Method == http.MethodPatch):
		rt.handleUpdateCollection(w, r, projectID, parts[2])
	case len(parts) == 3 && parts[1] == "collections" && r.Method == http.MethodDelete:
		rt.handleDeleteCollection(w, r, projectID, parts[2])
	case len(parts) == 2 && parts[1] == "domains" && r.Method == http.MethodGet:
		rt.handleListProjectDomains(w, r, projectID)
	case len(parts) == 2 && parts[1] == "domains" && r.Method == http.MethodPost:
		rt.handleCreateProjectDomain(w, r, projectID)
	case len(parts) == 2 && parts[1] == "repair-requests" && r.Method == http.MethodGet:
		rt.handleListProjectRepairRequests(w, r, projectID)
	case len(parts) == 2 && parts[1] == "repair-requests" && r.Method == http.MethodPost:
		rt.handleCreateRepairRequest(w, r, projectID)
	case len(parts) == 4 && parts[1] == "repair-requests" && parts[3] == "reply" && r.Method == http.MethodPost:
		rt.handleReplyRepairRequest(w, r, projectID, parts[2])
	case len(parts) == 5 && parts[1] == "repair-requests" && parts[3] == "ai" && parts[4] == "start" && r.Method == http.MethodPost:
		rt.handleStartRepairAI(w, r, projectID, parts[2], "")
	case len(parts) == 5 && parts[1] == "repair-requests" && parts[3] == "ai" && parts[4] == "stop" && r.Method == http.MethodPost:
		rt.handleStopRepairAI(w, r, projectID, parts[2])
	case len(parts) == 5 && parts[1] == "repair-requests" && parts[3] == "ai" && parts[4] == "feedback" && r.Method == http.MethodPost:
		rt.handleRepairAIFeedback(w, r, projectID, parts[2])
	case len(parts) == 5 && parts[1] == "repair-requests" && parts[3] == "ai" && parts[4] == "latest" && r.Method == http.MethodGet:
		rt.handleLatestRepairAI(w, r, projectID, parts[2])
	case len(parts) == 5 && parts[1] == "repair-requests" && parts[3] == "ai" && parts[4] == "ws" && r.Method == http.MethodGet:
		rt.handleRepairAIWebSocket(w, r, projectID, parts[2])
	case len(parts) == 5 && parts[1] == "repair-requests" && parts[3] == "ai" && parts[4] == "preview" && r.Method == http.MethodPost:
		rt.handleRepairAIPreview(w, r, projectID, parts[2])
	case len(parts) == 5 && parts[1] == "repair-requests" && parts[3] == "ai" && parts[4] == "publish" && r.Method == http.MethodPost:
		rt.handleRepairAIPublish(w, r, projectID, parts[2])
	case len(parts) == 2 && parts[1] == "source" && r.Method == http.MethodGet:
		rt.handleDownloadProjectSource(w, r, projectID)
	case len(parts) == 2 && parts[1] == "visibility" && r.Method == http.MethodPost:
		rt.handleUpdateProjectVisibility(w, r, projectID)
	case len(parts) == 2 && parts[1] == "path" && r.Method == http.MethodPost:
		rt.handleUpdateProjectPath(w, r, projectID)
	case len(parts) == 2 && parts[1] == "releases" && r.Method == http.MethodGet:
		rt.handleListReleases(w, r, projectID)
	case len(parts) == 2 && parts[1] == "releases" && r.Method == http.MethodPost:
		rt.handleCreateRelease(w, r, projectID)
	case len(parts) == 4 && parts[1] == "collections" && parts[3] == "records" && r.Method == http.MethodGet:
		rt.handleListRecords(w, r, projectID, parts[2])
	case len(parts) == 4 && parts[1] == "collections" && parts[3] == "records" && r.Method == http.MethodPost:
		rt.handleCreateRecord(w, r, projectID, parts[2])
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
	}
}

func (rt *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": rt.cfg.AppName,
	})
}

func (rt *Router) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok, err := rt.currentUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取登录状态失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "你还没有登录"})
		return
	}

	csrfToken, err := ensureCSRFCookie(w, r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "生成安全令牌失败"})
		return
	}

	w.Header().Set(csrfHeaderName, csrfToken)
	writeJSON(w, http.StatusOK, user)
}

func (rt *Router) handleRequestCode(w http.ResponseWriter, r *http.Request) {
	var input domain.AuthCodeRequestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Username = normalizePathSegment(input.Username)
	if input.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先填写邮箱"})
		return
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱地址格式不对"})
		return
	}

	code, err := generateDigits(6)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "生成验证码失败"})
		return
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	if err := rt.store.CreateOrRefreshAuthCode(r.Context(), input, code, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存验证码失败"})
		return
	}

	if err := rt.mailer.SendCode(input.Email, code); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "发送验证邮件失败"})
		return
	}

	response := map[string]any{
		"status":    "code-sent",
		"email":     input.Email,
		"expiresIn": 600,
	}
	if rt.mailer.IsNoop() {
		response["devCode"] = code
	}

	writeJSON(w, http.StatusAccepted, response)
}

func (rt *Router) handleVerifyCode(w http.ResponseWriter, r *http.Request) {
	var input domain.AuthCodeVerifyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Code = strings.TrimSpace(input.Code)
	if input.Email == "" || input.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱和验证码都要填"})
		return
	}

	user, ok, err := rt.store.ConsumeAuthCode(r.Context(), input, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "校验验证码失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "验证码不对，或者已经过期"})
		return
	}

	token, err := generateToken(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "创建登录会话失败"})
		return
	}

	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	if err := rt.store.CreateSession(r.Context(), user.ID, token, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存登录会话失败"})
		return
	}

	csrfToken, err := generateToken(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "生成安全令牌失败"})
		return
	}

	http.SetCookie(w, buildSessionCookie(token, expiresAt))
	http.SetCookie(w, buildCSRFCookie(csrfToken, expiresAt))
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "signed-in",
		"user":             user,
		"csrfToken":        csrfToken,
		"accessToken":      token,
		"tokenType":        "Bearer",
		"expiresAt":        expiresAt,
		"expiresInSeconds": int(time.Until(expiresAt).Seconds()),
	})
}

func (rt *Router) handleLogout(w http.ResponseWriter, r *http.Request) {
	if token := bearerToken(r); token != "" {
		_ = rt.store.DeleteSession(r.Context(), token)
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = rt.store.DeleteSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed-out"})
}

func (rt *Router) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}

	var input domain.UserProfileUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Username = normalizePathSegment(input.Username)
	if input.Username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "用户名不能为空"})
		return
	}

	updated, err := rt.store.UpdateUserUsername(r.Context(), user.ID, input.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存用户名失败"})
		return
	}
	if err := rt.store.SyncProjectUsernames(r.Context(), user.ID, updated.Username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "同步作品地址失败"})
		return
	}
	if shouldSendWelcomeAfterProfileUpdate(user, updated) {
		rt.notifyWelcomeUser(updated)
	}
	projects, err := rt.store.ListProjects(r.Context(), user.ID)
	if err == nil {
		for _, project := range projects {
			_ = rt.ensureProjectLiveLink(r.Context(), project)
		}
	}

	writeJSON(w, http.StatusOK, updated)
}

func (rt *Router) currentUser(r *http.Request) (domain.User, bool, error) {
	if token := bearerToken(r); token != "" {
		return rt.store.GetUserBySessionToken(r.Context(), token)
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return domain.User{}, false, nil
		}
		return domain.User{}, false, err
	}
	if cookie.Value == "" {
		return domain.User{}, false, nil
	}
	return rt.store.GetUserBySessionToken(r.Context(), cookie.Value)
}

func bearerToken(r *http.Request) string {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return ""
	}
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (rt *Router) requireUser(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	user, ok, err := rt.currentUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取登录状态失败"})
		return domain.User{}, false
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
		return domain.User{}, false
	}
	return user, true
}

func buildSessionCookie(token string, expiresAt time.Time) *http.Cookie {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   maxAge,
	}
}

func (rt *Router) withCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresCSRF(r) {
			next.ServeHTTP(w, r)
			return
		}
		if bearerToken(r) != "" {
			next.ServeHTTP(w, r)
			return
		}

		sessionCookie, err := r.Cookie(sessionCookieName)
		if err == http.ErrNoCookie || sessionCookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "安全令牌无效，请刷新页面后重试", "code": csrfErrorCode})
			return
		}

		csrfCookie, err := r.Cookie(csrfCookieName)
		if err != nil || csrfCookie.Value == "" || r.Header.Get(csrfHeaderName) == "" || csrfCookie.Value != r.Header.Get(csrfHeaderName) {
			if _, ok, userErr := rt.currentUser(r); userErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取登录状态失败"})
				return
			} else if ok {
				if token, tokenErr := ensureCSRFCookie(w, r); tokenErr == nil {
					w.Header().Set("X-CSRF-Token", token)
				}
			}
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "安全令牌已刷新，请重试", "code": csrfErrorCode})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func requiresCSRF(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/public/") {
		return false
	}
	if path == "/api/v1/auth/request-code" || path == "/api/v1/auth/verify-code" || path == "/api/v1/client-errors" {
		return false
	}
	return strings.HasPrefix(path, "/api/")
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) (string, error) {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	token, err := generateToken(32)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	http.SetCookie(w, buildCSRFCookie(token, expiresAt))
	return token, nil
}

func buildCSRFCookie(token string, expiresAt time.Time) *http.Cookie {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	return &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   maxAge,
	}
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

type authRateLimiter struct {
	mu             sync.Mutex
	emailRequests  map[string][]time.Time
	ipRequests     map[string][]time.Time
	verifyFailures map[string][]time.Time
}

func newAuthRateLimiter() *authRateLimiter {
	return &authRateLimiter{
		emailRequests:  map[string][]time.Time{},
		ipRequests:     map[string][]time.Time{},
		verifyFailures: map[string][]time.Time{},
	}
}

func (l *authRateLimiter) allowRequestCode(email, ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	emailKey := "email:" + email
	ipKey := "ip:" + ip
	l.emailRequests[emailKey] = pruneTimes(l.emailRequests[emailKey], now.Add(-time.Hour))
	l.ipRequests[ipKey] = pruneTimes(l.ipRequests[ipKey], now.Add(-time.Hour))
	if len(l.emailRequests[emailKey]) > 0 && now.Sub(l.emailRequests[emailKey][len(l.emailRequests[emailKey])-1]) < time.Minute {
		return false
	}
	if len(l.emailRequests[emailKey]) >= 5 || len(l.ipRequests[ipKey]) >= 30 {
		return false
	}
	l.emailRequests[emailKey] = append(l.emailRequests[emailKey], now)
	l.ipRequests[ipKey] = append(l.ipRequests[ipKey], now)
	return true
}

func (l *authRateLimiter) allowVerifyCode(email, ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := verifyLimiterKey(email, ip)
	l.verifyFailures[key] = pruneTimes(l.verifyFailures[key], now.Add(-15*time.Minute))
	return len(l.verifyFailures[key]) < 5
}

func (l *authRateLimiter) recordVerifyFailure(email, ip string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := verifyLimiterKey(email, ip)
	l.verifyFailures[key] = append(pruneTimes(l.verifyFailures[key], now.Add(-15*time.Minute)), now)
}

func (l *authRateLimiter) clearVerifyFailures(email, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.verifyFailures, verifyLimiterKey(email, ip))
}

func verifyLimiterKey(email, ip string) string {
	return email + "|" + ip
}

func pruneTimes(items []time.Time, cutoff time.Time) []time.Time {
	if len(items) == 0 {
		return items
	}
	kept := items[:0]
	for _, item := range items {
		if item.After(cutoff) {
			kept = append(kept, item)
		}
	}
	return kept
}

func generateDigits(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, item := range bytes {
		builder.WriteByte(byte('0' + (item % 10)))
	}
	return builder.String(), nil
}

func generateToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func inferUsernameFromEmail(email string) string {
	name := email
	if at := strings.Index(email, "@"); at > 0 {
		name = email[:at]
	}

	name = normalizePathSegment(name)
	if name == "" {
		return "新朋友"
	}
	return name
}

func normalizePathSegment(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Join(strings.Fields(value), "-")
	value = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r):
			return r
		case unicode.IsNumber(r):
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '-'
		}
	}, value)
	value = strings.Trim(value, "-._")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
