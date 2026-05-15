package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// cellPos 表示一个 chat 内部坐标:
//   - col  = 显示列 (0..viewport.Width-1)
//   - line = 经 ansi.Wrap 后的行号(在 chatContent 的总行集合内)
//
// 用"已 wrap 行号"而非"内容字节偏移"的好处:
//   - 用户在屏幕看到的就是 wrapped lines,鼠标拖拽方向跟它一一对应
//   - 内容只增不减(append-only),老的 line 号永远稳定
//   - 终端尺寸不变时,wrap 结果稳定;变了再用 WindowSizeMsg 清掉选区即可
type cellPos struct {
	col  int
	line int
}

// normalizeSel 把 selAnchor / selEnd 规约成块选择的左上、右下角。
// 块选择 = 矩形区域:col ∈ [minCol, maxCol], line ∈ [minLine, maxLine]。
func normalizeSel(a, b cellPos) (tl, br cellPos) {
	tl.col = min(a.col, b.col)
	br.col = max(a.col, b.col)
	tl.line = min(a.line, b.line)
	br.line = max(a.line, b.line)
	return
}

const (
	ansiReverseOn  = "\x1b[7m"
	ansiReverseOff = "\x1b[27m"
)

// applySelectionHighlight 在已 wrap 的 chat 内容里给选区矩形加反色。
// 短行先用空格 pad 到 width,这样矩形选区在没文字的位置也能可视化(显示反色空格)。
// width 必须等于 viewport.Width,否则坐标对不上。
func applySelectionHighlight(wrapped string, a, b cellPos, width int) string {
	tl, br := normalizeSel(a, b)
	if width <= 0 {
		return wrapped
	}

	lines := strings.Split(wrapped, "\n")
	left := tl.col
	right := br.col + 1
	if right > width {
		right = width
	}
	if left >= right {
		return wrapped
	}

	for i := tl.line; i <= br.line && i < len(lines); i++ {
		if i < 0 {
			continue
		}
		line := lines[i]
		// pad 短行,确保矩形高亮在空白处也可见
		if cur := ansi.StringWidth(line); cur < width {
			line = line + strings.Repeat(" ", width-cur)
		}
		pre := ansi.Cut(line, 0, left)
		mid := ansi.Cut(line, left, right)
		post := ansi.Cut(line, right, width)
		lines[i] = pre + ansiReverseOn + mid + ansiReverseOff + post
	}
	return strings.Join(lines, "\n")
}

// extractSelectionText 从 wrapped 内容里抠出选区矩形对应的纯文本,去除 ANSI 转义。
// 多行用 \n 连接。短行被裁掉超出实际内容的部分,不会引入末尾空格。
func extractSelectionText(wrapped string, a, b cellPos, width int) string {
	tl, br := normalizeSel(a, b)
	if width <= 0 {
		return ""
	}
	lines := strings.Split(wrapped, "\n")
	left := tl.col
	right := br.col + 1
	if right > width {
		right = width
	}
	if left >= right {
		return ""
	}

	var out []string
	for i := tl.line; i <= br.line && i < len(lines); i++ {
		if i < 0 {
			out = append(out, "")
			continue
		}
		seg := ansi.Cut(lines[i], left, right)
		seg = ansi.Strip(seg)
		// 块选择不需要 padding 出现在剪贴板里,trim 右侧空格
		seg = strings.TrimRight(seg, " ")
		out = append(out, seg)
	}
	return strings.Join(out, "\n")
}
