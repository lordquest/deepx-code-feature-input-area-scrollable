package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestInputClickCancelsSelectAll 钉死 #188 的修复:输入区"拖拽全选"高亮后,
// 在输入区里单击一下(无拖动)即取消高亮 —— 此前只能靠键盘(方向键/Esc…)解除,
// 鼠标用户最自然的"点一下取消"没接上。
func TestInputClickCancelsSelectAll(t *testing.T) {
	m := initModel()
	// 建立视口/输入框尺寸,使鼠标命中判定与 refreshViewport 正常工作。
	wm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = wm.(model)
	m.input.SetValue("hello world example text") // 非空才会触发全选

	leftW, vpH := m.layout()
	if vpH >= m.height {
		t.Fatalf("前置失败:vpH(%d) 不小于 height(%d),无输入区可点", vpH, m.height)
	}
	px, py := 2, vpH // 输入区内一点:Y ∈ [vpH, height),X ∈ [0, leftW)
	if px >= leftW {
		t.Fatalf("前置失败:px(%d) 不在内容列 [0,%d)", px, leftW)
	}
	dragX := px + inputDragThreshold + 1 // 横移超阈值 → 判定为真拖动
	if dragX >= leftW {
		t.Fatalf("前置失败:dragX(%d) 超出内容列 [0,%d)", dragX, leftW)
	}

	// 1) 按下 → 真拖动 → 松手:应进入全选态,且松手保留高亮(拖拽复制全文)。
	mm, _ := m.Update(tea.MouseClickMsg{X: px, Y: py, Button: tea.MouseLeft})
	m = mm.(model)
	mm, _ = m.Update(tea.MouseMotionMsg{X: dragX, Y: py, Button: tea.MouseLeft})
	m = mm.(model)
	if !m.inputAllSelected {
		t.Fatalf("真拖动后应进入全选态")
	}
	mm, _ = m.Update(tea.MouseReleaseMsg{X: dragX, Y: py, Button: tea.MouseLeft})
	m = mm.(model)
	if !m.inputAllSelected {
		t.Fatalf("拖拽松手应保留全选高亮(供再次操作 / 复制全文)")
	}

	// 2) 全选态下在输入区单击一下(按下即取消,松手无拖动)→ 高亮清除。
	mm, _ = m.Update(tea.MouseClickMsg{X: px, Y: py, Button: tea.MouseLeft})
	m = mm.(model)
	if m.inputAllSelected {
		t.Fatalf("全选态下在输入区按下应立即取消高亮(#188 点一下取消)")
	}
	mm, _ = m.Update(tea.MouseReleaseMsg{X: px, Y: py, Button: tea.MouseLeft})
	m = mm.(model)
	if m.inputAllSelected {
		t.Fatalf("单击松手后不应再是全选态")
	}
}
