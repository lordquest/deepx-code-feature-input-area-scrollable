package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// === deepx-code 文字 banner(右栏顶部)===
//
// 5 行布局:
//   - 顶部一条横线
//   - 3 行 5×3 block art "deepx",每个字母青→蓝渐变;底行右下角放 "code" 标签
//   - 底部一条横线
//
// 配色青→蓝、上下用横线分割,"code" 落在艺术字右下角,自有辨识度。
const (
	bannerArtRows  = 3
	bannerArtWidth = 3*5 + 4 // 5 字母 × 3 列 + 4 字母间空格 = 19
	bannerIndent   = 2       // art 左缩进
	bannerSuffix   = " -code"
	bannerMinWidth = bannerIndent + bannerArtWidth
)

// deepxLetters 5 个字母的 3×3 像素艺术。
var deepxLetters = [5][bannerArtRows]string{
	{"█▀▄", "█ █", "▀▀ "}, // d
	{"█▀▀", "█▀▀", "▀▀▀"}, // e
	{"█▀▀", "█▀▀", "▀▀▀"}, // e
	{"█▀▄", "█▀▀", "█  "}, // p
	{"█ █", "▀▄▀", "▀ ▀"}, // x
}

// deepxLetterColors 每个字母一色,组成青→蓝渐变。ANSI 256 调色板等距取色,跨终端稳。
var deepxLetterColors = [5]color.Color{
	lipgloss.Color("51"), // 亮青
	lipgloss.Color("45"), // 青
	lipgloss.Color("39"), // 青蓝
	lipgloss.Color("33"), // 蓝
	lipgloss.Color("27"), // 索蓝
}

// bannerSuffixColor "-code" 后缀色(浅灰,作字样副件);bannerDecoColor 分割线色(亦被
// scrollbar 轨道 / reasoning 角色名复用,留作通用蓝色强调)。
var (
	bannerSuffixColor color.Color = lipgloss.Color("250")
	bannerDecoColor   color.Color = lipgloss.Color("67") // 钢蓝
)

// renderBanner 返回 5 行 × width 列的 banner。width < bannerMinWidth 时返回空。
func renderBanner(width int) string {
	if width < bannerMinWidth {
		return ""
	}

	// 上下两条纯横线分割。
	deco := lipgloss.NewStyle().Foreground(bannerDecoColor).Render(strings.Repeat("─", width))
	label := strings.TrimLeft(strings.TrimSpace(bannerSuffix), "-") // "code"(去掉前导 -)

	pad := strings.Repeat(" ", bannerIndent)
	rows := make([]string, 0, 5)
	rows = append(rows, deco)
	for r := range bannerArtRows {
		var sb strings.Builder
		sb.WriteString(pad)
		for i, letter := range deepxLetters {
			if i > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(deepxLetterColors[i]).Render(letter[r]))
		}
		rowStr := sb.String()
		// 底行(基线)紧贴艺术字右侧放 "code" 标签(留 1 空格),靠近 deepx;放得下才放。
		if r == bannerArtRows-1 && ansi.StringWidth(rowStr)+1+ansi.StringWidth(label) <= width {
			rowStr += " " + lipgloss.NewStyle().Foreground(bannerSuffixColor).Render(label)
		}
		rows = append(rows, padBannerRow(rowStr, width))
	}
	rows = append(rows, deco)
	return strings.Join(rows, "\n")
}

// padBannerRow 把行右补空格到 width 列(按显示宽度算,忽略 ANSI 转义)。
func padBannerRow(s string, width int) string {
	if cur := ansi.StringWidth(s); cur < width {
		return s + strings.Repeat(" ", width-cur)
	}
	return s
}
