package http

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (rt *Router) handleProjectDomainSite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "这个独立网址只支持访问网页内容"})
		return
	}

	host := normalizeRequestHost(r.Host)
	if host == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少访问域名"})
		return
	}

	_, project, found, err := rt.store.GetActiveProjectDomainAccessByDomain(r.Context(), host)
	if err != nil {
		writeProjectDomainGonePage(w, http.StatusInternalServerError, "独立网址暂时不可访问", "平台读取独立网址配置失败，请稍后再试。")
		return
	}
	if !found || project.CurrentRelease == "" {
		writeProjectDomainGonePage(w, http.StatusNotFound, "当前作品已被删除", "这个独立网址对应的作品已经被删除，或者暂时没有可访问的发布版本。")
		return
	}

	rel := strings.TrimPrefix(r.URL.Path, "/api/v1/domain-site")
	serveProjectDomainFile(w, r, rt.releases.LivePublicDir(project.Username, project.Slug), rel)
}

func normalizeRequestHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		if end := strings.LastIndex(host, "]"); end >= 0 {
			return host[:end+1]
		}
	}
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		return host[:idx]
	}
	return host
}

func serveProjectDomainFile(w http.ResponseWriter, r *http.Request, root, requested string) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品目录失败"})
		return
	}
	clean := path.Clean("/" + strings.TrimPrefix(requested, "/"))
	if clean == "/" || clean == "." {
		clean = "/index.html"
	}
	rel := strings.TrimPrefix(clean, "/")
	target := filepath.Join(rootAbs, filepath.FromSlash(rel))
	if !isPathInside(rootAbs, target) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "访问路径不正确"})
		return
	}
	if isDir(target) {
		target = filepath.Join(target, "index.html")
	}
	if !fileExists(target) {
		target = filepath.Join(rootAbs, "index.html")
	}
	http.ServeFile(w, r, target)
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && !stat.IsDir()
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func isPathInside(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func writeProjectDomainGonePage(w http.ResponseWriter, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + htmlEscape(title) + `</title>
<style>
body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif;margin:0;min-height:100vh;display:grid;place-items:center;background:#f8fafc;color:#172033}
main{max-width:680px;margin:24px;padding:28px;border:1px solid #e2e8f0;border-radius:24px;background:white;box-shadow:0 18px 60px rgba(15,23,42,.08)}
h1{margin:0 0 12px;font-size:1.8rem}
p{margin:0;color:#475569;line-height:1.8}
</style>
</head>
<body>
<main>
<h1>` + htmlEscape(title) + `</h1>
<p>` + htmlEscape(message) + `</p>
</main>
</body>
</html>`))
}

func htmlEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return strings.ReplaceAll(value, "'", "&#39;")
}
