package http

import (
	"fmt"
	"html"
	"strings"

	"ai-static-host/api/internal/domain"
)

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
		case "feedback-center":
			body = renderRecordListTemplate(values, project.ID, publicKey, preview, recordListTemplateOptions{
				Collection: "feedbacks",
				FormTitle:  "提交反馈",
				ListTitle:  "反馈列表",
				SubmitText: "提交反馈",
				EmptyText:  "还没有反馈。",
				Fields: []recordListField{
					{Name: "nickname", Label: "昵称", Required: true},
					{Name: "title", Label: "问题标题", Required: true},
					{Name: "content", Label: "问题描述", Required: true, Multiline: true},
					{Name: "contact", Label: "联系方式"},
				},
				HiddenDefaults: map[string]string{"status": "待处理", "reply": ""},
				Preview: []map[string]string{
					{"nickname": "小明", "title": "按钮点了没反应", "content": "这里可以描述问题出现的页面、操作步骤和期望效果。", "contact": "example@qq.com", "status": "待处理"},
				},
			})
		case "game-scoreboard":
			body = renderRecordListTemplate(values, project.ID, publicKey, preview, recordListTemplateOptions{
				Collection: "scores",
				FormTitle:  "提交成绩",
				ListTitle:  "排行榜",
				SubmitText: "提交成绩",
				EmptyText:  "还没有成绩。",
				SortNumber: "score",
				Fields: []recordListField{
					{Name: "player", Label: "玩家名", Required: true},
					{Name: "score", Label: "分数", Required: true},
					{Name: "note", Label: "备注", Multiline: true},
				},
				Preview: []map[string]string{
					{"player": "汪汪", "score": "1200", "note": "第一关满分通过"},
					{"player": "小猫", "score": "860", "note": "还可以再冲一次"},
				},
			})
		case "announcement-log":
			body = renderRecordListTemplate(values, project.ID, publicKey, preview, recordListTemplateOptions{
				Collection: "announcements",
				FormTitle:  "发布公告",
				ListTitle:  "公告列表",
				SubmitText: "发布公告",
				EmptyText:  "还没有公告。",
				Fields: []recordListField{
					{Name: "title", Label: "标题", Required: true},
					{Name: "version", Label: "版本号"},
					{Name: "content", Label: "公告内容", Required: true, Multiline: true},
				},
				Preview: []map[string]string{
					{"title": "欢迎使用这个模板", "version": "v1.0", "content": "这里可以发布作品更新、维护通知或活动公告。"},
				},
			})
		case "event-registration":
			body = renderRecordListTemplate(values, project.ID, publicKey, preview, recordListTemplateOptions{
				Collection: "registrations",
				FormTitle:  "填写报名信息",
				ListTitle:  "报名名单",
				SubmitText: "提交报名",
				EmptyText:  "还没有人报名。",
				Fields: []recordListField{
					{Name: "nickname", Label: "昵称或姓名", Required: true},
					{Name: "contact", Label: "联系方式", Required: true},
					{Name: "note", Label: "备注", Multiline: true},
				},
				Preview: []map[string]string{
					{"nickname": "小红", "contact": "QQ：123456", "note": "周六下午可以参加"},
					{"nickname": "小明", "contact": "example@qq.com", "note": "需要提前提醒"},
				},
			})
		case "download-showcase":
			body = renderRecordListTemplate(values, project.ID, publicKey, preview, recordListTemplateOptions{
				Collection: "resources",
				FormTitle:  "添加作品或下载项",
				ListTitle:  "作品列表",
				SubmitText: "添加",
				EmptyText:  "还没有下载项。",
				Fields: []recordListField{
					{Name: "title", Label: "名称", Required: true},
					{Name: "version", Label: "版本"},
					{Name: "url", Label: "下载或访问地址", Required: true},
					{Name: "description", Label: "介绍", Multiline: true},
				},
				Preview: []map[string]string{
					{"title": "示例工具", "version": "v1.0", "url": "https://web.wangru.net", "description": "这里展示作品介绍和下载地址。"},
				},
			})
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
