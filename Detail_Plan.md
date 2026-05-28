# Detailed Test Coverage Plan

Language: [English](#english) | [zh-TW](#zh-tw)

Source: `Plan.md`

## English

### Atomicity Rule

Each item below is an independent modification. A developer can implement any item without first implementing another item in this file. If a change benefits from a shared helper, the helper must be introduced inside that same item or the item must remain locally self-contained.

Each item must include:

- One narrow behavior target.
- One narrow file or package boundary.
- Its own tests or verification command.
- Its own README and `.gitignore` check when the behavior, commands, generated files, or local artifacts change.

### DP-001: Add `web/ui.js` Pure Function Branch Tests

Target files:

- `web/ui.js`
- `web/ui.test.mjs`

Modification:

- Add focused tests for exported pure functions only.
- Cover empty query matching, invalid dates, invalid time zones, waterline capacity edge cases, waterline color/tone thresholds, and tomorrow date boundaries.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- The tests execute `web/ui.js` behavior directly.
- No browser app bootstrapping is required.
- Existing UI exports keep the same public names and behavior.

### DP-002: Add `web/app.js` Anonymous Startup Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`
- `package.json`, only if a DOM test dependency or command change is required.

Modification:

- Add a browser-like DOM fixture that loads `web/index.html`.
- Dynamically import `web/app.js` after installing DOM globals.
- Test anonymous startup behavior: login page visible, app shell hidden, fallback lines available, and start/due dates initialized.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- `web/app.js` is actually executed by the Node coverage run.
- The test does not call private functions directly.
- If a dependency such as `jsdom` is added, `package.json`, lockfile state, README test instructions, and `.gitignore` are checked in the same item.

### DP-003: Add `web/app.js` Login And Logout Session Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test successful login through DOM form submission.
- Mock `fetch` responses for login, lines, orders, calendar, and schedule history.
- Verify `woms.token` and `woms.user` are saved.
- Test logout through the UI and verify session storage and app state are cleared.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Login uses `POST /api/auth/login`.
- Logout uses `POST /api/auth/logout`.
- The test verifies visible UI state after both actions.

### DP-004: Add `web/app.js` Expired Session Test

Target files:

- `web/app.js`
- `web/app.test.mjs`

Modification:

- Mock an authenticated API call that returns `401`.
- Verify the app clears stored session data and returns to the login state.
- Verify the expired-login message is rendered.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- The test is independent from the login/logout test and creates its own stored session fixture.
- No real network calls are made.

### DP-005: Add Sales Preview And Draft Confirmation Browser Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test sales order submission with future due-date validation.
- Mock `POST /api/schedules/preview`.
- Verify preview dialog rendering.
- Confirm draft orders with `POST /api/orders/preview-confirm`.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Invalid due dates are rejected before the preview request.
- Valid submissions use the preview endpoint.
- Draft confirmation refreshes affected UI data.

### DP-006: Add Scheduler Selection And Bulk Action Browser Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test scheduler order filtering by status, customer, and priority.
- Test selected count updates.
- Test preview selected, reject selected, cancel selected, and schedule form submission through DOM events.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Each action verifies the expected API endpoint and payload.
- The test owns its own mock order data.

### DP-007: Add Calendar Mode And Drag Preview Browser Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test sales calendar modes for scheduled, pending, and all allocations.
- Test drag/drop or pointer fallback scheduling into a target date.
- Mock `document.elementFromPoint`, `DataTransfer`, and schedule preview requests.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Calendar mode changes are visible in DOM output.
- Drop behavior calls the preview endpoint with selected order IDs and target date.

### DP-008: Add Conflict Preview Browser Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test conflict preview actions: retry today, retry suggested start, update conflict due date, unselect conflict order, reject preview orders, preview conflict solution, and manual force validation.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Manual force rejects invalid inputs.
- Each conflict action verifies the expected request or UI state transition.

### DP-009: Add Production Flow Browser Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test production start and production confirmation.
- Validate quantity inputs.
- Verify production dialog open and close behavior.
- Mock `POST /api/production/start` and `POST /api/production/confirm`.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Remainder copy is rendered after confirmation.
- Invalid quantities do not call the API.

### DP-010: Add Admin User Management Browser Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test admin create user, assign line, reset password, and delete user form flows.
- Verify user list refresh after each successful operation.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Each form calls the correct endpoint and method.
- The test does not require prior login tests; it creates its own admin session fixture.

### DP-011: Add HPA Peak Browser Test

Target files:

- `web/app.js`
- `web/app.test.mjs`
- `web/index.html`

Modification:

- Test HPA peak create, refresh, and clear actions.
- Render autoscaling metadata.
- Verify polling starts only for active admin summaries.
- Verify polling avoids overlapping requests and stops when inactive.

Verification:

```sh
npm run test:web:coverage
```

Acceptance criteria:

- Timers are mocked.
- No Kubernetes cluster is required.

### DP-012: Add Startup TCP Tests

Target files:

- `internal/startup/startup.go`
- `internal/startup/startup_test.go`

Modification:

- Test `PingAnyTCP` with an empty address list, a local TCP listener, mixed reachable/unreachable addresses, and all unreachable addresses.
- Test that TCP connections close cleanly.

Verification:

```sh
go test ./internal/startup
```

Acceptance criteria:

- The joined error includes failed addresses.
- Tests use local listeners only.

### DP-013: Add Bearer Token Tests

Target files:

- `internal/auth/jwt.go`
- `internal/auth/jwt_test.go`

Modification:

- Test `BearerToken` with empty, malformed, lowercase, valid, extra-space, and non-Bearer authorization headers.

Verification:

```sh
go test ./internal/auth
```

Acceptance criteria:

- Existing JWT signing and validation behavior is unchanged.
- The tests cover both accepted and rejected header formats.

### DP-014: Add API Command Config Parsing Tests

Target files:

- `cmd/api/main.go`
- `cmd/api/main_test.go`

Modification:

- Extract API environment parsing into a pure helper if needed.
- Test default values, trimming, duration parsing, integer parsing, malformed values, and negative values.

Verification:

```sh
go test ./cmd/api
```

Acceptance criteria:

- Tests do not start the HTTP server.
- `main` still delegates to the same runtime behavior.

### DP-015: Add Scheduler Worker Config Parsing Tests

Target files:

- `cmd/scheduler-worker/main.go`
- `cmd/scheduler-worker/main_test.go`

Modification:

- Extract worker configuration parsing into a pure helper if needed.
- Test Kafka start offset labels: `earliest`, `latest`, aliases, empty, and invalid values.
- Test worker duration and integer environment parsing.

Verification:

```sh
go test ./cmd/scheduler-worker
```

Acceptance criteria:

- Tests do not start Kafka consumers or long-running worker loops.
- Invalid config returns clear errors.

### DP-016: Add Healthcheck Process Tests

Target files:

- `cmd/healthcheck/main.go`
- `cmd/healthcheck/main_test.go`

Modification:

- Refactor request execution into a testable helper if needed.
- Test success on 2xx.
- Test invalid timeout, invalid URL, request error, and non-2xx status.

Verification:

```sh
go test ./cmd/healthcheck
```

Acceptance criteria:

- Tests use `httptest.Server`.
- Process exit behavior or helper result behavior is covered without requiring an external service.

### DP-017: Expand Redis RESP Parser Unit Tests

Target files:

- `internal/api/token_session.go`
- Existing or new Redis parser test file under `internal/api`

Modification:

- Add unit tests for Redis protocol error replies, integer replies, nil bulk replies, malformed lengths, partial reads, and unsupported prefixes.

Verification:

```sh
go test ./internal/api
```

Acceptance criteria:

- No Redis server is required.
- Parser behavior is deterministic and table-driven.

### DP-018: Add Redis Token Session Integration Tests

Target files:

- `internal/api/token_session.go`
- New integration test file under `internal/api`
- Manual CI workflow file, only if this item introduces the Redis service job.
- README files, only if a new integration command or environment variable is documented.

Modification:

- Add tests gated by `WOMS_INTEGRATION_TESTS=1` and `REDIS_ADDR`.
- Test `Ping`, `Save`, `Verify`, `Revoke`, `TracksSessions`, `Close`, expired token behavior, empty token behavior, and reconnect after command errors.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 REDIS_ADDR=127.0.0.1:6379 go test ./internal/api
```

Acceptance criteria:

- Tests skip clearly when integration variables are absent.
- The item does not require PostgreSQL.

### DP-019: Add Redis Lock Integration Tests

Target files:

- `internal/lock/redis.go`
- New integration test file under `internal/lock`
- Manual CI workflow file, only if this item introduces the Redis service job.
- README files, only if a new integration command or environment variable is documented.

Modification:

- Add tests gated by `WOMS_INTEGRATION_TESTS=1` and `REDIS_ADDR`.
- Test `Ping`, `Acquire`, `Refresh`, `Release`, locked-key failure, success after release, success after expiry, owner mismatch, and empty Redis address errors.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 REDIS_ADDR=127.0.0.1:6379 go test ./internal/lock
```

Acceptance criteria:

- Tests use real Redis TTL, `SET NX PX`, and Lua `EVAL`.
- The item does not require PostgreSQL.

### DP-020: Add Scheduler Worker Payload And Lock Unit Tests

Target files:

- `cmd/scheduler-worker/main.go`
- `cmd/scheduler-worker/main_test.go`

Modification:

- Test invalid JSON, missing job ID, missing line ID, missing lock provider, lock acquisition timeout, and lock renewal failure.
- Use fake lock providers and fake store behavior.

Verification:

```sh
go test ./cmd/scheduler-worker
```

Acceptance criteria:

- No PostgreSQL, Redis, or Kafka service is required.
- Retryable and failed job states are asserted explicitly.

### DP-021: Add Scheduler Worker Job State Unit Tests

Target files:

- `cmd/scheduler-worker/main.go`
- `cmd/scheduler-worker/main_test.go`

Modification:

- Test that cancelled, running, completed, and missing jobs are not re-executed.
- Test queued jobs become running, increment attempt count, and complete on success.
- Use fake store behavior.

Verification:

```sh
go test ./cmd/scheduler-worker
```

Acceptance criteria:

- The test is independent from PostgreSQL fixtures.
- State transitions are asserted in the fake store.

### DP-022: Add Scheduler Worker PostgreSQL Persistence Integration Tests

Target files:

- `cmd/scheduler-worker/main.go`
- New integration test file under `cmd/scheduler-worker`
- Manual CI workflow file, only if this item introduces the PostgreSQL service job.
- README files, only if a new integration command or environment variable is documented.

Modification:

- Add PostgreSQL-backed tests gated by `WOMS_INTEGRATION_TESTS=1` and `DATABASE_URL`.
- Test full line schedule persistence, open allocation replacement, audit writes, preview allocation persistence, stale line revision checks, retry below max, retry at max, and queued-job backfill.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./cmd/scheduler-worker
```

Acceptance criteria:

- Kafka is not required.
- Redis may be replaced by a fake lock provider in this item.

### DP-023: Add PostgreSQL Store Construction And Auth Integration Tests

Target files:

- `internal/api/postgres_store.go`
- New integration test file under `internal/api`
- `db/migrations/*.sql`
- Manual CI workflow file, only if this item introduces the PostgreSQL service job.

Modification:

- Add tests gated by `WOMS_INTEGRATION_TESTS=1` and `DATABASE_URL`.
- Test empty `DATABASE_URL`, ping failure cleanup, migration failure cleanup, bcrypt auth, legacy SHA-256 auth, disabled user rejection, and UTF-8 status constraints.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api
```

Acceptance criteria:

- Tests create isolated database state.
- zh-TW values remain `待排程`, `已排程`, `生產中`, `已完成`, and `需業務處理`.

### DP-024: Add PostgreSQL User Management Integration Tests

Target files:

- `internal/api/postgres_store.go`
- New integration test file under `internal/api`

Modification:

- Test `CreateUser`, `AssignUser`, `ResetUserPassword`, and `DeleteUser`.
- Assert persisted users, role/line changes, password hash updates, disabled/deleted behavior, and audit logs.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api
```

Acceptance criteria:

- The item does not test order or schedule behavior.
- Each test seeds only required users and lines.

### DP-025: Add PostgreSQL Order State Integration Tests

Target files:

- `internal/api/postgres_store.go`
- New integration test file under `internal/api`

Modification:

- Test `CreateOrder`, `UpdateOrderDueDate`, `RejectOrders`, `ResubmitOrder`, and `CancelOrders`.
- Assert role, ownership, line, status, due-date, quantity, and high-priority manual intervention audit behavior.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api
```

Acceptance criteria:

- The item does not test schedule job execution.
- High-priority manual force paths must create audit logs.

### DP-026: Add PostgreSQL Schedule Job Lifecycle Integration Tests

Target files:

- `internal/api/postgres_store.go`
- New integration test file under `internal/api`

Modification:

- Test `CreateScheduleJob`, `DeleteQueuedScheduleJob`, `ExecuteScheduleJob`, and `GetScheduleJob`.
- Assert queued, cancelled, running, failed, completed, retry, attempt count, and stale job handling.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api
```

Acceptance criteria:

- The item does not test calendar rendering.
- State transitions are deterministic.

### DP-027: Add PostgreSQL Schedule Calendar Integration Tests

Target files:

- `internal/api/postgres_store.go`
- New integration test file under `internal/api`

Modification:

- Test `PreviewSchedule`, `ConfirmPreviewOrder`, `ScheduleCalendar`, and `ScheduleHistory`.
- Assert preview allocation persistence, order splitting or movement, deterministic calendar output, and deterministic history output.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api
```

Acceptance criteria:

- Tests do not depend on broad demo seed data.
- Same input produces same output order.

### DP-028: Add PostgreSQL Production Integration Tests

Target files:

- `internal/api/postgres_store.go`
- New integration test file under `internal/api`

Modification:

- Test `StartProduction` and `ConfirmProduction`.
- Assert valid state transitions, invalid transition rejection, quantity handling, and completed allocation persistence.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api
```

Acceptance criteria:

- The item does not test HPA demo behavior.
- Completed allocations are queryable after confirmation.

### DP-029: Add PostgreSQL HPA Demo Integration Tests

Target files:

- `internal/api/postgres_store.go`
- New integration test file under `internal/api`

Modification:

- Test `CreateHPAPeakDemo`, `ClearHPAPeakDemo`, `HPAPeakSummary`, and `HPAPeakJobs`.
- Assert creation, reset, summary counts, job rows, and clear behavior.

Verification:

```sh
WOMS_INTEGRATION_TESTS=1 DATABASE_URL=postgres://... go test ./internal/api
```

Acceptance criteria:

- The item does not require Kubernetes or Redis.
- Demo data is isolated from normal order fixtures.

### DP-030: Add Demo Conflict API Handler Tests

Target files:

- `internal/api/server.go`
- Existing or new server test file under `internal/api`

Modification:

- Add `httptest` coverage for `handleDemoConflictOrders` and memory-store demo conflict creation.
- Test success, unauthorized role, invalid method, JSON response shape, and audit-sensitive behavior if present.

Verification:

```sh
go test ./internal/api
```

Acceptance criteria:

- Tests use the memory store.
- No PostgreSQL service is required.

### DP-031: Add HPA Peak API Handler Tests

Target files:

- `internal/api/server.go`
- Existing or new server test file under `internal/api`

Modification:

- Add `httptest` coverage for HPA peak create, publish, jobs, summary, reset, and clear behavior.

Verification:

```sh
go test ./internal/api
```

Acceptance criteria:

- Tests use memory-store behavior.
- Kafka publishing is stubbed or isolated.

### DP-032: Add Kubernetes Autoscaling State Unit Tests

Target files:

- `internal/api/server.go`
- Existing or new server test file under `internal/api`

Modification:

- Test `loadHPAAutoscalingState` and `kubernetesGetJSON`.
- Cover success, HTTP errors, invalid JSON, missing environment variables, request timeout, and transport errors.

Verification:

```sh
go test ./internal/api
```

Acceptance criteria:

- Tests use `httptest.Server`.
- No Kubernetes cluster is required.

### DP-033: Add User-By-Username Handler Tests

Target files:

- `internal/api/server.go`
- Existing or new server test file under `internal/api`

Modification:

- Test scheduler line assignment, deletion, user not found, forbidden role, invalid method, and response shape for `handleUserByUsername`.

Verification:

```sh
go test ./internal/api
```

Acceptance criteria:

- Tests use memory-store fixtures.
- Each role scenario is isolated.

### DP-034: Add Schedule Line Resolution Tests

Target files:

- `internal/api/server.go`
- Existing or new server test file under `internal/api`

Modification:

- Test `scheduleLineID`, `hpaDemoLineID`, and `scheduleRequestLineID`.
- Cover default line, explicit line, HPA demo line, scheduler claims, sales claims, and invalid line IDs.

Verification:

```sh
go test ./internal/api
```

Acceptance criteria:

- Tests call helper behavior directly or through minimal request handlers.
- No database service is required.

### DP-035: Add Production Helper Tests

Target files:

- `internal/api/server.go`
- Existing or new server test file under `internal/api`

Modification:

- Test `completeProductionAllocationLocked` and `orderIDFromTime`.
- Cover normal completion, zero or invalid quantity behavior if supported, deterministic ID formatting, and collision-sensitive time values.

Verification:

```sh
go test ./internal/api
```

Acceptance criteria:

- Tests use memory-store state.
- Time-dependent values are fixed in test fixtures.

### DP-036: Add Manual Integration Test Script

Target files:

- `package.json`
- `scripts/`, if a wrapper script is needed.
- `README.md`
- `README.en.md`
- `README.zh-TW.md`
- `.gitignore`, only if the command creates local artifacts.

Modification:

- Add a manual integration command such as `npm run test:integration`.
- The command must require `WOMS_INTEGRATION_TESTS=1`.
- The command must not start Docker Compose as the default local fixture runner.

Verification:

```sh
npm run test:integration
```

Acceptance criteria:

- Missing `DATABASE_URL` or `REDIS_ADDR` causes clear skipped tests or clear setup errors.
- README documents the manual nature of the command in English and zh-TW.

### DP-037: Add Manual CI PostgreSQL Integration Workflow

Target files:

- `.github/workflows/*.yml`
- README files, only if workflow usage is documented.

Modification:

- Add or update a manual workflow dispatch job that starts PostgreSQL.
- Run only PostgreSQL-gated integration tests.
- Upload Go coverage artifacts from the integration run.

Verification:

```sh
gh workflow view
```

Acceptance criteria:

- The workflow does not run on every pull request by default.
- Docker Hub publishing is not triggered by this workflow.

### DP-038: Add Manual CI Redis Integration Workflow

Target files:

- `.github/workflows/*.yml`
- README files, only if workflow usage is documented.

Modification:

- Add or update a manual workflow dispatch job that starts Redis.
- Run only Redis-gated integration tests.
- Upload Go coverage artifacts from the integration run.

Verification:

```sh
gh workflow view
```

Acceptance criteria:

- The workflow does not run on every pull request by default.
- Docker Hub publishing is not triggered by this workflow.

### DP-039: Publish Fast Coverage Artifacts In CI

Target files:

- `.github/workflows/ci.yml`

Modification:

- Upload `coverage.out` and web coverage output from the existing fast CI jobs.

Verification:

```sh
npm run test:coverage
```

Acceptance criteria:

- Existing PR test behavior remains fast.
- Artifact upload does not change coverage thresholds.

### DP-040: Include Command Packages In Coverage Gate

Target files:

- `scripts/go-coverage.sh`
- README files, only if command output or coverage policy text changes.

Modification:

- Ensure command packages remain included in `go test -coverprofile=coverage.out -covermode=atomic ./...`.
- If filtering is added later, explicitly keep `cmd/api`, `cmd/healthcheck`, and `cmd/scheduler-worker` in scope.

Verification:

```sh
npm run test:go:coverage
```

Acceptance criteria:

- Coverage output includes command packages.
- Long-running `main` loops are not started by the coverage command.

### DP-041: Add Short-Term Coverage Threshold

Target files:

- `scripts/go-coverage.sh`
- `package.json`, only if a separate threshold command is introduced.
- README files, if coverage policy is documented.

Modification:

- Add a short-term threshold only when the current suite can pass it consistently.
- Suggested threshold from `Plan.md`: Go total coverage at least `55%`; web line coverage at least `95%`.

Verification:

```sh
npm run test:coverage
```

Acceptance criteria:

- Threshold failures print the actual coverage and required coverage.
- The threshold item does not add new tests.

### DP-042: Add Medium-Term Coverage Threshold

Target files:

- `scripts/go-coverage.sh`
- `package.json`, only if a separate threshold command is introduced.
- README files, if coverage policy is documented.

Modification:

- Add the medium-term threshold only when relevant tests are already present in the branch where this item is implemented.
- Suggested threshold from `Plan.md`: Go total coverage at least `70%`.

Verification:

```sh
npm run test:coverage
```

Acceptance criteria:

- The gate requires API handler, Redis, PostgreSQL integration, scheduler, and auth coverage according to the policy in that branch.
- The threshold item does not add new tests.

### DP-043: Add Long-Term Release Coverage Threshold

Target files:

- Release workflow files.
- README files, if release policy is documented.

Modification:

- Add the long-term release threshold only to release validation.
- Suggested threshold from `Plan.md`: Go total coverage at least `80%`, with manual CI integration tests required before release branches.

Verification:

```sh
npm run test:coverage
```

Acceptance criteria:

- Pull request fast CI is not unexpectedly blocked by release-only requirements.
- Release validation clearly reports missing manual integration runs.

### DP-044: Update Verification Documentation For Integration Tests

Target files:

- `docs/verification.en.md`
- `docs/verification.zh-TW.md`

Modification:

- Document how to run fast coverage and manual integration coverage.
- State that PostgreSQL and Redis integration tests are manual CI or developer-provided-service tests.
- State that Docker Compose is not the standard local integration fixture runner.

Verification:

```sh
npm run test:coverage
```

Acceptance criteria:

- English and zh-TW verification docs stay aligned.
- No command is documented unless it exists in the same branch.

### DP-045: Run Final Full Verification For Any Implemented Item

Target files:

- No source file target.

Modification:

- Run the available test command for the implemented atomic item.
- Run the full fast suite before opening a PR when practical.

Verification:

```sh
npm run test:coverage
```

Acceptance criteria:

- The final PR message lists the atomic items implemented.
- Generated coverage files, cache files, local volumes, `.env`, and IDE-private files are not committed.
