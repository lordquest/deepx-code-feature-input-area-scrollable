// Package session 把当前 workspace 的对话内容持久化到 ~/.deepx/sessions/{sessionID}/。
//
// 设计要点:
//   - sessionID = sha1(abs(workspace))[:16],workspace 切换自然换 session。
//   - 每天一个 jsonl 文件,append-only,一行一条 Entry。
//   - 只存 user/assistant content 主对话(tool_call / tool_result 不入文件,
//     避免 jsonl 巨大化;LLM 当下能从 message 里看到工具序列,
//     重加历史时看不到也无所谓 —— 它只需要语义上的"上次聊了什么")。
//   - 读时按文件名(日期)倒序聚合,N 个 user→assistant 对截止。
//   - Search 用纯 Go strings.Contains 大小写不敏感扫描,不走外部 grep,跨平台稳。
package session

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry 是 jsonl 里每行的结构。简洁版只 ts/role/content。
type Entry struct {
	Ts      time.Time `json:"ts"`
	Role    string    `json:"role"`
	Content string    `json:"content"`
}

// SearchHit Memory 工具的命中条目。
type SearchHit struct {
	Date  string // 文件名日期 (YYYY-MM-DD)
	Entry Entry
}

// Manager 管单个 workspace 的 session。线程不安全,TUI 单 goroutine 调用即可。
type Manager struct {
	workspace string
	sessionID string
	rootDir   string // ~/.deepx/sessions/{sessionID}
}

// metaFile 是 ~/.deepx/sessions/{sid}/meta.json 的结构。
type metaFile struct {
	Workspace  string    `json:"workspace"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// New 给指定 workspace 创建/打开 session。会自动建目录,刷新 meta.json 的 last_seen_at。
func New(workspace string) (*Manager, error) {
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}
	h := sha1.Sum([]byte(abs))
	sid := hex.EncodeToString(h[:])[:16]

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("user home: %w", err)
	}
	root := filepath.Join(home, ".deepx", "sessions", sid)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir session: %w", err)
	}

	m := &Manager{workspace: abs, sessionID: sid, rootDir: root}
	m.touchMeta()
	return m, nil
}

// SessionID 返回 16 字符 hex,用作目录名与诊断显示。
func (m *Manager) SessionID() string { return m.sessionID }

// RootDir 返回 session 目录绝对路径。
func (m *Manager) RootDir() string { return m.rootDir }

// touchMeta 创建或更新 meta.json。失败静默,不影响主流程。
func (m *Manager) touchMeta() {
	path := filepath.Join(m.rootDir, "meta.json")
	var info metaFile
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &info)
	}
	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}
	info.Workspace = m.workspace
	info.LastSeenAt = time.Now()
	data, _ := json.MarshalIndent(info, "", "  ")
	_ = os.WriteFile(path, data, 0o644)
}

// todayPath 当天文件路径。按本地时区命名,方便人类浏览。
func (m *Manager) todayPath() string {
	return filepath.Join(m.rootDir, time.Now().Format("2006-01-02")+".jsonl")
}

// Append 写一条记录到今天的 jsonl。
// 只接受 role = "user" / "assistant",其他 role(system/tool)静默丢弃 —— 主对话才入会话文件。
// 空 content 也跳过(流式中途的占位等)。
func (m *Manager) Append(role, content string) error {
	if role != "user" && role != "assistant" {
		return nil
	}
	if strings.TrimSpace(content) == "" {
		return nil
	}
	f, err := os.OpenFile(m.todayPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(Entry{Ts: time.Now(), Role: role, Content: content})
}

// LoadRecentTurns 从最新日期文件倒着读,凑足 n 个 user→assistant 对停。
// 返回按时间正序的 entries,可直接喂给 LLM history。
//
// "1 个 turn" 的定义:一条 user message 加它后续的 assistant 回复(可能多条 assistant
// 段或被工具调用插开,但本会话文件只记 content,所以约等于 1 个 user + 1 个 assistant)。
// 实现简化:反向数 user 出现次数到 n,从该 user 起截断。
func (m *Manager) LoadRecentTurns(n int) []Entry {
	if n <= 0 {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(m.rootDir, "*.jsonl"))
	sort.Strings(files) // 升序;旧 → 新
	var all []Entry
	for _, f := range files {
		all = append(all, readJSONL(f)...)
	}
	if len(all) == 0 {
		return nil
	}
	// 反向数 user,凑够 n 个就在 user 之前切
	userCount := 0
	cut := 0
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Role == "user" {
			userCount++
			if userCount == n {
				cut = i
				break
			}
		}
	}
	return all[cut:]
}

// Search 在当前 session 下扫描所有 jsonl,按关键词命中。
// mode = "and"(默认,全部关键词都在) / "or"(任一命中)
// 按日期降序遍历(新的优先),max 上限后立即返回。
func (m *Manager) Search(keywords []string, mode string, max int) []SearchHit {
	if max <= 0 {
		max = 20
	}
	if len(keywords) == 0 {
		return nil
	}
	if mode == "" {
		mode = "and"
	}
	files, _ := filepath.Glob(filepath.Join(m.rootDir, "*.jsonl"))
	sort.Sort(sort.Reverse(sort.StringSlice(files))) // 新 → 旧
	var hits []SearchHit
	for _, fp := range files {
		date := strings.TrimSuffix(filepath.Base(fp), ".jsonl")
		for _, e := range readJSONL(fp) {
			if matchKeywords(e.Content, keywords, mode) {
				hits = append(hits, SearchHit{Date: date, Entry: e})
				if len(hits) >= max {
					return hits
				}
			}
		}
	}
	return hits
}

// readJSONL 读一个 jsonl 文件,容错跳过解析失败的行。
// 单行容量 1MB(防写入超长内容时 scanner 默认 64KB 撑爆)。
func readJSONL(path string) []Entry {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []Entry
	for sc.Scan() {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err == nil {
			out = append(out, e)
		}
	}
	return out
}

// matchKeywords 大小写不敏感地匹配。mode="and" 全部命中, "or" 任一命中。
func matchKeywords(text string, kws []string, mode string) bool {
	lt := strings.ToLower(text)
	if mode == "or" {
		for _, k := range kws {
			if strings.Contains(lt, strings.ToLower(k)) {
				return true
			}
		}
		return false
	}
	for _, k := range kws {
		if !strings.Contains(lt, strings.ToLower(k)) {
			return false
		}
	}
	return true
}
