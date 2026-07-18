package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// mouseInputAt 返回输入框内一个屏幕坐标:X = gutter + col,Y = textarea 文本首行。
func (m model) mouseInputAt(col int) (x, y int) {
	return inputGutterWidth + col, m.inputTextTopY()
}

// TestInputDragSelectsFragment 验证 #188 的部分选择:输入区拖拽选中一段 → 复制该片段;
// 拖拽松手后保留高亮;随后单击一下(无拖动)→ 取消高亮(点一下取消)。
func TestInputDragSelectsFragment(t *testing.T) {
	m := initModel()
	wm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = wm.(model)
	m.input.SetValue("hello world")

	ax, ay := m.mouseInputAt(0) // 锚点:第 0 列
	ex, ey := m.mouseInputAt(5) // 拖到第 5 列("hello" 之后)

	// 按下 → 拖动到不同列 → 应进入选区态
	mm, _ := m.Update(tea.MouseClickMsg{X: ax, Y: ay, Button: tea.MouseLeft})
	m = mm.(model)
	if m.inputSelecting {
		t.Fatalf("刚按下(未拖动)不应有选区")
	}
	mm, _ = m.Update(tea.MouseMotionMsg{X: ex, Y: ey, Button: tea.MouseLeft})
	m = mm.(model)
	if !m.inputSelecting {
		t.Fatalf("拖到不同列后应进入选区态")
	}
	if got := m.inputSelectionText(); !strings.Contains(got, "hello") || strings.Contains(got, "world") {
		t.Fatalf("选区文本应为片段 \"hello\",实得 %q", got)
	}

	// 松手:保留高亮(供再操作 / 复制已发出)
	mm, _ = m.Update(tea.MouseReleaseMsg{X: ex, Y: ey, Button: tea.MouseLeft})
	m = mm.(model)
	if !m.inputSelecting {
		t.Fatalf("拖拽松手后应保留选区高亮")
	}

	// 单击一下(无拖动)→ 点一下取消
	cx, cy := m.mouseInputAt(2)
	mm, _ = m.Update(tea.MouseClickMsg{X: cx, Y: cy, Button: tea.MouseLeft})
	m = mm.(model)
	if m.inputSelecting {
		t.Fatalf("单击按下应立即取消选区高亮(点一下取消)")
	}
	mm, _ = m.Update(tea.MouseReleaseMsg{X: cx, Y: cy, Button: tea.MouseLeft})
	m = mm.(model)
	if m.inputSelecting {
		t.Fatalf("单击松手后不应再有选区")
	}
}

// TestInputSelectionClearedByKey 验证选区是"复制标记":任何按键都取消它、再走正常处理
// (不删 value)。
func TestInputSelectionClearedByKey(t *testing.T) {
	m := initModel()
	wm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = wm.(model)
	m.input.SetValue("abcdef")

	ax, ay := m.mouseInputAt(0)
	ex, ey := m.mouseInputAt(3)
	mm, _ := m.Update(tea.MouseClickMsg{X: ax, Y: ay, Button: tea.MouseLeft})
	m = mm.(model)
	mm, _ = m.Update(tea.MouseMotionMsg{X: ex, Y: ey, Button: tea.MouseLeft})
	m = mm.(model)
	if !m.inputSelecting {
		t.Fatalf("前置失败:未进入选区态")
	}

	// 按方向键:取消选区,但 value 不动(不再是旧的"全选态清空"语义)
	mm, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	m = mm.(model)
	if m.inputSelecting {
		t.Fatalf("按键后应取消选区标记")
	}
	if m.input.Value() != "abcdef" {
		t.Fatalf("取消选区不应改动 value,实得 %q", m.input.Value())
	}
}
