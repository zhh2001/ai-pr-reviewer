package llm

import (
	"context"
	"errors"

	openai "github.com/sashabaranov/go-openai"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// suggestions 与 risks 维度互补：
//   - risks 报"明显的问题"，需要结构化 JSON + 行号 + 误报控制。曾用 v4-pro 追求准，
//     实测在 48KB 上下文 + 思维链下经常超过 180s 分析超时，改回 v4-flash 后稳定能拿到结果。
//   - suggestions 报"可以更好的建议"，本来就非阻断、容错性高，flash 完全够用。
const suggestionsSystemPrompt = `你是一个资深工程师，正在为 PR 提改进建议。
注意：bug / 安全 / 性能 这类"问题/风险"由 risks 通道负责，不要在这里重复。
suggestions 只覆盖可读性、命名、补测试、结构、文档等增强性意见。

输出规则（必须严格遵守）：
- 只输出一个 JSON 对象，不要任何解释、不要 markdown 代码块、不要前后缀。
- 顶层结构必须是 {"suggestions": [...]}；没有任何建议时返回 {"suggestions": []}。
- 每个 suggestion 字段：
    file (string): 文件路径
    line (int): 行号，无法定位时填 0
    category (string): "readability" | "testing" | "naming" | "structure" | "docs"
    suggestion (string): 简体中文，具体、可操作

判断准则：
- 必须是具体、可操作的建议（"把 X 抽成 helper"、"为 Y 的零长度入参补一个测试用例"）。
- 不要无意义的风格吹毛求疵，不要"建议加注释"这种泛泛之谈。
- 没有可提的就返回空数组，不要凑数。`

func (c *Client) GenerateSuggestions(ctx context.Context, changes *pr.PRChanges) ([]pr.Suggestion, error) {
	p := BuildSuggestionsPrompt(changes)
	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.suggestionsModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: p.System},
			{Role: openai.ChatMessageRoleUser, Content: p.User},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("deepseek returned no choices")
	}
	return ParseSuggestionsJSON(resp.Choices[0].Message.Content)
}

// ParseSuggestionsJSON 解析模型返回的 suggestions JSON。
func ParseSuggestionsJSON(raw string) ([]pr.Suggestion, error) {
	return parseEnvelopedJSON[pr.Suggestion](raw, "suggestions")
}
