package llm

import (
	"context"
	"errors"

	openai "github.com/sashabaranov/go-openai"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// 系统提示同时显式要求"只输出 JSON、不要 markdown 代码块"：
// json_object 模式只保证返回是合法 JSON，并不阻止模型把 JSON 包在 ```json fence 里
// 或加解释段；而 DeepSeek 若没有这条硬约束有时会直接吐空白卡住。
const risksSystemPrompt = `你是一个资深工程师，正在做 PR 风险 review。

输出规则（必须严格遵守）：
- 只输出一个 JSON 对象，不要任何解释、不要 markdown 代码块、不要前后缀。
- 顶层结构必须是 {"risks": [...]}；没有任何风险时返回 {"risks": []}。
- 每个 risk 字段：
    file (string): 文件路径
    line (int): 行号，无法定位时填 0
    severity (string): "high" | "medium" | "low"
    category (string): "bug" | "security" | "performance" | "maintainability" | "style"
    description (string): 简体中文，简洁，落到具体行为或位置
    confidence (float): 0~1，对不确定的判断给低分

判断准则：
- 只报真正值得关注的问题，宁可少报，不要为凑数误报。
- description 不要"建议加测试"这类泛泛之谈，要指出具体代码层面的隐患。
- 这是"问题/风险"维度。可读性、命名、补测试等改进性意见不在这里输出。`

func (c *Client) DetectRisks(ctx context.Context, changes *pr.PRChanges) ([]pr.Risk, error) {
	p := BuildRisksPrompt(changes)
	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.risksModel,
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
	return ParseRisksJSON(resp.Choices[0].Message.Content)
}

// ParseRisksJSON 解析模型返回的 risks JSON。
func ParseRisksJSON(raw string) ([]pr.Risk, error) {
	return parseEnvelopedJSON[pr.Risk](raw, "risks")
}
