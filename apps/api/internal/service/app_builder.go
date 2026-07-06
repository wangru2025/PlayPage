package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
)

type AppBuilder struct {
	token    string
	owner    string
	repo     string
	workflow string
	client   *http.Client
}

func NewAppBuilder(token, owner, repo, workflow string) *AppBuilder {
	return &AppBuilder{
		token:    strings.TrimSpace(token),
		owner:    strings.TrimSpace(owner),
		repo:     strings.TrimSpace(repo),
		workflow: strings.TrimSpace(workflow),
		client:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (b *AppBuilder) Enabled() bool {
	return b.token != "" && b.owner != "" && b.repo != "" && b.workflow != ""
}

func (b *AppBuilder) Dispatch(ctx context.Context, job domain.AppBuildJob, callbackBaseURL, callbackToken string) error {
	if !b.Enabled() {
		return fmt.Errorf("安装包构建仓库还没有配置")
	}
	payload := map[string]any{
		"ref": "main",
		"inputs": map[string]string{
			"job_id":         job.ID,
			"project_id":     job.ProjectID,
			"release_id":     job.ReleaseID,
			"platform":       job.Platform,
			"callback_base":  strings.TrimRight(callbackBaseURL, "/"),
			"callback_token": callbackToken,
		},
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", b.owner, b.repo, b.workflow)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub 构建仓库返回状态码 %d", resp.StatusCode)
	}
	return nil
}
