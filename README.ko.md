# deepx-code

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md) | **한국어**

> DeepSeek 표준 코딩 에이전트. 로컬 OCR 이미지 인식, 자동 컨텍스트 압축, 네이티브 codegraph 지원. 토큰 소비를 근본적으로 줄입니다.

![deepx screenshot](assets/screenshot.jpg)

## 왜 deepx 인가

- 🚀 Go 로 개발되어 작고 빠르며 모든 플랫폼 지원.
- 🚀 gob 바이너리 영속화. `tool_calls`, tool results, `reasoning_content` 를 모두 보존해 LLM 이 끊김 없이 이어받음.
- 🚀 계층 압축 + 기존 요약 병합.
- 🚀 skill, MCP 기본 탑재로 기존 Claude 생태계에 매끄럽게 통합.
- 🚀 로컬 키워드 라우팅. 제로 레이턴시, 제로 토큰으로 매칭 시 곧바로 pro 로 승급.
- 🚀 자동 모델 전환. 문제 복잡도에 따라 pro 모델로 자동 승급.
- 🚀 Plan DAG 동시 스케줄링. 의존 관계에 따라 서브 에이전트를 병렬 실행하며 각 노드가 독자적으로 모델 선택.
- 🚀 로컬 OCR (PaddleOCR). 오프라인 인식으로 멀티모달 API 에 의존하지 않음.
- 🚀 코드 그래프 (codeGraph). read/glob/grep 의 토큰 낭비를 크게 줄임.

## 빠른 시작

### 설치

- macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/itmisx/deepx-code/main/scripts/install.sh | bash
```

- Windows (PowerShell)

```bash
irm https://raw.githubusercontent.com/itmisx/deepx-code/main/scripts/install.ps1 | iex
```

## 사용법

### 워크스페이스 진입

```bash
cd <당신의 프로젝트 디렉터리>
deepx
```

### DeepSeek API KEY 설정

최초 실행 시 설정 창이 뜨며, 거기서 API key 를 설정합니다.

### Skill 설정

현재 디렉터리의 `.deepx/skills/` 에 배치합니다.

### MCP 설정

`/mcp-add` 명령으로 MCP 를 추가합니다.

## 핵심 메커니즘

### 모델 라우팅 (로컬, 제로 레이턴시)

사용자 메시지가 도착하면 deepx 는 로컬에서 키워드 매칭 + 길이 판정을 수행합니다:

```
메시지에 "重构/refactor/architecture/调试…" 포함 → 곧바로 pro 로 승급
메시지 길이 < 100 자                            → flash
메시지 길이 > 500 자                            → pro
```

중국어(간체/번체) / 영어 / 일본어 / 한국어 5개 언어 지원. **라우팅은 순식간에 일어나며 추가 LLM 토큰을 전혀 소비하지 않습니다.**

### 세션 영속화 (gob 바이너리)

```
~/.deepx/sessions/<sha1(workspace)[:16]>/
├── meta.json          # 워크스페이스 메타 정보
├── state.json         # 압축 상태 (summary + total_turns)
├── YYYY-MM-DD.jsonl   # 텍스트 로그 (Memory 검색용)
└── history.gob        # 완전한 바이너리 히스토리
```

| 포맷               | 저장 내용                                                                            | 용도                              |
| :----------------- | :----------------------------------------------------------------------------------- | :-------------------------------- |
| `history.gob`      | system + user + assistant (`tool_calls`, tool results, `reasoning_content` 포함)     | **재시작 복원, LLM 끊김 없는 이어받기** |
| `YYYY-MM-DD.jsonl` | user / assistant 순수 텍스트 (모드 알림 포함)                                         | Memory 도구 검색                  |

재시작 시 gob 을 우선 로드하고, 실패하면 JSONL 로 폴백합니다. system prompt 가 build 升级나 skill 변경으로 바뀌면 gob 복원 시 현재 버전으로 그 자리에서 교체됩니다.

### 세션 압축

긴 대화가 컨텍스트 윈도우의 70% 를 초과하면 자동으로 트리거됩니다. 꼬리 부분을 계층적으로 약 20K 토큰 유지하고, 오래된 내용을 LLM 이 생성한 일관된 요약으로 압축해 기존 요약과 병합합니다. **압축 후 gob 도 동기화 업데이트**되어 재시작해도 일관됩니다.

### Plan DAG 동시 스케줄링

모델은 `CreatePlan` 도구로 복잡한 작업을 DAG 노드로 분할하고, deepx 는 의존 관계에 따라 동시 서브 에이전트를 실행합니다:

```
PlanCreated
  ├─ plan-1: Read (flash) ─────┐
  ├─ plan-2: Read (flash) ─────┤
  ├─ plan-3: Grep (flash) ─────┤
  └─ plan-4: Write (pro) ──────┘ depends_on: [1,2,3]
```

### 리뷰 모드 (기본값)

| 모드               | Write / Update / Bash | 그 외 도구 | 전환 명령 |
| :----------------- | :-------------------- | :--------- | :-------- |
| `review` (기본값)  | 수동 YES/NO 확인      | 자동 실행  | `/review` |
| `auto`             | 자동 실행             | 자동 실행  | `/auto`   |
| `plan`             | 비활성화              | 자동 실행  | `/plan`   |

### 로컬 OCR

Ctrl+V 로 이미지 붙여넣기 → deepx 가 자동으로 디스크에 저장 → LLM 이 `OCR` 도구(PaddleOCR PP-OCRv5)로 이미지 속 텍스트를 인식. 최초에는 ~37MB 모델을 자동 다운로드하고 이후에는 초 단위로 응답합니다. **DeepSeek 은 멀티모달을 지원하지 않으며, 로컬 OCR 이 최대 약점을 보완합니다.**

### 코드 그래프

deepx 는 코드 그래프 엔진을 내장하고 있습니다. 모델은 심볼 수준 내비게이션 + 호출 관계 쿼리를 직접 수행할 수 있어, 저장소 전체 grep 과 파일을 하나씩 넘겨보는 작업을 대체합니다.

**연산 빠른 참조표**

| op             | 용도                       | 필수 파라미터               | 설명                                                  |
| :------------- | :------------------------- | :-------------------------- | :---------------------------------------------------- |
| `def`          | 심볼 정의 위치             | `name`                      | 함수/타입/메서드/변수의 정의 위치                     |
| `refs`         | 심볼을 사용하는 곳         | `name`                      | 모든 참조 (정의 + 호출 + 값 취득)                     |
| `symbols`      | 이름으로 심볼 퍼지 검색    | `name`(선택), `kind`(선택)  | `kind` 필터: func/method/type/var/const/field         |
| `outline`      | 파일 내 심볼 목록          | `path`                      | 파일 아웃라인                                         |
| `imports`      | 파일이 import 하는 패키지  | `path`                      | 의존성 개요                                           |
| `callers`      | 함수를 호출하는 곳         | `name`                      | **함수 변경 시 영향 범위 확인**, Go 암묵 인터페이스도 포함 |
| `callees`      | 함수가 호출하는 것         | `name`                      | 함수 내부 처리 흐름 이해                              |
| `implementers` | 인터페이스 구현체          | `name`                      | Go 암묵 인터페이스를 **심볼 수준으로 정확히**, grep 으로는 안 나옴 |
| `subtypes`     | 타입을 상속/임베드한 것    | `name`                      | 서브타입 추적                                         |
| `supertypes`   | 타입의 파생 원본           | `name`                      | 부모 타입 / 임베드 인터페이스                         |
| `impact`       | 심볼 변경이 미치는 하류    | `name`, `depth`(기본3)      | 추이 폐포, blast radius 분석                          |
| `reindex`      | 인덱스 강제 재구축         | —                           | 캐시 이상 시 수동 트리거                              |

**CodeGraph 와 Grep 의 구분**

| 시나리오                              |               사용                 |
| :------------------------------------ | :--------------------------------: |
| 함수/타입/변수 정의 또는 참조         |    ✅ CodeGraph `def` / `refs`     |
| 호출 체인 상류/하류                   | ✅ CodeGraph `callers` / `callees` |
| 인터페이스 구현 관계                  |    ✅ CodeGraph `implementers`     |
| 코드 변경의 영향 범위                 |       ✅ CodeGraph `impact`        |
| 파일 내 심볼                          |       ✅ CodeGraph `outline`       |
| 주석/문자열/설정 내 텍스트            |              ❌ Grep               |
| 비코드 파일 (JSON/MD/Shell/YAML)      |              ❌ Grep               |
| 심볼명이 불확실하고 퍼지 검색         |     ✅ `symbols` + `kind` 필터     |

**지원 언어**: Go(stdlib 정밀 파싱) + TypeScript / JavaScript / Python / Java / Rust / C / C++ / C# / Ruby / PHP / Kotlin / Swift / Scala / Dart / Vue / Svelte.

**동작 원리**: 시작 시 백그라운드 `Prewarm` 이 자동으로 인덱스를 구축하며, 상태 표시줄에 `loading → ready` 가 표시됩니다. 파일이 Write/Update 도구로 수정되면 `stale` 로 표시되고 다음 쿼리 시 증분 재구축됩니다. 결과는 `파일:줄`(시그니처/호출자 포함)로 표시되며 상한을 초과하면 자동으로 잘려 페이징됩니다.

## 도구 세트

| 유형        | 도구                               |         plan | auto | review |
| :---------- | :--------------------------------- | -----------: | :--: | :----: |
| 파일 읽기   | `Read` `List` `Tree` `Glob` `Grep` |            ✓ |  ✓   |   ✓    |
| 코드 그래프 | `CodeGraph`                        |            ✓ |  ✓   |   ✓    |
| 파일 쓰기   | `Write` `Update`                   |            ✗ |  ✓   |   ⏳   |
| Shell       | `Bash`                             |            ✗ |  ✓   |   ⏳   |
| 네트워크    | `Search` `Fetch`                   |            ✓ |  ✓   |   ✓    |
| 메모리      | `Memory`                           |            ✓ |  ✓   |   ✓    |
| 스킬        | `LoadSkill`                        |            ✓ |  ✓   |   ✓    |
| 이미지      | `OCR`                              |            ✓ |  ✓   |   ✓    |
| 플래닝      | `CreatePlan` `UpdatePlanStatus`    | LLM 자율 호출 |     |        |
| 승급        | `SwitchModel`                      | LLM 자율 호출 |     |        |

> ⏳ = 자동 실행이지만 수동 확인 필요.

## Slash 명령어

| 명령어    | 작용                  |
| :-------- | :-------------------- |
| `/plan`   | 읽기 전용 모드로 전환 |
| `/auto`   | 완전 자동 모드로 전환 |
| `/review` | 리뷰 모드로 전환      |
| `/mode`   | 현재 모드 표시        |
| `/config` | API key 재설정        |
| `/skills` | 사용 가능한 skill 목록 |
| `/help`   | 도움말                |

## Skills 생태계

```
workspace 레벨  <wd>/.deepx/skills/
global 레벨     ~/.agents/skills/ → ~/.claude/skills/ → ~/.deepx/skills/
```

- workspace 레벨은 `git add` 로 팀에 공유 가능.
- global 레벨은 Claude Code 생태계와 호환되어 기존 skill 을 그대로 재사용.

## 아키텍처

```
단일 턴:
  사용자 입력
    ↓
  RouteByKeyword (로컬) ─► flash 또는 pro
    ↓
  StartStream (메인 루프)
    ├─ 직접 응답
    ├─ 도구 호출 → review 가 write/Shell 차단 → 실행 → 결과 환류 → 계속
    ├─ SwitchModel → pro 로 승급
    └─ CreatePlan → DAG scheduler → 서브 에이전트 동시 → 집계

세션 영속화:
  HistoryUpdateMsg → SaveGob (history.gob, 완전 fidelity)
  StreamDoneMsg  → Append JSONL (순수 텍스트, Memory 검색)
  재시작         → LoadGob (우선) / JSONL (폴백)

세션 압축:
  tokens ≥ ctxWindow × 70% → runCompression (비동기)
    → 꼬리를 계층적으로 ~20K 토큰 유지
    → LLM 이 신구 요약 병합
    → gob + state.json 업데이트
```

## 토큰 경제

- **제로 토큰 라우팅**: 순수 로컬 키워드, LLM 호출 없음.
- **도구 사전 주입 없음**: `Memory` / `LoadSkill` 은 호출 시에만 context 에 진입.
- **극도로 간결한 system prompt**: 도구 간 규약 + workspace 만, 각 도구의 트리거 조건은 각자의 description 에.
- **DeepSeek KV cache 친화적**: tools 배열은 모드에 따라 바뀌지 않고, system prompt 는 gob 복원 시 버전을 인식.
- **코드 그래프 네이티브 지원**: 토큰 낭비를 근본부터 줄임.

## 프로젝트 구조

```
deepx/
├── main.go
├── agent/          StartStream 도구 루프 + 라우팅 + DAG 스케줄링 + 서브 에이전트
├── config/         ~/.deepx/model.yaml 읽기/쓰기
├── session/        gob 영속화 + JSONL 로그 + 세션 압축 상태
├── tools/          전체 도구 구현 (읽기/쓰기/검색/OCR/Memory/Skill/Plan/CodeGraph)
├── codegraph/      코드 그래프: 정의 점프 / 호출자 찾기 / 상속 구현 / 영향 범위
├── skill/          멀티 경로 skill 발견과 로드
├── ocr/            PaddleOCR 래퍼 (ONNX Runtime)
├── tui/            bubbletea TUI (입력/렌더링/클립보드/선택/대시보드)
└── scripts/        설치 스크립트
```

## 제거

```bash
# macOS / Linux
rm -f ~/.local/bin/deepx && rm -rf ~/.deepx

# Windows
# %LOCALAPPDATA%\Programs\deepx 와 %USERPROFILE%\.deepx 삭제
```

## License

[MIT](LICENSE) © 2026 itmisx
