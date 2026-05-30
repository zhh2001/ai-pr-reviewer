package llm

import "testing"

// 把三个模型常量锁进测试。risksModel 历史上是 v4-pro，因实测频繁超过分析超时
// 改回 v4-flash（详见 client.go 注释）。这条测试防止未来不读注释又改回去。
func TestModelConstants(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"summary", summaryModel, "deepseek-v4-flash"},
		{"risks", risksModel, "deepseek-v4-flash"},
		{"suggestions", suggestionsModel, "deepseek-v4-flash"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%sModel = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestNewClient_BindsModelsToFields(t *testing.T) {
	c := NewClient("test-key")
	if c.summaryModel != summaryModel {
		t.Errorf("summaryModel field = %q, want %q", c.summaryModel, summaryModel)
	}
	if c.risksModel != risksModel {
		t.Errorf("risksModel field = %q, want %q", c.risksModel, risksModel)
	}
	if c.suggestionsModel != suggestionsModel {
		t.Errorf("suggestionsModel field = %q, want %q", c.suggestionsModel, suggestionsModel)
	}
}
