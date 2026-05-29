package pr

import "context"

// Ref 唯一定位一个 PR。
type Ref struct {
	Owner  string
	Repo   string
	Number int
}

// PRChanges 是一个 PR 的领域视图，不依赖 GitHub SDK 类型。
type PRChanges struct {
	Owner       string       `json:"owner"`
	Repo        string       `json:"repo"`
	Number      int          `json:"number"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Author      string       `json:"author"`
	BaseRef     string       `json:"base_ref"`
	HeadRef     string       `json:"head_ref"`
	Files       []FileChange `json:"files"`
}

// FileChange 是 PR 中单个文件的变更。
type FileChange struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

// Risk 是单条 review 风险点。Line 无法定位时为 0。
type Risk struct {
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Severity    string  `json:"severity"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
}

// ReviewResult 是 /api/review 的返回信封。
// 之后做风险建议时往这里加字段。
type ReviewResult struct {
	Changes      *PRChanges `json:"changes"`
	Summary      string     `json:"summary,omitempty"`
	SummaryError string     `json:"summary_error,omitempty"`
	Risks        []Risk     `json:"risks,omitempty"`
	RisksError   string     `json:"risks_error,omitempty"`
}

// Fetcher 拉取 PR 变更。具体实现见 internal/github。
type Fetcher interface {
	Fetch(ctx context.Context, ref Ref) (*PRChanges, error)
}

// Summarizer 对 PR 变更产出自然语言总结。具体实现见 internal/llm。
type Summarizer interface {
	Summarize(ctx context.Context, changes *PRChanges) (string, error)
}

// RiskDetector 对 PR 变更产出结构化风险点。具体实现见 internal/llm。
type RiskDetector interface {
	DetectRisks(ctx context.Context, changes *PRChanges) ([]Risk, error)
}
