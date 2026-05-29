package llm

import (
	"context"
	"errors"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

const (
	// DeepSeekBaseURL 是 DeepSeek 提供的 OpenAI 兼容 endpoint。
	DeepSeekBaseURL = "https://api.deepseek.com"
	// SummaryModel：CLAUDE.md 指定 deepseek-v4-flash 用于总结。
	SummaryModel = "deepseek-v4-flash"
)

// Client 用 go-openai 走 DeepSeek，实现 pr.Summarizer。
type Client struct {
	api   *openai.Client
	model string
}

func NewClient(apiKey string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = DeepSeekBaseURL
	return &Client{
		api:   openai.NewClientWithConfig(cfg),
		model: SummaryModel,
	}
}

func (c *Client) Summarize(ctx context.Context, changes *pr.PRChanges) (string, error) {
	p := BuildSummaryPrompt(changes)
	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
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
