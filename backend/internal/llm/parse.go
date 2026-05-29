package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseEnvelopedJSON 解析形如 {"key":[...]} 的对象，把数组反序列化为 []T。
// 即便系统提示已禁止 markdown，模型偶尔仍包 ```json fence，这里做一层兜底剥离。
// 字段缺失 / 为 null 时视作空列表，不视为错误。
func parseEnvelopedJSON[T any](raw, key string) ([]T, error) {
	s := stripFences(strings.TrimSpace(raw))
	var env map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		snippet := s
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("parse %s json: %w; got=%q", key, err, snippet)
	}
	rawArr, ok := env[key]
	if !ok {
		return []T{}, nil
	}
	var out []T
	if err := json.Unmarshal(rawArr, &out); err != nil {
		return nil, fmt.Errorf("parse %s array: %w", key, err)
	}
	if out == nil {
		out = []T{}
	}
	return out, nil
}

// stripFences 去掉首尾 ```（可带语言标签如 ```json），其它内容原样保留。
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		s = s[nl+1:]
	} else {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimRight(s, " \n\r\t")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
