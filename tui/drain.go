package tui

import tea "charm.land/bubbletea/v2"

// drainAndDiscard 起一个常驻 goroutine 把 ch 里的消息读完丢掉,直到 ch 被 sender 关闭。
// 用于 Ctrl+C 中断流式时:UI 已经"抛弃"这条流,但 agent goroutine 还在 send,
// 不消费会让它 block 在 channel 上泄漏(buffer 128 满后)。
// 调用方调完应当把 streamCh 字段置 nil,避免再走 ListenToStream。
func drainAndDiscard(ch <-chan tea.Msg) {
	go func() {
		for range ch {
		}
	}()
}
