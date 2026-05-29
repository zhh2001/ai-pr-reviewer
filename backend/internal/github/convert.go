package github

import (
	gh "github.com/google/go-github/v66/github"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func toChanges(ref pr.Ref, p *gh.PullRequest, files []*gh.CommitFile) *pr.PRChanges {
	c := &pr.PRChanges{
		Owner:       ref.Owner,
		Repo:        ref.Repo,
		Number:      ref.Number,
		Title:       p.GetTitle(),
		Description: p.GetBody(),
		Author:      p.GetUser().GetLogin(),
		BaseRef:     p.GetBase().GetRef(),
		HeadRef:     p.GetHead().GetRef(),
		Files:       make([]pr.FileChange, 0, len(files)),
	}
	for _, f := range files {
		c.Files = append(c.Files, pr.FileChange{
			Path:      f.GetFilename(),
			Status:    f.GetStatus(),
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
			Patch:     f.GetPatch(),
		})
	}
	return c
}
