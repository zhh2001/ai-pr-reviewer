package github

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

var shorthandRe = regexp.MustCompile(`^([^/\s#]+)/([^/\s#]+)#(\d+)$`)

// ParseRef 解析两种 PR 引用：
//   - https://github.com/{owner}/{repo}/pull/{number}
//   - {owner}/{repo}#{number}
func ParseRef(s string) (pr.Ref, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return pr.Ref{}, errors.New("empty pr reference")
	}
	if strings.Contains(s, "#") {
		return parseShorthand(s)
	}
	return parseURL(s)
}

func parseURL(s string) (pr.Ref, error) {
	u, err := url.Parse(s)
	if err != nil {
		return pr.Ref{}, fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return pr.Ref{}, fmt.Errorf("unsupported url scheme: %q", u.Scheme)
	}
	if u.Host != "github.com" {
		return pr.Ref{}, fmt.Errorf("not a github.com url: host=%q", u.Host)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" {
		return pr.Ref{}, fmt.Errorf("path does not match /{owner}/{repo}/pull/{number}: %q", u.Path)
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil || n <= 0 {
		return pr.Ref{}, fmt.Errorf("invalid pr number: %q", parts[3])
	}
	return pr.Ref{Owner: parts[0], Repo: parts[1], Number: n}, nil
}

func parseShorthand(s string) (pr.Ref, error) {
	m := shorthandRe.FindStringSubmatch(s)
	if m == nil {
		return pr.Ref{}, fmt.Errorf("not a valid shorthand {owner}/{repo}#{number}: %q", s)
	}
	n, err := strconv.Atoi(m[3])
	if err != nil || n <= 0 {
		return pr.Ref{}, fmt.Errorf("invalid pr number: %q", m[3])
	}
	return pr.Ref{Owner: m[1], Repo: m[2], Number: n}, nil
}
