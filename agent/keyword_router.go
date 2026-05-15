package agent

import "strings"

// RouteByKeyword 是入口路由的确定性版本 — 纯本地、零延迟,替代之前的 LLM classifier。
//
// 决策顺序:
//  1. 用户消息小写化后,任意命中 complexKeywords 中的关键词 → "pro"
//  2. 否则,按消息长度(rune 数)兜底:
//     - < 100 → "flash"  (短消息默认走快模型)
//     - > 500 → "pro"    (长消息一般有深度)
//     -        → "flash" (中间长度默认 flash 省钱)
//
// 关键词覆盖英文 + 简繁中文 + 日韩,这样不同语言用户的"复杂任务直觉"都能被路由捕获。
func RouteByKeyword(userMsg string) string {
	lower := strings.ToLower(userMsg)
	for _, kw := range complexKeywords {
		// 关键词本身已经是小写或 CJK(CJK 无大小写概念),
		// 用 Contains 即可正确匹配;不做边界检测以保持简单。
		if strings.Contains(lower, kw) {
			return "pro"
		}
	}

	runeCount := len([]rune(userMsg))
	if runeCount < 100 {
		return "flash"
	}
	if runeCount > 500 {
		return "pro"
	}
	return "flash"
}

// complexKeywords 触发 pro 路由的关键词列表。
// 维护原则:
//  1. 整体偏中性,避免把日常查询误判(比如别加"看一下")
//  2. 多语言覆盖 — 国际用户可能用本地语言描述同样的复杂任务
//  3. 优先包含"动词+范围"组合(refactor / 重构 / 分析整个),不是单一动词
//
// 顺序按地区分组,便于维护。
var complexKeywords = []string{
	// === English ===
	"refactor",
	"architecture",
	"design",
	"debug",
	"security",
	"review",
	"audit",
	"migrate",
	"optimize",
	"rewrite",
	"implement",
	"analyze",
	"investigate",
	"root cause",
	"multi-file",
	"end-to-end",
	"cross-file",

	// === 简体中文 ===
	"重构",
	"架构",
	"设计",
	"调试",
	"安全",
	"审查",
	"审计",
	"迁移",
	"优化",
	"重写",
	"实现",
	"分析",
	"规划",
	"排查",
	"根因",
	"整个",
	"跨多个",
	"跨文件",
	"方案",
	"调研",

	// === 繁体中文 ===
	"重構",
	"架構",
	"設計",
	"調試",
	"審查",
	"審計",
	"遷移",
	"優化",
	"重寫",
	"實現",
	"規劃",
	"排查",
	"整個",
	"跨多個",
	"跨檔案",
	"方案",
	"調研",

	// === 日本語 ===
	"リファクタリング",
	"リファクタ",
	"アーキテクチャ",
	"設計",
	"デバッグ",
	"セキュリティ",
	"レビュー",
	"監査",
	"マイグレーション",
	"移行",
	"最適化",
	"書き直し",
	"リライト",
	"実装",
	"解析",
	"計画",
	"調査",
	"根本原因",
	"複数ファイル",
	"エンドツーエンド",

	// === 한국어 ===
	"리팩토링",
	"리팩터링",
	"아키텍처",
	"구조",
	"설계",
	"디자인",
	"디버깅",
	"디버그",
	"보안",
	"리뷰",
	"검토",
	"감사",
	"마이그레이션",
	"이전",
	"최적화",
	"재작성",
	"구현",
	"분석",
	"계획",
	"조사",
	"근본 원인",
}
