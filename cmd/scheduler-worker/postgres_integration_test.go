package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/d11nn/woms/internal/domain"
	womslock "github.com/d11nn/woms/internal/lock"
	"github.com/d11nn/woms/internal/scheduler"
	_ "github.com/lib/pq"
)

func requireWorkerPostgresIntegration(t *testing.T) string {
	t.Helper()
	if os.Getenv("WOMS_INTEGRATION_TESTS") != "1" {
		t.Skip("set WOMS_INTEGRATION_TESTS=1 and DATABASE_URL to run PostgreSQL integration tests")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}
	chdirWorkerRepoRoot(t)
	return databaseURL
}

func chdirWorkerRepoRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "db", "migrations", "001_init.sql")); err == nil {
			if err := os.Chdir(dir); err != nil {
				t.Fatalf("chdir repo root: %v", err)
			}
			t.Cleanup(func() {
				_ = os.Chdir(wd)
			})
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", wd)
		}
		dir = parent
	}
}

func newWorkerIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := requireWorkerPostgresIntegration(t)
	schema := fmt.Sprintf("woms_worker_it_%d", time.Now().UnixNano())
	adminDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + workerPQIdentifier(schema)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + workerPQIdentifier(schema) + ` CASCADE`)
		_ = adminDB.Close()
	})

	db, err := sql.Open("postgres", workerDatabaseURLWithSearchPath(t, databaseURL, schema))
	if err != nil {
		t.Fatalf("open schema database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	schemaSQL, err := os.ReadFile("db/migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(schemaSQL)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	return db
}

func workerDatabaseURLWithSearchPath(t *testing.T, databaseURL, schema string) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	query := parsed.Query()
	query.Set("options", "-c search_path="+schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func workerPQIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func seedWorkerUser(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if _, err := db.Exec("INSERT INTO users (id, username, password_hash, role) VALUES ($1, $2, 'demo', 'scheduler')", id, id); err != nil {
		t.Fatalf("insert worker user: %v", err)
	}
}

func seedWorkerOrder(t *testing.T, db *sql.DB, id string, status domain.OrderStatus) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO orders (id, customer, line_id, quantity, priority, status, due_date, created_by, created_at, updated_at)
		VALUES ($1, 'Worker', 'A', 100, 'low', $2, '2026-06-10', 'worker-user', NOW(), NOW())
	`, id, status); err != nil {
		t.Fatalf("insert worker order %s: %v", id, err)
	}
}

func seedWorkerJob(t *testing.T, db *sql.DB, job domain.ScheduleJob) {
	t.Helper()
	orderJSON, _ := json.Marshal(job.OrderIDs)
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = job.CreatedAt
	}
	if _, err := db.Exec(`
		INSERT INTO schedule_jobs (id, line_id, status, message, source, preview_id, request_hash, line_revision, attempt_count, order_ids, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12)
	`, job.ID, job.LineID, job.Status, job.Message, job.Source, job.PreviewID, job.RequestHash, job.LineRevision, job.AttemptCount, string(orderJSON), job.CreatedAt, job.UpdatedAt); err != nil {
		t.Fatalf("insert worker job %s: %v", job.ID, err)
	}
	if _, err := db.Exec(`
		INSERT INTO audit_logs (id, actor_id, action, resource, reason, created_at)
		VALUES ($1, 'worker-user', 'schedule.job.create', $2, 'integration', NOW())
	`, "AUD-CREATE-"+job.ID, job.ID); err != nil {
		t.Fatalf("insert worker job audit: %v", err)
	}
}

func workerJob(t *testing.T, db *sql.DB, id string) domain.ScheduleJob {
	t.Helper()
	var job domain.ScheduleJob
	var orderJSON []byte
	if err := db.QueryRow(`
		SELECT id, line_id, status, COALESCE(message, ''), COALESCE(source, ''), COALESCE(preview_id, ''),
		       COALESCE(request_hash, ''), line_revision, attempt_count, order_ids, created_at, updated_at
		FROM schedule_jobs
		WHERE id = $1
	`, id).Scan(&job.ID, &job.LineID, &job.Status, &job.Message, &job.Source, &job.PreviewID, &job.RequestHash, &job.LineRevision, &job.AttemptCount, &orderJSON, &job.CreatedAt, &job.UpdatedAt); err != nil {
		t.Fatalf("read worker job %s: %v", id, err)
	}
	_ = json.Unmarshal(orderJSON, &job.OrderIDs)
	return job
}

func TestWorkerPostgresPersistsLineScheduleAndAudit(t *testing.T) {
	db := newWorkerIntegrationDB(t)
	seedWorkerUser(t, db, "worker-user")
	seedWorkerOrder(t, db, "ORD-WORKER-LINE", domain.StatusPending)
	job := domain.ScheduleJob{ID: "JOB-WORKER-LINE", LineID: "A", Status: domain.JobQueued, OrderIDs: []string{"ORD-WORKER-LINE"}, LineRevision: 0}
	seedWorkerJob(t, db, job)

	if err := processDBJobLocked(context.Background(), db, job, 3); err != nil {
		t.Fatalf("process line schedule job: %v", err)
	}
	persisted := workerJob(t, db, job.ID)
	if persisted.Status != domain.JobCompleted || persisted.AttemptCount != 1 {
		t.Fatalf("unexpected completed job: %+v", persisted)
	}
	var orderStatus domain.OrderStatus
	var allocationStatus domain.OrderStatus
	if err := db.QueryRow(`
		SELECT o.status, a.status
		FROM orders o
		JOIN schedule_allocations a ON a.order_id = o.id
		WHERE o.id = 'ORD-WORKER-LINE'
	`).Scan(&orderStatus, &allocationStatus); err != nil {
		t.Fatalf("read scheduled order/allocation: %v", err)
	}
	if orderStatus != domain.StatusScheduled || allocationStatus != domain.StatusScheduled {
		t.Fatalf("expected scheduled statuses, order=%q allocation=%q", orderStatus, allocationStatus)
	}
	var audits int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'schedule.job.complete' AND resource = $1", job.ID).Scan(&audits); err != nil {
		t.Fatalf("count complete audit: %v", err)
	}
	if audits != 1 {
		t.Fatalf("expected complete audit, got %d", audits)
	}
}

func TestWorkerPostgresPersistsPreviewAllocationsAndRejectsStaleRevision(t *testing.T) {
	db := newWorkerIntegrationDB(t)
	seedWorkerUser(t, db, "worker-user")
	seedWorkerOrder(t, db, "ORD-WORKER-PREVIEW", domain.StatusPending)
	if _, err := db.Exec(`
		INSERT INTO schedule_allocations (order_id, line_id, allocation_date, quantity, priority, locked, status)
		VALUES ('ORD-WORKER-PREVIEW', 'A', '2026-05-30', 100, 'low', FALSE, '已排程')
	`); err != nil {
		t.Fatalf("insert old allocation: %v", err)
	}
	allocations, _ := json.Marshal([]scheduler.Allocation{{
		OrderID:  "ORD-WORKER-PREVIEW",
		LineID:   "A",
		Date:     time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Quantity: 100,
		Priority: domain.PriorityLow,
	}})
	if _, err := db.Exec(`
		INSERT INTO schedule_previews (id, actor_id, actor_role, line_id, line_revision, request_hash, request, allocations, conflicts, created_at, expires_at)
		VALUES ('PREVIEW-WORKER', 'worker-user', 'scheduler', 'A', 0, 'hash', '{}'::jsonb, $1::jsonb, '[]'::jsonb, NOW(), NOW() + INTERVAL '10 minutes')
	`, string(allocations)); err != nil {
		t.Fatalf("insert preview: %v", err)
	}
	job := domain.ScheduleJob{ID: "JOB-WORKER-PREVIEW", LineID: "A", Status: domain.JobQueued, PreviewID: "PREVIEW-WORKER", RequestHash: "hash", LineRevision: 0, OrderIDs: []string{"ORD-WORKER-PREVIEW"}}
	seedWorkerJob(t, db, job)
	if err := processDBJobLocked(context.Background(), db, job, 3); err != nil {
		t.Fatalf("process preview job: %v", err)
	}
	var allocationDate time.Time
	var allocationCount int
	if err := db.QueryRow("SELECT COUNT(*), MIN(allocation_date) FROM schedule_allocations WHERE order_id = 'ORD-WORKER-PREVIEW'").Scan(&allocationCount, &allocationDate); err != nil {
		t.Fatalf("read preview allocations: %v", err)
	}
	if allocationCount != 1 || allocationDate.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("preview should replace open allocations, count=%d date=%s", allocationCount, allocationDate.Format("2006-01-02"))
	}

	seedWorkerOrder(t, db, "ORD-WORKER-STALE", domain.StatusPending)
	if _, err := db.Exec("UPDATE production_lines SET schedule_revision = 5 WHERE id = 'A'"); err != nil {
		t.Fatalf("bump line revision: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO schedule_previews (id, actor_id, actor_role, line_id, line_revision, request_hash, request, allocations, conflicts, created_at, expires_at)
		VALUES ('PREVIEW-STALE', 'worker-user', 'scheduler', 'A', 0, 'hash', '{}'::jsonb, $1::jsonb, '[]'::jsonb, NOW(), NOW() + INTERVAL '10 minutes')
	`, string(allocations)); err != nil {
		t.Fatalf("insert stale preview: %v", err)
	}
	stale := domain.ScheduleJob{ID: "JOB-WORKER-STALE", LineID: "A", Status: domain.JobQueued, PreviewID: "PREVIEW-STALE", RequestHash: "hash", LineRevision: 0}
	seedWorkerJob(t, db, stale)
	if err := processDBJobLocked(context.Background(), db, stale, 3); err != nil {
		t.Fatalf("process stale job: %v", err)
	}
	if got := workerJob(t, db, stale.ID); got.Status != domain.JobFailed || !strings.Contains(got.Message, "排程資料已變更") {
		t.Fatalf("stale job should fail deterministically, got %+v", got)
	}
}

func TestWorkerPostgresRetryAndBackfillQueuedJobs(t *testing.T) {
	db := newWorkerIntegrationDB(t)
	seedWorkerUser(t, db, "worker-user")
	retryJob := domain.ScheduleJob{ID: "JOB-WORKER-RETRY", LineID: "missing-line", Status: domain.JobQueued}
	seedWorkerJob(t, db, retryJob)
	if err := processDBJobLocked(context.Background(), db, retryJob, 2); err == nil {
		t.Fatal("missing line should return a retryable persistence error")
	}
	if got := workerJob(t, db, retryJob.ID); got.Status != domain.JobQueued || got.AttemptCount != 1 {
		t.Fatalf("first failure should requeue below max retries, got %+v", got)
	}
	if err := processDBJobLocked(context.Background(), db, retryJob, 1); err != nil {
		t.Fatalf("max retry failure should be persisted and swallowed: %v", err)
	}
	if got := workerJob(t, db, retryJob.ID); got.Status != domain.JobFailed || got.AttemptCount != 2 {
		t.Fatalf("max retry should fail job, got %+v", got)
	}

	seedWorkerOrder(t, db, "ORD-WORKER-BACKFILL", domain.StatusPending)
	backfillJob := domain.ScheduleJob{ID: "JOB-WORKER-BACKFILL", LineID: "A", Status: domain.JobQueued, OrderIDs: []string{"ORD-WORKER-BACKFILL"}}
	seedWorkerJob(t, db, backfillJob)
	if err := backfillQueuedJobs(context.Background(), db, workerLockProvider{}, 2, time.Second, 0, time.Second); err != nil {
		t.Fatalf("backfill queued jobs: %v", err)
	}
	if got := workerJob(t, db, backfillJob.ID); got.Status != domain.JobCompleted {
		t.Fatalf("backfill should complete eligible job, got %+v", got)
	}
}

type workerLockProvider struct{}

func (workerLockProvider) Acquire(context.Context, string, time.Duration) (womslock.Lock, error) {
	return workerLock{}, nil
}

type workerLock struct{}

func (workerLock) Refresh(context.Context, time.Duration) error { return nil }
func (workerLock) Release(context.Context) error                { return nil }
