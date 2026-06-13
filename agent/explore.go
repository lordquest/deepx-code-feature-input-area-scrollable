package agent

import (
	"context"
	"deepx/tools"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// exploreToolNames 是「探索」子 agent 允许的工具(只读)。两类目标:
//   - 本地代码库:List / Tree / Glob / Grep / Read / CodeGraph
//   - 外部仓库 / 网页:Search(搜索)/ Fetch(抓网页,如 GitHub 仓库主页 / README / raw 源码)
//
// 显式 allowlist:不能简单用 ReadOnly 字段过滤——CreatePlan / UpdatePlanStatus / Memory 等
// 也标了 ReadOnly:true,但属于编排/状态工具,不该给探索子 agent。刻意不含:
//   - Explore 自身 → 防止递归套娃(探索里再派探索)
//   - 写工具 / Bash → 保持纯只读,且只读不靠 prompt 兜底(对比官方 Explore 保留 Bash 靠 prompt 约束,
//     deepx 走 #108 的教训:硬不给比"信任模型只读用 Bash"更稳)。外部抓取用 Fetch 足够,不需要 gh/git。
var exploreToolNames = map[string]bool{
	"List":      true,
	"Tree":      true,
	"Glob":      true,
	"Grep":      true,
	"Read":      true,
	"CodeGraph": true,
	"Search":    true,
	"Fetch":     true,
}

// buildExploreToolSpecs 探索子 agent 的工具表:只取 exploreToolNames 里的只读搜索/读取工具。
// 所有 Explore 调用共用这同一份工具表 → 它们彼此之间前缀可复用(虽不命中主会话缓存)。
func buildExploreToolSpecs() []tools.OpenAIToolSpec {
	var out []tools.OpenAIToolSpec
	for _, t := range tools.Tools {
		if exploreToolNames[t.Name] {
			out = append(out, t.ToOpenAISpec())
		}
	}
	return out
}

// exploreInput 是一次 Explore 调用的依赖。由主 agent 工具循环拦截 Explore 时构造。
type exploreInput struct {
	Entry        ModelEntry // 跑探索用的模型连接参数(默认 flash,便宜)
	Task         string     // 上层要探索的问题 / 要带回的结论
	Thoroughness string     // 详尽程度:quick / medium / thorough(空 = medium)
	Workspace    string
}

// exploreSummaryMaxRunes 是探索结论回灌主上下文前的软上限(rune 数,~8k 字符 ≈ 2~3k token)。
// 隔离的价值在于"只回精炼结论":prompt 已约束别贴原文,这里再加一道硬兜底(同 #108 思路:
// 别只信 prompt)——模型万一吐一大段,也不让它原样进主 convo、之后每轮重发。
const exploreSummaryMaxRunes = 8000

// clampExploreSummary 把超长结论按 rune 截断,并附一行提示让上层知道结论被截、可缩小探索范围重试。
func clampExploreSummary(s string) string {
	r := []rune(s)
	if len(r) <= exploreSummaryMaxRunes {
		return s
	}
	return string(r[:exploreSummaryMaxRunes]) +
		"\n\n…(探索结论过长已截断。如需更细的内容,请缩小探索范围、用更聚焦的 task 重新 Explore,或直接用 Read/Grep 定位。)"
}

// thoroughnessGuidance 把 quick/medium/thorough 映射成给探索子 agent 的深度指引。
func thoroughnessGuidance(level string) string {
	switch level {
	case "quick":
		return "快速:做基本搜索,够回答就停,优先速度。"
	case "thorough":
		return "彻底:跨多个位置、多种命名/措辞全面排查,交叉验证,尽量不漏。"
	default: // medium / 空
		return "中等:适度展开,覆盖主要位置后即可下结论。"
	}
}

// runExplore 执行一个只读「探索」子 agent:在独立上下文里搜索本地代码库或外部仓库/网页,只返回结论。
//
// 与 runSubAgent 同构(独立 history、静默 channel、ctx 预算熔断、无进展断路器),区别:
//   - 工具表只有 exploreToolNames 里的只读搜索/抓取工具(不含 Explore 自身 → 不递归)
//   - 以 AgentMode_Plan 调 executeTool,纵深防御:即便工具表被改坏混进写工具,plan 硬拦也挡住
//   - 专用精简 system prompt:强调"只读、查透、尽量并行、只回结论、别贴原文"
func runExplore(ctx context.Context, in exploreInput) (string, error) {
	sys := fmt.Sprintf(`你是 DeepX 的「探索」子 agent:只读、一次性,职责是搜索/定位并回答上层 agent 交给你的探索任务。

你能探索两类目标:
1. 本地代码库 —— 用 List / Tree / Glob / Grep / Read / CodeGraph 在当前工作目录里搜索、定位、读代码。
2. 外部仓库 / 网页 —— 用 Search 搜索、Fetch 抓取网页(GitHub 仓库主页 / README / raw 源码文件 / 文档等),搞清一个外部项目或链接是干嘛的、怎么用、关键实现在哪。

规则:
- 只读:你只有上述搜索 / 读取 / 抓取工具,不能改文件、不能执行命令、不能写任何东西。
- 查透再回:多角度交叉搜索(本地多用 Grep/Glob/CodeGraph;外部多用 Search + Fetch 抓关键页面),必要时读关键片段确认,别只看一两处就下结论。
- 尽量并行:能一次发起多个搜索 / 读取 / 抓取调用就并行发,别一个一个串行等——你要快。
- 只回「结论」,不要回「原文」:最终只输出精炼答案——本地给相关位置(file:line)+ 关键发现;外部给项目用途 / 关键事实 + 来源链接。绝对不要把大段源码或网页原文贴回来;上层要的是结论,不是原文。

详尽程度:%s
当前工作目录:%s`, thoroughnessGuidance(in.Thoroughness), in.Workspace)

	convo := []ChatMessage{
		{Role: "system", Content: sys},
		{Role: "user", Content: in.Task},
	}

	toolSpecs := buildExploreToolSpecs()

	// 静默 channel:streamOnce 的 TokenMsg 不进 UI,内部 drain 掉(探索中间过程对用户隐藏)。
	silent := make(chan tea.Msg, 64)
	drained := make(chan struct{})
	go func() {
		for range silent {
		}
		close(drained)
	}()

	ctxWin := in.Entry.ContextWindow
	if ctxWin <= 0 {
		ctxWin = 65536
	}
	ctxBudget := ctxWin * subAgentCtxBudgetPct / 100

	var lastFile string
	noProgressRounds := 0

	for {
		if ctx.Err() != nil {
			close(silent)
			<-drained
			return "", ctx.Err()
		}
		if est := estimateConvoTokens(convo); est >= ctxBudget {
			close(silent)
			<-drained
			return "", fmt.Errorf("探索子 agent 上下文超预算(~%d/%d tokens),中止", est, ctxWin)
		}
		content, reasoning, toolCalls, _, _, err := streamOnce(
			ctx,
			in.Entry.APIKey, in.Entry.BaseURL, in.Entry.Model,
			convo, clampMaxTokens(in.Entry.MaxTokens, in.Entry.ContextWindow, convo), toolSpecs,
			in.Entry.ReasoningEffort, in.Entry.Thinking,
			silent,
		)
		if err != nil {
			close(silent)
			<-drained
			return "", err
		}

		convo = append(convo, ChatMessage{
			Role:             "assistant",
			Content:          content,
			ReasoningContent: reasoning,
			ToolCalls:        toolCalls,
		})

		if len(toolCalls) == 0 {
			close(silent)
			<-drained
			summary := strings.TrimSpace(content)
			if summary == "" {
				summary = "(探索子 agent 未给出明确结论)"
			}
			return clampExploreSummary(summary), nil
		}

		roundProgress := false
		for _, tc := range toolCalls {
			// 以 plan 模式执行:工具表已只含只读工具,这里再加一道 plan 硬拦兜底(issue #108 同款)。
			result := executeTool(tc, AgentMode_Plan, &lastFile)
			if result.Success {
				roundProgress = true
			}
			convo = append(convo, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result.Output,
			})
		}

		if roundProgress {
			noProgressRounds = 0
		} else {
			noProgressRounds++
			if noProgressRounds >= maxNoProgressRounds {
				close(silent)
				<-drained
				return "", fmt.Errorf("探索子 agent 连续 %d 轮工具调用均未成功,疑似卡死,中止", maxNoProgressRounds)
			}
		}
	}
}
