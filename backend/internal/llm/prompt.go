package llm

import (
	"fmt"
	"strings"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// MaxPromptBytes 总结调用的输入字节预算。超出后按文件均分预算截断各自的 patch。
// 32KB 对 deepseek-v4-flash 已经够 review 一个中等 PR，又不至于把上下文窗口挤爆。
const MaxPromptBytes = 32 * 1024

const truncatedMarker = "\n... [truncated]"

const summarySystemPrompt = `你是一个资深工程师，正在做 code review 总结。要求：
- 用中文输出 5-10 句以内
- 直接说这个 PR 改了什么、为什么改、影响面
- 不要"本 PR 旨在……" / "综上所述" 这类套话，不要 emoji
- 不要逐个复述文件名，按主题归纳
- 只基于给到的 diff 说事，不要编造背景`

// Prompt 是一次 chat completion 的 system+user 双消息内容。
type Prompt struct {
	System string
	User   string
}

// BuildSummaryPrompt 把 PRChanges 组织成总结用的 prompt。
// 总输入受 MaxPromptBytes 控制。
func BuildSummaryPrompt(c *pr.PRChanges) Prompt {
	return buildSummaryPromptWithBudget(c, MaxPromptBytes)
}

func buildSummaryPromptWithBudget(c *pr.PRChanges, budget int) Prompt {
	var head strings.Builder
	fmt.Fprintf(&head, "PR: %s/%s#%d\n", c.Owner, c.Repo, c.Number)
	fmt.Fprintf(&head, "Title: %s\n", c.Title)
	if strings.TrimSpace(c.Description) != "" {
		fmt.Fprintf(&head, "Description:\n%s\n", c.Description)
	}
	fmt.Fprintf(&head, "Author: %s\nBranch: %s -> %s\nFiles changed: %d\n",
		c.Author, c.BaseRef, c.HeadRef, len(c.Files))

	patchBudget := budget - head.Len()
	if patchBudget < 0 {
		patchBudget = 0
	}
	perFile := patchBudget
	if len(c.Files) > 0 {
		perFile = patchBudget / len(c.Files)
	}

	var body strings.Builder
	body.WriteString(head.String())
	for _, f := range c.Files {
		fmt.Fprintf(&body, "\n--- %s (%s, +%d/-%d) ---\n",
			f.Path, f.Status, f.Additions, f.Deletions)
		body.WriteString(truncatePatch(f.Patch, perFile))
		body.WriteString("\n")
	}
	return Prompt{System: summarySystemPrompt, User: body.String()}
}

// truncatePatch 把 patch 限制在 max 字节内，超出则截断并附 [truncated] 标记。
// 返回值长度 <= max（除非 max < len(marker)，此时返回 marker 本身）。
func truncatePatch(patch string, max int) string {
	if len(patch) <= max {
		return patch
	}
	if max <= len(truncatedMarker) {
		return truncatedMarker
	}
	keep := max - len(truncatedMarker)
	return patch[:keep] + truncatedMarker
}
