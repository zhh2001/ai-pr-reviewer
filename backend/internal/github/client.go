package github

import gh "github.com/google/go-github/v66/github"

// Client 仅占位，本期不接线。后续会用 go-github 拉取 PR diff。
type Client struct {
	api *gh.Client
}
