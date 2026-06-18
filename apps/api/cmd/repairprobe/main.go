package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-static-host/api/internal/service"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	endpoint := strings.TrimSpace(os.Getenv("AI_PROVIDER_BASE_URL"))
	apiKey := strings.TrimSpace(os.Getenv("AI_PROVIDER_API_KEY"))
	model := strings.TrimSpace(os.Getenv("AI_PROVIDER_MODEL"))
	if endpoint == "" || apiKey == "" || model == "" {
		fmt.Fprintln(os.Stderr, "missing AI_PROVIDER_BASE_URL / AI_PROVIDER_API_KEY / AI_PROVIDER_MODEL")
		os.Exit(1)
	}
	targetURL := strings.TrimSpace(os.Getenv("PROBE_URL"))
	if targetURL == "" {
		targetURL = "https://web.wangru.net/@%E6%B1%AA%E6%B1%AA/%E6%B1%AA%E6%B1%AA%E5%A5%B6%E8%8C%B6%E5%BA%97"
	}
	projectName := strings.TrimSpace(os.Getenv("PROBE_PROJECT_NAME"))
	if projectName == "" {
		projectName = "示例作品"
	}
	userIssue := strings.TrimSpace(os.Getenv("PROBE_ISSUE"))
	if userIssue == "" {
		userIssue = "用户反馈作品体验不平衡、修复起来容易跑偏。"
	}
	expected := strings.TrimSpace(os.Getenv("PROBE_EXPECTED"))
	if expected == "" {
		expected = "期望保留策略性，但修复结果必须明显改善可玩性或流程。"
	}

	html, err := fetchURL(targetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch html failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("HTML bytes: %d\n", len(html))
	fmt.Printf("HTML head:\n%s\n\n", trim(html, 3000))

	client := service.NewAIClient(endpoint, apiKey, model)
	if !client.Enabled() {
		fmt.Fprintln(os.Stderr, "AI client disabled")
		os.Exit(1)
	}

	system := `你是网页急救圆桌里的修复员。你必须基于用户给你的 HTML 和问题，返回一个 JSON。
JSON 字段：
- visible_message：中文简短总结
- patch：apply_patch 风格补丁文本

硬性要求：
- 必须局部修改，不要重写完整 HTML。
- patch 必须是有效的 apply_patch 格式。
- 不要输出 Markdown，不要输出解释。`

	contextHTML := selectRelevantHTML(html)
	prompt := fmt.Sprintf(`作品名：%s
用户问题：%s
期望效果：%s
当前入口 HTML 关键片段：
%s

请直接给出修复补丁。`, projectName, userIssue, expected, contextHTML)

	if os.Getenv("PROBE_THINKING") == "1" {
		fmt.Println("== round 1: no tools, thinking on ==")
		msg, finish, err := client.Chat(ctx, []service.AIMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		}, nil, nil, 6000, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chat failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("finish_reason=%s tool_calls=%d\n", finish, len(msg.ToolCalls))
		if len(msg.ToolCalls) > 0 {
			for i, call := range msg.ToolCalls {
				fmt.Printf("tool[%d]=%s args=%s\n", i, call.Function.Name, trim(call.Function.Arguments, 1200))
			}
		}
		fmt.Printf("content:\n%s\n", trim(msg.Content, 12000))
	}

	fmt.Println("\n== round 1: no tools, thinking off ==")
	msg1b, finish1b, err := client.Chat(ctx, []service.AIMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: prompt},
	}, nil, nil, 8000, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat no-thinking failed: %v\n", err)
	} else {
		fmt.Printf("finish_reason=%s tool_calls=%d\n", finish1b, len(msg1b.ToolCalls))
		fmt.Printf("content:\n%s\n", trim(msg1b.Content, 12000))
		if patch := extractPatchForProbe(msg1b.Content); patch != "" {
			fmt.Printf("\nconverted patch head:\n%s\n", trim(patch, 3000))
		} else {
			fmt.Println("\nconverted patch head: <empty>")
		}
	}

	if os.Getenv("PROBE_TOOLS") != "1" {
		return
	}

	fmt.Println("\n== round 2: with tools ==")
	tools := []service.AITool{
		{
			Type: "function",
			Function: service.AIToolFunction{
				Name:        "read_current_html",
				Description: "Read the current entry HTML.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"mode":  map[string]any{"type": "string"},
						"query": map[string]any{"type": "string"},
					},
					"required": []string{"mode"},
				},
			},
		},
		{
			Type: "function",
			Function: service.AIToolFunction{
				Name:        "apply_patch",
				Description: "Apply a patch to the current HTML.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"patch": map[string]any{"type": "string"},
					},
					"required": []string{"patch"},
				},
			},
		},
	}

	msg2, finish2, err := client.Chat(ctx, []service.AIMessage{
		{Role: "system", Content: system + "\n必须先调用 read_current_html。"},
		{Role: "user", Content: prompt},
	}, tools, "auto", 8000, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat with tools failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("finish_reason=%s tool_calls=%d\n", finish2, len(msg2.ToolCalls))
	if len(msg2.ToolCalls) > 0 {
		for i, call := range msg2.ToolCalls {
			fmt.Printf("tool[%d]=%s id=%s args=%s\n", i, call.Function.Name, call.ID, trim(call.Function.Arguments, 1600))
		}
	}
	fmt.Printf("content:\n%s\n", trim(msg2.Content, 12000))

	fmt.Println("\n== round 3: short context, no tools, thinking off ==")
	msg3, finish3, err := client.Chat(ctx, []service.AIMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: prompt},
	}, nil, nil, 5000, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "short no-tools failed: %v\n", err)
	} else {
		fmt.Printf("finish_reason=%s tool_calls=%d content_len=%d\n", finish3, len(msg3.ToolCalls), len(msg3.Content))
		fmt.Printf("content:\n%s\n", trim(msg3.Content, 12000))
	}
}

func extractPatchForProbe(content string) string {
	raw := strings.TrimSpace(content)
	var out struct {
		Patch string `json:"patch"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		patch := strings.TrimSpace(out.Patch)
		if strings.Contains(patch, "*** Begin Patch") {
			return patch
		}
		if converted := extractUnifiedDiffAsApplyPatchForProbe(patch); converted != "" {
			return converted
		}
	}
	return extractUnifiedDiffAsApplyPatchForProbe(content)
}

func extractUnifiedDiffAsApplyPatchForProbe(s string) string {
	raw := strings.TrimSpace(s)
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var hunks []string
	inHunk := false
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			hunks = append(hunks, "@@")
			continue
		}
		if !inHunk {
			continue
		}
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			continue
		}
		if line == "" {
			hunks = append(hunks, " ")
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			hunks = append(hunks, line)
		}
	}
	if len(hunks) == 0 {
		return ""
	}
	return "*** Begin Patch\n*** Update File: index.html\n" + strings.Join(hunks, "\n") + "\n*** End Patch"
}

func fetchURL(u string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status=%d body=%s", resp.StatusCode, trim(string(body), 1000))
	}
	return string(body), nil
}

func trim(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}

func quote(s string) string {
	out := strings.Builder{}
	out.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			out.WriteString(`\\`)
		case '"':
			out.WriteString(`\"`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte('"')
	return out.String()
}

func selectRelevantHTML(html string) string {
	keys := []string{
		"function getTodayCosts",
		"function buyStock",
		"function openDay",
		"function restDay",
		"const DEFAULT_STATE",
	}
	var b strings.Builder
	for _, key := range keys {
		part := lineBlock(html, key, 80)
		if part == "" {
			continue
		}
		b.WriteString("\n\n===== ")
		b.WriteString(key)
		b.WriteString(" =====\n")
		b.WriteString(part)
	}
	if b.Len() == 0 {
		return trim(html, 30000)
	}
	return b.String()
}

func around(s, key string, width int) string {
	idx := strings.Index(s, key)
	if idx < 0 {
		return ""
	}
	start := idx - width/2
	if start < 0 {
		start = 0
	}
	end := idx + width/2
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}

func lineBlock(s, key string, radius int) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	idx := -1
	for i, line := range lines {
		if strings.Contains(line, key) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}
	start := idx - radius/2
	if start < 0 {
		start = 0
	}
	end := idx + radius/2
	if end > len(lines) {
		end = len(lines)
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		b.WriteString(fmt.Sprintf("%d: %s\n", i+1, lines[i]))
	}
	return b.String()
}
