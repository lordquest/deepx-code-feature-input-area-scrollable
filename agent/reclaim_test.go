package agent

import (
	"fmt"
	"strings"
	"testing"
)

func toolMsg(id, name, content string) ChatMessage {
	return ChatMessage{Role: "tool", ToolCallID: id, Name: name, Content: content}
}

func asstCall(id, name, argsJSON string) ChatMessage {
	return ChatMessage{Role: "assistant", ToolCalls: []ToolCall{
		{ID: id, Type: "function", Function: ToolCallFunc{Name: name, Arguments: argsJSON}},
	}}
}

// buildConvo 造 1 system + 1 user + n 轮 (assistant call + tool result),每条工具结果内容为 content。
func buildConvo(n int, name, content string) []ChatMessage {
	convo := []ChatMessage{{Role: "system", Content: "sys"}, {Role: "user", Content: "任务"}}
	for k := 0; k < n; k++ {
		id := fmt.Sprintf("c%d", k)
		convo = append(convo, asstCall(id, name, fmt.Sprintf(`{"path":"src/f%d.go"}`, k)))
		convo = append(convo, toolMsg(id, name, content))
	}
	return convo
}

func countTool(convo []ChatMessage) (reclaimed, kept int) {
	for _, m := range convo {
		if m.Role != "tool" {
			continue
		}
		if strings.HasPrefix(m.Content, reclaimMarkerPrefix) {
			reclaimed++
		} else {
			kept++
		}
	}
	return
}

func TestReclaim_OldReplacedRecentKept(t *testing.T) {
	big := strings.Repeat("这是一大段读取内容。", 500) // 每条远超小预算
	convo := buildConvo(10, "Read", big)
	before := len(convo)

	if !reclaimToolOutputs(convo, 4096) { // 预算 = 4096*20% ≈ 819 token
		t.Fatal("应发生回收")
	}
	reclaimed, kept := countTool(convo)
	if reclaimed == 0 {
		t.Fatal("较旧工具结果应被回收")
	}
	if kept < reclaimMinTailToolMsgs {
		t.Fatalf("最近 %d 条应保留,实际保留 %d", reclaimMinTailToolMsgs, kept)
	}
	if len(convo) != before {
		t.Fatalf("reclaim 不应增删消息: %d -> %d", before, len(convo))
	}
}

func TestReclaim_TailProtected(t *testing.T) {
	convo := buildConvo(reclaimMinTailToolMsgs+3, "Bash", strings.Repeat("x", 5000))
	reclaimToolOutputs(convo, 1024) // 预算极小

	seen := 0
	for i := len(convo) - 1; i >= 0; i-- {
		if convo[i].Role != "tool" {
			continue
		}
		seen++
		if seen <= reclaimMinTailToolMsgs && strings.HasPrefix(convo[i].Content, reclaimMarkerPrefix) {
			t.Fatalf("最近第 %d 条工具结果被误回收", seen)
		}
	}
}

func TestReclaim_ReferenceHasNameAndPath(t *testing.T) {
	convo := buildConvo(8, "Read", strings.Repeat("y", 8000))
	reclaimToolOutputs(convo, 2048)

	found := false
	for _, m := range convo {
		if m.Role == "tool" && strings.HasPrefix(m.Content, reclaimMarkerPrefix) {
			if !strings.Contains(m.Content, "Read") || !strings.Contains(m.Content, "src/f") {
				t.Fatalf("引用应含工具名和 path, got=%q", m.Content)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("应有被回收的工具结果")
	}
}

func TestReclaim_Idempotent(t *testing.T) {
	convo := buildConvo(8, "Read", strings.Repeat("z", 6000))
	if !reclaimToolOutputs(convo, 2048) {
		t.Fatal("首次应有改动")
	}
	if reclaimToolOutputs(convo, 2048) {
		t.Fatal("再次 reclaim 不应有新改动(幂等)")
	}
}

func TestReclaim_NonToolUntouched(t *testing.T) {
	userBig := strings.Repeat("u", 9000)
	asstBig := strings.Repeat("a", 9000)
	convo := []ChatMessage{
		{Role: "system", Content: "s"},
		{Role: "user", Content: userBig},
		asstCall("c1", "Read", `{"path":"a.go"}`),
		{Role: "assistant", Content: asstBig},
	}
	before := len(convo)
	reclaimToolOutputs(convo, 1024)
	if convo[1].Content != userBig {
		t.Fatal("user 消息不应被改")
	}
	if convo[3].Content != asstBig {
		t.Fatal("assistant 文本不应被改")
	}
	if len(convo) != before {
		t.Fatal("消息数不应变")
	}
}

func TestReclaim_UnderBudgetNoChange(t *testing.T) {
	convo := buildConvo(6, "Read", "tiny") // 工具输出很小
	if reclaimToolOutputs(convo, 1_000_000) {
		t.Fatal("预算充足时不应回收")
	}
}

func TestReclaim_PairingPreserved(t *testing.T) {
	// 回收后每条 tool 消息的 ToolCallID / Name 不变,配对完整。
	convo := buildConvo(10, "Read", strings.Repeat("w", 6000))
	ids := map[int]string{}
	for i, m := range convo {
		if m.Role == "tool" {
			ids[i] = m.ToolCallID
		}
	}
	reclaimToolOutputs(convo, 2048)
	for i, m := range convo {
		if m.Role == "tool" {
			if m.ToolCallID != ids[i] {
				t.Fatalf("第 %d 条 ToolCallID 被改动", i)
			}
			if m.Name == "" {
				t.Fatalf("第 %d 条 Name 丢失", i)
			}
		}
	}
}
