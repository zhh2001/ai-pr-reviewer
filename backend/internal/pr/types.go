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

// Fetcher 拉取 PR 变更。具体实现见 internal/github。
type Fetcher interface {
	Fetch(ctx context.Context, ref Ref) (*PRChanges, error)
}
