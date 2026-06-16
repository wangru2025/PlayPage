package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"log"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
	"ai-static-host/api/internal/storage"
	"golang.org/x/net/html/charset"
)

var allowedStaticExtensions = map[string]bool{
	".html": true,
	".htm":  true,
	".css":  true,
	".js":   true,
	".json": true,
	".txt":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".mp3":  true,
	".wav":  true,
	".ogg":  true,
	".m4a":  true,
}

const maxUploadBytes = 10 << 20
const maxArchiveEntries = 200

type Publisher struct {
	store  storage.Store
	layout *storage.ReleaseLayout
}

type releaseValidationReport struct {
	Warnings []string
}

const (
	projectKeyPlaceholder = "__AUTO_FILLED_PROJECT_KEY__"
	projectIDPlaceholder  = "__PROJECT_ID__"
)

func NewPublisher(store storage.Store, layout *storage.ReleaseLayout) *Publisher {
	return &Publisher{store: store, layout: layout}
}

func (p *Publisher) PublishZip(ctx context.Context, project domain.Project, file multipart.File, filename string) (domain.Release, error) {
	releaseKey, archiveDir, releaseDir, liveDir, archivePath, err := p.prepareRelease(project, filename)
	if err != nil {
		log.Printf("publish_zip_error stage=prepare_release project_id=%s project_name=%q filename=%q error=%q", project.ID, project.Name, filename, err.Error())
		return domain.Release{}, err
	}
	log.Printf("publish_zip_start project_id=%s project_name=%q release_key=%s filename=%q archive_path=%q release_dir=%q", project.ID, project.Name, releaseKey, filename, archivePath, releaseDir)

	if err := writeUpload(file, archivePath, maxUploadBytes); err != nil {
		log.Printf("publish_zip_error stage=write_upload project_id=%s release_key=%s archive_path=%q error=%q", project.ID, releaseKey, archivePath, err.Error())
		return domain.Release{}, err
	}
	if err := unzipToDir(archivePath, releaseDir); err != nil {
		log.Printf("publish_zip_error stage=unzip project_id=%s release_key=%s archive_path=%q error=%q", project.ID, releaseKey, archivePath, err.Error())
		return domain.Release{}, err
	}

	return p.finishRelease(ctx, project, releaseKey, archiveDir, releaseDir, liveDir, archivePath)
}

func (p *Publisher) PublishSingleHTML(ctx context.Context, project domain.Project, filename string, body []byte) (domain.Release, error) {
	releaseKey, archiveDir, releaseDir, liveDir, archivePath, err := p.prepareRelease(project, filename)
	if err != nil {
		return domain.Release{}, err
	}
	if len(body) == 0 {
		return domain.Release{}, fmt.Errorf("HTML 内容不能为空")
	}
	if len(body) > maxUploadBytes {
		return domain.Release{}, fmt.Errorf("上传内容不能超过 10MB")
	}
	if err := validateHTMLDocument(body); err != nil {
		return domain.Release{}, err
	}

	if err := os.WriteFile(archivePath, body, 0o644); err != nil {
		return domain.Release{}, err
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "index.html"), body, 0o644); err != nil {
		return domain.Release{}, err
	}

	return p.finishRelease(ctx, project, releaseKey, archiveDir, releaseDir, liveDir, archivePath)
}

func (p *Publisher) PublishHTMLText(ctx context.Context, project domain.Project, body string) (domain.Release, error) {
	return p.PublishSingleHTML(ctx, project, "index.txt", []byte(body))
}

func (p *Publisher) PublishPatchedHTML(ctx context.Context, project domain.Project, sourcePublicDir, entryFile string, body []byte) (domain.Release, error) {
	releaseKey, archiveDir, releaseDir, liveDir, archivePath, err := p.prepareRelease(project, "ai-repair.html")
	if err != nil {
		return domain.Release{}, err
	}
	_ = archiveDir
	if err := copyStaticDir(sourcePublicDir, releaseDir); err != nil {
		return domain.Release{}, err
	}
	if entryFile == "" {
		entryFile = "index.html"
	}
	if err := validateHTMLDocument(body); err != nil {
		return domain.Release{}, err
	}
	if err := os.WriteFile(filepath.Join(releaseDir, entryFile), body, 0o644); err != nil {
		return domain.Release{}, err
	}
	if err := os.WriteFile(archivePath, body, 0o644); err != nil {
		return domain.Release{}, err
	}
	return p.finishRelease(ctx, project, releaseKey, archiveDir, releaseDir, liveDir, archivePath)
}

func (p *Publisher) CreatePreview(project domain.Project, sourcePublicDir, entryFile, jobID string, body []byte) (string, error) {
	if strings.TrimSpace(jobID) == "" {
		return "", fmt.Errorf("预览任务 ID 不能为空")
	}
	previewDir := p.layout.PreviewDir(jobID)
	if err := os.RemoveAll(previewDir); err != nil {
		return "", err
	}
	if err := copyStaticDir(sourcePublicDir, previewDir); err != nil {
		return "", err
	}
	if entryFile == "" {
		entryFile = "index.html"
	}
	if err := validateHTMLDocument(body); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(previewDir, entryFile), body, 0o644); err != nil {
		return "", err
	}
	if err := ensureHTMLCharset(previewDir); err != nil {
		return "", err
	}
	return "_previews/" + jobID + "/", nil
}

func copyStaticDir(sourceDir, targetDir string) error {
	if strings.TrimSpace(sourceDir) == "" {
		return fmt.Errorf("源目录不能为空")
	}
	sourceAbs, err := filepath.Abs(sourceDir)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}
	if sourceAbs == targetAbs {
		return fmt.Errorf("源目录和目标目录不能相同")
	}
	return filepath.WalkDir(sourceAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceAbs, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(targetAbs, 0o755)
		}
		if strings.HasPrefix(rel, "..") {
			return fmt.Errorf("文件路径越界")
		}
		dest := filepath.Join(targetAbs, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("不允许复制软链接")
		}
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		if err := ensureAllowedStaticFile(rel); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func (p *Publisher) EnsureLivePublicLink(project domain.Project, publicDir string) error {
	liveDir := p.layout.LivePublicDir(project.Username, project.Slug)
	if err := os.MkdirAll(filepath.Dir(liveDir), 0o755); err != nil {
		return err
	}

	_ = os.Remove(liveDir)
	_ = os.RemoveAll(liveDir)
	return os.Symlink(publicDir, liveDir)
}

func (p *Publisher) prepareRelease(project domain.Project, filename string) (string, string, string, string, string, error) {
	releaseKey := "rel_" + time.Now().UTC().Format("20060102T150405")
	archiveDir := p.layout.UploadsDir(project.ID)
	releaseDir := p.layout.PublicDir(project.ID, releaseKey)
	liveDir := p.layout.LivePublicDir(project.Username, project.Slug)
	archivePath := filepath.Join(archiveDir, strings.ReplaceAll(releaseKey+"-"+sanitizeFileName(filename), " ", "-"))

	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return "", "", "", "", "", err
	}
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return "", "", "", "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(liveDir), 0o755); err != nil {
		return "", "", "", "", "", err
	}

	return releaseKey, archiveDir, releaseDir, liveDir, archivePath, nil
}

func (p *Publisher) finishRelease(
	ctx context.Context,
	project domain.Project,
	_ string,
	_ string,
	releaseDir string,
	liveDir string,
	archivePath string,
) (domain.Release, error) {
	report, err := validateReleaseTree(releaseDir)
	if err != nil {
		log.Printf("publish_release_error stage=validate_tree project_id=%s release_dir=%q archive_path=%q error=%q", project.ID, releaseDir, archivePath, err.Error())
		return domain.Release{}, err
	}
	if err := ensureReleaseEntryFile(releaseDir, project.Name); err != nil {
		log.Printf("publish_release_error stage=ensure_entry project_id=%s release_dir=%q archive_path=%q error=%q", project.ID, releaseDir, archivePath, err.Error())
		return domain.Release{}, err
	}
	if err := p.replaceProjectKeyPlaceholders(ctx, project.ID, releaseDir); err != nil {
		log.Printf("publish_release_error stage=replace_project_key project_id=%s release_dir=%q archive_path=%q error=%q", project.ID, releaseDir, archivePath, err.Error())
		return domain.Release{}, err
	}
	if err := ensureHTMLCharset(releaseDir); err != nil {
		log.Printf("publish_release_error stage=ensure_charset project_id=%s release_dir=%q archive_path=%q error=%q", project.ID, releaseDir, archivePath, err.Error())
		return domain.Release{}, err
	}
	if project.AnalyticsEnabled {
		if err := injectPlayPageAnalytics(releaseDir, project.ID); err != nil {
			log.Printf("publish_release_error stage=inject_analytics project_id=%s release_dir=%q archive_path=%q error=%q", project.ID, releaseDir, archivePath, err.Error())
			return domain.Release{}, err
		}
	}

	entryFile := detectEntryFile(releaseDir)
	if entryFile == "" {
		log.Printf("publish_release_error stage=detect_entry project_id=%s release_dir=%q archive_path=%q error=%q", project.ID, releaseDir, archivePath, "missing entry file")
		return domain.Release{}, fmt.Errorf("上传内容里缺少 index.html 或 index.htm")
	}

	_ = os.Remove(liveDir)
	_ = os.RemoveAll(liveDir)
	if err := os.Symlink(releaseDir, liveDir); err != nil {
		log.Printf("publish_release_error stage=symlink_live project_id=%s live_dir=%q release_dir=%q error=%q", project.ID, liveDir, releaseDir, err.Error())
		return domain.Release{}, err
	}

	release := domain.Release{
		ProjectID:   project.ID,
		Status:      "ready",
		ArchivePath: archivePath,
		PublicPath:  releaseDir,
		EntryFile:   entryFile,
		Warnings:    report.Warnings,
	}

	created, err := p.store.CreateRelease(ctx, release)
	if err != nil {
		log.Printf("publish_release_error stage=create_release_record project_id=%s archive_path=%q release_dir=%q error=%q", project.ID, archivePath, releaseDir, err.Error())
		return domain.Release{}, err
	}
	log.Printf("publish_release_ok project_id=%s release_id=%s entry_file=%q archive_path=%q release_dir=%q live_dir=%q", project.ID, created.ID, created.EntryFile, archivePath, releaseDir, liveDir)
	return created, nil
}

func writeUpload(file multipart.File, archivePath string, maxBytes int64) error {
	dst, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(file, maxBytes+1))
	if err != nil {
		return err
	}
	if written > maxBytes {
		return fmt.Errorf("上传文件不能超过 10MB")
	}
	return nil
}

func unzipToDir(archivePath, targetDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if len(reader.File) > maxArchiveEntries {
		return fmt.Errorf("压缩包里的文件数量不能超过 %d 个", maxArchiveEntries)
	}

	var totalUncompressed uint64
	for _, file := range reader.File {
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("压缩包里不能包含软链接")
		}
		if file.UncompressedSize64 > uint64(maxUploadBytes) {
			return fmt.Errorf("压缩包里的单个文件不能超过 10MB")
		}
		totalUncompressed += file.UncompressedSize64
		if totalUncompressed > uint64(maxUploadBytes*3) {
			return fmt.Errorf("压缩包解压后的总大小不能超过 30MB")
		}

		name, err := safeArchivePath(file.Name)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := ensureAllowedStaticFile(name); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.Create(targetPath)
		if err != nil {
			_ = src.Close()
			return err
		}

		if _, err := io.Copy(dst, io.LimitReader(src, int64(file.UncompressedSize64)+1)); err != nil {
			_ = src.Close()
			_ = dst.Close()
			return err
		}

		_ = src.Close()
		if err := dst.Close(); err != nil {
			return err
		}
	}

	return nil
}

func validateReleaseTree(root string) (releaseValidationReport, error) {
	var htmlCount int
	var nonEmptyHTMLCount int
	emptyHTMLFiles := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if _, err := safeArchivePath(rel); err != nil {
			return err
		}
		if err := ensureAllowedStaticFile(rel); err != nil {
			return err
		}
		if strings.EqualFold(filepath.Ext(rel), ".html") || strings.EqualFold(filepath.Ext(rel), ".htm") {
			htmlCount += 1
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if len(data) == 0 {
				emptyHTMLFiles = append(emptyHTMLFiles, rel)
				return nil
			}
			nonEmptyHTMLCount += 1
			return validateHTMLDocument(data)
		}
		return nil
	})
	if err != nil {
		return releaseValidationReport{}, err
	}
	if htmlCount > 0 && nonEmptyHTMLCount == 0 {
		return releaseValidationReport{}, fmt.Errorf("压缩包里的 HTML 文件全部都是空的")
	}
	report := releaseValidationReport{}
	if len(emptyHTMLFiles) > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("检测到 %d 个空的 HTML 文件，平台已忽略它们并继续发布。", len(emptyHTMLFiles)))
	}
	return report, nil
}

func safeArchivePath(name string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(name)))
	switch {
	case clean == "", clean == ".", clean == "/":
		return "", fmt.Errorf("压缩包里有无效路径")
	case strings.HasPrefix(clean, "/"):
		return "", fmt.Errorf("压缩包里有不安全的路径")
	case strings.Contains(clean, "../"):
		return "", fmt.Errorf("压缩包里有不安全的路径")
	case clean == "..":
		return "", fmt.Errorf("压缩包里有不安全的路径")
	case hasUnsafeControlChars(clean):
		return "", fmt.Errorf("文件路径里不能包含控制字符")
	default:
		return filepath.FromSlash(clean), nil
	}
}

func ensureAllowedStaticFile(name string) error {
	ext := strings.ToLower(filepath.Ext(name))
	if !allowedStaticExtensions[ext] {
		return fmt.Errorf("暂不支持这个文件类型：%s", ext)
	}
	return nil
}

func detectEntryFile(root string) string {
	switch {
	case fileExists(filepath.Join(root, "index.html")):
		return "index.html"
	case fileExists(filepath.Join(root, "index.htm")):
		return "index.htm"
	default:
		return ""
	}
}

func ensureReleaseEntryFile(root, projectName string) error {
	if detectEntryFile(root) != "" {
		return nil
	}

	htmlFiles, err := listTopLevelHTMLFiles(root)
	if err != nil {
		return err
	}
	if len(htmlFiles) == 0 {
		return fmt.Errorf("压缩包里没有可以打开的 HTML 文件")
	}

	index := buildGeneratedIndexHTML(projectName, htmlFiles)
	return os.WriteFile(filepath.Join(root, "index.html"), []byte(index), 0o644)
}

func listTopLevelHTMLFiles(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	items := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".html" && ext != ".htm" {
			continue
		}
		if strings.EqualFold(entry.Name(), "index.html") || strings.EqualFold(entry.Name(), "index.htm") {
			continue
		}
		items = append(items, entry.Name())
	}
	sort.Strings(items)
	return items, nil
}

func buildGeneratedIndexHTML(projectName string, files []string) string {
	title := strings.TrimSpace(projectName)
	if title == "" {
		title = "作品目录"
	}

	var items strings.Builder
	for _, name := range files {
		label := html.EscapeString(name)
		href := html.EscapeString("./" + url.PathEscape(name))
		items.WriteString(`<li><a href="`)
		items.WriteString(href)
		items.WriteString(`">`)
		items.WriteString(label)
		items.WriteString(`</a></li>`)
	}

	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + html.EscapeString(title) + `</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f8f3ea;
      --card: #fffaf3;
      --line: #dfcfba;
      --text: #2b2118;
      --muted: #6f6255;
      --brand: #b85c38;
      --shadow: 0 18px 40px rgba(73, 46, 20, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans SC", sans-serif;
      color: var(--text);
      background: linear-gradient(180deg, #f8f3ea 0%, #f3eadf 100%);
    }
    main {
      width: min(960px, calc(100% - 32px));
      margin: 0 auto;
      padding: 32px 0 48px;
      display: grid;
      gap: 20px;
    }
    .panel {
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 24px;
      box-shadow: var(--shadow);
      padding: 24px;
    }
    h1, p { margin: 0; }
    p { color: var(--muted); line-height: 1.7; }
    ul {
      margin: 0;
      padding-left: 20px;
      display: grid;
      gap: 10px;
    }
    a {
      color: var(--brand);
      text-decoration: none;
      word-break: break-all;
    }
    a:hover { text-decoration: underline; }
  </style>
</head>
<body>
  <main>
    <section class="panel">
      <h1>` + html.EscapeString(title) + `</h1>
      <p>这个压缩包里没有 index.html，平台已经自动帮你生成了一个目录页。你可以直接点下面的文件进入对应页面。</p>
    </section>
    <section class="panel">
      <ul>` + items.String() + `</ul>
    </section>
  </main>
</body>
</html>`
}

func injectPlayPageAnalytics(root, projectID string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".htm" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("data-playpage-visit-tracker")) {
			return nil
		}
		script := buildPlayPageAnalyticsScript(projectID)
		content := string(data)
		lower := strings.ToLower(content)
		switch {
		case strings.Contains(lower, "</head>"):
			idx := strings.Index(lower, "</head>")
			content = content[:idx] + script + content[idx:]
		case strings.Contains(lower, "</body>"):
			idx := strings.Index(lower, "</body>")
			content = content[:idx] + script + content[idx:]
		default:
			content += script
		}
		return os.WriteFile(path, []byte(content), 0o644)
	})
}

func buildPlayPageAnalyticsScript(projectID string) string {
	return `
<!-- PlayPage 访问量统计：你在创建作品时已启用此功能。它只记录作品访问次数和互动 API 请求统计，不读取页面输入内容、密码或互动数据。 -->
<script data-playpage-visit-tracker>
(function () {
  try {
    var url = '/api/v1/public/projects/` + projectID + `/visit';
    var body = JSON.stringify({ path: location.pathname, title: document.title || '' });
    if (navigator.sendBeacon) {
      navigator.sendBeacon(url, new Blob([body], { type: 'application/json' }));
      return;
    }
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body,
      keepalive: true
    }).catch(function () {});
  } catch (error) {}
})();
</script>
`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	if name == "." || name == "" {
		return "upload.bin"
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if name == "" {
		return "upload.bin"
	}
	return name
}

func ensureHTMLCharset(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".htm" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte{0x00}) {
			return nil
		}
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "charset=") {
			return nil
		}

		charsetName := detectHTMLCharsetName(data)
		inject := `<meta charset="` + charsetName + `">`
		switch {
		case strings.Contains(lower, "<head>"):
			content := strings.Replace(string(data), "<head>", "<head>"+inject, 1)
			return os.WriteFile(path, []byte(content), 0o644)
		case strings.Contains(lower, "<html>"):
			content := strings.Replace(string(data), "<html>", "<html><head>"+inject+"</head>", 1)
			return os.WriteFile(path, []byte(content), 0o644)
		default:
			buffer := bytes.NewBufferString(inject)
			buffer.Write(data)
			return os.WriteFile(path, buffer.Bytes(), 0o644)
		}
	})
}

func validateHTMLDocument(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("HTML 内容不能为空")
	}
	if hasUnsafeControlBytes(data) {
		return fmt.Errorf("HTML 内容里不能包含危险控制字符")
	}
	return nil
}

func detectHTMLCharsetName(data []byte) string {
	if len(data) == 0 {
		return "utf-8"
	}
	_, name, _ := charset.DetermineEncoding(data, "text/html")
	if strings.TrimSpace(name) == "" {
		return "utf-8"
	}
	return strings.ToLower(name)
}

func (p *Publisher) replaceProjectKeyPlaceholders(ctx context.Context, projectID, root string) error {
	access, found, err := p.store.GetProjectPublicAccess(ctx, projectID)
	if err != nil {
		return err
	}
	projectKey := ""
	if found {
		projectKey = strings.TrimSpace(access.PublicKey)
	}

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".htm" && ext != ".js" && ext != ".json" && ext != ".txt" && ext != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Contains(data, []byte(projectIDPlaceholder)) && (projectKey == "" || !bytes.Contains(data, []byte(projectKeyPlaceholder))) {
			return nil
		}

		replaced := bytes.ReplaceAll(data, []byte(projectIDPlaceholder), []byte(projectID))
		if projectKey != "" {
			replaced = bytes.ReplaceAll(replaced, []byte(projectKeyPlaceholder), []byte(projectKey))
		}
		return os.WriteFile(path, replaced, 0o644)
	})
}

func hasUnsafeControlBytes(data []byte) bool {
	for _, b := range data {
		if b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		if b < 32 || b == 127 {
			return true
		}
	}
	return false
}

func hasUnsafeControlChars(value string) bool {
	for _, r := range value {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if r < 32 || r == 127 {
			return true
		}
	}
	return false
}
