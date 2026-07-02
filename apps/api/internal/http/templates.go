package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

const maxTemplateSubmissionBytes = 12 << 20

func (rt *Router) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))
	category := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("category")))

	items := make([]domain.ProjectTemplate, 0, len(builtInTemplates))
	for _, item := range builtInTemplates {
		item = normalizeTemplateForJSON(item)
		if category != "" && item.Category != category {
			continue
		}
		if query != "" && !templateMatchesQuery(item, query) {
			continue
		}
		items = append(items, item)
	}
	submissions, err := rt.store.ListPublishedTemplateSubmissions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取模板失败"})
		return
	}
	for _, submission := range submissions {
		item := normalizeTemplateForJSON(templateFromSubmission(submission))
		if category != "" && item.Category != category {
			continue
		}
		if query != "" && !templateMatchesQuery(item, query) {
			continue
		}
		items = append(items, item)
	}

	categories := templateCategories(items)
	writeJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"categories": categories,
	})
}

func (rt *Router) handleTemplateRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/templates/"), "/")
	parts := strings.Split(path, "/")
	if path == "" || len(parts) > 2 || parts[0] == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个模板"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
		return
	}
	item, ok, err := rt.findTemplate(r.Context(), parts[0])
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取模板失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个模板"})
		return
	}
	if len(parts) == 2 {
		if parts[1] != "preview" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
			return
		}
		rt.handleTemplatePreview(w, r, item)
		return
	}
	item = normalizeTemplateForJSON(item)
	writeJSON(w, http.StatusOK, map[string]any{"template": item})
}

func (rt *Router) handleTemplatePreview(w http.ResponseWriter, r *http.Request, item domain.ProjectTemplate) {
	params := map[string]string{}
	for _, field := range item.ConfigFields {
		params[field.Name] = field.Default
	}
	project := domain.Project{ID: "template-preview", Name: item.Name, Slug: item.Slug, Interactive: item.InteractiveRequired}
	body := renderTemplateHTML(item, project, "template-preview-key", params, true)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func normalizeTemplateForJSON(item domain.ProjectTemplate) domain.ProjectTemplate {
	if item.AuthorName == "" {
		item.AuthorName = "官方模板"
	}
	if item.Source == "" {
		item.Source = "official"
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ConfigFields == nil {
		item.ConfigFields = []domain.TemplateConfigField{}
	}
	if item.Collections == nil {
		item.Collections = []domain.TemplateCollectionDefinition{}
	}
	return item
}

func templateMatchesQuery(item domain.ProjectTemplate, query string) bool {
	parts := []string{item.ID, item.Slug, item.Name, item.Category, item.CategoryLabel, item.Summary, item.Description}
	parts = append(parts, item.Tags...)
	return strings.Contains(strings.ToLower(strings.Join(parts, " ")), query)
}

func templateCategories(items []domain.ProjectTemplate) []map[string]string {
	seen := map[string]string{}
	for _, item := range items {
		if item.Category != "" {
			seen[item.Category] = item.CategoryLabel
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, map[string]string{"id": key, "label": seen[key]})
	}
	return result
}

func (rt *Router) findTemplate(ctx context.Context, idOrSlug string) (domain.ProjectTemplate, bool, error) {
	for _, item := range builtInTemplates {
		if item.ID == idOrSlug || item.Slug == idOrSlug {
			item.Source = "official"
			item.AuthorName = "官方模板"
			return item, true, nil
		}
	}
	submission, ok, err := rt.store.GetPublishedTemplateSubmissionByIDOrSlug(ctx, idOrSlug)
	if err != nil || !ok {
		return domain.ProjectTemplate{}, false, err
	}
	return templateFromSubmission(submission), true, nil
}

func templateFromSubmission(item domain.TemplateSubmission) domain.ProjectTemplate {
	return domain.ProjectTemplate{
		ID:                   "submission:" + item.ID,
		Slug:                 item.Slug,
		Name:                 item.Name,
		Category:             item.Category,
		CategoryLabel:        item.CategoryLabel,
		Description:          item.Description,
		Summary:              item.Summary,
		Tags:                 item.Tags,
		AuthorUserID:         item.AuthorUserID,
		AuthorName:           item.AuthorName,
		Source:               "community",
		InteractiveRequired:  item.InteractiveRequired,
		AnalyticsRecommended: item.AnalyticsRecommended,
		ConfigFields:         item.ConfigFields,
		Collections:          item.Collections,
		CreatedAtText:        item.CreatedAt.Format("2006-01-02"),
		HTMLSource:           item.HTMLSource,
	}
}

func (rt *Router) handleCreateTemplateSubmission(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTemplateSubmissionBytes)
	input, err := parseTemplateSubmissionInput(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	submission, err := rt.buildTemplateSubmission(user, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	created, err := rt.store.CreateTemplateSubmission(r.Context(), submission)
	if err != nil {
		var pgErr *pgconn.PgError
		if strings.Contains(err.Error(), "模板链接名") || strings.Contains(err.Error(), "duplicate") || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "这个模板链接名已经被使用"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "提交模板失败"})
		return
	}
	created.HTMLSource = ""
	writeJSON(w, http.StatusCreated, created)
}

func parseTemplateSubmissionInput(r *http.Request) (domain.TemplateSubmissionCreateInput, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		return parseMultipartTemplateSubmissionInput(r)
	}
	var input domain.TemplateSubmissionCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return input, fmt.Errorf("请求内容格式不正确")
	}
	return input, nil
}

func parseMultipartTemplateSubmissionInput(r *http.Request) (domain.TemplateSubmissionCreateInput, error) {
	var input domain.TemplateSubmissionCreateInput
	if err := r.ParseMultipartForm(maxTemplateSubmissionBytes); err != nil {
		return input, fmt.Errorf("上传表单无效")
	}
	input.Slug = r.FormValue("slug")
	input.Name = r.FormValue("name")
	input.Category = r.FormValue("category")
	input.CategoryLabel = r.FormValue("categoryLabel")
	input.Summary = r.FormValue("summary")
	input.Description = r.FormValue("description")
	input.SourceType = r.FormValue("sourceType")
	input.InteractiveRequired = r.FormValue("interactiveRequired") == "true"
	input.AnalyticsRecommended = r.FormValue("analyticsRecommended") != "false"
	input.Tags = splitTags(r.FormValue("tags"))
	if err := json.Unmarshal([]byte(defaultJSON(r.FormValue("configFields"), "[]")), &input.ConfigFields); err != nil {
		return input, fmt.Errorf("参数声明 JSON 格式不正确")
	}
	if err := json.Unmarshal([]byte(defaultJSON(r.FormValue("collections"), "[]")), &input.Collections); err != nil {
		return input, fmt.Errorf("数据表声明 JSON 格式不正确")
	}
	htmlText := strings.TrimSpace(r.FormValue("htmlSource"))
	file, header, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		body, readErr := readTemplateSubmissionFile(file, header)
		if readErr != nil {
			return input, readErr
		}
		htmlText = body
		if input.SourceType == "" {
			input.SourceType = strings.TrimPrefix(strings.ToLower(filepath.Ext(header.Filename)), ".")
		}
	}
	input.HTMLSource = htmlText
	return input, nil
}

func readTemplateSubmissionFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	body, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取上传文件失败")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".zip" {
		return string(body), nil
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", fmt.Errorf("ZIP 文件格式不正确")
	}
	var fallback *zip.File
	for _, item := range reader.File {
		if item.FileInfo().IsDir() {
			continue
		}
		name := strings.ToLower(filepath.Base(item.Name))
		if filepath.Ext(name) != ".html" && filepath.Ext(name) != ".htm" {
			continue
		}
		if name == "index.html" || name == "index.htm" {
			return readZipTextFile(item)
		}
		if fallback == nil {
			fallback = item
		}
	}
	if fallback != nil {
		return readZipTextFile(fallback)
	}
	return "", fmt.Errorf("ZIP 里没有找到 HTML 文件")
}

func readZipTextFile(file *zip.File) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("读取 ZIP 内文件失败")
	}
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取 ZIP 内文件失败")
	}
	return string(body), nil
}

func (rt *Router) buildTemplateSubmission(user domain.User, input domain.TemplateSubmissionCreateInput) (domain.TemplateSubmission, error) {
	input.Slug = normalizePathSegment(input.Slug)
	if input.Slug == "" && input.Name != "" {
		input.Slug = normalizePathSegment(input.Name)
	}
	if input.Name = strings.TrimSpace(input.Name); input.Name == "" {
		return domain.TemplateSubmission{}, fmt.Errorf("模板名称不能为空")
	}
	if input.Slug == "" {
		return domain.TemplateSubmission{}, fmt.Errorf("模板链接名不能为空")
	}
	for _, item := range builtInTemplates {
		if item.ID == input.Slug || item.Slug == input.Slug {
			return domain.TemplateSubmission{}, fmt.Errorf("这个模板链接名已经被使用")
		}
	}
	input.Category = normalizePathSegment(input.Category)
	if input.Category == "" {
		input.Category = "community"
	}
	input.CategoryLabel = trimLimit(input.CategoryLabel, 40)
	if input.CategoryLabel == "" {
		input.CategoryLabel = "用户投稿"
	}
	input.Summary = trimLimit(input.Summary, 200)
	input.Description = trimLimit(input.Description, 2000)
	input.HTMLSource = strings.TrimSpace(input.HTMLSource)
	if input.HTMLSource == "" {
		return domain.TemplateSubmission{}, fmt.Errorf("模板 HTML 不能为空")
	}
	if !strings.Contains(strings.ToLower(input.HTMLSource), "<html") {
		return domain.TemplateSubmission{}, fmt.Errorf("模板 HTML 需要包含完整 html 页面")
	}
	if len(input.ConfigFields) > 50 {
		return domain.TemplateSubmission{}, fmt.Errorf("模板参数太多")
	}
	if len(input.Collections) > 20 {
		return domain.TemplateSubmission{}, fmt.Errorf("数据表声明太多")
	}
	for i := range input.ConfigFields {
		input.ConfigFields[i].Name = normalizeTemplateFieldName(input.ConfigFields[i].Name)
		input.ConfigFields[i].Label = trimLimit(input.ConfigFields[i].Label, 80)
		input.ConfigFields[i].Default = trimLimit(input.ConfigFields[i].Default, 2000)
		input.ConfigFields[i].Placeholder = trimLimit(input.ConfigFields[i].Placeholder, 200)
		input.ConfigFields[i].Help = trimLimit(input.ConfigFields[i].Help, 300)
		if input.ConfigFields[i].Name == "" || input.ConfigFields[i].Label == "" {
			return domain.TemplateSubmission{}, fmt.Errorf("模板参数必须包含 name 和 label")
		}
	}
	for i := range input.Collections {
		input.Collections[i].Name = normalizePathSegment(input.Collections[i].Name)
		if input.Collections[i].Name == "" {
			return domain.TemplateSubmission{}, fmt.Errorf("数据表名称不能为空")
		}
		input.Collections[i].Permissions = domain.PermissionSet{
			PublicRead:  input.Collections[i].Permissions.PublicRead,
			PublicWrite: input.Collections[i].Permissions.PublicWrite,
		}
		for j := range input.Collections[i].Fields {
			input.Collections[i].Fields[j].Name = normalizeTemplateFieldName(input.Collections[i].Fields[j].Name)
			input.Collections[i].Fields[j].Type = normalizePathSegment(input.Collections[i].Fields[j].Type)
			if input.Collections[i].Fields[j].Name == "" || input.Collections[i].Fields[j].Type == "" {
				return domain.TemplateSubmission{}, fmt.Errorf("数据表字段必须包含 name 和 type")
			}
		}
	}
	sourceType := normalizePathSegment(input.SourceType)
	if sourceType == "" {
		sourceType = "text"
	}
	authorName := user.Username
	if authorName == "" {
		authorName = user.Email
	}
	return domain.TemplateSubmission{
		AuthorUserID:         user.ID,
		AuthorEmail:          user.Email,
		AuthorName:           authorName,
		Slug:                 input.Slug,
		Name:                 input.Name,
		Category:             input.Category,
		CategoryLabel:        input.CategoryLabel,
		Summary:              input.Summary,
		Description:          input.Description,
		Tags:                 input.Tags,
		InteractiveRequired:  input.InteractiveRequired,
		AnalyticsRecommended: input.AnalyticsRecommended,
		ConfigFields:         input.ConfigFields,
		Collections:          input.Collections,
		HTMLSource:           input.HTMLSource,
		SourceType:           sourceType,
		Status:               "pending",
	}, nil
}

func normalizeTemplateFieldName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-':
			return r
		default:
			return -1
		}
	}, value)
	return strings.Trim(value, "-_")
}

func splitTags(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '，' || r == '\n' || r == '\t' })
	items := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		part = trimLimit(strings.TrimSpace(part), 20)
		if part == "" || seen[part] {
			continue
		}
		items = append(items, part)
		seen[part] = true
		if len(items) >= 10 {
			break
		}
	}
	return items
}

func defaultJSON(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (rt *Router) handleListMyTemplateSubmissions(w http.ResponseWriter, r *http.Request) {
	user, ok := rt.requireUser(w, r)
	if !ok {
		return
	}
	items, err := rt.store.ListMyTemplateSubmissions(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取模板投稿失败"})
		return
	}
	for i := range items {
		items[i].HTMLSource = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleAdminListTemplateSubmissions(w http.ResponseWriter, r *http.Request) {
	if _, ok := rt.requireAdminUser(w, r); !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	items, err := rt.store.ListAdminTemplateSubmissions(r.Context(), status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取模板投稿失败"})
		return
	}
	for i := range items {
		items[i].HTMLSource = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (rt *Router) handleAdminTemplateSubmissionRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/template-submissions/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个模板投稿"})
		return
	}
	switch {
	case parts[1] == "preview" && r.Method == http.MethodGet:
		rt.handleAdminTemplateSubmissionPreview(w, r, parts[0])
	case parts[1] == "review" && r.Method == http.MethodPost:
		rt.handleAdminReviewTemplateSubmission(w, r, parts[0])
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "没有找到这个接口"})
	}
}

func (rt *Router) handleAdminTemplateSubmissionPreview(w http.ResponseWriter, r *http.Request, submissionID string) {
	if _, ok := rt.requireAdminUser(w, r); !ok {
		return
	}
	submission, ok, err := rt.store.GetTemplateSubmission(r.Context(), submissionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取模板投稿失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到这个模板投稿"})
		return
	}
	item := templateFromSubmission(submission)
	params := map[string]string{}
	for _, field := range item.ConfigFields {
		params[field.Name] = field.Default
	}
	body := renderTemplateHTML(item, domain.Project{ID: "template-review", Name: item.Name, Slug: item.Slug, Interactive: item.InteractiveRequired}, "template-review-key", params, true)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func (rt *Router) handleAdminReviewTemplateSubmission(w http.ResponseWriter, r *http.Request, submissionID string) {
	admin, ok := rt.requireAdminUser(w, r)
	if !ok {
		return
	}
	var input domain.TemplateSubmissionReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if input.Status != "published" && input.Status != "rejected" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "审核状态只能是 published 或 rejected"})
		return
	}
	item, err := rt.store.UpdateTemplateSubmissionReview(r.Context(), submissionID, input.Status, trimLimit(input.AdminNote, 1000), admin.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存审核结果失败"})
		return
	}
	item.HTMLSource = ""
	writeJSON(w, http.StatusOK, item)
}

func formatTemplateDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}
