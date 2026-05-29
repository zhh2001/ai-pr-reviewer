package github

import (
	"context"

	gh "github.com/google/go-github/v66/github"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// Fetcher 用 go-github 实现 pr.Fetcher。
type Fetcher struct {
	api *gh.Client
}

func NewFetcher(token string) *Fetcher {
	api := gh.NewClient(nil)
	if token != "" {
		api = api.WithAuthToken(token)
	}
	return &Fetcher{api: api}
}

func (f *Fetcher) Fetch(ctx context.Context, ref pr.Ref) (*pr.PRChanges, error) {
	p, _, err := f.api.PullRequests.Get(ctx, ref.Owner, ref.Repo, ref.Number)
	if err != nil {
		return nil, err
	}
	files, err := f.listAllFiles(ctx, ref)
	if err != nil {
		return nil, err
	}
	return toChanges(ref, p, files), nil
}

func (f *Fetcher) listAllFiles(ctx context.Context, ref pr.Ref) ([]*gh.CommitFile, error) {
	var all []*gh.CommitFile
	opt := &gh.ListOptions{PerPage: 100}
	for {
		batch, resp, err := f.api.PullRequests.ListFiles(ctx, ref.Owner, ref.Repo, ref.Number, opt)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}
