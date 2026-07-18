package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestRenderUserBubbleNoOverflow 验证用户消息里的超长无空格单行(日志/JSON/minified
// 代码)会被硬折行、且每行都补白到统一宽度铺满 chat 区。这里同时钉死两件事:
//   - 不横向溢出:任何行显示宽度都不超过 viewportW(= 色条 1 列 + boxW)。
//   - 不双倍行距:lipgloss 的 Width 已经硬折行 + 补白,绝不能再套 ansi.Wrap 二次折行 ——
//     那会把行尾补白顶到新行、生成交替空行并打散补白。故此处要求每行都恰好铺满 viewportW、
//     且每行都含实际内容(无空的 "┃ " 行)。任一条不满足即说明二次折行回归复现。
func TestRenderUserBubbleNoOverflow(t *testing.T) {
	const viewportW = 40
	longNoSpace := strings.Repeat("x", 200) // 200 字符无空格,远超 boxW
	out := renderUserBubble(longNoSpace, viewportW)

	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		t.Fatal("empty output")
	}
	for i, line := range lines {
		plain := ansi.Strip(line)
		w := ansi.StringWidth(plain)
		// 每行都应铺满 viewportW:不足=补白被打散(二次折行回归),超过=横向溢出。
		if w != viewportW {
			t.Fatalf("line %d width %d != viewportW %d (overflow or broken padding): %q", i, w, viewportW, plain)
		}
		// 去掉色条与补白后必须还有实际内容:出现空行=二次折行插入的交替空行。
		if strings.Trim(plain, "┃ ") == "" {
			t.Fatalf("line %d is a blank/padding-only row (double-wrap regression): %q", i, plain)
		}
	}
}

// TestRenderUserBubbleNormalText 验证普通带空格文本仍正常按行渲染。
func TestRenderUserBubbleNormalText(t *testing.T) {
	out := renderUserBubble("hello world\nsecond line", 40)
	if !strings.Contains(out, "hello world") || !strings.Contains(out, "second line") {
		t.Fatalf("normal text lost: %q", out)
	}
}
