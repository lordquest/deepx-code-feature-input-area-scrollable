package tui

import (
	"deepx/agent"
	"strings"
	"testing"
)

// TestPlanStateApplyPlanLevel 验证 plan 状态更新能落到对应项。
func TestPlanStateApplyPlanLevel(t *testing.T) {
	p := &planState{items: []agent.PlanItem{
		{ID: "plan1", Status: agent.PlanStatusPending},
	}}
	p.apply(agent.TaskStatusMsg{ID: "plan1", Status: agent.PlanStatusDone, Summary: "ok"})
	if got := p.items[0].Status; got != agent.PlanStatusDone {
		t.Errorf("plan1 want done, got %s", got)
	}
	if got := p.items[0].Summary; got != "ok" {
		t.Errorf("plan1 summary want ok, got %q", got)
	}
}

// TestRenderPlanForChatReflectsStatus 验证 renderPlanForChat 输出会随 status 变化。
func TestRenderPlanForChatReflectsStatus(t *testing.T) {
	p := &planState{items: []agent.PlanItem{
		{ID: "p1", Title: "read A", Status: agent.PlanStatusDone},
		{ID: "p2", Title: "read B", Status: agent.PlanStatusRunning},
	}}
	out := renderPlanForChat(p)
	if !strings.Contains(out, "[✓]") {
		t.Errorf("expected [✓] for done plan in output:\n%s", out)
	}
	if !strings.Contains(out, "[⏵]") {
		t.Errorf("expected [⏵] for running plan in output:\n%s", out)
	}
}
