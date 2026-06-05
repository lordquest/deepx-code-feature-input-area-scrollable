package agent

import "strings"

// WorkingMode 工作模式:圈定本轮该用 / 不该用哪些 skill,把 LLM 引导到该方法论的规划范围内。
//
// 每轮请求时,由 renderWorkingMode 把对应提示追加到"最后一条 user 消息"尾部——
// 参考 OCR/视觉的 renderConvoImages:在请求副本上变换、**不写回 history**,所以每轮都新鲜注入、
// 切换模式立刻生效、也不污染历史。三种模式各自对应一个内置 skill,且明确要求不用另外两个。
type WorkingMode string

const (
	WorkingModeKarpathy    WorkingMode = "karpathy"    // kp:务实工匠,karpathy-guidelines
	WorkingModeOpenSpec    WorkingMode = "openspec"    // spec:规格驱动,openspec
	WorkingModeSuperpowers WorkingMode = "superpowers" // sp:全流程严谨,superpowers
	WorkingModeDefault                 = WorkingModeKarpathy
)

// NormalizeWorkingMode 把别名 / 空值归一到合法模式(默认 kp)。
func NormalizeWorkingMode(s string) WorkingMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "openspec", "spec":
		return WorkingModeOpenSpec
	case "superpowers", "sp":
		return WorkingModeSuperpowers
	default: // karpathy / kp / 空 / 未知
		return WorkingModeKarpathy
	}
}

// workingModePrompt 返回该模式每轮追加到 user 消息尾部的引导。
// 关键:明确"加载并遵循本模式的 skill,且不要加载/遵循另外两个相关 skill",把 LLM 锁在规划内。
func workingModePrompt(m WorkingMode) string {
	switch m {
	case WorkingModeOpenSpec:
		return "[工作模式 openspec] 本轮按「规格驱动」工作:请使用 `openspec`技能 并严格遵循——" +
			"动手写代码前,先写 / 更新改动规格(spec),与用户对齐后再按规格实现。" +
			"**本模式只使用 openspec 这一 skill;不要加载或遵循 karpathy-guidelines、superpowers。**"
	case WorkingModeSuperpowers:
		return "[工作模式 superpowers] 本轮按「全流程严谨」工作:请使用`superpowers`技能 并" +
			"**严格遵循它定义的完整工作流**(入口会按需引导到 brainstorming / 计划 / TDD / 子 agent 执行 / 代码审查 / 收尾 / 调试 / 完成前验证 等子 skill)——别只做其中几步。" +
			"**本模式只使用 superpowers 这套 skill;不要加载或遵循 karpathy-guidelines、openspec。**"
	default: // karpathy(默认)
		return "[工作模式 karpathy] 本轮按 karpathy-guidelines 工作:请使用 `karpathy-guidelines`技能 并严格遵循——" +
			"想清楚再写、最小改动、显式暴露假设、定义可验证的成功标准。" +
			"**本模式只使用 karpathy-guidelines 这一 skill;不要加载或遵循 openspec、superpowers。**"
	}
}

// renderWorkingMode 在请求副本上,把当前工作模式提示追加到最后一条 user 消息尾部。
// 每轮调用,不写回 history(同 renderConvoImages)。返回新切片,不改原 convo。
func renderWorkingMode(convo []ChatMessage, mode WorkingMode) []ChatMessage {
	prompt := workingModePrompt(mode)
	if prompt == "" || len(convo) == 0 {
		return convo
	}
	out := make([]ChatMessage, len(convo))
	copy(out, convo)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Role != "user" {
			continue
		}
		msg := out[i] // 值拷贝,改副本
		if msg.Content != "" {
			msg.Content += "\n\n" + prompt
		} else {
			msg.Content = prompt
		}
		out[i] = msg
		break
	}
	return out
}
