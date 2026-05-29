package llm

import (
	"fmt"
	"strings"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// 不同任务给不同输入预算：
//   - Summary 只需把改动说清，32KB 够用。
//   - Risks 需要更完整的 patch 上下文以精确定位行号、判断危险代码，
//     截断越激进越容易漏报或误报，预算适当放大。
const (
	MaxSummaryPromptBytes = 32 * 1024
	MaxRisksPromptBytes   = 48 * 1024
)

const truncatedMarker = "\n... [truncated]"

// Prompt 是一次 chat completion 的 system+user 双消息内容。
type Prompt struct {
	System string
	User   string
}

// BuildSummaryPrompt 把 PRChanges 组织成总结用的 prompt。
func BuildSummaryPrompt(c *pr.PRChanges) Prompt {
	return buildSummaryPromptWithBudget(c, MaxSummaryPromptBytes)
}

func buildSummaryPromptWithBudget(c *pr.PRChanges, budget int) Prompt {
	return Prompt{System: summarySystemPrompt, User: renderContext(c, budget)}
}

// BuildRisksPrompt 把 PRChanges 组织成风险识别用的 prompt。
func BuildRisksPrompt(c *pr.PRChanges) Prompt {
	return buildRisksPromptWithBudget(c, MaxRisksPromptBytes)
}

func buildRisksPromptWithBudget(c *pr.PRChanges, budget int) Prompt {
	return Prompt{System: risksSystemPrompt, User: renderContext(c, budget)}
}

// renderContext 渲染 PR 上下文（标题、描述、文件列表、各文件 patch）成一段文本，
// 总输入受 budget 控制：剩余预算均分给各文件 patch，超出按字节截断并标注 [truncated]。
func renderContext(c *pr.PRChanges, budget int) string {
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
	return body.String()
}

// truncatePatch 把 patch 限制在 max 字节内，超出则截断并附 [truncated] 标记。
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
