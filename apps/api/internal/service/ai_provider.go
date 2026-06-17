package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type AIClient struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

type AIMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
	ToolCalls  []AIToolCall `json:"tool_calls,omitempty"`
}

type aiChatRequest struct {
	Model              string         `json:"model"`
	Messages           []AIMessage    `json:"messages"`
	Temperature        float64        `json:"temperature,omitempty"`
	MaxTokens          int            `json:"max_tokens,omitempty"`
	ChatTemplateKwargs map[string]any `json:"chat_template_kwargs,omitempty"`
	Tools              []AITool       `json:"tools,omitempty"`
	ToolChoice         any            `json:"tool_choice,omitempty"`
}

type aiChatResponse struct {
	Choices []struct {
		Message      AIMessage `json:"message"`
		FinishReason string    `json:"finish_reason"`
	} `json:"choices"`
	Error any `json:"error,omitempty"`
}

type AITool struct {
	Type     string         `json:"type"`
	Function AIToolFunction `json:"function"`
}

type AIToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type AIToolCall struct {
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function AIToolCallFunction `json:"function"`
}

type AIToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func NewAIClient(endpoint, apiKey, model string) *AIClient {
	timeout := 360 * time.Second
	if value := strings.TrimSpace(os.Getenv("AI_PROVIDER_TIMEOUT_SECONDS")); value != "" {
		if parsed, err := time.ParseDuration(value + "s"); err == nil && parsed > 0 {
			timeout = parsed
		}
	}
	return &AIClient{
		endpoint:   strings.TrimSpace(endpoint),
		apiKey:     strings.TrimSpace(apiKey),
		model:      strings.TrimSpace(model),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *AIClient) Enabled() bool {
	return c != nil && c.endpoint != "" && c.apiKey != "" && c.model != ""
}

func (c *AIClient) Complete(ctx context.Context, messages []AIMessage, maxTokens int) (string, error) {
	return c.complete(ctx, messages, maxTokens, true)
}

func (c *AIClient) CompletePlain(ctx context.Context, messages []AIMessage, maxTokens int) (string, error) {
	return c.complete(ctx, messages, maxTokens, false)
}

func (c *AIClient) complete(ctx context.Context, messages []AIMessage, maxTokens int, enableThinking bool) (string, error) {
	message, _, err := c.Chat(ctx, messages, nil, nil, maxTokens, enableThinking)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(message.Content)
	if content == "" {
		return "", fmt.Errorf("AI 返回了空内容")
	}
	return content, nil
}

func (c *AIClient) Chat(ctx context.Context, messages []AIMessage, tools []AITool, toolChoice any, maxTokens int, enableThinking bool) (AIMessage, string, error) {
	if !c.Enabled() {
		return AIMessage{}, "", fmt.Errorf("AI 圆桌还没有配置好，请稍后再试")
	}
	reqBody := aiChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.2,
		MaxTokens:   maxTokens,
		Tools:       tools,
		ToolChoice:  toolChoice,
	}
	if enableThinking {
		reqBody.ChatTemplateKwargs = map[string]any{"enable_thinking": true}
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return AIMessage{}, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		return AIMessage{}, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			if ctx.Err() == context.Canceled {
				return AIMessage{}, "", fmt.Errorf("AI 圆桌已叫停")
			}
			return AIMessage{}, "", fmt.Errorf("AI 服务响应超时，请稍后重试；如果作品 HTML 很大，建议先简化页面后再启动圆桌")
		}
		if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			return AIMessage{}, "", fmt.Errorf("AI 服务响应超时，请稍后重试；如果作品 HTML 很大，建议先简化页面后再启动圆桌")
		}
		return AIMessage{}, "", fmt.Errorf("连接 AI 服务失败：%w", err)
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, 8<<20)
	body, err := io.ReadAll(limited)
	if err != nil {
		return AIMessage{}, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AIMessage{}, "", fmt.Errorf("AI 服务暂时没有返回可用结果（状态码 %d）", resp.StatusCode)
	}
	var parsed aiChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return AIMessage{}, "", fmt.Errorf("AI 返回内容格式不正确")
	}
	if len(parsed.Choices) == 0 {
		return AIMessage{}, "", fmt.Errorf("AI 没有返回修复内容")
	}
	return parsed.Choices[0].Message, parsed.Choices[0].FinishReason, nil
}
