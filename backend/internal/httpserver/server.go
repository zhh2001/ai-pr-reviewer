package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/config"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// New 用真实的 GitHub Fetcher 装配 handler。
func New(cfg config.Config) http.Handler {
	return NewWithFetcher(github.NewFetcher(cfg.GitHubToken))
}

// NewWithFetcher 暴露给测试，允许注入 mock Fetcher。
func NewWithFetcher(fetcher pr.Fetcher) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.Handle("POST /api/review", reviewHandler(fetcher))
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
