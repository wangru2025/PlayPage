package http

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
)

type dataExportInput struct {
	Collections []string `json:"collections"`
	Format      string   `json:"format"`
}

type dataExportPayload struct {
	Version     int                    `json:"version"`
	ExportedAt  string                 `json:"exportedAt"`
	Project     domain.Project         `json:"project"`
	Collections []dataExportCollection `json:"collections"`
}

type dataExportCollection struct {
	Collection domain.Collection `json:"collection"`
	Records    []domain.Record   `json:"records"`
}

var unsafeDownloadNameChars = regexp.MustCompile(`[\\/:*?"<>|\r\n\t]+`)

func (rt *Router) handleExportProjectData(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}

	var input dataExportInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.Format = strings.TrimSpace(strings.ToLower(input.Format))
	if input.Format != "json" && input.Format != "word" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "导出格式只能是 json 或 word"})
		return
	}

	selected := map[string]bool{}
	for _, name := range input.Collections {
		name = strings.TrimSpace(name)
		if name != "" {
			selected[name] = true
		}
	}
	if len(selected) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请至少选择一个数据表"})
		return
	}

	collections, err := rt.store.ListCollections(r.Context(), project.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品数据表失败"})
		return
	}

	payload := dataExportPayload{
		Version:     1,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		Project:     project,
		Collections: []dataExportCollection{},
	}
	found := map[string]bool{}
	for _, collection := range collections {
		if !selected[collection.Name] {
			continue
		}
		records, err := rt.store.ListRecords(r.Context(), project.ID, collection.Name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取数据表记录失败"})
			return
		}
		payload.Collections = append(payload.Collections, dataExportCollection{
			Collection: collection,
			Records:    records,
		})
		found[collection.Name] = true
	}
	for name := range selected {
		if !found[name] {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "找不到数据表：" + name})
			return
		}
	}

	now := time.Now()
	baseName := safeDownloadName(project.Name) + "-数据表导出-" + now.Format("20060102-150405")
	switch input.Format {
	case "json":
		body, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "生成导出文件失败"})
			return
		}
		writeAttachment(w, baseName+".json", "application/json; charset=utf-8", body)
	case "word":
		body := []byte(buildDataExportWordHTML(payload))
		writeAttachment(w, baseName+".doc", "application/msword; charset=utf-8", body)
	}
}

func writeAttachment(w http.ResponseWriter, filename, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(
		`attachment; filename="%s"; filename*=UTF-8''%s`,
		downloadASCIIFallback(filename),
		url.PathEscape(filename),
	))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func safeDownloadName(value string) string {
	value = unsafeDownloadNameChars.ReplaceAllString(value, "-")
	value = strings.TrimSpace(value)
	if value == "" {
		return "作品"
	}
	return value
}

func downloadASCIIFallback(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return '-'
		}
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, value)
	value = strings.TrimSpace(value)
	if value == "" {
		return "download"
	}
	return value
}

func buildDataExportWordHTML(payload dataExportPayload) string {
	var builder strings.Builder
	builder.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>")
	builder.WriteString(html.EscapeString(payload.Project.Name))
	builder.WriteString(" 数据表导出</title><style>")
	builder.WriteString(`body{font-family:"Microsoft YaHei",Arial,sans-serif;line-height:1.6;}table{border-collapse:collapse;width:100%;margin:12px 0 28px;}th,td{border:1px solid #999;padding:6px 8px;vertical-align:top;}th{background:#f2f2f2;}`)
	builder.WriteString("</style></head><body>")
	builder.WriteString("<h1>")
	builder.WriteString(html.EscapeString(payload.Project.Name))
	builder.WriteString(" 数据表导出</h1>")
	builder.WriteString("<p>导出时间：")
	builder.WriteString(html.EscapeString(payload.ExportedAt))
	builder.WriteString("</p><p>作品地址：")
	builder.WriteString(html.EscapeString(payload.Project.PublicURL))
	builder.WriteString("</p>")

	for _, item := range payload.Collections {
		fieldNames := make([]string, 0, len(item.Collection.Fields))
		for _, field := range item.Collection.Fields {
			fieldNames = append(fieldNames, field.Name)
		}
		headers := append([]string{"记录 ID", "状态", "创建时间", "更新时间"}, fieldNames...)
		builder.WriteString("<h2>")
		builder.WriteString(html.EscapeString(item.Collection.Name))
		builder.WriteString("</h2><p>字段数：")
		builder.WriteString(fmt.Sprintf("%d", len(item.Collection.Fields)))
		builder.WriteString("；记录数：")
		builder.WriteString(fmt.Sprintf("%d", len(item.Records)))
		builder.WriteString("</p><table><thead><tr>")
		for _, header := range headers {
			builder.WriteString("<th>")
			builder.WriteString(html.EscapeString(header))
			builder.WriteString("</th>")
		}
		builder.WriteString("</tr></thead><tbody>")
		if len(item.Records) == 0 {
			builder.WriteString(fmt.Sprintf(`<tr><td colspan="%d">没有记录</td></tr>`, len(headers)))
		}
		for _, record := range item.Records {
			builder.WriteString("<tr>")
			cells := []string{record.ID, record.Status, record.CreatedAt.Format(time.RFC3339), record.UpdatedAt.Format(time.RFC3339)}
			for _, fieldName := range fieldNames {
				cells = append(cells, stringifyExportCell(record.Data[fieldName]))
			}
			for _, cell := range cells {
				builder.WriteString("<td>")
				builder.WriteString(html.EscapeString(cell))
				builder.WriteString("</td>")
			}
			builder.WriteString("</tr>")
		}
		builder.WriteString("</tbody></table>")
	}
	builder.WriteString("</body></html>")
	return builder.String()
}

func stringifyExportCell(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64, bool, int, int64:
		return fmt.Sprint(typed)
	default:
		body, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(body)
	}
}
