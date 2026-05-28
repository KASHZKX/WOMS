# Test Coverage Plan

Language: [English](#english) | [zh-TW](#zh-tw)

## English

### Coverage Baseline

Command run on 2026-05-28:

```sh
npm run test:coverage
```

Result:

- Go tests passed. Total Go statement coverage is `38.8%`.
- Web and Helm static tests passed. Node coverage is `96.90%` line coverage, `84.72%` branch coverage, and `96.84%` function coverage.
- The largest untested area is database-backed Go behavior, especially `internal/api/postgres_store.go`.
- The next largest gaps are Redis-backed session/lock code, scheduler-worker database job execution, HPA/KEDA demo behavior, command entrypoints, and a small number of web UI branches.
- `web/app.js` is not actually executed by the current Node coverage run. Existing web tests read it as text, so behavior in this 2,231-line browser module is effectively untested by coverage.

### Confirmed Decisions

- PostgreSQL and Redis integration tests should run only in manual CI, not on every pull request.
- Docker Compose should not be the standard local fixture runner for integration tests.
- Command `main` functions should count toward required coverage.

### Priority 1: PostgreSQL Store Integration Tests

Target files:

- `internal/api/postgres_store.go`
- `db/migrations/*.sql`

Why:

Most functions in `internal/api/postgres_store.go` are currently `0.0%` covered. These functions own persistence, migrations, order state transitions, audit logs, schedule jobs, schedule calendar data, production confirmation, and HPA peak demo data. Unit tests around the in-memory store do not prove the real deployment path works.

Plan:

- Add integration tests gated by an environment variable such as `WOMS_INTEGRATION_TESTS=1`.
- Start tests against a real PostgreSQL instance in manual CI. Do not require Docker Compose as the standard local runner.
- For local development, allow tests to connect to a developer-provided `DATABASE_URL` when `WOMS_INTEGRATION_TESTS=1` is set.
- Create a fresh test database per test package or per test run.
- Run migrations through `NewPostgresStoreContext`.
- Seed only the records needed for each test; avoid depending on broad demo seed data except in explicit demo tests.
- Verify all zh-TW status values stay UTF-8 and valid: `待排程`, `已排程`, `生產中`, `已完成`, `需業務處理`.

Test cases:

- Store construction rejects empty `DATABASE_URL` and closes the DB on migration or ping failure.
- `Authenticate` accepts bcrypt and legacy SHA-256 password hashes and rejects disabled users.
- `CreateUser`, `AssignUser`, `ResetUserPassword`, and `DeleteUser` persist expected user records and audit logs.
- `CreateOrder`, `UpdateOrderDueDate`, `RejectOrders`, `ResubmitOrder`, and `CancelOrders` enforce role, ownership, line, status, due-date, and quantity rules.
- High-priority manual intervention paths create audit logs.
- `CreateScheduleJob`, `DeleteQueuedScheduleJob`, `ExecuteScheduleJob`, and `GetScheduleJob` correctly move jobs through queued, cancelled, running, failed, and completed states.
- `PreviewSchedule`, `ConfirmPreviewOrder`, `ScheduleCalendar`, and `ScheduleHistory` persist preview allocations, split or move orders correctly, and expose deterministic calendar/history output.
- `StartProduction` and `ConfirmProduction` enforce production state transitions and produce completed allocations.
- `CreateHPAPeakDemo`, `ClearHPAPeakDemo`, `HPAPeakSummary`, and `HPAPeakJobs` create, reset, and summarize demo load correctly.

### Priority 2: Redis Session And Lock Tests

Target files:

- `internal/api/token_session.go`
- `internal/lock/redis.go`

Why:

The Redis-backed token session store and Redis lock provider are mostly untested. These are deployment-critical for JWT session revocation and deterministic scheduler-worker line locking.

Plan:

- Keep existing RESP parser unit tests and expand them for error, integer, nil bulk, malformed length, partial read, and unsupported prefix cases.
- Add integration tests gated by `WOMS_INTEGRATION_TESTS=1` and `REDIS_ADDR`.
- Use real Redis in manual CI for behavior that depends on TTL, `SET NX PX`, Lua `EVAL`, and connection reuse.
- For local development, allow tests to connect to a developer-provided `REDIS_ADDR` when `WOMS_INTEGRATION_TESTS=1` is set.

Test cases:

- `RedisTokenSessionStore.Ping`, `Save`, `Verify`, `Revoke`, `TracksSessions`, and `Close`.
- Expired or empty token save/verify failures.
- Redis connection reconnects after command errors.
- `RedisProvider.Ping`, `Acquire`, `Refresh`, and `Release`.
- Lock acquisition fails when the key is already locked and succeeds after release or expiry.
- Refresh fails after ownership value mismatch.
- Empty `REDIS_ADDR` returns a clear error.

### Priority 3: Scheduler Worker Database Job Tests

Target file:

- `cmd/scheduler-worker/main.go`

Why:

`processDBJob`, `processDBJobLocked`, `startLockRenewal`, retry/fail helpers, allocation persistence, preview allocation persistence, and backfill are currently `0.0%` covered. This is a high-risk path because it joins Kafka payloads, PostgreSQL jobs, Redis locks, deterministic scheduling, retries, and audit logs.

Plan:

- Prefer function-level tests with PostgreSQL integration fixtures and fake lock providers.
- Keep Kafka itself out of these tests where possible; test Kafka reader behavior separately only if needed.
- Use table-driven tests for job payload, status, source, retry count, and stale schedule data.

Test cases:

- Invalid JSON and missing job ID/line ID are handled safely.
- Missing lock provider marks the job failed with the expected zh-TW message.
- Lock acquisition timeout marks the job retryable.
- Cancelled, running, completed, and missing jobs are not re-executed.
- Queued job becomes running, increments attempt count, and then becomes completed.
- Persisting a full line schedule replaces open allocations and writes audit records.
- Persisting preview allocations respects line revision and stale data checks.
- Retry count below max retries marks retry; retry count at max marks failed.
- `startLockRenewal` cancels work when lock refresh fails.
- `backfillQueuedJobs` picks eligible queued jobs without duplicating locked line work.

### Priority 4: API Server Gaps

Target file:

- `internal/api/server.go`

Why:

The API server already has useful coverage, but several request handlers and helper branches remain uncovered or weak: demo conflict orders, HPA peak publishing and Kubernetes state loading, user-by-username edge cases, schedule line resolution, completed production allocation, and order ID generation.

Plan:

- Add focused `httptest` request tests using the memory store for handler routing, auth, RBAC, JSON validation, and response shape.
- Add direct store tests for memory-only behavior that does not require PostgreSQL.
- Stub HPA/Kubernetes HTTP calls with `httptest.Server`; do not depend on a real cluster for unit tests.

Test cases:

- `handleDemoConflictOrders` and `CreateDemoConflictOrders` success, unauthorized role, and invalid method cases.
- `handleHPAPeakDemo`, `publishHPAPeakJobs`, `CreateHPAPeakDemo`, `HPAPeakJobs`, reset, and clear behavior.
- `loadHPAAutoscalingState` and `kubernetesGetJSON` for success, HTTP errors, invalid JSON, missing env vars, and timeout/error paths.
- `handleUserByUsername` for scheduler line assignment, deletion, not found, and forbidden role cases.
- `scheduleLineID`, `hpaDemoLineID`, and `scheduleRequestLineID` for default line, explicit line, HPA demo line, scheduler claims, sales claims, and invalid line IDs.
- `completeProductionAllocationLocked` and `orderIDFromTime` direct helper coverage.

### Priority 5: Command Entrypoint And Config Tests

Target files:

- `cmd/api/main.go`
- `cmd/healthcheck/main.go`
- `cmd/scheduler-worker/main.go`

Why:

Main functions are currently `0.0%` covered. Full process tests are more expensive than unit tests, but config parsing and healthcheck behavior can be covered cheaply.

Plan:

- Extract pure config parsing from `main` into small functions that can be tested without starting servers.
- For `cmd/healthcheck`, add process-level tests with `os/exec` or refactor request execution into a testable function.
- Keep long-running server and Kafka loops out of normal unit tests.
- Count command `main` packages in coverage gates. If a `main` function cannot be directly unit-tested without starting a long-running process, move its decisions into testable helpers and cover those helpers plus short process-level exit behavior.

Test cases:

- Environment fallback and trimming behavior.
- Duration and integer parsing for valid, empty, negative, and malformed values.
- Worker start offset labels: `earliest`, `latest`, aliases, empty, and invalid values.
- Healthcheck exits successfully on 2xx and fails on invalid timeout, invalid URL, request error, and non-2xx status.

### Priority 6: Startup And Auth Small Gaps

Target files:

- `internal/startup/startup.go`
- `internal/auth/jwt.go`

Why:

These modules are small but deployment-facing. `PingAnyTCP`, `pingTCP`, and `BearerToken` are currently uncovered.

Plan:

- Test `PingAnyTCP` with a local TCP listener and unreachable addresses.
- Test `BearerToken` with empty, malformed, lowercase, valid, and extra-space authorization headers.

Test cases:

- Empty address list returns `no addresses configured`.
- Multiple addresses succeed if any address is reachable.
- All unreachable addresses return a joined error that includes failed addresses.
- TCP connection closes cleanly after ping.

### Priority 7: Web UI And `app.js` Browser Module Tests

Target files:

- `web/ui.js`
- `web/app.js`
- `web/index.html`

Why:

`web/ui.js` coverage is already strong, but `web/app.js` is not executed in coverage at all. It is a large side-effect browser module that wires forms, buttons, dialogs, `fetch`, `localStorage`, timers, drag/drop, role-based rendering, schedule preview, production reporting, rejection flows, HPA polling, and calendar interactions. Text assertions can catch copy regressions, but they do not prove the app behavior works.

Plan:

- Add targeted tests in `web/ui.test.mjs`.
- Keep tests pure and DOM-light unless behavior requires rendering.
- Add a new `web/app.test.mjs` that executes `web/app.js` inside a browser-like DOM fixture.
- Prefer a real DOM test harness such as `jsdom` as a dev dependency. If no dependency is allowed, create a minimal custom DOM fixture only for smoke tests, but that will cover less behavior.
- Build the DOM fixture from `web/index.html` so required element IDs, forms, dialogs, and buttons match production markup.
- Mock `window.fetch`, `localStorage`, `window.confirm`, `HTMLDialogElement.showModal/close`, `setInterval`, `clearInterval`, `setTimeout`, `FormData`, `DataTransfer`, and `document.elementFromPoint` where needed.
- Import `web/app.js` dynamically after installing the DOM globals, then drive behavior through real DOM events instead of calling private functions directly.
- Add a small optional refactor only if tests become too brittle: export a factory such as `createAppController({ document, window, fetch, storage, clock })` while keeping the current browser entrypoint behavior unchanged.

Test cases:

- `matchesOrder` returns `true` for an empty query.
- `waterlineMetrics` handles zero or negative capacity.
- `dateKeyInTimeZone` returns an empty string for invalid dates and falls back to ISO date for invalid time zones.
- `tomorrowDateKey` handles month and year boundaries.
- `waterlineColor` and `waterlineTone` cover safe, warning, danger, and clamped values.
- Anonymous startup renders the login page, hides the app shell, uses fallback lines, and initializes start/due dates.
- Successful login sends `POST /api/auth/login`, saves `woms.token` and `woms.user`, renders the correct role UI, and loads lines, orders, calendar, and schedule history.
- Logout sends `POST /api/auth/logout`, clears session storage, resets orders/calendar/preview state, and returns to login UI.
- A 401 response clears the session and shows the expired-login message.
- Sales order submission validates future due dates, calls `POST /api/schedules/preview`, opens the preview dialog, and confirms draft orders through `POST /api/orders/preview-confirm`.
- Scheduler selection, status/customer/priority filters, selected count, preview selected, reject selected, cancel selected, and schedule form submit work through DOM clicks/submits.
- Calendar mode toggles show scheduled, pending, and all allocations for sales users.
- Drag/drop and pointer fallback scheduling call the preview endpoint with selected order IDs and the target date.
- Conflict preview actions cover retry-today, retry-suggested-start, update-conflict-due-date, unselect-conflict-order, reject-preview-orders, preview-conflict-solution, and retry-manual-force validation.
- Production start and production confirmation validate quantities, open and close the production dialog, call `POST /api/production/start` and `POST /api/production/confirm`, and show remainder copy.
- Admin user management forms call create, assign, reset password, and delete user endpoints and refresh user lists.
- HPA peak actions call create, refresh, and clear endpoints, render autoscaling metadata, start polling only for active admin summaries, avoid overlapping polling requests, and stop polling when inactive.

### Priority 8: Coverage Workflow Improvements

Target files:

- `scripts/go-coverage.sh`
- `package.json`
- CI workflow files, if coverage is added to CI gates.

Plan:

- Keep `npm run test:coverage` as the main local command.
- Add an optional integration command, for example `npm run test:integration`, only after PostgreSQL and Redis manual CI services are defined.
- Do not require Docker Compose for local integration fixtures.
- Include command `main` packages in strict coverage gates after config/runtime helpers and healthcheck process tests are added.
- Update `test:web:coverage` so it executes `web/app.test.mjs`; otherwise `web/app.js` will remain invisible to coverage.
- Publish `coverage.out` as a CI artifact.
- Add a minimum coverage threshold only after Priority 1 and Priority 2 tests are in place.

Suggested gates after implementing this plan:

- Short term: Go total coverage at least `55%`; web line coverage at least `95%`.
- Medium term: Go total coverage at least `70%`; `internal/scheduler`, `internal/auth`, API handler tests, Redis lock/session tests, and PostgreSQL store integration tests required.
- Long term: Go total coverage at least `80%`, with manual CI integration tests required before release branches.

### Implementation Order

1. Add web UI branch tests because they are low risk and fast.
2. Add `web/app.js` DOM execution tests so the main frontend workflow is covered by behavior tests.
3. Add startup/auth small-gap tests.
4. Add command config parsing tests after small refactors, and keep command packages included in coverage.
5. Add Redis parser tests, then Redis integration tests in manual CI.
6. Add scheduler-worker tests with fake locks and PostgreSQL fixtures.
7. Add PostgreSQL store integration tests in manual CI.
8. Add HPA/Kubernetes API server tests using HTTP stubs.
9. Add CI artifacts and coverage thresholds after the suite is stable.

### Verification Commands

Run fast tests:

```sh
npm run test:coverage
```

Run Go tests only:

```sh
npm run test:go:coverage
```

Run web and Helm tests only:

```sh
npm run test:web:coverage
```

Run future manual CI integration tests only after PostgreSQL and Redis services are available:

```sh
WOMS_INTEGRATION_TESTS=1 npm run test:integration
```

### Implementation Answers

- PostgreSQL and Redis integration tests run only in manual CI.
- Docker Compose is not the standard local fixture runner.
- Command `main` functions count toward required coverage.
