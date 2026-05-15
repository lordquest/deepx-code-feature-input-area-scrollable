package tools

import (
	"fmt"
	"strings"

	"deepx/session"
)

// currentSession 由 tui 启动时通过 SetMemorySession 注入。
// 工具的 Executor 签名固定为 map[string]any → ToolResult,没法直接传依赖,
// 用包级变量是最小侵入。运行时单 session,不存在竞态。
var currentSession *session.Manager

// SetMemorySession 注入当前 session 管理器。tui.initialModel 启动时调一次即可。
func SetMemorySession(m *session.Manager) {
	currentSession = m
}

// Memory 检索当前 workspace 的历史对话。
// LLM 决定关键词(可多个),工具内部用大小写不敏感的全文匹配扫所有日期文件。
// 命中输出:每条 = 日期 / 角色 / 截断 snippet,LLM 据此判断是否需要展开继续追问。
func Memory(args map[string]any) ToolResult {
	if currentSession == nil {
		return ToolResult{Output: "记忆系统未启用(session 未初始化)", Success: false}
	}

	kwsAny, _ := args["keywords"].([]any)
	if len(kwsAny) == 0 {
		return ToolResult{Output: "keywords 不能为空,至少给 1 个关键词", Success: false}
	}
	kws := make([]string, 0, len(kwsAny))
	for _, k := range kwsAny {
		s, ok := k.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			kws = append(kws, s)
		}
	}
	if len(kws) == 0 {
		return ToolResult{Output: "keywords 全为空字符串", Success: false}
	}

	mode, _ := args["mode"].(string)
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "or" {
		mode = "and"
	}
	maxResults := toInt(args["max_results"], 10)
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 50 {
		maxResults = 50
	}

	hits := currentSession.Search(kws, mode, maxResults)
	if len(hits) == 0 {
		return ToolResult{
			Output: fmt.Sprintf("未命中(keywords=[%s], mode=%s)。可尝试换近义词、拆短、或改 mode=or",
				strings.Join(kws, ", "), mode),
			Success: true,
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "命中 %d 条历史对话(keywords=[%s], mode=%s):\n\n",
		len(hits), strings.Join(kws, ", "), mode)
	for i, h := range hits {
		snippet := h.Entry.Content
		// 单条裁到 400 字符,避免一次返回把上下文塞爆
		runes := []rune(snippet)
		if len(runes) > 400 {
			snippet = string(runes[:400]) + "..."
		}
		fmt.Fprintf(&sb, "[%d] %s | %s | %s\n%s\n\n",
			i+1, h.Date, h.Entry.Ts.Format("15:04"), h.Entry.Role, snippet)
	}
	return ToolResult{Output: sb.String(), Success: true}
}
