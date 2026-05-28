# DP Progress

Language: [English](#english) | [zh-TW](#zh-tw)

Source: `Plan.md` and `Detail_Plan.md`  
Generated: 2026-05-28

## English

### Summary

- DP prompt split: completed.
- Task prompt files generated: 45.
- Implementation progress: 1/45 completed.
- Git usage: disabled for task prompts, the multi-agent prompt, and this progress tracker.
- Current implementation status is initialized from the plan only; no DP implementation was performed while generating these prompt files.

### Status Legend

- Not started: prompt exists, implementation has not been started in this tracker.
- In progress: implementation has begun but verification is not complete.
- Verified: DP verification command passed.
- Blocked: implementation or verification needs a decision, service, or dependency.

### Task Tracker

| DP | Task | Prompt | Status | Verification |
| --- | --- | --- | --- | --- |
| DP-001 | Add `web/ui.js` Pure Function Branch Tests | [doc/task/DP-001.md](task/DP-001.md) | Verified | `npm run test:web:coverage` |
| DP-002 | Add `web/app.js` Anonymous Startup Test | [doc/task/DP-002.md](task/DP-002.md) | Verified | `npm run test:web:coverage` |
| DP-003 | Add `web/app.js` Login And Logout Session Test | [doc/task/DP-003.md](task/DP-003.md) | Verified | `npm run test:web:coverage` |
| DP-004 | Add `web/app.js` Expired Session Test | [doc/task/DP-004.md](task/DP-004.md) | Verified | `npm run test:web:coverage` |
| DP-005 | Add Sales Preview And Draft Confirmation Browser Test | [doc/task/DP-005.md](task/DP-005.md) | Verified | `npm run test:web:coverage` |
| DP-006 | Add Scheduler Selection And Bulk Action Browser Test | [doc/task/DP-006.md](task/DP-006.md) | Verified | `npm run test:web:coverage` |
| DP-007 | Add Calendar Mode And Drag Preview Browser Test | [doc/task/DP-007.md](task/DP-007.md) | Verified | `npm run test:web:coverage` |
| DP-008 | Add Conflict Preview Browser Test | [doc/task/DP-008.md](task/DP-008.md) | Verified | `npm run test:web:coverage` |
| DP-009 | Add Production Flow Browser Test | [doc/task/DP-009.md](task/DP-009.md) | Verified | `npm run test:web:coverage` |
| DP-010 | Add Admin User Management Browser Test | [doc/task/DP-010.md](task/DP-010.md) | Verified | `npm run test:web:coverage` |
| DP-011 | Add HPA Peak Browser Test | [doc/task/DP-011.md](task/DP-011.md) | Verified | `npm run test:web:coverage` |
| DP-012 | Add Startup TCP Tests | [doc/task/DP-012.md](task/DP-012.md) | Verified | `go test ./internal/startup` |
| DP-013 | Add Bearer Token Tests | [doc/task/DP-013.md](task/DP-013.md) | Verified | `go test ./internal/auth` |
| DP-014 | Add API Command Config Parsing Tests | [doc/task/DP-014.md](task/DP-014.md) | Verified | `go test ./cmd/api` |
| DP-015 | Add Scheduler Worker Config Parsing Tests | [doc/task/DP-015.md](task/DP-015.md) | Verified | `go test ./cmd/scheduler-worker` |
| DP-016 | Add Healthcheck Process Tests | [doc/task/DP-016.md](task/DP-016.md) | Verified | `go test ./cmd/healthcheck` |
| DP-017 | Expand Redis RESP Parser Unit Tests | [doc/task/DP-017.md](task/DP-017.md) | Verified | `go test ./internal/api` |
| DP-018 | Add Redis Token Session Integration Tests | [doc/task/DP-018.md](task/DP-018.md) | In progress | `WOMS_INTEGRATION_TESTS=1 REDIS_ADDR=127.0.0.1:6379 go test ./internal/api` |
| DP-019 | Add Redis Lock Integration Tests | [doc/task/DP-019.md](task/DP-019.md) | In progress | `WOMS_INTEGRATION_TESTS=1 REDIS_ADDR=127.0.0.1:6379 go test ./internal/lock` |
| DP-020 | Add Scheduler Worker Payload And Lock Unit Tests | [doc/task/DP-020.md](task/DP-020.md) | Verified | `go test ./cmd/scheduler-worker` |
| DP-021 | Add Scheduler Worker Job State Unit Tests | [doc/task/DP-021.md](task/DP-021.md) | Verified | `go test ./cmd/scheduler-worker` |
| DP-022 | Add Scheduler Worker PostgreSQL Persistence Integration Tests | [doc/task/DP-022.md](task/DP-022.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./cmd/scheduler-worker` |
| DP-023 | Add PostgreSQL Store Construction And Auth Integration Tests | [doc/task/DP-023.md](task/DP-023.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-024 | Add PostgreSQL User Management Integration Tests | [doc/task/DP-024.md](task/DP-024.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-025 | Add PostgreSQL Order State Integration Tests | [doc/task/DP-025.md](task/DP-025.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-026 | Add PostgreSQL Schedule Job Lifecycle Integration Tests | [doc/task/DP-026.md](task/DP-026.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-027 | Add PostgreSQL Schedule Calendar Integration Tests | [doc/task/DP-027.md](task/DP-027.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-028 | Add PostgreSQL Production Integration Tests | [doc/task/DP-028.md](task/DP-028.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-029 | Add PostgreSQL HPA Demo Integration Tests | [doc/task/DP-029.md](task/DP-029.md) | In progress | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-030 | Add Demo Conflict API Handler Tests | [doc/task/DP-030.md](task/DP-030.md) | Verified | `go test ./internal/api` |
| DP-031 | Add HPA Peak API Handler Tests | [doc/task/DP-031.md](task/DP-031.md) | Verified | `go test ./internal/api` |
| DP-032 | Add Kubernetes Autoscaling State Unit Tests | [doc/task/DP-032.md](task/DP-032.md) | Verified | `go test ./internal/api` |
| DP-033 | Add User-By-Username Handler Tests | [doc/task/DP-033.md](task/DP-033.md) | Verified | `go test ./internal/api` |
| DP-034 | Add Schedule Line Resolution Tests | [doc/task/DP-034.md](task/DP-034.md) | Verified | `go test ./internal/api` |
| DP-035 | Add Production Helper Tests | [doc/task/DP-035.md](task/DP-035.md) | Verified | `go test ./internal/api` |
| DP-036 | Add Manual Integration Test Script | [doc/task/DP-036.md](task/DP-036.md) | Verified | `npm run test:integration` |
| DP-037 | Add Manual CI PostgreSQL Integration Workflow | [doc/task/DP-037.md](task/DP-037.md) | Verified | `npm run test:coverage` |
| DP-038 | Add Manual CI Redis Integration Workflow | [doc/task/DP-038.md](task/DP-038.md) | Verified | `npm run test:coverage` |
| DP-039 | Publish Fast Coverage Artifacts In CI | [doc/task/DP-039.md](task/DP-039.md) | Verified | `npm run test:coverage` |
| DP-040 | Include Command Packages In Coverage Gate | [doc/task/DP-040.md](task/DP-040.md) | Verified | `npm run test:go:coverage` |
| DP-041 | Add Short-Term Coverage Threshold | [doc/task/DP-041.md](task/DP-041.md) | Verified | `npm run test:coverage` |
| DP-042 | Add Medium-Term Coverage Threshold | [doc/task/DP-042.md](task/DP-042.md) | Verified | `npm run test:coverage` |
| DP-043 | Add Long-Term Release Coverage Threshold | [doc/task/DP-043.md](task/DP-043.md) | Verified | `npm run test:coverage` |
| DP-044 | Update Verification Documentation For Integration Tests | [doc/task/DP-044.md](task/DP-044.md) | Verified | `npm run test:coverage` |
| DP-045 | Run Final Full Verification For Any Implemented Item | [doc/task/DP-045.md](task/DP-045.md) | Not started | `npm run test:coverage` |

### Progress Notes

- 2026-05-28 17:42 CST: DP-001 assigned to Codex slave agent Schrodinger and verified. Changed files: `web/ui.test.mjs`, `doc/task/DP-001.md`. Verification: `npm run test:web:coverage` passed. Notes: added direct pure-function branch coverage for empty queries, invalid dates/time zones, date boundaries, and waterline capacity/threshold behavior.

## zh-TW

### 摘要

- DP prompt 拆分：已完成。
- 已產生 task prompt 檔案：45 份。
- 實作進度：1/45 完成。
- Git 使用：task prompts、multi-agent prompt 與本進度追蹤檔皆停用 git commands。
- 目前實作狀態只依計畫初始化；產生 prompt 檔案時尚未實作任何 DP。

### 狀態說明

- 未開始：prompt 已存在，但 tracker 尚未記錄實作開始。
- 進行中：已開始實作，但驗證尚未完成。
- 已驗證：DP 指定驗證命令已通過。
- 阻塞：實作或驗證需要決策、服務或 dependency。

### Task Tracker

| DP | 任務 | Prompt | 狀態 | 驗證 |
| --- | --- | --- | --- | --- |
| DP-001 | Add `web/ui.js` Pure Function Branch Tests | [doc/task/DP-001.md](task/DP-001.md) | 已驗證 | `npm run test:web:coverage` |
| DP-002 | Add `web/app.js` Anonymous Startup Test | [doc/task/DP-002.md](task/DP-002.md) | 已驗證 | `npm run test:web:coverage` |
| DP-003 | Add `web/app.js` Login And Logout Session Test | [doc/task/DP-003.md](task/DP-003.md) | 已驗證 | `npm run test:web:coverage` |
| DP-004 | Add `web/app.js` Expired Session Test | [doc/task/DP-004.md](task/DP-004.md) | 已驗證 | `npm run test:web:coverage` |
| DP-005 | Add Sales Preview And Draft Confirmation Browser Test | [doc/task/DP-005.md](task/DP-005.md) | 已驗證 | `npm run test:web:coverage` |
| DP-006 | Add Scheduler Selection And Bulk Action Browser Test | [doc/task/DP-006.md](task/DP-006.md) | 已驗證 | `npm run test:web:coverage` |
| DP-007 | Add Calendar Mode And Drag Preview Browser Test | [doc/task/DP-007.md](task/DP-007.md) | 已驗證 | `npm run test:web:coverage` |
| DP-008 | Add Conflict Preview Browser Test | [doc/task/DP-008.md](task/DP-008.md) | 已驗證 | `npm run test:web:coverage` |
| DP-009 | Add Production Flow Browser Test | [doc/task/DP-009.md](task/DP-009.md) | 已驗證 | `npm run test:web:coverage` |
| DP-010 | Add Admin User Management Browser Test | [doc/task/DP-010.md](task/DP-010.md) | 已驗證 | `npm run test:web:coverage` |
| DP-011 | Add HPA Peak Browser Test | [doc/task/DP-011.md](task/DP-011.md) | 已驗證 | `npm run test:web:coverage` |
| DP-012 | Add Startup TCP Tests | [doc/task/DP-012.md](task/DP-012.md) | 已驗證 | `go test ./internal/startup` |
| DP-013 | Add Bearer Token Tests | [doc/task/DP-013.md](task/DP-013.md) | 已驗證 | `go test ./internal/auth` |
| DP-014 | Add API Command Config Parsing Tests | [doc/task/DP-014.md](task/DP-014.md) | 已驗證 | `go test ./cmd/api` |
| DP-015 | Add Scheduler Worker Config Parsing Tests | [doc/task/DP-015.md](task/DP-015.md) | 已驗證 | `go test ./cmd/scheduler-worker` |
| DP-016 | Add Healthcheck Process Tests | [doc/task/DP-016.md](task/DP-016.md) | 已驗證 | `go test ./cmd/healthcheck` |
| DP-017 | Expand Redis RESP Parser Unit Tests | [doc/task/DP-017.md](task/DP-017.md) | 已驗證 | `go test ./internal/api` |
| DP-018 | Add Redis Token Session Integration Tests | [doc/task/DP-018.md](task/DP-018.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 REDIS_ADDR=127.0.0.1:6379 go test ./internal/api` |
| DP-019 | Add Redis Lock Integration Tests | [doc/task/DP-019.md](task/DP-019.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 REDIS_ADDR=127.0.0.1:6379 go test ./internal/lock` |
| DP-020 | Add Scheduler Worker Payload And Lock Unit Tests | [doc/task/DP-020.md](task/DP-020.md) | 已驗證 | `go test ./cmd/scheduler-worker` |
| DP-021 | Add Scheduler Worker Job State Unit Tests | [doc/task/DP-021.md](task/DP-021.md) | 已驗證 | `go test ./cmd/scheduler-worker` |
| DP-022 | Add Scheduler Worker PostgreSQL Persistence Integration Tests | [doc/task/DP-022.md](task/DP-022.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./cmd/scheduler-worker` |
| DP-023 | Add PostgreSQL Store Construction And Auth Integration Tests | [doc/task/DP-023.md](task/DP-023.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-024 | Add PostgreSQL User Management Integration Tests | [doc/task/DP-024.md](task/DP-024.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-025 | Add PostgreSQL Order State Integration Tests | [doc/task/DP-025.md](task/DP-025.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-026 | Add PostgreSQL Schedule Job Lifecycle Integration Tests | [doc/task/DP-026.md](task/DP-026.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-027 | Add PostgreSQL Schedule Calendar Integration Tests | [doc/task/DP-027.md](task/DP-027.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-028 | Add PostgreSQL Production Integration Tests | [doc/task/DP-028.md](task/DP-028.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-029 | Add PostgreSQL HPA Demo Integration Tests | [doc/task/DP-029.md](task/DP-029.md) | 進行中 | `WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api` |
| DP-030 | Add Demo Conflict API Handler Tests | [doc/task/DP-030.md](task/DP-030.md) | 已驗證 | `go test ./internal/api` |
| DP-031 | Add HPA Peak API Handler Tests | [doc/task/DP-031.md](task/DP-031.md) | 已驗證 | `go test ./internal/api` |
| DP-032 | Add Kubernetes Autoscaling State Unit Tests | [doc/task/DP-032.md](task/DP-032.md) | 已驗證 | `go test ./internal/api` |
| DP-033 | Add User-By-Username Handler Tests | [doc/task/DP-033.md](task/DP-033.md) | 已驗證 | `go test ./internal/api` |
| DP-034 | Add Schedule Line Resolution Tests | [doc/task/DP-034.md](task/DP-034.md) | 已驗證 | `go test ./internal/api` |
| DP-035 | Add Production Helper Tests | [doc/task/DP-035.md](task/DP-035.md) | 已驗證 | `go test ./internal/api` |
| DP-036 | Add Manual Integration Test Script | [doc/task/DP-036.md](task/DP-036.md) | 已驗證 | `npm run test:integration` |
| DP-037 | Add Manual CI PostgreSQL Integration Workflow | [doc/task/DP-037.md](task/DP-037.md) | 已驗證 | `npm run test:coverage` |
| DP-038 | Add Manual CI Redis Integration Workflow | [doc/task/DP-038.md](task/DP-038.md) | 已驗證 | `npm run test:coverage` |
| DP-039 | Publish Fast Coverage Artifacts In CI | [doc/task/DP-039.md](task/DP-039.md) | 已驗證 | `npm run test:coverage` |
| DP-040 | Include Command Packages In Coverage Gate | [doc/task/DP-040.md](task/DP-040.md) | 已驗證 | `npm run test:go:coverage` |
| DP-041 | Add Short-Term Coverage Threshold | [doc/task/DP-041.md](task/DP-041.md) | 已驗證 | `npm run test:coverage` |
| DP-042 | Add Medium-Term Coverage Threshold | [doc/task/DP-042.md](task/DP-042.md) | 已驗證 | `npm run test:coverage` |
| DP-043 | Add Long-Term Release Coverage Threshold | [doc/task/DP-043.md](task/DP-043.md) | 已驗證 | `npm run test:coverage` |
| DP-044 | Update Verification Documentation For Integration Tests | [doc/task/DP-044.md](task/DP-044.md) | 已驗證 | `npm run test:coverage` |
| DP-045 | Run Final Full Verification For Any Implemented Item | [doc/task/DP-045.md](task/DP-045.md) | 未開始 | `npm run test:coverage` |

### 進度備註

- 2026-05-28 17:42 CST：DP-001 已指派給 Codex slave agent Schrodinger 並完成驗證。變更檔案：`web/ui.test.mjs`、`doc/task/DP-001.md`。驗證：`npm run test:web:coverage` 通過。備註：已加入直接測試 pure functions 的分支覆蓋，包含空查詢、無效日期/時區、日期邊界，以及 waterline capacity/threshold 行為。
