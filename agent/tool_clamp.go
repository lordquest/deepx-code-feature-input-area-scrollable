package agent

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// 工具返回值在写入会话历史前的两道闸,根治 issue #135:
// 单条超大工具结果(real-browser-mcp 的 base64 截图、大文件读取、海量 grep/Explore 结果)
// 会原样进历史且只增不减,累积后超模型上下文窗口(如 1M tokens),导致 HTTP 400 且会话不可恢复。
// 严重时单条消息就 ~1.3M 字符,序列化都坏掉(messages[N]: missing field `content`)。
// 这里在唯一执行入口 executeTool 出口统一收口:剥 base64 二进制 + 总字节硬上限,任何单条结果都不可能独占窗口。
const (
	// maxToolOutputBytes:单条工具结果写入历史的硬上限。超出按 UTF-8 边界截断并附说明。
	// 96KB(≈ 数万 token):够装正常的大文件读取 / grep 结果,又远小于上下文窗口。
	maxToolOutputBytes = 96 * 1024
	// minBase64RunBytes:连续 base64 字符达到这个长度即判定为二进制 blob(截图 / 附件等),
	// 整段替换为占位符。正常文本 / 代码不会出现这么长且不含空白的连续串。
	minBase64RunBytes = 4096
)

// clampToolOutput 把工具结果压到可安全入历史的大小:
// 先剥掉 base64 二进制 blob(替换为占位符),再对剩余文本做总字节上限截断。
// name 仅用于占位 / 截断说明,方便模型理解发生了什么、如何缩小范围重试。
func clampToolOutput(name, out string) string {
	out = stripBase64Blobs(out)
	if len(out) <= maxToolOutputBytes {
		return out
	}
	b := []byte(out)[:maxToolOutputBytes]
	// 回退到合法 UTF-8 边界,避免截出半个多字节字符(gob 持久化没事,但发给 API 会乱码 / 被拒)。
	for len(b) > 0 && !utf8.Valid(b) {
		b = b[:len(b)-1]
	}
	return string(b) + fmt.Sprintf(
		"\n\n[…%s 返回 %d 字节,已截断至 %d 字节,防止撑爆上下文(issue #135)。"+
			"请缩小范围重试:读文件用 offset/limit 分页、grep 收窄匹配、命令只取必要输出。]",
		name, len(out), len(b))
}

// stripBase64Blobs 把每一段足够长的连续 base64 字符串替换为简短占位符。
// 用途:real-browser-mcp 的 browser_screenshot 等工具会把截图编成单行 base64 直接塞进文本结果,
// 对非视觉模型纯属上下文垃圾。按字节扫描(base64 字符全是 ASCII,非 ASCII 字节天然断开,保证不破坏 UTF-8 文本)。
func stripBase64Blobs(s string) string {
	if len(s) < minBase64RunBytes {
		return s
	}
	var b strings.Builder
	i := 0
	for i < len(s) {
		if isBase64Byte(s[i]) {
			j := i
			for j < len(s) && isBase64Byte(s[j]) {
				j++
			}
			if j-i >= minBase64RunBytes {
				fmt.Fprintf(&b, "[…%d 字节 base64 二进制数据已省略(截图 / 附件不入上下文,issue #135)]", j-i)
				i = j
				continue
			}
			b.WriteString(s[i:j])
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// isBase64Byte 判断是否为 base64 字母表字符(含标准 +/ 和 URL-safe -_ 以及填充 =)。
func isBase64Byte(c byte) bool {
	return c >= 'A' && c <= 'Z' ||
		c >= 'a' && c <= 'z' ||
		c >= '0' && c <= '9' ||
		c == '+' || c == '/' || c == '=' || c == '-' || c == '_'
}
