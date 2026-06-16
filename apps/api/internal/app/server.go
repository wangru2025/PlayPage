package app

import (
	"net/http"
	"time"

	"ai-static-host/api/internal/config"
	aphttp "ai-static-host/api/internal/http"
)

func NewServer(cfg config.Config) *http.Server {
	return &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           aphttp.NewRouter(cfg),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
