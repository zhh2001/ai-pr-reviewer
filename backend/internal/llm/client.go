package llm

import (
	openai "github.com/sashabaranov/go-openai"
)

// DeepSeekBaseURL 是 DeepSeek 提供的 OpenAI 兼容 endpoint。
const DeepSeekBaseURL = "https://api.deepseek.com"

// 模型分工：
//   - summaryModel: 总结只需要"说清楚改了什么"，延迟敏感，用便宜快的 flash。
//   - risksModel:   风险识别要求结构化 JSON、行号定位、误报控制，对推理能力比对
//     延迟更敏感，用 pro。这是有意的成本/质量分层。
const (
	summaryModel = "deepseek-v4-flash"
	risksModel   = "deepseek-v4-pro"
)

// Client 用 go-openai 走 DeepSeek，同时实现 pr.Summarizer 和 pr.RiskDetector。
type Client struct {
	api          *openai.Client
	summaryModel string
	risksModel   string
}

func NewClient(apiKey string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = DeepSeekBaseURL
	return &Client{
		api:          openai.NewClientWithConfig(cfg),
		summaryModel: summaryModel,
		risksModel:   risksModel,
	}
}
