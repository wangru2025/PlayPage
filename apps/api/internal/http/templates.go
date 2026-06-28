package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
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

var builtInTemplates = []domain.ProjectTemplate{
	{
		ID:                   "blank-page",
		Slug:                 "blank-page",
		Name:                 "空白起步页",
		Category:             "basic",
		CategoryLabel:        "基础页面",
		Summary:              "一个极简页面模板，用来验证模板创建流程。",
		Description:          "这是模板市场功能骨架里的官方占位模板。它只包含标题、副标题和主题色，后续可以替换成真正的论坛、主页、工具、小游戏等模板。",
		Tags:                 []string{"官方", "基础", "占位"},
		InteractiveRequired:  false,
		AnalyticsRecommended: false,
		UsageCount:           0,
		CreatedAtText:        "2026-06-21",
		ConfigFields: []domain.TemplateConfigField{
			{Name: "siteTitle", Label: "页面标题", Type: "string", Required: true, Default: "我的 PlayPage 作品", Placeholder: "例如：我的个人主页"},
			{Name: "subtitle", Label: "副标题", Type: "text", Required: false, Default: "这是从模板创建的作品。", Placeholder: "一句话介绍这个作品"},
			{Name: "themeColor", Label: "主题色", Type: "color", Required: false, Default: "#d6652f"},
		},
	},
	{
		ID:                   "personal-home",
		Slug:                 "personal-home",
		Name:                 "个人主页",
		Category:             "site",
		CategoryLabel:        "个人网站",
		Summary:              "适合做个人介绍、社交链接和作品展示。",
		Description:          "一个简洁的个人主页模板，包含昵称、简介、三个链接和强调色配置。适合新用户快速做一个自己的主页。",
		Tags:                 []string{"官方", "个人主页", "展示"},
		InteractiveRequired:  false,
		AnalyticsRecommended: true,
		UsageCount:           0,
		CreatedAtText:        "2026-06-22",
		ConfigFields: []domain.TemplateConfigField{
			{Name: "siteTitle", Label: "主页标题", Type: "string", Required: true, Default: "我的个人主页", Placeholder: "例如：汪汪的小站"},
			{Name: "nickname", Label: "昵称", Type: "string", Required: true, Default: "PlayPage 用户"},
			{Name: "bio", Label: "个人简介", Type: "text", Required: false, Default: "这里写一段自我介绍、兴趣或作品说明。"},
			{Name: "link1", Label: "链接一", Type: "string", Required: false, Default: "https://web.wangru.net"},
			{Name: "link2", Label: "链接二", Type: "string", Required: false, Default: ""},
			{Name: "link3", Label: "链接三", Type: "string", Required: false, Default: ""},
			{Name: "themeColor", Label: "主题色", Type: "color", Required: false, Default: "#7c3aed"},
		},
	},
	{
		ID:                   "link-directory",
		Slug:                 "link-directory",
		Name:                 "资源导航页",
		Category:             "site",
		CategoryLabel:        "个人网站",
		Summary:              "把常用网站、资料、作品链接整理成一个导航页。",
		Description:          "适合做班级资源导航、个人收藏夹、项目入口页。先提供 6 个链接位，后续可以再扩展为可编辑数据表版本。",
		Tags:                 []string{"官方", "导航", "资源"},
		InteractiveRequired:  false,
		AnalyticsRecommended: true,
		UsageCount:           0,
		CreatedAtText:        "2026-06-22",
		ConfigFields: []domain.TemplateConfigField{
			{Name: "siteTitle", Label: "导航标题", Type: "string", Required: true, Default: "我的资源导航"},
			{Name: "subtitle", Label: "说明文字", Type: "text", Required: false, Default: "把常用链接集中放在这里。"},
			{Name: "link1Title", Label: "链接1标题", Type: "string", Required: false, Default: "PlayPage"},
			{Name: "link1Url", Label: "链接1地址", Type: "string", Required: false, Default: "https://web.wangru.net"},
			{Name: "link2Title", Label: "链接2标题", Type: "string", Required: false, Default: "我的作品"},
			{Name: "link2Url", Label: "链接2地址", Type: "string", Required: false, Default: "https://web.wangru.net/projects"},
			{Name: "link3Title", Label: "链接3标题", Type: "string", Required: false, Default: "模板市场"},
			{Name: "link3Url", Label: "链接3地址", Type: "string", Required: false, Default: "https://web.wangru.net/templates"},
			{Name: "themeColor", Label: "主题色", Type: "color", Required: false, Default: "#0f766e"},
		},
	},
	{
		ID:                   "message-board",
		Slug:                 "message-board",
		Name:                 "留言板",
		Category:             "interactive",
		CategoryLabel:        "互动作品",
		Summary:              "带云端保存的简单留言板，适合反馈、祝福墙、班级留言。",
		Description:          "创建后会自动开启互动功能并创建 messages 数据表。访客可以填写昵称和留言，留言会保存到作品数据表。",
		Tags:                 []string{"官方", "留言", "互动"},
		InteractiveRequired:  true,
		AnalyticsRecommended: true,
		UsageCount:           0,
		CreatedAtText:        "2026-06-22",
		ConfigFields: []domain.TemplateConfigField{
			{Name: "siteTitle", Label: "留言板标题", Type: "string", Required: true, Default: "我的留言板"},
			{Name: "subtitle", Label: "说明文字", Type: "text", Required: false, Default: "欢迎留下你的想法。"},
			{Name: "themeColor", Label: "主题色", Type: "color", Required: false, Default: "#d6652f"},
		},
		Collections: []domain.TemplateCollectionDefinition{
			{
				Name:        "messages",
				Permissions: domain.PermissionSet{PublicRead: true, PublicWrite: true},
				Fields: []domain.FieldSchema{
					{Name: "nickname", Type: "string", Required: true},
					{Name: "content", Type: "text", Required: true},
				},
			},
		},
	},
	{
		ID:                   "mini-forum",
		Slug:                 "mini-forum",
		Name:                 "轻论坛",
		Category:             "interactive",
		CategoryLabel:        "互动作品",
		Summary:              "一个轻量论坛模板，支持发帖和回复。",
		Description:          "创建后会自动开启互动功能并创建 posts、comments 数据表。适合小圈子讨论、班级交流、项目问答。",
		Tags:                 []string{"官方", "论坛", "互动"},
		InteractiveRequired:  true,
		AnalyticsRecommended: true,
		UsageCount:           0,
		CreatedAtText:        "2026-06-22",
		ConfigFields: []domain.TemplateConfigField{
			{Name: "siteTitle", Label: "论坛标题", Type: "string", Required: true, Default: "我的轻论坛"},
			{Name: "subtitle", Label: "论坛简介", Type: "text", Required: false, Default: "发帖、回复，开始交流。"},
			{Name: "themeColor", Label: "主题色", Type: "color", Required: false, Default: "#2563eb"},
		},
		Collections: []domain.TemplateCollectionDefinition{
			{
				Name:        "posts",
				Permissions: domain.PermissionSet{PublicRead: true, PublicWrite: true},
				Fields: []domain.FieldSchema{
					{Name: "title", Type: "string", Required: true},
					{Name: "content", Type: "text", Required: true},
					{Name: "author", Type: "string", Required: true},
				},
			},
			{
				Name:        "comments",
				Permissions: domain.PermissionSet{PublicRead: true, PublicWrite: true},
				Fields: []domain.FieldSchema{
					{Name: "postId", Type: "string", Required: true},
					{Name: "content", Type: "text", Required: true},
					{Name: "author", Type: "string", Required: true},
				},
			},
		},
	},
}

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

func renderTemplateHTML(t domain.ProjectTemplate, project domain.Project, publicKey string, params map[string]string, preview bool) string {
	values := map[string]string{}
	for _, field := range t.ConfigFields {
		value := strings.TrimSpace(params[field.Name])
		if value == "" {
			value = field.Default
		}
		values[field.Name] = html.EscapeString(value)
	}
	var body string
	if strings.TrimSpace(t.HTMLSource) != "" {
		body = renderCustomTemplateHTML(t.HTMLSource, project, publicKey, values, preview)
	} else {
		switch t.ID {
		case "personal-home":
			body = renderPersonalHomeTemplate(values)
		case "link-directory":
			body = renderLinkDirectoryTemplate(values)
		case "message-board":
			body = renderMessageBoardTemplate(values, project.ID, publicKey, preview)
		case "mini-forum":
			body = renderMiniForumTemplate(values, project.ID, publicKey, preview)
		default:
			body = renderBlankTemplate(values)
		}
	}
	if preview {
		body = addTemplatePreviewBanner(body)
	}
	return body
}

func renderCustomTemplateHTML(source string, project domain.Project, publicKey string, values map[string]string, preview bool) string {
	replacements := map[string]string{
		"PLAYPAGE_API_BASE":   "/api/v1/public/projects/" + project.ID,
		"PLAYPAGE_PUBLIC_KEY": publicKey,
		"PLAYPAGE_PREVIEW":    fmt.Sprintf("%v", preview),
		"PROJECT_ID":          project.ID,
		"PROJECT_NAME":        html.EscapeString(project.Name),
	}
	for key, value := range values {
		replacements[key] = value
	}
	body := source
	for key, value := range replacements {
		body = strings.ReplaceAll(body, "{{"+key+"}}", value)
	}
	return body
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

func addTemplatePreviewBanner(body string) string {
	banner := `<div style="position:fixed;left:16px;right:16px;bottom:16px;z-index:99999;padding:10px 14px;border:1px solid rgba(15,23,42,.18);border-radius:14px;background:rgba(255,255,255,.94);box-shadow:0 12px 34px rgba(15,23,42,.16);font:14px/1.5 system-ui,'Microsoft YaHei',sans-serif;color:#0f172a">当前为 PlayPage 模板预览。互动数据仅用于演示，不会保存。</div>`
	if strings.Contains(body, "<body>") {
		return strings.Replace(body, "<body>", "<body>"+banner, 1)
	}
	return strings.Replace(body, "<body", "<body", 1) + banner
}

func renderBlankTemplate(values map[string]string) string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + values["siteTitle"] + `</title>
  <style>
    :root { color-scheme: light; --theme: ` + values["themeColor"] + `; }
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; font-family: system-ui, "Microsoft YaHei", sans-serif; background: #fff8ef; color: #241b15; }
    main { width: min(760px, calc(100vw - 32px)); padding: 42px; border: 1px solid #ead8c2; border-radius: 28px; background: #fffdf8; box-shadow: 0 18px 50px rgba(120, 72, 24, .12); }
    h1 { margin: 0 0 12px; color: var(--theme); font-size: clamp(2rem, 7vw, 4rem); }
    p { margin: 0; font-size: 1.12rem; line-height: 1.8; }
  </style>
</head>
<body>
  <main>
    <h1>` + values["siteTitle"] + `</h1>
    <p>` + values["subtitle"] + `</p>
  </main>
</body>
</html>`
}

func renderPersonalHomeTemplate(values map[string]string) string {
	links := []string{}
	for _, key := range []string{"link1", "link2", "link3"} {
		if strings.TrimSpace(values[key]) != "" {
			links = append(links, `<a href="`+values[key]+`" target="_blank" rel="noreferrer">`+values[key]+`</a>`)
		}
	}
	if len(links) == 0 {
		links = append(links, `<span>还没有填写链接</span>`)
	}
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + values["siteTitle"] + `</title><style>
:root{--theme:` + values["themeColor"] + `}*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:system-ui,"Microsoft YaHei",sans-serif;background:linear-gradient(135deg,#fff7ed,#eef2ff);color:#1f2937}.wrap{width:min(920px,calc(100vw - 32px));margin:0 auto;padding:72px 0}.card{background:rgba(255,255,255,.88);border:1px solid rgba(0,0,0,.08);border-radius:32px;padding:40px;box-shadow:0 24px 70px rgba(31,41,55,.12)}.avatar{width:96px;height:96px;border-radius:28px;background:var(--theme);color:#fff;display:grid;place-items:center;font-size:3rem;font-weight:900}h1{font-size:clamp(2rem,7vw,4rem);margin:22px 0 8px;color:var(--theme)}p{font-size:1.12rem;line-height:1.8}.links{display:grid;gap:12px;margin-top:24px}.links a,.links span{padding:14px 16px;border:1px solid #e5e7eb;border-radius:16px;background:#fff;color:#111827;text-decoration:none;overflow-wrap:anywhere}</style></head><body><main class="wrap"><section class="card"><div class="avatar" aria-hidden="true">` + firstRune(values["nickname"]) + `</div><h1>` + values["nickname"] + `</h1><p>` + values["bio"] + `</p><div class="links">` + strings.Join(links, "") + `</div></section></main></body></html>`
}

func renderLinkDirectoryTemplate(values map[string]string) string {
	var cards []string
	for i := 1; i <= 3; i++ {
		title := values["link"+string(rune('0'+i))+"Title"]
		url := values["link"+string(rune('0'+i))+"Url"]
		if strings.TrimSpace(title) != "" && strings.TrimSpace(url) != "" {
			cards = append(cards, `<a class="card" href="`+url+`" target="_blank" rel="noreferrer"><strong>`+title+`</strong><span>`+url+`</span></a>`)
		}
	}
	if len(cards) == 0 {
		cards = append(cards, `<div class="card"><strong>还没有链接</strong><span>请编辑 HTML 添加更多链接。</span></div>`)
	}
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + values["siteTitle"] + `</title><style>
:root{--theme:` + values["themeColor"] + `}body{margin:0;font-family:system-ui,"Microsoft YaHei",sans-serif;background:#f8fafc;color:#0f172a}.wrap{width:min(980px,calc(100vw - 32px));margin:0 auto;padding:56px 0}header{margin-bottom:28px}h1{font-size:clamp(2rem,7vw,4rem);margin:0;color:var(--theme)}p{color:#475569;font-size:1.1rem}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:16px}.card{display:grid;gap:8px;padding:22px;border:1px solid #e2e8f0;border-radius:22px;background:#fff;text-decoration:none;color:inherit;box-shadow:0 14px 40px rgba(15,23,42,.08)}.card strong{font-size:1.25rem}.card span{color:#64748b;overflow-wrap:anywhere}</style></head><body><main class="wrap"><header><h1>` + values["siteTitle"] + `</h1><p>` + values["subtitle"] + `</p></header><section class="grid">` + strings.Join(cards, "") + `</section></main></body></html>`
}

func renderMessageBoardTemplate(values map[string]string, projectID, publicKey string, preview bool) string {
	cfg := templateRuntimeConfig(projectID, publicKey, preview)
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + values["siteTitle"] + `</title><style>
:root{--theme:` + values["themeColor"] + `}body{margin:0;font-family:system-ui,"Microsoft YaHei",sans-serif;background:#fff7ed;color:#211812}.wrap{width:min(860px,calc(100vw - 32px));margin:0 auto;padding:48px 0}h1{margin:0;color:var(--theme);font-size:clamp(2rem,7vw,3.5rem)}p{line-height:1.7}.panel{background:#fff;border:1px solid #ead7c4;border-radius:24px;padding:22px;margin-top:18px;box-shadow:0 14px 40px rgba(120,72,24,.08)}label{display:grid;gap:6px;margin-bottom:12px}input,textarea,button{font:inherit}input,textarea{width:100%;padding:12px;border:1px solid #decbb8;border-radius:14px}button{border:0;border-radius:999px;padding:12px 18px;background:var(--theme);color:#fff;font-weight:700}.msg{border-top:1px solid #f1e1cf;padding:14px 0}.meta{color:#7c6f64;font-size:.92rem}</style></head><body><main class="wrap"><h1>` + values["siteTitle"] + `</h1><p>` + values["subtitle"] + `</p><section class="panel"><form id="form"><label>昵称<input id="nickname" required maxlength="40"></label><label>留言<textarea id="content" required rows="4" maxlength="1000"></textarea></label><button>发送留言</button></form><div id="status" role="status" aria-live="polite"></div></section><section class="panel"><h2>留言</h2><div id="list">正在读取留言……</div></section></main><script>const CFG=` + cfg + `;
const api=(p)=>CFG.apiBase+p;const headers={'Content-Type':'application/json','X-Project-Key':CFG.publicKey};
const previewMessages=[{id:'preview-msg-1',data:{nickname:'小明',content:'这个留言板可以用来收集反馈。'},createdAt:new Date(Date.now()-86400000).toISOString()},{id:'preview-msg-2',data:{nickname:'小红',content:'当前是模板预览，留言只会临时显示。'},createdAt:new Date().toISOString()}];
function esc(s){return String(s||'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]))}
async function request(path,opt={}){if(CFG.preview){return previewRequest(path,opt)}const r=await fetch(api(path),{...opt,headers:{...headers,...opt.headers}});const j=await r.json().catch(()=>({}));if(!r.ok)throw new Error(j.error||'请求失败');return j}
async function previewRequest(path,opt={}){await new Promise(r=>setTimeout(r,120));if(path==='/collections/messages/records'&&(!opt.method||opt.method==='GET'))return{items:previewMessages};if(path==='/collections/messages/records'&&opt.method==='POST'){const body=JSON.parse(opt.body||'{}');previewMessages.push({id:'preview-msg-'+(previewMessages.length+1),data:body.data||{},createdAt:new Date().toISOString()});return previewMessages[previewMessages.length-1]}throw new Error('预览模式不支持这个操作')}
async function load(){try{const data=await request('/collections/messages/records');const items=(data.items||[]).slice().reverse();list.innerHTML=items.length?items.map(x=>'<article class="msg"><strong>'+esc(x.data.nickname)+'</strong><div>'+esc(x.data.content)+'</div><div class="meta">'+new Date(x.createdAt).toLocaleString()+'</div></article>').join(''):'还没有留言。'}catch(e){list.textContent=e.message}}
form.addEventListener('submit',async e=>{e.preventDefault();status.textContent='正在发送……';try{await request('/collections/messages/records',{method:'POST',body:JSON.stringify({data:{nickname:nickname.value.trim(),content:content.value.trim()}})});content.value='';status.textContent=CFG.preview?'预览留言已临时显示，刷新后会恢复默认演示数据。':'已发送。';await load()}catch(err){status.textContent=err.message}});load();</script></body></html>`
}

func renderMiniForumTemplate(values map[string]string, projectID, publicKey string, preview bool) string {
	cfg := templateRuntimeConfig(projectID, publicKey, preview)
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + values["siteTitle"] + `</title><style>
:root{--theme:` + values["themeColor"] + `}body{margin:0;font-family:system-ui,"Microsoft YaHei",sans-serif;background:#eff6ff;color:#172033}.wrap{width:min(960px,calc(100vw - 32px));margin:0 auto;padding:42px 0}h1{color:var(--theme);font-size:clamp(2rem,7vw,3.5rem);margin:0}.panel,.post{background:#fff;border:1px solid #dbeafe;border-radius:22px;padding:20px;margin-top:16px;box-shadow:0 12px 36px rgba(30,64,175,.08)}label{display:grid;gap:6px;margin-bottom:10px}input,textarea,button{font:inherit}input,textarea{width:100%;padding:12px;border:1px solid #bfdbfe;border-radius:14px}button{border:0;border-radius:999px;padding:10px 16px;background:var(--theme);color:#fff;font-weight:700}.muted{color:#64748b}.comment{border-top:1px solid #e0edff;margin-top:10px;padding-top:10px}</style></head><body><main class="wrap"><h1>` + values["siteTitle"] + `</h1><p class="muted">` + values["subtitle"] + `</p><section class="panel"><h2>发布新帖</h2><form id="postForm"><label>昵称<input id="author" required maxlength="40"></label><label>标题<input id="title" required maxlength="80"></label><label>内容<textarea id="content" required rows="4" maxlength="2000"></textarea></label><button>发布</button></form><div id="status" role="status" aria-live="polite"></div></section><section id="posts"></section></main><script>const CFG=` + cfg + `;
const api=(p)=>CFG.apiBase+p;const headers={'Content-Type':'application/json','X-Project-Key':CFG.publicKey};let allComments=[];
const previewPosts=[{id:'preview-post-1',data:{author:'模板管理员',title:'欢迎来到轻论坛',content:'这里可以发布帖子，也可以回复讨论。'},createdAt:new Date(Date.now()-7200000).toISOString()},{id:'preview-post-2',data:{author:'路过的同学',title:'模板预览说明',content:'当前互动数据只存在浏览器内存里，不会保存到数据库。'},createdAt:new Date(Date.now()-3600000).toISOString()}];
const previewComments=[{id:'preview-comment-1',data:{postId:'preview-post-1',author:'小明',content:'这个模板适合做班级交流区。'},createdAt:new Date(Date.now()-1800000).toISOString()}];
function esc(s){return String(s||'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]))}
async function request(path,opt={}){if(CFG.preview){return previewRequest(path,opt)}const r=await fetch(api(path),{...opt,headers:{...headers,...opt.headers}});const j=await r.json().catch(()=>({}));if(!r.ok)throw new Error(j.error||'请求失败');return j}
async function previewRequest(path,opt={}){await new Promise(r=>setTimeout(r,120));const method=opt.method||'GET';if(path==='/collections/posts/records'&&method==='GET')return{items:previewPosts};if(path==='/collections/comments/records'&&method==='GET')return{items:previewComments};if(path==='/collections/posts/records'&&method==='POST'){const body=JSON.parse(opt.body||'{}');const item={id:'preview-post-'+(previewPosts.length+1),data:body.data||{},createdAt:new Date().toISOString()};previewPosts.push(item);return item}if(path==='/collections/comments/records'&&method==='POST'){const body=JSON.parse(opt.body||'{}');const item={id:'preview-comment-'+(previewComments.length+1),data:body.data||{},createdAt:new Date().toISOString()};previewComments.push(item);return item}throw new Error('预览模式不支持这个操作')}
async function load(){posts.textContent='正在读取帖子……';try{const [p,c]=await Promise.all([request('/collections/posts/records'),request('/collections/comments/records')]);allComments=c.items||[];const items=(p.items||[]).slice().reverse();posts.innerHTML=items.length?items.map(renderPost).join(''):'<section class="panel">还没有帖子。</section>';document.querySelectorAll('[data-reply]').forEach(btn=>btn.addEventListener('click',reply))}catch(e){posts.innerHTML='<section class="panel">'+esc(e.message)+'</section>'}}
function renderPost(x){const cs=allComments.filter(c=>c.data.postId===x.id);return '<article class="post"><h2>'+esc(x.data.title)+'</h2><p>'+esc(x.data.content)+'</p><p class="muted">'+esc(x.data.author)+' · '+new Date(x.createdAt).toLocaleString()+'</p><button data-reply="'+x.id+'">回复</button><div>'+cs.map(c=>'<div class="comment"><strong>'+esc(c.data.author)+'</strong>：'+esc(c.data.content)+'</div>').join('')+'</div></article>'}
async function reply(e){const postId=e.target.getAttribute('data-reply');const author=prompt('你的昵称')||'';if(!author.trim())return;const content=prompt('回复内容')||'';if(!content.trim())return;await request('/collections/comments/records',{method:'POST',body:JSON.stringify({data:{postId,author:author.trim(),content:content.trim()}})});await load()}
postForm.addEventListener('submit',async e=>{e.preventDefault();status.textContent='正在发布……';try{await request('/collections/posts/records',{method:'POST',body:JSON.stringify({data:{author:author.value.trim(),title:title.value.trim(),content:content.value.trim()}})});title.value='';content.value='';status.textContent=CFG.preview?'预览帖子已临时显示，刷新后会恢复默认演示数据。':'已发布。';await load()}catch(err){status.textContent=err.message}});load();</script></body></html>`
}

func templateRuntimeConfig(projectID, publicKey string, preview bool) string {
	payload := map[string]any{
		"apiBase":   "/api/v1/public/projects/" + projectID,
		"publicKey": publicKey,
		"preview":   preview,
	}
	raw, _ := json.Marshal(payload)
	return string(raw)
}

func firstRune(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "P"
	}
	for _, r := range value {
		return html.EscapeString(string(r))
	}
	return "P"
}
