# deepx-code

**简体中文** | [English](README.en.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

> deepseek标配coding agent,支持本地OCR图片识别，自动上下文压缩，原生codegraph支持，根本上减少token的消耗

![deepx screenshot](assets/screenshot.jpg)

## 为什么选 deepx

- 🚀 golang开发，小巧快速，全平台覆盖。
- 🚀 gob二进制持久化。tool_calls、tool results、reasoning_content 全部保留，LLM 无缝续接
- 🚀 分层压缩 + 旧摘要合并
- 🚀 标配skill，MCP，无缝融合现有claude生态
- 🚀 本地关键词路由。零延迟、零 token 消耗，命中直接升 pro
- 🚀 自动模型切换。根据问题复杂度，自动升级为pro模型
- 🚀 Plan DAG 并发调度。按依赖关系并行跑子 agent，每个节点独立选模型
- 🚀 本地OCR(PaddleOCR)。离线识别，不依赖多模态 API
- 🚀 代码图谱(codeGraph)。极大减少read,glob,grep token浪费

## 快速开始

### 安装

- macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/itmisx/deepx-code/main/scripts/install.sh | bash
```

- Windows (PowerShell)

```bash
irm https://raw.githubusercontent.com/itmisx/deepx-code/main/scripts/install.ps1 | iex
```

## 使用

### 进入工作空间

```bash
cd <你的项目目录>
deepx
```

### 配置deepseek API KEY

首次启动弹出配置弹窗，配置 API key

### 配置Skill

当前目录配置 .deepx/skills/

### 配置MCP

通过指令 /mcp-add 添加mcp

## 核心机制

### 模型路由（本地，零延迟）

用户消息发来时，deepx 在本地做关键词匹配 + 长度判定：

```
消息含 "重构/refactor/architecture/调试…" → 直接升 pro
消息长度 < 100 字符                       → flash
消息长度 > 500 字符                       → pro
```

覆盖中（简/繁）/ 英 / 日 / 韩五种语言。**路由发生在一瞬间，不额外消耗任何 LLM token。**

### 会话持久化（gob 二进制）

```
~/.deepx/sessions/<sha1(workspace)[:16]>/
├── meta.json          # 工作区元信息
├── state.json         # 压缩状态 (summary + total_turns)
├── YYYY-MM-DD.jsonl   # 文本日志（Memory 搜索用）
└── history.gob        # 二进制完整历史
```

| 格式               | 存储内容                                                                          | 用途                       |
| :----------------- | :-------------------------------------------------------------------------------- | :------------------------- |
| `history.gob`      | system + user + assistant（含 `tool_calls`、`tool results`、`reasoning_content`） | **重启恢复，LLM 无缝续接** |
| `YYYY-MM-DD.jsonl` | user / assistant 纯文本（含模式通知）                                             | Memory 工具搜索            |

重启时优先加载 gob；失败则回退 JSONL。system prompt 如因 build 升级 / skill 变化而变动，gob 恢复时自动原地替换为当前版本。

### 会话压缩

长对话超出上下文窗口 70% 时自动触发。尾部分层保留约 20K token，旧内容压缩为 LLM 生成的连贯摘要并合并已有摘要。**压缩后也同步更新 gob**，重启一致。

### Plan DAG 并发调度

模型通过 `CreatePlan` 工具将复杂任务拆为 DAG 节点，deepx 按依赖关系启动并发子 agent：

```
PlanCreated
  ├─ plan-1: Read (flash) ─────┐
  ├─ plan-2: Read (flash) ─────┤
  ├─ plan-3: Grep (flash) ─────┤
  └─ plan-4: Write (pro) ──────┘ depends_on: [1,2,3]
```

### 审核模式（默认）

| 模式             | Write / Update / Bash | 其余工具 | 切换命令  |
| :--------------- | :-------------------- | :------- | :-------- |
| `review`（默认） | 人工 YES/NO 确认      | 自动执行 | `/review` |
| `auto`           | 自动执行              | 自动执行 | `/auto`   |
| `plan`           | 禁用                  | 自动执行 | `/plan`   |

### 本地 OCR

Ctrl+V 粘贴图片 → deepx 自动落盘 → LLM 通过 `OCR` 工具（PaddleOCR PP-OCRv5）识别图片中的文字。首次自动下载 ~37MB 模型，后续秒级响应。**DeepSeek 不支持多模态，本地 OCR 补齐最大短板。**

### 代码图谱

deepx 内置代码图谱引擎，模型可以直接做符号级导航 + 调用关系查询，代替满仓库 grep + 一个个翻文件。

**操作速查表**

| op             | 用途                 | 必填参数                   | 说明                                            |
| :------------- | :------------------- | :------------------------- | :---------------------------------------------- |
| `def`          | 符号定义在哪         | `name`                     | 函数/类型/方法/变量的定义位置                   |
| `refs`         | 谁用到了某符号       | `name`                     | 全部引用（定义 + 调用 + 取值）                  |
| `symbols`      | 按名模糊搜索符号     | `name`(可选), `kind`(可选) | `kind` 可过滤: func/method/type/var/const/field |
| `outline`      | 一个文件有哪些符号   | `path`                     | 文件大纲                                        |
| `imports`      | 文件 import 了哪些包 | `path`                     | 依赖概览                                        |
| `callers`      | 谁调用了某函数       | `name`                     | **改函数时查影响面**，Go 隐式接口也覆盖         |
| `callees`      | 某函数调用了哪些     | `name`                     | 理解函数内部实现流程                            |
| `implementers` | 谁实现了某接口       | `name`                     | 对 Go 隐式接口**精确到符号级**，grep 查不出     |
| `subtypes`     | 谁继承/嵌入了某类型  | `name`                     | 子类型追踪                                      |
| `supertypes`   | 某类型派生自什么     | `name`                     | 父类型/嵌入接口                                 |
| `impact`       | 改某符号牵连哪些下游 | `name`, `depth`(默认3)     | 传递闭包，blast radius 分析                     |
| `reindex`      | 强制重建索引         | —                          | 缓存异常时手动触发                              |

**何时用 CodeGraph vs Grep**

| 场景                             |                 用                 |
| :------------------------------- | :--------------------------------: |
| 函数/类型/变量定义或引用         |    ✅ CodeGraph `def` / `refs`     |
| 调用链上下游                     | ✅ CodeGraph `callers` / `callees` |
| 接口实现关系                     |    ✅ CodeGraph `implementers`     |
| 改代码的影响范围                 |       ✅ CodeGraph `impact`        |
| 文件里有哪些符号                 |       ✅ CodeGraph `outline`       |
| 注释/字符串/配置里的文本         |              ❌ Grep               |
| 非代码文件（JSON/MD/Shell/YAML） |              ❌ Grep               |
| 不确定符号名且模糊搜索           |     ✅ `symbols` + `kind` 过滤     |

**覆盖语言**：Go（stdlib 精确解析） + TypeScript / JavaScript / Python / Java / Rust / C / C++ / C# / Ruby / PHP / Kotlin / Swift / Scala / Dart / Vue / Svelte。

**工作机制**：启动时后台 `Prewarm` 自动建立索引，状态栏显示 `loading → ready`。文件被 Write/Update 工具修改后自动标记为 `stale`，下次查询时增量重建。查询结果按 `文件:行` 展示（含签名/调用方），超出上限自动截断分页。

## 工具集

| 类型     | 工具                               |         plan | auto | review |
| :------- | :--------------------------------- | -----------: | :--: | :----: |
| 文件只读 | `Read` `List` `Tree` `Glob` `Grep` |            ✓ |  ✓   |   ✓    |
| 代码图谱 | `CodeGraph`                        |            ✓ |  ✓   |   ✓    |
| 文件写入 | `Write` `Update`                   |            ✗ |  ✓   |   ⏳   |
| Shell    | `Bash`                             |            ✗ |  ✓   |   ⏳   |
| 联网     | `Search` `Fetch`                   |            ✓ |  ✓   |   ✓    |
| 记忆     | `Memory`                           |            ✓ |  ✓   |   ✓    |
| 技能     | `LoadSkill`                        |            ✓ |  ✓   |   ✓    |
| 图片     | `OCR`                              |            ✓ |  ✓   |   ✓    |
| 规划     | `CreatePlan` `UpdatePlanStatus`    | LLM 自主调用 |      |        |
| 升级     | `SwitchModel`                      | LLM 自主调用 |      |        |

> ⏳ = 自动执行，但需人工确认。

## Slash 命令

| 命令      | 作用             |
| :-------- | :--------------- |
| `/plan`   | 切只读模式       |
| `/auto`   | 切全自动模式     |
| `/review` | 切审核模式       |
| `/mode`   | 显示当前模式     |
| `/config` | 重新配置 API key |
| `/skills` | 列出可用 skill   |
| `/help`   | 帮助             |

## Skills 生态

```
workspace 级  <wd>/.deepx/skills/
global 级     ~/.agents/skills/ → ~/.claude/skills/ → ~/.deepx/skills/
```

- workspace 级可 `git add` 共享给团队
- global 兼容 Claude Code 生态，已有 skill 直接复用

## 架构

```
单轮对话:
  用户输入
    ↓
  RouteByKeyword (本地) ─► flash 或 pro
    ↓
  StartStream (主循环)
    ├─ 直接答
    ├─ 调工具 → review 拦截写/Shell → 执行 → 结果回灌 → 继续
    ├─ SwitchModel → 升 pro
    └─ CreatePlan → DAG scheduler → 子 agent 并发 → 汇总

会话持久化:
  HistoryUpdateMsg → SaveGob (history.gob, 完整 fidelity)
  StreamDoneMsg  → Append JSONL (纯文本, Memory 搜索)
  重启           → LoadGob (优先) / JSONL (回退)

会话压缩:
  tokens ≥ ctxWindow × 70% → runCompression (异步)
    → 尾部分层保留 ~20K token
    → LLM 合并新旧摘要
    → 更新 gob + state.json
```

## Token 经济

- **路由零 token**：纯本地关键词，不发 LLM 调用
- **工具不预注入**：`Memory` / `LoadSkill` 只在调用时才进 context
- **system prompt 极简**：仅跨工具规约 + workspace，工具触发条件在各自 description 里
- **DeepSeek KV cache 友好**：tools 数组不随模式变化；system prompt gob 恢复时版本感知
- **原生支持代码图谱**：根上减少token的浪费

## 项目结构

```
deepx/
├── main.go
├── agent/          StartStream 工具循环 + 路由 + DAG 调度 + 子 agent
├── config/         ~/.deepx/model.yaml 读写
├── session/        gob 持久化 + JSONL 日志 + 会话压缩状态
├── tools/          全部工具实现（读写/搜索/OCR/Memory/Skill/Plan/CodeGraph）
├── codegraph/      代码图谱：跳定义 / 找调用 / 继承实现 / 影响面
├── skill/          多路径 skill 发现与加载
├── ocr/            PaddleOCR 包装（ONNX Runtime）
├── tui/            bubbletea TUI（输入/渲染/剪贴板/选中/仪表盘）
└── scripts/        安装脚本
```

## 卸载

```bash
# macOS / Linux
rm -f ~/.local/bin/deepx && rm -rf ~/.deepx

# Windows
# 删除 %LOCALAPPDATA%\Programs\deepx 和 %USERPROFILE%\.deepx
```

## License

[MIT](LICENSE) © 2026 itmisx
