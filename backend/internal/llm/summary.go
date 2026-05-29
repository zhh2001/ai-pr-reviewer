package llm

import (
	"context"
	"errors"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

const summarySystemPrompt = `你是一个资深工程师，正在做 code review 总结。要求：
- 用中文输出 5-10 句以内
- 直接说这个 PR 改了什么、为什么改、影响面
- 不要"本 PR 旨在……" / "综上所述" 这类套话，不要 emoji
- 不要逐个复述文件名，按主题归纳
- 只基于给到的 diff 说事，不要编造背景`

func (c *Client) Summarize(ctx context.Context, changes *pr.PRChanges) (string, error) {
	p := BuildSummaryPrompt(changes)
	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.summaryModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: p.System},
			{Role: openai.ChatMessageRoleUser, Content: p.User},
		},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("deepseek returned no choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
