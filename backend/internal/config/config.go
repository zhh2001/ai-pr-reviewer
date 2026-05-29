package config

import (
	"os"
	"strconv"
	"time"
)

// DefaultAnalyzeTimeout 是 LLM 分析阶段的兜底超时。90s 覆盖三个并发任务里最慢的
// 一个调用（含 DeepSeek 偶发慢响应），又不至于让客户端等到天荒地老。
const DefaultAnalyzeTimeout = 90 * time.Second

// DefaultRiskConfidenceThreshold 是 risks 通道的误报过滤默认阈值。
// 0.5 偏中性：低于这个值的判断通常更接近"猜测"，剔掉能显著降低误报。
const DefaultRiskConfidenceThreshold = 0.5

type Config struct {
	Addr                    string
	GitHubToken             string
	DeepSeekAPIKey          string
	AnalyzeTimeout          time.Duration
	RiskConfidenceThreshold float64
}

func Load() Config {
	return Config{
		Addr:                    envOr("ADDR", ":8080"),
		GitHubToken:             os.Getenv("GITHUB_TOKEN"),
		DeepSeekAPIKey:          os.Getenv("DEEPSEEK_API_KEY"),
		AnalyzeTimeout:          envDurationSeconds("ANALYZE_TIMEOUT_SECONDS", DefaultAnalyzeTimeout),
		RiskConfidenceThreshold: envFloatClamped("RISK_CONFIDENCE_THRESHOLD", 0, 1, DefaultRiskConfidenceThreshold),
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

// envFloatClamped 解析浮点环境变量：
//   - 缺省 / 非法 → fallback
//   - 越界 → 夹紧到 [minVal, maxVal]
func envFloatClamped(key string, minVal, maxVal, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	if f < minVal {
		return minVal
	}
	if f > maxVal {
		return maxVal
	}
	return f
}
