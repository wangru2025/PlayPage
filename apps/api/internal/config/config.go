package config

import "os"

type Config struct {
	ListenAddr            string
	AppName               string
	DatabaseURL           string
	DataRoot              string
	PublicBase            string
	SMTPHost              string
	SMTPPort              string
	SMTPUsername          string
	SMTPPassword          string
	SMTPFromEmail         string
	AIProviderBaseURL     string
	AIProviderAPIKey      string
	AIProviderModel       string
	GitHubToken           string
	GitHubOwner           string
	GitHubRepo            string
	GitHubAppWorkflow     string
	AppBuildCallbackToken string
}

func Load() Config {
	return Config{
		ListenAddr:            env("LISTEN_ADDR", "127.0.0.1:8080"),
		AppName:               env("APP_NAME", "AI Static Host"),
		DatabaseURL:           env("DATABASE_URL", ""),
		DataRoot:              env("DATA_ROOT", "/data/apps"),
		PublicBase:            env("PUBLIC_BASE_URL", "https://example.com"),
		SMTPHost:              env("SMTP_HOST", ""),
		SMTPPort:              env("SMTP_PORT", "587"),
		SMTPUsername:          env("SMTP_USERNAME", ""),
		SMTPPassword:          env("SMTP_PASSWORD", ""),
		SMTPFromEmail:         env("SMTP_FROM_EMAIL", ""),
		AIProviderBaseURL:     env("AI_PROVIDER_BASE_URL", "https://apihub.agnes-ai.com/v1/chat/completions"),
		AIProviderAPIKey:      env("AI_PROVIDER_API_KEY", ""),
		AIProviderModel:       env("AI_PROVIDER_MODEL", "agnes-2.0-flash"),
		GitHubToken:           env("GITHUB_TOKEN", ""),
		GitHubOwner:           env("GITHUB_OWNER", "wangru2025"),
		GitHubRepo:            env("GITHUB_APP_BUILD_REPO", "PlayPageAppBuilder"),
		GitHubAppWorkflow:     env("GITHUB_APP_BUILD_WORKFLOW", "playpage-app-build.yml"),
		AppBuildCallbackToken: env("APP_BUILD_CALLBACK_TOKEN", ""),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
