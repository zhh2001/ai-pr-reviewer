package llm

import (
	openai "github.com/sashabaranov/go-openai"
)

// DeepSeekBaseURL 是 DeepSeek 提供的 OpenAI 兼容 endpoint。
const DeepSeekBaseURL = "https://api.deepseek.com"

// 模型分工：
//   - summaryModel / suggestionsModel: 总结与建议都是延迟敏感、容错性高的任务，
//     用便宜快的 flash。
//   - risksModel: 风险识别要求结构化 JSON、行号定位、误报控制，对推理能力比对延迟
//     更敏感，用 pro。把贵的推理预算留给最需要"准"的通道，这是有意的分层。
const (
	summaryModel     = "deepseek-v4-flash"
	risksModel       = "deepseek-v4-pro"
	suggestionsModel = "deepseek-v4-flash"
)

// Client 用 go-openai 走 DeepSeek，同时实现 pr.Summarizer、pr.RiskDetector、
// pr.SuggestionGenerator。三个接口共享同一份 http client，仅在调用时选择不同模型。
type Client struct {
	api              *openai.Client
	summaryModel     string
	risksModel       string
	suggestionsModel string
}

func NewClient(apiKey string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = DeepSeekBaseURL
	return &Client{
		api:              openai.NewClientWithConfig(cfg),
		summaryModel:     summaryModel,
		risksModel:       risksModel,
		suggestionsModel: suggestionsModel,
	}
}
