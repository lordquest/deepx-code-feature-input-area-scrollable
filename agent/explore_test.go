package agent

import (
	"strings"
	"testing"
)

// Explore 子 agent 的工具表必须:只含只读搜索工具、不含任何写工具、不含 Explore 自身(防递归)。
func TestExploreToolSpecs_ReadOnlyNoRecursion(t *testing.T) {
	specs := buildExploreToolSpecs()
	if len(specs) == 0 {
		t.Fatal("探索工具表不应为空")
	}

	got := map[string]bool{}
	for _, s := range specs {
		got[s.Function.Name] = true
	}

	// 必须拿到的工具:本地搜索/读取 + 外部仓库/网页(Search/Fetch)
	for _, want := range []string{"Grep", "Glob", "Read", "List", "Tree", "CodeGraph", "Search", "Fetch"} {
		if !got[want] {
			t.Errorf("探索工具表应包含只读工具 %q", want)
		}
	}

	// 绝不能出现的工具:写/副作用、编排/状态、以及 Explore 自身(套娃)
	for _, bad := range []string{"Write", "Update", "Bash", "Explore", "CreatePlan", "Todo", "UpdatePlanStatus", "SwitchModel", "AskUser", "Remember", "Memory"} {
		if got[bad] {
			t.Errorf("探索工具表不应包含 %q(只读/防递归/非编排)", bad)
		}
	}
}

func TestParseExploreArgs(t *testing.T) {
	if task, th := parseExploreArgs(`{"task":"找鉴权中间件","thoroughness":"thorough"}`); task != "找鉴权中间件" || th != "thorough" {
		t.Errorf("parseExploreArgs = (%q,%q), 期望 (\"找鉴权中间件\",\"thorough\")", task, th)
	}
	if task, th := parseExploreArgs(`{"task":"x"}`); task != "x" || th != "" {
		t.Errorf("缺 thoroughness 时应为空串,得到 (%q,%q)", task, th)
	}
	if task, _ := parseExploreArgs(`{}`); task != "" {
		t.Errorf("缺 task 时应返回空串,得到 %q", task)
	}
	if task, _ := parseExploreArgs(`not json`); task != "" {
		t.Errorf("非法 JSON 时应返回空串,得到 %q", task)
	}
}

func TestClampExploreSummary(t *testing.T) {
	// 短结论原样返回。
	short := "在 agent/llm.go:1180 定义 executeTool"
	if got := clampExploreSummary(short); got != short {
		t.Errorf("短结论不应被改动,得到 %q", got)
	}
	// 超长结论被截到上限内并带提示。
	long := strings.Repeat("行", exploreSummaryMaxRunes+500)
	got := clampExploreSummary(long)
	if len([]rune(got)) <= exploreSummaryMaxRunes {
		t.Errorf("截断后应保留上限内容 + 提示,rune 数 = %d", len([]rune(got)))
	}
	if !strings.Contains(got, "已截断") {
		t.Error("截断结论应包含提示文字")
	}
	// 恰好等于上限不截断。
	exact := strings.Repeat("x", exploreSummaryMaxRunes)
	if got := clampExploreSummary(exact); got != exact {
		t.Error("恰好等于上限不应截断")
	}
}

func TestThoroughnessGuidance(t *testing.T) {
	// 空 / 未知值都应退到 medium 档(不 panic、有合理默认)。
	for _, level := range []string{"quick", "medium", "thorough", "", "bogus"} {
		if thoroughnessGuidance(level) == "" {
			t.Errorf("thoroughnessGuidance(%q) 不应为空", level)
		}
	}
}
