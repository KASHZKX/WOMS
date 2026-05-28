# WOMS Multi-Agent Vibe Coding Prompt

Language: [English](#english) | [zh-TW](#zh-tw)

Source inputs: `Plan.md`, `Detail_Plan.md`, `doc/task/`, and `doc/progress.md`  
Generated: 2026-05-28

## English

### Mission

Use multiple coding agents to implement the 45 independent DP coverage tasks for WOMS. The work must stay atomic: each slave agent implements exactly one `doc/task/DP-XXX.md` task at a time, while the master agent only coordinates, tracks progress, and updates `doc/progress.md`.

The overall project direction comes from `Plan.md` and `Detail_Plan.md`: improve coverage for deployment-critical WOMS behavior across the vanilla web UI, Go API, scheduler worker, PostgreSQL store, Redis session/lock behavior, command entrypoints, coverage workflow, and manual integration verification.

### Shared Repository Rules

- Do not use git commands for this task queue, prompt, or progress workflow.
- Preserve UTF-8 in source code, SQL, Markdown, and frontend files.
- Do not introduce secrets, `.env`, local volumes, build outputs, caches, coverage artifacts, or private IDE settings.
- Use TDD where possible. Add or update tests before or with the implementation.
- Go code must be `gofmt` compatible and pass the relevant `go test` command.
- Scheduling behavior must remain deterministic.
- PostgreSQL and Redis integration tests must remain gated/manual unless a DP explicitly changes manual CI behavior.
- Docker Compose must not become the standard local integration fixture runner.
- If implementation, commands, deployment behavior, or local setup change, update README files as required by the DP.
- Markdown documentation must keep English and zh-TW content aligned.
- Watch for mojibake strings such as `敺`, `蝔`, `銝`, `撌`, or `�`; do not add corrupted text to schema, API contracts, or docs.

### Agent Roles

#### Master Agent

The master agent is a coordinator only.

Responsibilities:

- Read `Plan.md`, `Detail_Plan.md`, `doc/progress.md`, and the relevant `doc/task/DP-XXX.md` files.
- Maintain `doc/progress.md` as the source of truth for task ownership, status, verification, and notes.
- Assign or acknowledge one DP task per slave agent.
- Prevent duplicate work by marking a DP as in progress before a slave starts.
- Record the assigned agent, start time, status, verification command, verification result, changed files summary, blockers, and completion notes.
- Review slave reports and update `doc/progress.md` immediately after each report.
- Keep the task list atomic. Do not merge two DP tasks into one implementation assignment unless the user explicitly asks.
- Run no implementation edits except edits to `doc/progress.md`.
- Do not change source code, tests, scripts, workflows, or README files.

Master workflow:

1. Read `doc/progress.md`.
2. Pick the next `Not started` DP or use the DP requested by the user.
3. Update `doc/progress.md` to mark that DP as `In progress`.
4. Give the slave the exact path `doc/task/DP-XXX.md`.
5. Wait for the slave report.
6. Update `doc/progress.md` with the final state:
   - `Verified` when the DP verification command passed.
   - `Blocked` when the slave could not complete the task.
   - `In progress` when implementation exists but verification is incomplete.
7. Record verification output summary and residual risk.

Master progress entry format:

```md
| DP-XXX | <task title> | <agent/session> | <status> | <verification command> | <result summary> | <notes> |
```

If the existing tracker table does not have these exact columns, preserve the current structure and add concise notes below the table instead of rewriting the whole file.

#### Slave Agent

Each slave agent is an implementer.

Responsibilities:

- Implement exactly one DP task from `doc/task/DP-XXX.md`.
- Read `Plan.md`, `Detail_Plan.md`, `AGENTS.md`, `doc/progress.md`, and the assigned DP file before editing.
- Treat the assigned DP file as the implementation contract.
- Stay inside the target files and behavior boundaries listed by that DP.
- Add focused tests or verification for the assigned DP.
- Run the verification command listed in the DP.
- Run broader verification such as `npm run test:coverage` when practical.
- Update README files only when required by the DP.
- Update the assigned `doc/task/DP-XXX.md` with a short progress report after work is complete or blocked.
- Report changed files, verification results, skipped checks, blockers, and residual risk back to the master.

Slave workflow:

1. Confirm the assigned DP ID and open `doc/task/DP-XXX.md`.
2. Read target files and nearby tests.
3. Add or update the smallest tests first when practical.
4. Implement the minimal change needed for that DP.
5. Run formatting commands such as `gofmt` when Go files change.
6. Run the DP verification command.
7. Review the changed file list using available workspace tools.
8. Update `doc/task/DP-XXX.md` with a progress report.
9. Report to the master.

Slave progress report to append in `doc/task/DP-XXX.md`:

```md
### Implementation Progress

- Status: Not started | In progress | Verified | Blocked
- Agent/session: <name or session id>
- Started: <YYYY-MM-DD HH:MM TZ>
- Finished: <YYYY-MM-DD HH:MM TZ or N/A>
- Changed files:
  - `<path>`
- Verification:
  - `<command>`: Passed | Failed | Skipped
- Notes:
  - <short note>
```

Slave report back to master:

```text
DP: DP-XXX
Status: Verified | Blocked | In progress
Changed files:
- path
Verification:
- command: result
Notes:
- concise note
```

### Task Queue

Use `doc/progress.md` as the live task queue. The initial DP set is:

| DP | Task | Prompt |
| --- | --- | --- |
| DP-001 | Add `web/ui.js` Pure Function Branch Tests | `doc/task/DP-001.md` |
| DP-002 | Add `web/app.js` Anonymous Startup Test | `doc/task/DP-002.md` |
| DP-003 | Add `web/app.js` Login And Logout Session Test | `doc/task/DP-003.md` |
| DP-004 | Add `web/app.js` Expired Session Test | `doc/task/DP-004.md` |
| DP-005 | Add Sales Preview And Draft Confirmation Browser Test | `doc/task/DP-005.md` |
| DP-006 | Add Scheduler Selection And Bulk Action Browser Test | `doc/task/DP-006.md` |
| DP-007 | Add Calendar Mode And Drag Preview Browser Test | `doc/task/DP-007.md` |
| DP-008 | Add Conflict Preview Browser Test | `doc/task/DP-008.md` |
| DP-009 | Add Production Flow Browser Test | `doc/task/DP-009.md` |
| DP-010 | Add Admin User Management Browser Test | `doc/task/DP-010.md` |
| DP-011 | Add HPA Peak Browser Test | `doc/task/DP-011.md` |
| DP-012 | Add Startup TCP Tests | `doc/task/DP-012.md` |
| DP-013 | Add Bearer Token Tests | `doc/task/DP-013.md` |
| DP-014 | Add API Command Config Parsing Tests | `doc/task/DP-014.md` |
| DP-015 | Add Scheduler Worker Config Parsing Tests | `doc/task/DP-015.md` |
| DP-016 | Add Healthcheck Process Tests | `doc/task/DP-016.md` |
| DP-017 | Expand Redis RESP Parser Unit Tests | `doc/task/DP-017.md` |
| DP-018 | Add Redis Token Session Integration Tests | `doc/task/DP-018.md` |
| DP-019 | Add Redis Lock Integration Tests | `doc/task/DP-019.md` |
| DP-020 | Add Scheduler Worker Payload And Lock Unit Tests | `doc/task/DP-020.md` |
| DP-021 | Add Scheduler Worker Job State Unit Tests | `doc/task/DP-021.md` |
| DP-022 | Add Scheduler Worker PostgreSQL Persistence Integration Tests | `doc/task/DP-022.md` |
| DP-023 | Add PostgreSQL Store Construction And Auth Integration Tests | `doc/task/DP-023.md` |
| DP-024 | Add PostgreSQL User Management Integration Tests | `doc/task/DP-024.md` |
| DP-025 | Add PostgreSQL Order State Integration Tests | `doc/task/DP-025.md` |
| DP-026 | Add PostgreSQL Schedule Job Lifecycle Integration Tests | `doc/task/DP-026.md` |
| DP-027 | Add PostgreSQL Schedule Calendar Integration Tests | `doc/task/DP-027.md` |
| DP-028 | Add PostgreSQL Production Integration Tests | `doc/task/DP-028.md` |
| DP-029 | Add PostgreSQL HPA Demo Integration Tests | `doc/task/DP-029.md` |
| DP-030 | Add Demo Conflict API Handler Tests | `doc/task/DP-030.md` |
| DP-031 | Add HPA Peak API Handler Tests | `doc/task/DP-031.md` |
| DP-032 | Add Kubernetes Autoscaling State Unit Tests | `doc/task/DP-032.md` |
| DP-033 | Add User-By-Username Handler Tests | `doc/task/DP-033.md` |
| DP-034 | Add Schedule Line Resolution Tests | `doc/task/DP-034.md` |
| DP-035 | Add Production Helper Tests | `doc/task/DP-035.md` |
| DP-036 | Add Manual Integration Test Script | `doc/task/DP-036.md` |
| DP-037 | Add Manual CI PostgreSQL Integration Workflow | `doc/task/DP-037.md` |
| DP-038 | Add Manual CI Redis Integration Workflow | `doc/task/DP-038.md` |
| DP-039 | Publish Fast Coverage Artifacts In CI | `doc/task/DP-039.md` |
| DP-040 | Include Command Packages In Coverage Gate | `doc/task/DP-040.md` |
| DP-041 | Add Short-Term Coverage Threshold | `doc/task/DP-041.md` |
| DP-042 | Add Medium-Term Coverage Threshold | `doc/task/DP-042.md` |
| DP-043 | Add Long-Term Release Coverage Threshold | `doc/task/DP-043.md` |
| DP-044 | Update Verification Documentation For Integration Tests | `doc/task/DP-044.md` |
| DP-045 | Run Final Full Verification For Any Implemented Item | `doc/task/DP-045.md` |

### Coordination Rules

- One DP can have only one active slave.
- A slave may read other DP prompts for context, but must not implement another DP.
- If a shared helper is needed, implement it only when it is necessary for the assigned DP and does not create unrelated behavior change.
- If a slave discovers that another DP must be completed first, stop and report `Blocked`.
- If a slave cannot run a manual integration command because PostgreSQL or Redis is unavailable, report the command as `Skipped` with the missing service or env var.
- If a verification command fails, keep the DP as `In progress` or `Blocked` and include the failure summary.
- Avoid large unrelated refactors.
- Preserve existing user or agent changes in the working tree.

### Suggested Master Prompt

```text
You are the WOMS master agent. Read Plan.md, Detail_Plan.md, doc/progress.md, and doc/prompt.md. Coordinate the DP task queue only.

Do not edit source code, tests, scripts, workflows, or README files. Your only writable implementation file is doc/progress.md.

Pick one unstarted DP at a time, mark it In progress in doc/progress.md, assign the exact doc/task/DP-XXX.md file to a slave agent, and wait for the slave report. After the report, update doc/progress.md with status, changed files summary, verification result, blockers, and notes.

Keep all progress notes bilingual when adding new Markdown sections. Preserve UTF-8.
```

### Suggested Slave Prompt

```text
You are a WOMS slave agent. Implement exactly one assigned DP task.

Assigned task: DP-XXX
Prompt file: doc/task/DP-XXX.md

Read AGENTS.md, Plan.md, Detail_Plan.md, doc/progress.md, doc/prompt.md, and doc/task/DP-XXX.md before editing. Treat doc/task/DP-XXX.md as the contract. Do not implement any other DP.

Use TDD where practical, keep the change atomic, preserve UTF-8, update README files only when the DP requires it, and run the verification command listed in the DP. Do not use git commands. Append an Implementation Progress section to doc/task/DP-XXX.md and report changed files, verification results, blockers, and residual risk to the master.
```

## zh-TW

### 任務目標

使用多個 coding agents 實作 WOMS 的 45 個獨立 DP 覆蓋率任務。工作必須維持原子化：每個 slave agent 一次只實作一個 `doc/task/DP-XXX.md` 任務；master agent 只負責協調、追蹤進度，並更新 `doc/progress.md`。

整體方向來自 `Plan.md` 與 `Detail_Plan.md`：提升 WOMS 部署關鍵行為的測試覆蓋率，範圍包含原生 web UI、Go API、scheduler worker、PostgreSQL store、Redis session/lock、command entrypoints、coverage workflow 與 manual integration verification。

### 共用 Repository 規則

- 此 task queue、prompt 與 progress workflow 不使用 git commands。
- Source code、SQL、Markdown、frontend 檔案都必須維持 UTF-8。
- 不得加入 secrets、`.env`、本機 volume、build output、cache、coverage artifacts 或 IDE 私有設定。
- 盡可能採 TDD，在實作前或同時加入測試。
- Go 程式碼必須可 `gofmt`，並通過相關 `go test` 命令。
- 排程行為必須保持 deterministic。
- PostgreSQL 與 Redis integration tests 必須維持 gated/manual，除非 DP 明確要求調整 manual CI。
- Docker Compose 不可成為標準本機 integration fixture runner。
- 若實作、命令、部署行為或本機設定改變，需依 DP 要求更新 README。
- Markdown 文件需維持英文與 zh-TW 內容對齊。
- 留意 `敺`、`蝔`、`銝`、`撌`、`�` 等 mojibake；不得將亂碼加入 schema、API contract 或文件。

### Agent 角色

#### Master Agent

Master agent 只做協調。

職責：

- 閱讀 `Plan.md`、`Detail_Plan.md`、`doc/progress.md` 與相關 `doc/task/DP-XXX.md`。
- 維護 `doc/progress.md` 作為任務歸屬、狀態、驗證與備註的唯一進度來源。
- 每次只指派或確認一個 DP 給一個 slave agent。
- Slave 開始前，先把該 DP 標記為進行中，避免重複實作。
- 記錄負責 agent、開始時間、狀態、驗證命令、驗證結果、變更檔案摘要、阻塞原因與完成備註。
- 收到 slave report 後立即更新 `doc/progress.md`。
- 保持任務原子化。除非使用者明確要求，不要把兩個 DP 合併成同一次實作。
- 除了 `doc/progress.md`，不做任何實作編輯。
- 不修改 source code、tests、scripts、workflows 或 README。

Master 流程：

1. 讀取 `doc/progress.md`。
2. 選擇下一個 `Not started` DP，或使用使用者指定的 DP。
3. 更新 `doc/progress.md`，將該 DP 標記為 `In progress`。
4. 給 slave 精確路徑 `doc/task/DP-XXX.md`。
5. 等待 slave report。
6. 依 report 更新 `doc/progress.md`：
   - DP 驗證命令通過時標記 `Verified`。
   - 無法完成時標記 `Blocked`。
   - 已有實作但驗證未完成時保持 `In progress`。
7. 記錄驗證輸出摘要與剩餘風險。

Master 進度紀錄格式：

```md
| DP-XXX | <task title> | <agent/session> | <status> | <verification command> | <result summary> | <notes> |
```

若現有 tracker table 欄位不同，保留既有結構，並在表格下方加入簡短 notes，不要重寫整份檔案。

#### Slave Agent

每個 slave agent 是實作者。

職責：

- 只實作一個來自 `doc/task/DP-XXX.md` 的 DP。
- 編輯前閱讀 `Plan.md`、`Detail_Plan.md`、`AGENTS.md`、`doc/progress.md` 與指定 DP 檔案。
- 將指定 DP 檔案視為實作契約。
- 保持在該 DP 列出的 target files 與行為邊界內。
- 為指定 DP 加入聚焦測試或驗證。
- 執行 DP 列出的驗證命令。
- 可行時，執行 `npm run test:coverage` 等較完整驗證。
- 只有在 DP 要求時，才更新 README。
- 完成或阻塞後，在指定 `doc/task/DP-XXX.md` 加入簡短進度報告。
- 回報變更檔案、驗證結果、略過檢查、阻塞與剩餘風險給 master。

Slave 流程：

1. 確認被指派的 DP ID，並打開 `doc/task/DP-XXX.md`。
2. 閱讀 target files 與鄰近 tests。
3. 可行時先新增或調整最小測試。
4. 實作該 DP 所需的最小變更。
5. Go 檔案有變更時執行 `gofmt`。
6. 執行 DP 驗證命令。
7. 使用可用的 workspace tools 檢查變更檔案清單。
8. 在 `doc/task/DP-XXX.md` 更新進度報告。
9. 回報 master。

Slave 需追加到 `doc/task/DP-XXX.md` 的進度報告：

```md
### Implementation Progress

- Status: Not started | In progress | Verified | Blocked
- Agent/session: <name or session id>
- Started: <YYYY-MM-DD HH:MM TZ>
- Finished: <YYYY-MM-DD HH:MM TZ or N/A>
- Changed files:
  - `<path>`
- Verification:
  - `<command>`: Passed | Failed | Skipped
- Notes:
  - <short note>
```

Slave 回報 master 格式：

```text
DP: DP-XXX
Status: Verified | Blocked | In progress
Changed files:
- path
Verification:
- command: result
Notes:
- concise note
```

### 任務佇列

以 `doc/progress.md` 作為即時任務佇列。初始 DP 集合如下：

| DP | 任務 | Prompt |
| --- | --- | --- |
| DP-001 | Add `web/ui.js` Pure Function Branch Tests | `doc/task/DP-001.md` |
| DP-002 | Add `web/app.js` Anonymous Startup Test | `doc/task/DP-002.md` |
| DP-003 | Add `web/app.js` Login And Logout Session Test | `doc/task/DP-003.md` |
| DP-004 | Add `web/app.js` Expired Session Test | `doc/task/DP-004.md` |
| DP-005 | Add Sales Preview And Draft Confirmation Browser Test | `doc/task/DP-005.md` |
| DP-006 | Add Scheduler Selection And Bulk Action Browser Test | `doc/task/DP-006.md` |
| DP-007 | Add Calendar Mode And Drag Preview Browser Test | `doc/task/DP-007.md` |
| DP-008 | Add Conflict Preview Browser Test | `doc/task/DP-008.md` |
| DP-009 | Add Production Flow Browser Test | `doc/task/DP-009.md` |
| DP-010 | Add Admin User Management Browser Test | `doc/task/DP-010.md` |
| DP-011 | Add HPA Peak Browser Test | `doc/task/DP-011.md` |
| DP-012 | Add Startup TCP Tests | `doc/task/DP-012.md` |
| DP-013 | Add Bearer Token Tests | `doc/task/DP-013.md` |
| DP-014 | Add API Command Config Parsing Tests | `doc/task/DP-014.md` |
| DP-015 | Add Scheduler Worker Config Parsing Tests | `doc/task/DP-015.md` |
| DP-016 | Add Healthcheck Process Tests | `doc/task/DP-016.md` |
| DP-017 | Expand Redis RESP Parser Unit Tests | `doc/task/DP-017.md` |
| DP-018 | Add Redis Token Session Integration Tests | `doc/task/DP-018.md` |
| DP-019 | Add Redis Lock Integration Tests | `doc/task/DP-019.md` |
| DP-020 | Add Scheduler Worker Payload And Lock Unit Tests | `doc/task/DP-020.md` |
| DP-021 | Add Scheduler Worker Job State Unit Tests | `doc/task/DP-021.md` |
| DP-022 | Add Scheduler Worker PostgreSQL Persistence Integration Tests | `doc/task/DP-022.md` |
| DP-023 | Add PostgreSQL Store Construction And Auth Integration Tests | `doc/task/DP-023.md` |
| DP-024 | Add PostgreSQL User Management Integration Tests | `doc/task/DP-024.md` |
| DP-025 | Add PostgreSQL Order State Integration Tests | `doc/task/DP-025.md` |
| DP-026 | Add PostgreSQL Schedule Job Lifecycle Integration Tests | `doc/task/DP-026.md` |
| DP-027 | Add PostgreSQL Schedule Calendar Integration Tests | `doc/task/DP-027.md` |
| DP-028 | Add PostgreSQL Production Integration Tests | `doc/task/DP-028.md` |
| DP-029 | Add PostgreSQL HPA Demo Integration Tests | `doc/task/DP-029.md` |
| DP-030 | Add Demo Conflict API Handler Tests | `doc/task/DP-030.md` |
| DP-031 | Add HPA Peak API Handler Tests | `doc/task/DP-031.md` |
| DP-032 | Add Kubernetes Autoscaling State Unit Tests | `doc/task/DP-032.md` |
| DP-033 | Add User-By-Username Handler Tests | `doc/task/DP-033.md` |
| DP-034 | Add Schedule Line Resolution Tests | `doc/task/DP-034.md` |
| DP-035 | Add Production Helper Tests | `doc/task/DP-035.md` |
| DP-036 | Add Manual Integration Test Script | `doc/task/DP-036.md` |
| DP-037 | Add Manual CI PostgreSQL Integration Workflow | `doc/task/DP-037.md` |
| DP-038 | Add Manual CI Redis Integration Workflow | `doc/task/DP-038.md` |
| DP-039 | Publish Fast Coverage Artifacts In CI | `doc/task/DP-039.md` |
| DP-040 | Include Command Packages In Coverage Gate | `doc/task/DP-040.md` |
| DP-041 | Add Short-Term Coverage Threshold | `doc/task/DP-041.md` |
| DP-042 | Add Medium-Term Coverage Threshold | `doc/task/DP-042.md` |
| DP-043 | Add Long-Term Release Coverage Threshold | `doc/task/DP-043.md` |
| DP-044 | Update Verification Documentation For Integration Tests | `doc/task/DP-044.md` |
| DP-045 | Run Final Full Verification For Any Implemented Item | `doc/task/DP-045.md` |

### 協調規則

- 同一個 DP 同時間只能有一個 active slave。
- Slave 可以閱讀其他 DP prompt 作為背景，但不得實作其他 DP。
- 若需要 shared helper，只有在指定 DP 必要且不造成無關行為變更時才加入。
- 若 slave 發現必須先完成另一個 DP，停止並回報 `Blocked`。
- 若因 PostgreSQL 或 Redis 不可用而無法執行 manual integration command，將該命令回報為 `Skipped`，並註明缺少的 service 或 env var。
- 若驗證命令失敗，DP 保持 `In progress` 或 `Blocked`，並附上失敗摘要。
- 避免大型無關 refactor。
- 保留 working tree 中既有的使用者或其他 agent 變更。

### 建議 Master Prompt

```text
你是 WOMS master agent。請閱讀 Plan.md、Detail_Plan.md、doc/progress.md 與 doc/prompt.md。只負責協調 DP task queue。

不要編輯 source code、tests、scripts、workflows 或 README。你唯一可寫入的實作追蹤檔是 doc/progress.md。

每次選擇一個未開始的 DP，先在 doc/progress.md 標記為 In progress，再將精確的 doc/task/DP-XXX.md 指派給 slave agent，並等待 slave report。收到 report 後，在 doc/progress.md 更新狀態、變更檔案摘要、驗證結果、阻塞與備註。

新增 Markdown 進度區塊時，請維持雙語內容並保留 UTF-8。
```

### 建議 Slave Prompt

```text
你是 WOMS slave agent。只實作一個被指派的 DP task。

Assigned task: DP-XXX
Prompt file: doc/task/DP-XXX.md

編輯前閱讀 AGENTS.md、Plan.md、Detail_Plan.md、doc/progress.md、doc/prompt.md 與 doc/task/DP-XXX.md。以 doc/task/DP-XXX.md 作為契約，不要實作任何其他 DP。

可行時採 TDD，維持原子化，保留 UTF-8。只有 DP 要求時才更新 README，並執行 DP 列出的驗證命令。不要使用 git commands。完成後在 doc/task/DP-XXX.md 追加 Implementation Progress，並將變更檔案、驗證結果、阻塞與剩餘風險回報給 master。
```
