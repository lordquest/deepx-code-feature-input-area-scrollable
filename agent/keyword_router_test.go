package agent

import "testing"

func TestRouteByKeyword(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want string
	}{
		// 关键词命中(短消息也升级)
		{"en-refactor", "refactor the auth module", "pro"},
		{"en-debug-short", "debug this", "pro"},
		{"zh-refactor", "帮我重构这个", "pro"},
		{"zh-trad", "幫我重構這個", "pro"},
		{"ja-debug", "デバッグして", "pro"},
		{"ja-refactor", "リファクタリングしてください", "pro"},
		{"ko-refactor", "리팩토링 해주세요", "pro"},
		{"ko-debug", "디버깅 도와줘", "pro"},

		// 长度阈值
		{"short-no-keyword", "你好", "flash"},
		{"short-en", "hi", "flash"},
		{"short-question", "main.go 第 50 行写的什么", "flash"},

		// 中等长度无关键词 → flash
		{"medium-no-keyword-zh", "我想问问你这个 main.go 里的 loadEnvFile 函数读取顺序是怎么样的,优先级是怎么定的", "flash"},

		// 长消息 > 500 → pro (300 个"内容" = 600 字)
		{"long-no-keyword", "我有一个" + repeat("内容", 300), "pro"},

		// 大小写不敏感
		{"uppercase-en", "REFACTOR THE CODE", "pro"},
		{"mixed-case", "Debug this issue", "pro"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := RouteByKeyword(c.msg)
			if got != c.want {
				t.Errorf("RouteByKeyword(%q) = %q, want %q", c.msg, got, c.want)
			}
		})
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
