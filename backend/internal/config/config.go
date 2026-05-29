package config

import (
	"os"
	"strconv"
	"time"
)

// DefaultAnalyzeTimeout 是 LLM 分析阶段的兜底超时。90s 覆盖三个并发任务里最慢的
// 一个调用（含 DeepSeek 偶发慢响应），又不至于让客户端等到天荒地老。
const DefaultAnalyzeTimeout = 90 * time.Second

type Config struct {
	Addr           string
	GitHubToken    string
	DeepSeekAPIKey string
	AnalyzeTimeout time.Duration
}

func Load() Config {
	return Config{
		Addr:           envOr("ADDR", ":8080"),
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		DeepSeekAPIKey: os.Getenv("DEEPSEEK_API_KEY"),
		AnalyzeTimeout: envDurationSeconds("ANALYZE_TIMEOUT_SECONDS", DefaultAnalyzeTimeout),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}
