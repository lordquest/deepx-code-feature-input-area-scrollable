package tui

import (
	"strings"
	"testing"
)

func TestFormatPastedTextRef(t *testing.T) {
	if got := formatPastedTextRef(1, 0); got != "[Pasted text #1]" {
		t.Fatalf("numLines=0: got %q", got)
	}
	if got := formatPastedTextRef(3, 12); got != "[Pasted text #3 +12 lines]" {
		t.Fatalf("numLines>0: got %q", got)
	}
}

func TestExpandPastedTextRefs(t *testing.T) {
	store := map[int]string{
		1: "line1\nline2\nline3",
		2: "alpha\nbeta\n",
	}
	// 单占位符
	if got := expandPastedTextRefs("[Pasted text #1 +2 lines]", store); got != "line1\nline2\nline3" {
		t.Fatalf("single: got %q", got)
	}
	// 多占位符(混合文本),验证顺序与偏移
	in := "head [Pasted text #1 +2 lines] mid [Pasted text #2 +1 lines] tail"
	want := "head line1\nline2\nline3 mid alpha\nbeta\n tail"
	if got := expandPastedTextRefs(in, store); got != want {
		t.Fatalf("multi: got %q want %q", got, want)
	}
	// 无对应存储 → 保留占位符原样
	if got := expandPastedTextRefs("[Pasted text #9 +5 lines]", store); got != "[Pasted text #9 +5 lines]" {
		t.Fatalf("missing: got %q", got)
	}
}

// TestPasteThresholdLogic 验证触发占位符的判定与 free-code 一致:>800 字符或 >2 行。
func TestPasteThresholdLogic(t *testing.T) {
	// 与源码一致:行数阈值 = min(h-10, pasteLineThresholdCap);常态终端 h=30 → 2。
	check := func(text string, h int) bool {
		c := h - 10
		if pasteLineThresholdCap < c {
			c = pasteLineThresholdCap
		}
		return len(text) > pasteTextThreshold || strings.Count(text, "\n") > c
	}
	// 短单行 → 不触发
	if check("hello world", 30) {
		t.Fatal("short single line should NOT trigger")
	}
	// 恰好 800 字符、1 行 → 不触发(严格大于)
	if check(strings.Repeat("a", 800), 30) {
		t.Fatal("exactly 800 chars, 1 line should NOT trigger")
	}
	// 801 字符 → 触发
	if !check(strings.Repeat("a", 801), 30) {
		t.Fatal("801 chars should trigger")
	}
	// 4 行(3 个换行) → 触发(free-code: numLines>2 才占位)
	if !check("a\nb\nc\nd", 30) {
		t.Fatal("4 lines (3 newlines) should trigger")
	}
	// 3 行(2 个换行) → 不触发
	if check("a\nb\nc", 30) {
		t.Fatal("3 lines (2 newlines) should NOT trigger")
	}
	// 2 行(1 个换行) → 不触发
	if check("a\nb", 30) {
		t.Fatal("2 lines should NOT trigger")
	}
	// 矮终端(height=11 → cap=1):2 个换行(3 行)也会触发(numLines>1)
	if !check("a\nb\nc", 11) {
		t.Fatal("short terminal (h=11,cap=1) should trigger on 2 newlines")
	}
}

// TestPrunePastedTexts 验证回收逻辑:只删"占位符已无处引用"的条目,
// 保留仍在 input / pendingInputOriginal / queuedInput 中的 id(尤其 Esc 打断回填路径依赖 pendingInputOriginal)。
func TestPrunePastedTexts(t *testing.T) {
	// 场景1:input 含 id1、id2 占位符,id3 无引用 → 只留 1、2
	m := initModel()
	m.pastedTexts = map[int]string{1: "full1", 2: "full2", 3: "full3"}
	m.input.SetValue("[Pasted text #1] keep [Pasted text #2]")
	m.pendingInputOriginal = ""
	m.queuedInput = nil
	m.prunePastedTexts()
	if len(m.pastedTexts) != 2 {
		t.Fatalf("场景1:期望保留 2 条,实际 %d: %v", len(m.pastedTexts), m.pastedTexts)
	}
	if _, ok := m.pastedTexts[3]; ok {
		t.Errorf("❌ 未回收无引用的 id3")
	}
	if _, ok := m.pastedTexts[1]; !ok {
		t.Errorf("❌ 误删仍在 input 的 id1")
	}

	// 场景2:input 空,但 pendingInputOriginal 含 id1(模拟 Esc 打断回填前状态)
	// → id1 必须保留,无引用的 id2 回收。
	m2 := initModel()
	m2.pastedTexts = map[int]string{1: "full1", 2: "full2"}
	m2.input.SetValue("")
	m2.pendingInputOriginal = "[Pasted text #1]"
	m2.queuedInput = nil
	m2.prunePastedTexts()
	if _, ok := m2.pastedTexts[1]; !ok {
		t.Errorf("❌ 误删 pendingInputOriginal 引用的 id1(Esc 回填会丢字)")
	}
	if _, ok := m2.pastedTexts[2]; ok {
		t.Errorf("❌ 未回收无引用的 id2")
	}

	// 场景3:占位符数字与 store id 不一致 → 不匹配的 id 应被回收
	m3 := initModel()
	m3.pastedTexts = map[int]string{5: "full5"}
	m3.input.SetValue("[Pasted text #9 +3 lines]") // id9 无存储
	m3.pendingInputOriginal = ""
	m3.queuedInput = nil
	m3.prunePastedTexts()
	if len(m3.pastedTexts) != 0 {
		t.Fatalf("场景3:占位符 id9 无存储,store 应清空,实际 %v", m3.pastedTexts)
	}
}
