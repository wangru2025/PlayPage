package http

import (
	"archive/zip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ai-static-host/api/internal/domain"
)

type releaseFile struct {
	absolute string
	relative string
}

func (rt *Router) handleDeleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	user, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	if project.Visibility == "public" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先把作品从广场隐藏，再删除作品"})
		return
	}

	deleted, err := rt.store.DeleteProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "删除作品失败"})
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个作品"})
		return
	}

	liveDir := rt.releases.LivePublicDir(project.Username, project.Slug)
	_ = os.Remove(liveDir)
	_ = os.RemoveAll(liveDir)
	_ = os.RemoveAll(rt.releases.ProjectDir(projectID))

	parent := filepath.Dir(liveDir)
	if entries, err := os.ReadDir(parent); err == nil && len(entries) == 0 {
		_ = os.Remove(parent)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (rt *Router) handleDownloadProjectSource(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	if project.CurrentRelease == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品还没有可下载的发布内容"})
		return
	}

	releases, err := rt.store.ListReleases(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取发布版本失败"})
		return
	}

	var current domain.Release
	found := false
	for _, release := range releases {
		if release.ID == project.CurrentRelease {
			current = release
			found = true
			break
		}
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到当前发布版本"})
		return
	}

	files, err := collectReleaseFiles(current.PublicPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品文件失败"})
		return
	}
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "当前作品没有可下载的文件"})
		return
	}

	timestamp := current.CreatedAt.Local().Format("20060102-150405")
	if len(files) == 1 {
		rt.downloadSingleReleaseFile(w, r, project.Name, timestamp, files[0])
		return
	}

	rt.downloadReleaseZip(w, project.Name, timestamp, files)
}

func collectReleaseFiles(root string) ([]releaseFile, error) {
	items := make([]releaseFile, 0)
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
		items = append(items, releaseFile{absolute: path, relative: filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].relative < items[j].relative
	})
	return items, nil
}

func (rt *Router) downloadSingleReleaseFile(w http.ResponseWriter, r *http.Request, projectName, timestamp string, file releaseFile) {
	ext := filepath.Ext(file.relative)
	downloadName := projectName + "-" + timestamp + ext
	setAttachmentHeaders(w, downloadName)
	if contentType := mime.TypeByExtension(strings.ToLower(ext)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	http.ServeFile(w, r, file.absolute)
}

func (rt *Router) downloadReleaseZip(w http.ResponseWriter, projectName, timestamp string, files []releaseFile) {
	downloadName := projectName + "-" + timestamp + ".zip"
	setAttachmentHeaders(w, downloadName)
	w.Header().Set("Content-Type", "application/zip")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, file := range files {
		src, err := os.Open(file.absolute)
		if err != nil {
			http.Error(w, "读取作品文件失败", http.StatusInternalServerError)
			return
		}

		entry, err := zipWriter.Create(file.relative)
		if err != nil {
			_ = src.Close()
			http.Error(w, "生成压缩包失败", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(entry, src); err != nil {
			_ = src.Close()
			http.Error(w, "写入压缩包失败", http.StatusInternalServerError)
			return
		}
		_ = src.Close()
	}
}

func setAttachmentHeaders(w http.ResponseWriter, filename string) {
	fallback := sanitizeDownloadName(filename)
	w.Header().Set("Content-Disposition", fmt.Sprintf(
		`attachment; filename="%s"; filename*=UTF-8''%s`,
		fallback,
		url.PathEscape(filename),
	))
}

func sanitizeDownloadName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	if strings.TrimSpace(name) == "" {
		return "download"
	}
	return name
}
