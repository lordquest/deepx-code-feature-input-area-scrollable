# deepx-code

[简体中文](README.md) | **English** | [日本語](README.ja.md) | [한국어](README.ko.md)

> The coding agent built for DeepSeek — local OCR image recognition, automatic context compression, native codegraph support. Fundamentally cuts token consumption.

![deepx screenshot](assets/screenshot.jpg)

## Why deepx

- 🚀 Written in Go — small, fast, cross-platform.
- 🚀 gob binary persistence. `tool_calls`, tool results, and `reasoning_content` are all preserved, so the LLM resumes seamlessly.
- 🚀 Layered compression + merging of old summaries.
- 🚀 Ships with skills and MCP, integrating seamlessly into the existing Claude ecosystem.
- 🚀 Local keyword routing. Zero latency, zero token cost — a hit upgrades straight to pro.
- 🚀 Automatic model switching. Upgrades to the pro model based on problem complexity.
- 🚀 Plan DAG concurrent scheduling. Runs sub-agents in parallel by dependency order, each node picking its own model.
- 🚀 Local OCR (PaddleOCR). Offline recognition, no dependency on a multimodal API.
- 🚀 Code graph (codeGraph). Drastically reduces token waste from read/glob/grep.

## Quick Start

### Install

- macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/itmisx/deepx-code/main/scripts/install.sh | bash
```

- Windows (PowerShell)

```bash
irm https://raw.githubusercontent.com/itmisx/deepx-code/main/scripts/install.ps1 | iex
```

## Usage

### Enter a workspace

```bash
cd <your project directory>
deepx
```

### Configure the DeepSeek API KEY

A configuration dialog pops up on first launch — set your API key there.

### Configure Skills

Place them under `.deepx/skills/` in the current directory.

### Configure MCP

Add an MCP server with the `/mcp-add` command.

## Core Mechanisms

### Model routing (local, zero latency)

When a user message arrives, deepx does local keyword matching + length checks:

```
Message contains "重构/refactor/architecture/调试…" → upgrade to pro directly
Message length < 100 chars                        → flash
Message length > 500 chars                        → pro
```

Covers five languages: Chinese (Simplified/Traditional) / English / Japanese / Korean. **Routing happens instantly and consumes no extra LLM tokens.**

### Session persistence (gob binary)

```
~/.deepx/sessions/<sha1(workspace)[:16]>/
├── meta.json          # workspace metadata
├── state.json         # compression state (summary + total_turns)
├── YYYY-MM-DD.jsonl   # text log (for Memory search)
└── history.gob        # full binary history
```

| Format             | Stored content                                                                       | Purpose                          |
| :----------------- | :----------------------------------------------------------------------------------- | :------------------------------- |
| `history.gob`      | system + user + assistant (incl. `tool_calls`, tool results, `reasoning_content`)    | **Restart recovery, seamless LLM resume** |
| `YYYY-MM-DD.jsonl` | user / assistant plain text (incl. mode notices)                                     | Memory tool search               |

On restart, gob is loaded first; on failure it falls back to JSONL. If the system prompt changes due to a build upgrade or skill change, it is replaced in place with the current version during gob recovery.

### Session compression

Triggered automatically when a long conversation exceeds 70% of the context window. The tail is kept in layers (~20K tokens), older content is compressed into a coherent LLM-generated summary and merged with the existing summary. **The gob is updated in sync after compression**, staying consistent across restarts.

### Plan DAG concurrent scheduling

The model splits a complex task into DAG nodes via the `CreatePlan` tool, and deepx launches concurrent sub-agents by dependency order:

```
PlanCreated
  ├─ plan-1: Read (flash) ─────┐
  ├─ plan-2: Read (flash) ─────┤
  ├─ plan-3: Grep (flash) ─────┤
  └─ plan-4: Write (pro) ──────┘ depends_on: [1,2,3]
```

### Review mode (default)

| Mode               | Write / Update / Bash | Other tools | Switch command |
| :----------------- | :-------------------- | :---------- | :------------- |
| `review` (default) | Manual YES/NO confirm | Auto-run    | `/review`      |
| `auto`             | Auto-run              | Auto-run    | `/auto`        |
| `plan`             | Disabled              | Auto-run    | `/plan`        |

### Local OCR

Paste an image with Ctrl+V → deepx saves it to disk automatically → the LLM recognizes the text in the image via the `OCR` tool (PaddleOCR PP-OCRv5). The ~37MB model is downloaded automatically on first use; responses are sub-second afterward. **DeepSeek has no multimodal support, and local OCR fills its biggest gap.**

### Code graph

deepx has a built-in code graph engine. The model can do symbol-level navigation + call-relationship queries directly, replacing repo-wide grep and flipping through files one by one.

**Operation cheat sheet**

| op             | Purpose                              | Required params              | Notes                                                       |
| :------------- | :----------------------------------- | :--------------------------- | :---------------------------------------------------------- |
| `def`          | Where a symbol is defined            | `name`                       | Definition location of a function/type/method/variable      |
| `refs`         | Who uses a symbol                    | `name`                       | All references (definition + calls + accesses)              |
| `symbols`      | Fuzzy search symbols by name         | `name`(opt), `kind`(opt)     | `kind` filters: func/method/type/var/const/field            |
| `outline`      | What symbols a file has              | `path`                       | File outline                                                |
| `imports`      | What packages a file imports         | `path`                       | Dependency overview                                         |
| `callers`      | Who calls a function                 | `name`                       | **Check impact when changing a function**; covers Go implicit interfaces |
| `callees`      | What a function calls                | `name`                       | Understand a function's internal flow                       |
| `implementers` | Who implements an interface          | `name`                       | **Symbol-level precision** for Go implicit interfaces; grep can't find these |
| `subtypes`     | Who inherits/embeds a type           | `name`                       | Subtype tracking                                            |
| `supertypes`   | What a type derives from             | `name`                       | Parent types / embedded interfaces                          |
| `impact`       | Downstream affected by a symbol change | `name`, `depth`(default 3)  | Transitive closure, blast-radius analysis                   |
| `reindex`      | Force-rebuild the index              | —                            | Manually trigger when the cache is off                      |

**When to use CodeGraph vs Grep**

| Scenario                              |               Use                  |
| :------------------------------------ | :--------------------------------: |
| Function/type/variable def or refs    |    ✅ CodeGraph `def` / `refs`     |
| Call-chain upstream/downstream        | ✅ CodeGraph `callers` / `callees` |
| Interface implementation relations    |    ✅ CodeGraph `implementers`     |
| Impact scope of a code change         |       ✅ CodeGraph `impact`        |
| What symbols are in a file            |       ✅ CodeGraph `outline`       |
| Text in comments/strings/config       |              ❌ Grep               |
| Non-code files (JSON/MD/Shell/YAML)   |              ❌ Grep               |
| Unknown symbol name, fuzzy search     |     ✅ `symbols` + `kind` filter   |

**Supported languages**: Go (precise stdlib parsing) + TypeScript / JavaScript / Python / Java / Rust / C / C++ / C# / Ruby / PHP / Kotlin / Swift / Scala / Dart / Vue / Svelte.

**How it works**: On startup a background `Prewarm` builds the index automatically, with the status bar showing `loading → ready`. After a file is modified by the Write/Update tool it is marked `stale` and incrementally rebuilt on the next query. Results are shown as `file:line` (with signatures/callers), auto-truncated and paginated beyond the limit.

## Toolset

| Type        | Tools                              |         plan | auto | review |
| :---------- | :--------------------------------- | -----------: | :--: | :----: |
| File read   | `Read` `List` `Tree` `Glob` `Grep` |            ✓ |  ✓   |   ✓    |
| Code graph  | `CodeGraph`                        |            ✓ |  ✓   |   ✓    |
| File write  | `Write` `Update`                   |            ✗ |  ✓   |   ⏳   |
| Shell       | `Bash`                             |            ✗ |  ✓   |   ⏳   |
| Network     | `Search` `Fetch`                   |            ✓ |  ✓   |   ✓    |
| Memory      | `Memory`                           |            ✓ |  ✓   |   ✓    |
| Skill       | `LoadSkill`                        |            ✓ |  ✓   |   ✓    |
| Image       | `OCR`                              |            ✓ |  ✓   |   ✓    |
| Planning    | `CreatePlan` `UpdatePlanStatus`    | LLM-invoked  |      |        |
| Upgrade     | `SwitchModel`                      | LLM-invoked  |      |        |

> ⏳ = auto-run, but requires manual confirmation.

## Slash Commands

| Command   | Action                |
| :-------- | :-------------------- |
| `/plan`   | Switch to read-only mode |
| `/auto`   | Switch to fully automatic mode |
| `/review` | Switch to review mode |
| `/mode`   | Show the current mode |
| `/config` | Reconfigure the API key |
| `/skills` | List available skills |
| `/help`   | Help                  |

## Skills ecosystem

```
workspace level  <wd>/.deepx/skills/
global level     ~/.agents/skills/ → ~/.claude/skills/ → ~/.deepx/skills/
```

- Workspace-level skills can be `git add`ed and shared with the team.
- Global level is compatible with the Claude Code ecosystem — reuse existing skills directly.

## Architecture

```
Single turn:
  user input
    ↓
  RouteByKeyword (local) ─► flash or pro
    ↓
  StartStream (main loop)
    ├─ answer directly
    ├─ call tool → review intercepts write/Shell → execute → feed result back → continue
    ├─ SwitchModel → upgrade to pro
    └─ CreatePlan → DAG scheduler → concurrent sub-agents → aggregate

Session persistence:
  HistoryUpdateMsg → SaveGob (history.gob, full fidelity)
  StreamDoneMsg  → Append JSONL (plain text, Memory search)
  restart        → LoadGob (preferred) / JSONL (fallback)

Session compression:
  tokens ≥ ctxWindow × 70% → runCompression (async)
    → keep tail in layers ~20K tokens
    → LLM merges new and old summaries
    → update gob + state.json
```

## Token economy

- **Zero-token routing**: pure local keywords, no LLM call.
- **No tool pre-injection**: `Memory` / `LoadSkill` enter the context only when invoked.
- **Minimal system prompt**: only cross-tool conventions + workspace; per-tool trigger conditions live in each tool's description.
- **DeepSeek KV-cache friendly**: the tools array doesn't change with mode; the system prompt is version-aware on gob recovery.
- **Native code graph support**: cuts token waste at the root.

## Project structure

```
deepx/
├── main.go
├── agent/          StartStream tool loop + routing + DAG scheduling + sub-agents
├── config/         ~/.deepx/model.yaml read/write
├── session/        gob persistence + JSONL log + session compression state
├── tools/          all tool implementations (read/write/search/OCR/Memory/Skill/Plan/CodeGraph)
├── codegraph/      code graph: jump-to-def / find-callers / inheritance & impl / impact
├── skill/          multi-path skill discovery and loading
├── ocr/            PaddleOCR wrapper (ONNX Runtime)
├── tui/            bubbletea TUI (input/render/clipboard/selection/dashboard)
└── scripts/        install scripts
```

## Uninstall

```bash
# macOS / Linux
rm -f ~/.local/bin/deepx && rm -rf ~/.deepx

# Windows
# Delete %LOCALAPPDATA%\Programs\deepx and %USERPROFILE%\.deepx
```

## License

[MIT](LICENSE) © 2026 itmisx
