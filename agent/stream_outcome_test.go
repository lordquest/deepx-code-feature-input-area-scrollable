package agent

import "testing"

// TestClassifyStreamResult 覆盖 issue #169 的两类异常响应分类:
// tool_call 被长度截断 / 空响应,以及正常情形不被误判。
func TestClassifyStreamResult(t *testing.T) {
	oneTool := []ToolCall{{ID: "x", Function: ToolCallFunc{Name: "Write", Arguments: `{"path":"a"`}}}

	cases := []struct {
		name      string
		content   string
		reasoning string
		toolCalls []ToolCall
		truncated bool
		want      streamOutcome
	}{
		// 会话 A:超大 Write 撞输出上限,tool_call arguments 残缺。
		{"truncated tool call", "", "", oneTool, true, outcomeTruncatedTool},
		// 有 content 的截断也归为截断工具(只要还带着工具调用)。
		{"truncated with content and tool", "思考中", "", oneTool, true, outcomeTruncatedTool},
		// 会话 B:供应商返回完全空。
		{"empty response", "", "", nil, false, outcomeEmpty},
		// finish_reason=length 但什么都没生成 → 也当空响应,催重试。
		{"truncated but nothing generated", "", "", nil, true, outcomeEmpty},
		// 正常:有文本、无工具。
		{"normal text", "结果如下", "", nil, false, outcomeNormal},
		// 正常:只有 reasoning(thinking 模型),非空。
		{"reasoning only", "", "让我想想", nil, false, outcomeNormal},
		// 正常:完整工具调用,未截断。
		{"complete tool call", "", "", oneTool, false, outcomeNormal},
		// 纯文本被截断(无工具)→ 走正常路径,交给 completionGate 催继续,不算截断工具。
		{"truncated plain text", "写到一半就断了", "", nil, true, outcomeNormal},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := classifyStreamResult(c.content, c.reasoning, c.toolCalls, c.truncated)
			if got != c.want {
				t.Fatalf("classifyStreamResult(%q,%q,tools=%d,trunc=%v) = %v, want %v",
					c.content, c.reasoning, len(c.toolCalls), c.truncated, got, c.want)
			}
		})
	}
}

// TestCompletionGateCap 验证连续催继续不会无限循环:达到 maxGateNudges 后放行(返回空)。
func TestCompletionGateCap(t *testing.T) {
	nudges := 0
	got := 0
	for range maxGateNudges + 3 {
		if completionGate(true, nil, &nudges) != "" {
			got++
		}
	}
	if got != maxGateNudges {
		t.Fatalf("completionGate 应最多催 %d 次,实际 %d 次", maxGateNudges, got)
	}
}
