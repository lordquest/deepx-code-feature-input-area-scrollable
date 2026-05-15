package tui

import "charm.land/lipgloss/v2"

const (
	rightPanelWidth = 32
	maxTokens       = 384 * 1024
)

var (
	frameColor     = lipgloss.Color("62")
	dimColor       = lipgloss.Color("240")
	subtleColor    = lipgloss.Color("8")
	accentColor    = lipgloss.Color("12")
	highlightColor = lipgloss.Color("13")

	outerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(frameColor)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().Foreground(frameColor)
	thinSepStyle = lipgloss.NewStyle().Foreground(dimColor)

	// 三种角色统一用 "**emoji name**: " 形式写进 chatContent。
	// glamour 渲染时把 ** ** 转成终端粗体,跨终端比内嵌 ANSI 稳。
	deepxPrefix  = "**🐋 deepx**: "
	userPrefix   = "**👤 我**: "
	systemPrefix = "**⚙ System**: "
)

// thinSepStyle 预留未用。
var _ = thinSepStyle
