package tools

import (
	"fmt"
	"strings"

	"deepx/skill"
)

// currentSkillLoader 由 tui 启动时通过 SetSkillLoader 注入。
// 跟 Memory 的依赖注入模式一致 —— 工具 Executor 签名固定 (map[string]any → ToolResult),
// 包级变量是最小侵入。运行时单 loader,不存在竞态。
var currentSkillLoader *skill.Loader

// SetSkillLoader 注入 skill 加载器(tui.initialModel 启动时调用一次)。
func SetSkillLoader(l *skill.Loader) {
	currentSkillLoader = l
}

// LoadSkill 按名加载一个 skill 的完整正文塞给 LLM。
//
// 设计:
//   - frontmatter 里的 name + description 已经在 system prompt 里挂过摘要列表,LLM
//     看到合适的就调本工具拿正文
//   - 项目级 (session/<sid>/skills/) 优先于全局 (~/.deepx/skills/),同名时覆盖
//   - 不做本会话缓存 —— LLM history 自带 tool result,重复调用本身少见
func LoadSkill(args map[string]any) ToolResult {
	if currentSkillLoader == nil {
		return ToolResult{Output: "skill 系统未启用 (loader 未初始化)", Success: false}
	}
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return ToolResult{Output: "name 不能为空,请从 system prompt 的 Available Skills 列表里挑一个", Success: false}
	}
	s, err := currentSkillLoader.Load(name)
	if err != nil {
		return ToolResult{Output: err.Error(), Success: false}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Skill: %s (%s)\n", s.Name, s.Scope)
	if s.Description != "" {
		fmt.Fprintf(&sb, "_%s_\n\n", s.Description)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimSpace(s.Body))
	return ToolResult{Output: sb.String(), Success: true}
}
