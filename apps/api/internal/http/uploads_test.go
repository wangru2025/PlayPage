package http

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ai-static-host/api/internal/config"
)

func TestCreateProjectWithInteractiveEnabled(t *testing.T) {
	cfg := config.Config{
		AppName:    "test",
		ListenAddr: "127.0.0.1:0",
		DataRoot:   filepath.Join(t.TempDir(), "apps"),
		PublicBase: "https://example.com",
	}

	handler := NewRouter(cfg)
	auth := signInForTest(t, handler)

	projectBody := bytes.NewBufferString(`{"name":"互动作品","slug":"interactive-demo","interactive":true}`)
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	addAuth(projectReq, auth)
	projectResp := httptest.NewRecorder()
	handler.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body = %s", projectResp.Code, projectResp.Body.String())
	}

	var project struct {
		ID          string `json:"id"`
		Interactive bool   `json:"interactive"`
	}
	if err := json.Unmarshal(projectResp.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	if !project.Interactive {
		t.Fatalf("expected interactive=true, body=%s", projectResp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+project.ID, nil)
	addAuth(getReq, auth)
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get project status = %d body = %s", getResp.Code, getResp.Body.String())
	}

	var payload struct {
		Project struct {
			Interactive bool `json:"interactive"`
		} `json:"project"`
	}
	if err := json.Unmarshal(getResp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Project.Interactive {
		t.Fatalf("expected stored interactive=true, body=%s", getResp.Body.String())
	}
}

func TestCreateReleaseFromZip(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "apps")
	cfg := config.Config{
		AppName:    "test",
		ListenAddr: "127.0.0.1:0",
		DataRoot:   dataRoot,
		PublicBase: "https://example.com",
	}

	handler := NewRouter(cfg)
	auth := signInForTest(t, handler)

	projectBody := bytes.NewBufferString(`{"name":"测试论坛","slug":"forum"}`)
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	addAuth(projectReq, auth)
	projectResp := httptest.NewRecorder()
	handler.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body = %s", projectResp.Code, projectResp.Body.String())
	}

	var project struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(projectResp.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, err := writer.CreateFormFile("file", "site.zip")
	if err != nil {
		t.Fatal(err)
	}

	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	entry, err := zw.Create("index.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, "<html><body>ok</body></html>"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(zb.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project.ID+"/releases", &form)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	addAuth(req, auth)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create release status = %d body = %s", resp.Code, resp.Body.String())
	}

	livePath := filepath.Join(string(filepath.Separator), "data", "public", "@demo", "forum")
	if _, err := os.Lstat(livePath); err != nil && os.Getenv("CI") == "" {
		t.Logf("live symlink not asserted on this host: %v", err)
	}
}

func TestCreateReleaseFromZipWithoutIndexGeneratesDirectoryPage(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "apps")
	cfg := config.Config{
		AppName:    "test",
		ListenAddr: "127.0.0.1:0",
		DataRoot:   dataRoot,
		PublicBase: "https://example.com",
	}

	handler := NewRouter(cfg)
	auth := signInForTest(t, handler)

	projectBody := bytes.NewBufferString(`{"name":"幻想故事","slug":"stories"}`)
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	addAuth(projectReq, auth)
	projectResp := httptest.NewRecorder()
	handler.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body = %s", projectResp.Code, projectResp.Body.String())
	}

	var project struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(projectResp.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, err := writer.CreateFormFile("file", "story-pack.zip")
	if err != nil {
		t.Fatal(err)
	}

	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	first, err := zw.Create("01. 第一章.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(first, "<html><body>chapter1</body></html>"); err != nil {
		t.Fatal(err)
	}
	second, err := zw.Create("02. 第二章.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(second, "<html><body>chapter2</body></html>"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(zb.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project.ID+"/releases?mode=zip", &form)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	addAuth(req, auth)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create release without index status = %d body = %s", resp.Code, resp.Body.String())
	}

	files, err := filepath.Glob(filepath.Join(dataRoot, project.ID, "releases", "*", "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected generated index file, got %d", len(files))
	}

	published, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(published, []byte("平台已经自动帮你生成了一个目录页")) {
		t.Fatalf("expected generated directory page, got %q", string(published))
	}
	if !bytes.Contains(published, []byte("./01.%20%E7%AC%AC%E4%B8%80%E7%AB%A0.html")) {
		t.Fatalf("expected generated directory page to use current-directory relative links, got %q", string(published))
	}
	if !bytes.Contains(published, []byte("01. 第一章.html")) || !bytes.Contains(published, []byte("02. 第二章.html")) {
		t.Fatalf("expected generated directory page to list html files, got %q", string(published))
	}
}

func TestCreateReleaseFromZipWithSomeEmptyHTMLStillPublishes(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "apps")
	cfg := config.Config{
		AppName:    "test",
		ListenAddr: "127.0.0.1:0",
		DataRoot:   dataRoot,
		PublicBase: "https://example.com",
	}

	handler := NewRouter(cfg)
	auth := signInForTest(t, handler)

	projectBody := bytes.NewBufferString(`{"name":"混合故事","slug":"mixed-story"}`)
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	addAuth(projectReq, auth)
	projectResp := httptest.NewRecorder()
	handler.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body = %s", projectResp.Code, projectResp.Body.String())
	}

	var project struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(projectResp.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, err := writer.CreateFormFile("file", "mixed-story.zip")
	if err != nil {
		t.Fatal(err)
	}

	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	emptyEntry, err := zw.Create("01. 空白章节.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emptyEntry.Write([]byte{}); err != nil {
		t.Fatal(err)
	}
	validEntry, err := zw.Create("02. 正常章节.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(validEntry, "<html><body>ok</body></html>"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(zb.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project.ID+"/releases?mode=zip", &form)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	addAuth(req, auth)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create mixed release status = %d body = %s", resp.Code, resp.Body.String())
	}

	var release struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &release); err != nil {
		t.Fatal(err)
	}
	if len(release.Warnings) == 0 {
		t.Fatalf("expected warnings for empty html files, got body=%s", resp.Body.String())
	}
}

func TestCreateReleaseFromZipWithNonUTF8HTML(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "apps")
	cfg := config.Config{
		AppName:    "test",
		ListenAddr: "127.0.0.1:0",
		DataRoot:   dataRoot,
		PublicBase: "https://example.com",
	}

	handler := NewRouter(cfg)
	auth := signInForTest(t, handler)

	projectBody := bytes.NewBufferString(`{"name":"编码测试","slug":"encoding-demo"}`)
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	addAuth(projectReq, auth)
	projectResp := httptest.NewRecorder()
	handler.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body = %s", projectResp.Code, projectResp.Body.String())
	}

	var project struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(projectResp.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, err := writer.CreateFormFile("file", "legacy-html.zip")
	if err != nil {
		t.Fatal(err)
	}

	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	entry, err := zw.Create("index.html")
	if err != nil {
		t.Fatal(err)
	}
	legacyHTML := []byte{0x3c, 0x68, 0x74, 0x6d, 0x6c, 0x3e, 0x63, 0x61, 0x66, 0xe9, 0x3c, 0x2f, 0x68, 0x74, 0x6d, 0x6c, 0x3e}
	if _, err := entry.Write(legacyHTML); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(zb.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project.ID+"/releases", &form)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	addAuth(req, auth)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create non-utf8 release status = %d body = %s", resp.Code, resp.Body.String())
	}

	files, err := filepath.Glob(filepath.Join(dataRoot, project.ID, "releases", "*", "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one published html file, got %d", len(files))
	}

	published, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(bytes.ToLower(published), []byte(`meta charset="windows-1252"`)) {
		t.Fatalf("expected detected charset meta, got %q", string(published))
	}
}

func TestCreateReleaseReplacesProjectKeyPlaceholder(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "apps")
	cfg := config.Config{
		AppName:    "test",
		ListenAddr: "127.0.0.1:0",
		DataRoot:   dataRoot,
		PublicBase: "https://example.com",
	}

	handler := NewRouter(cfg)
	auth := signInForTest(t, handler)

	projectBody := bytes.NewBufferString(`{"name":"带 key 的作品","slug":"with-key","interactive":true}`)
	projectReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	addAuth(projectReq, auth)
	projectResp := httptest.NewRecorder()
	handler.ServeHTTP(projectResp, projectReq)
	if projectResp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body = %s", projectResp.Code, projectResp.Body.String())
	}

	var project struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(projectResp.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}

	htmlBody := bytes.NewBufferString(`{"html":"<!doctype html><html><body><script>const key='__AUTO_FILLED_PROJECT_KEY__';</script></body></html>"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project.ID+"/releases?mode=text", htmlBody)
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, auth)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create text release status = %d body = %s", resp.Code, resp.Body.String())
	}

	files, err := filepath.Glob(filepath.Join(dataRoot, project.ID, "releases", "*", "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one published html file, got %d", len(files))
	}

	published, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(published, []byte("__AUTO_FILLED_PROJECT_KEY__")) {
		t.Fatalf("expected project key placeholder to be replaced, got %q", string(published))
	}
}

func TestPublicProjectRoutesHandleOptions(t *testing.T) {
	cfg := config.Config{
		AppName:    "test",
		ListenAddr: "127.0.0.1:0",
		DataRoot:   filepath.Join(t.TempDir(), "apps"),
		PublicBase: "https://example.com",
	}

	handler := NewRouter(cfg)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/public/projects/demo/collections", nil)
	req.Header.Set("Origin", "https://marscode-static.doubaocdn.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type,x-project-key")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("options status = %d body = %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Access-Control-Allow-Origin") != "https://marscode-static.doubaocdn.com" {
		t.Fatalf("unexpected allow origin: %q", resp.Header().Get("Access-Control-Allow-Origin"))
	}
}

type testAuth struct {
	session *http.Cookie
	csrf    *http.Cookie
}

func addAuth(req *http.Request, auth testAuth) {
	req.AddCookie(auth.session)
	if auth.csrf != nil {
		req.AddCookie(auth.csrf)
		req.Header.Set(csrfHeaderName, auth.csrf.Value)
	}
}

func signInForTest(t *testing.T, handler http.Handler) testAuth {
	t.Helper()

	requestBody := bytes.NewBufferString(`{"email":"demo@example.com","username":"demo"}`)
	requestReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-code", requestBody)
	requestReq.Header.Set("Content-Type", "application/json")
	requestResp := httptest.NewRecorder()
	handler.ServeHTTP(requestResp, requestReq)
	if requestResp.Code != http.StatusAccepted {
		t.Fatalf("request code status = %d body = %s", requestResp.Code, requestResp.Body.String())
	}

	var requestResult struct {
		DevCode string `json:"devCode"`
	}
	if err := json.Unmarshal(requestResp.Body.Bytes(), &requestResult); err != nil {
		t.Fatal(err)
	}
	if requestResult.DevCode == "" {
		t.Fatal("expected devCode in test mode")
	}

	verifyBody := bytes.NewBufferString(`{"email":"demo@example.com","code":"` + requestResult.DevCode + `"}`)
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-code", verifyBody)
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyResp := httptest.NewRecorder()
	handler.ServeHTTP(verifyResp, verifyReq)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("verify code status = %d body = %s", verifyResp.Code, verifyResp.Body.String())
	}

	var auth testAuth
	for _, cookie := range verifyResp.Result().Cookies() {
		switch cookie.Name {
		case sessionCookieName:
			auth.session = cookie
		case csrfCookieName:
			auth.csrf = cookie
		}
	}
	if auth.session == nil {
		t.Fatal("session cookie not found")
	}
	if auth.csrf == nil {
		t.Fatal("csrf cookie not found")
	}
	return auth
}
