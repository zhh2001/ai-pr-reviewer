package llm

import (
	openai "github.com/sashabaranov/go-openai"
)

// DeepSeekBaseURL 是 DeepSeek 提供的 OpenAI 兼容 endpoint。
const DeepSeekBaseURL = "https://api.deepseek.com"

// 模型分工：
//   - summaryModel / suggestionsModel: 总结与建议都是延迟敏感、容错性高的任务，
//     用便宜快的 flash。
//   - risksModel: 风险识别原先选 pro 是为了"准"——更强的推理 + 严格的结构化输出。
//     真实联调发现 pro 上的 risks 通道在 48KB 上下文 + 思维链推理下经常超过
//     180s 分析超时，结果整段 risks 通过 risks_error 降级丢掉。
//     v4-flash 默认开启思考模式，推理质量已经接近 pro 而延迟显著低，
//     于是改成 flash：准确性轻微让步，换回稳定能返回的 risks——一个稍弱但
//     能拿到结果的判断，胜过一个更强但永远 timeout 的判断。
const (
	summaryModel     = "deepseek-v4-flash"
	risksModel       = "deepseek-v4-flash"
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
