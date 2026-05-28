package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/d11nn/woms/internal/auth"
	"github.com/d11nn/woms/internal/domain"
	_ "github.com/lib/pq"
)

func requirePostgresIntegration(t *testing.T) string {
	t.Helper()
	if os.Getenv("WOMS_INTEGRATION_TESTS") != "1" {
		t.Skip("set WOMS_INTEGRATION_TESTS=1 and DATABASE_URL to run PostgreSQL integration tests")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}
	parsedURL, err := url.Parse(databaseURL)
	if err == nil {
		query := parsedURL.Query()
		query.Set("sslmode", "disable")
		parsedURL.RawQuery = query.Encode()
		databaseURL = parsedURL.String()
	}
	chdirRepoRoot(t)
	return databaseURL
}

func chdirRepoRoot(t *testing.T) {
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

func newIntegrationPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()
	databaseURL := requirePostgresIntegration(t)
	parsedURL, _ := url.Parse(databaseURL)
	query := parsedURL.Query()
	query.Set("sslmode", "disable")
	parsedURL.RawQuery = query.Encode()
	databaseURL = parsedURL.String()

	schema := fmt.Sprintf("woms_it_%d", time.Now().UnixNano())
	adminDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + pqIdentifier(schema)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + pqIdentifier(schema) + ` CASCADE`)
		_ = adminDB.Close()
	})

	storeURL := databaseURLWithSearchPath(t, databaseURL, schema)
	store, err := NewPostgresStoreContext(context.Background(), storeURL, false)
	if err != nil {
		t.Fatalf("create postgres store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func databaseURLWithSearchPath(t *testing.T, databaseURL, schema string) string {
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

func pqIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func withFixedNow(t *testing.T, value time.Time) {
	t.Helper()
	previous := nowUTC
	nowUTC = func() time.Time { return value }
	t.Cleanup(func() { nowUTC = previous })
}

func integrationClaims(role domain.Role, subject, lineID string) auth.Claims {
	return auth.Claims{Subject: subject, Role: role, LineID: lineID}
}

func insertIntegrationUser(t *testing.T, store *PostgresStore, id, username string, role domain.Role, lineID string, hash string, disabled bool) {
	t.Helper()
	if hash == "" {
		var err error
		hash, err = auth.HashPassword("secret")
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
	}
	_, err := store.db.Exec(`
		INSERT INTO users (id, username, password_hash, role, line_id, disabled)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
	`, id, username, hash, role, lineID, disabled)
	if err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}
}

func insertIntegrationOrder(t *testing.T, store *PostgresStore, order domain.Order) {
	t.Helper()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}
	if order.UpdatedAt.IsZero() {
		order.UpdatedAt = order.CreatedAt
	}
	_, err := store.db.Exec(`
		INSERT INTO orders (id, customer, line_id, quantity, priority, status, due_date, note, created_by, source_order, rejection_reason, rejected_by, rejected_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), $13, $14, $15)
	`, order.ID, order.Customer, order.LineID, order.Quantity, order.Priority, order.Status, order.DueDate, order.Note, order.CreatedBy, order.SourceOrder, order.RejectionReason, order.RejectedBy, nullableTime(order.RejectedAt), order.CreatedAt, order.UpdatedAt)
	if err != nil {
		t.Fatalf("insert order %s: %v", order.ID, err)
	}
}

func countAuditActions(t *testing.T, store *PostgresStore, action, resource string) int {
	t.Helper()
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = $1 AND resource = $2", action, resource).Scan(&count); err != nil {
		t.Fatalf("count audit logs: %v", err)
	}
	return count
}

func legacySHA256TestHash(password string) string {
	salt := []byte("0123456789abcdef")
	digest := append(append([]byte{}, salt...), []byte(password)...)
	sum := sha256.Sum256(digest)
	digest = sum[:]
	for i := 1; i < 2; i++ {
		next := sha256.Sum256(digest)
		digest = next[:]
	}
	return "sha256$2$" + base64.RawURLEncoding.EncodeToString(salt) + "$" + base64.RawURLEncoding.EncodeToString(digest)
}

func TestPostgresStoreConstructionAuthAndUTF8Constraints(t *testing.T) {
	databaseURL := requirePostgresIntegration(t)
	if _, err := NewPostgresStoreContext(context.Background(), "", false); err == nil {
		t.Fatal("empty DATABASE_URL should fail")
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	if _, err := NewPostgresStoreContext(pingCtx, "postgres://127.0.0.1:1/woms?sslmode=disable", false); err == nil {
		cancel()
		t.Fatal("unreachable DATABASE_URL should fail ping")
	}
	cancel()
	migrationFailureURL := newMigrationFailureDatabaseURL(t, databaseURL)
	if _, err := NewPostgresStoreContext(context.Background(), migrationFailureURL, false); err == nil {
		t.Fatal("migration failure should fail store construction")
	}

	store := newIntegrationPostgresStore(t)
	insertIntegrationUser(t, store, "user-bcrypt", "bcrypt-user", domain.RoleSales, "", "", false)
	insertIntegrationUser(t, store, "user-legacy", "legacy-user", domain.RoleSales, "", legacySHA256TestHash("legacy-secret"), false)
	insertIntegrationUser(t, store, "user-disabled", "disabled-user", domain.RoleSales, "", "", true)

	if user, ok := store.Authenticate("bcrypt-user", "secret"); !ok || user.ID != "user-bcrypt" {
		t.Fatalf("bcrypt auth failed: user=%+v ok=%v", user, ok)
	}
	if user, ok := store.Authenticate("legacy-user", "legacy-secret"); !ok || user.ID != "user-legacy" {
		t.Fatalf("legacy sha256 auth failed: user=%+v ok=%v", user, ok)
	}
	if _, ok := store.Authenticate("disabled-user", "secret"); ok {
		t.Fatal("disabled user should not authenticate")
	}

	validStatuses := []domain.OrderStatus{
		domain.StatusPending,
		domain.StatusScheduled,
		domain.StatusInProgress,
		domain.StatusCompleted,
		domain.StatusRejected,
	}
	for index, status := range validStatuses {
		orderID := fmt.Sprintf("ORD-UTF8-%d", index)
		insertIntegrationOrder(t, store, domain.Order{
			ID:        orderID,
			Customer:  "UTF8",
			LineID:    "A",
			Quantity:  100,
			Priority:  domain.PriorityLow,
			Status:    status,
			DueDate:   time.Date(2026, 6, 10+index, 0, 0, 0, 0, time.UTC),
			CreatedBy: "user-bcrypt",
		})
		_, err := store.db.Exec(`
			INSERT INTO schedule_allocations (order_id, line_id, allocation_date, quantity, priority, locked, status)
			VALUES ($1, 'A', $2, 100, 'low', FALSE, $3)
		`, orderID, time.Date(2026, 6, 10+index, 0, 0, 0, 0, time.UTC), status)
		if err != nil {
			t.Fatalf("status %q should satisfy UTF-8 allocation constraint: %v", status, err)
		}
	}
	if _, err := store.db.Exec("INSERT INTO schedule_allocations (order_id, line_id, allocation_date, quantity, priority, locked, status) VALUES ('ORD-UTF8-0', 'A', '2026-07-01', 1, 'low', FALSE, 'invalid-status')"); err == nil {
		t.Fatal("unknown status should be rejected by allocation constraint")
	}
}

func newMigrationFailureDatabaseURL(t *testing.T, databaseURL string) string {
	t.Helper()
	schema := fmt.Sprintf("woms_it_bad_%d", time.Now().UnixNano())
	adminDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("open admin database for migration failure fixture: %v", err)
	}
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + pqIdentifier(schema)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create migration failure schema: %v", err)
	}
	if _, err := adminDB.Exec(`CREATE TABLE ` + pqIdentifier(schema) + `.users (id INTEGER PRIMARY KEY)`); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create incompatible users table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + pqIdentifier(schema) + ` CASCADE`)
		_ = adminDB.Close()
	})
	return databaseURLWithSearchPath(t, databaseURL, schema)
}

func TestPostgresUserManagementPersistsChangesAndAudit(t *testing.T) {
	store := newIntegrationPostgresStore(t)
	insertIntegrationUser(t, store, "user-admin", "admin", domain.RoleAdmin, "", "", false)

	created, err := store.CreateUser(createUserRequest{Username: "planner", Password: "secret", Role: domain.RoleScheduler, LineID: "A"}, "user-admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Role != domain.RoleScheduler || created.LineID != "A" {
		t.Fatalf("unexpected created user: %+v", created)
	}
	if countAuditActions(t, store, "user.create", created.ID) != 1 {
		t.Fatal("expected user.create audit")
	}

	assigned, err := store.AssignUser(assignUserRequest{Username: "planner", Role: domain.RoleSales, LineID: "A"}, "user-admin")
	if err != nil {
		t.Fatalf("assign user: %v", err)
	}
	if assigned.Role != domain.RoleSales || assigned.LineID != "" {
		t.Fatalf("non-scheduler line assignment should be cleared: %+v", assigned)
	}
	reset, err := store.ResetUserPassword(resetUserPasswordRequest{Username: "planner", Password: "changed"}, "user-admin")
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if reset.PasswordHash == created.PasswordHash {
		t.Fatal("password hash should change after reset")
	}
	if _, ok := store.Authenticate("planner", "changed"); !ok {
		t.Fatal("reset password should authenticate")
	}

	deleted, err := store.DeleteUser("planner", "user-admin")
	if err != nil {
		t.Fatalf("delete unreferenced user: %v", err)
	}
	if !deleted.Deleted || deleted.Disabled {
		t.Fatalf("unreferenced user should be deleted, got %+v", deleted)
	}

	referenced, err := store.CreateUser(createUserRequest{Username: "referenced", Password: "secret", Role: domain.RoleSales}, "user-admin")
	if err != nil {
		t.Fatalf("create referenced user: %v", err)
	}
	insertIntegrationOrder(t, store, domain.Order{
		ID:        "ORD-USER-REF",
		Customer:  "Referenced",
		LineID:    "A",
		Quantity:  100,
		Priority:  domain.PriorityLow,
		Status:    domain.StatusPending,
		DueDate:   time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
		CreatedBy: referenced.ID,
	})
	disabled, err := store.DeleteUser("referenced", "user-admin")
	if err != nil {
		t.Fatalf("disable referenced user: %v", err)
	}
	if !disabled.Disabled || disabled.Deleted {
		t.Fatalf("referenced user should be disabled, got %+v", disabled)
	}
	if _, ok := store.Authenticate("referenced", "secret"); ok {
		t.Fatal("disabled referenced user should not authenticate")
	}
}

func TestPostgresOrderStateTransitionsAndAudit(t *testing.T) {
	withFixedNow(t, time.Date(2026, 5, 28, 4, 0, 0, 0, time.UTC))
	store := newIntegrationPostgresStore(t)
	insertIntegrationUser(t, store, "user-sales-a", "sales-a", domain.RoleSales, "", "", false)
	insertIntegrationUser(t, store, "user-sales-b", "sales-b", domain.RoleSales, "", "", false)
	insertIntegrationUser(t, store, "user-scheduler-a", "scheduler-a", domain.RoleScheduler, "A", "", false)

	salesClaims := integrationClaims(domain.RoleSales, "user-sales-a", "")
	schedulerClaims := integrationClaims(domain.RoleScheduler, "user-scheduler-a", "A")
	order, err := store.CreateOrder(createOrderRequest{Customer: "Acme", LineID: "A", Quantity: 100, Priority: domain.PriorityHigh, DueDate: "2026-06-10", Note: "first"}, salesClaims.Subject)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Status != domain.StatusPending || order.CreatedBy != salesClaims.Subject {
		t.Fatalf("unexpected created order: %+v", order)
	}
	if _, err := store.UpdateOrderDueDate(order.ID, updateOrderRequest{DueDate: "2026-05-28"}, salesClaims); err == nil {
		t.Fatal("same-day due date should be rejected")
	}
	updated, err := store.UpdateOrderDueDate(order.ID, updateOrderRequest{DueDate: "2026-06-11", Quantity: 125}, salesClaims)
	if err != nil {
		t.Fatalf("update order: %v", err)
	}
	if updated.Quantity != 125 || updated.DueDate.Format(dateLayout) != "2026-06-11" {
		t.Fatalf("unexpected updated order: %+v", updated)
	}
	if _, err := store.UpdateOrderDueDate(order.ID, updateOrderRequest{DueDate: "2026-06-12"}, integrationClaims(domain.RoleSales, "user-sales-b", "")); err == nil {
		t.Fatal("sales should not update another user's order")
	}
	rejected, err := store.RejectOrders(rejectOrdersRequest{OrderIDs: []string{order.ID}, Reason: "capacity review"}, schedulerClaims)
	if err != nil {
		t.Fatalf("reject order: %v", err)
	}
	if len(rejected.Orders) != 1 || rejected.Orders[0].Status != domain.StatusRejected {
		t.Fatalf("unexpected rejected response: %+v", rejected)
	}
	resubmitted, err := store.ResubmitOrder(resubmitOrderRequest{OrderID: order.ID, DueDate: "2026-06-13", Quantity: 150}, salesClaims)
	if err != nil {
		t.Fatalf("resubmit order: %v", err)
	}
	if resubmitted.Status != domain.StatusPending || resubmitted.RejectionReason != "" || resubmitted.Quantity != 150 {
		t.Fatalf("unexpected resubmitted order: %+v", resubmitted)
	}
	cancelled, err := store.CancelOrders(cancelOrdersRequest{OrderIDs: []string{order.ID, "missing"}}, salesClaims)
	if err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	if fmt.Sprint(cancelled.CancelledOrderIDs) != fmt.Sprint([]string{order.ID}) || fmt.Sprint(cancelled.SkippedOrderIDs) != fmt.Sprint([]string{"missing"}) {
		t.Fatalf("unexpected cancel response: %+v", cancelled)
	}
	for _, action := range []string{"order.create", "order.update_due_date", "order.reject", "order.resubmit", "order.cancel"} {
		if countAuditActions(t, store, action, order.ID) == 0 {
			t.Fatalf("expected %s audit for %s", action, order.ID)
		}
	}
}

func TestPostgresScheduleJobLifecycle(t *testing.T) {
	withFixedNow(t, time.Date(2026, 5, 28, 4, 0, 0, 0, time.UTC))
	store := newIntegrationPostgresStore(t)
	insertIntegrationUser(t, store, "user-sales", "sales", domain.RoleSales, "", "", false)
	insertIntegrationUser(t, store, "user-scheduler-a", "scheduler-a", domain.RoleScheduler, "A", "", false)
	insertIntegrationOrder(t, store, domain.Order{ID: "ORD-JOB-1", Customer: "Job", LineID: "A", Quantity: 100, Priority: domain.PriorityLow, Status: domain.StatusPending, DueDate: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), CreatedBy: "user-sales"})

	claims := integrationClaims(domain.RoleScheduler, "user-scheduler-a", "A")
	req := scheduleRequest{LineID: "A", StartDate: "2026-06-01", CurrentDate: "2026-05-28", OrderIDs: []string{"ORD-JOB-1"}}
	preview, err := store.PreviewSchedule(req, claims)
	if err != nil {
		t.Fatalf("preview schedule: %v", err)
	}
	req.PreviewID = preview.PreviewID
	job, err := store.CreateScheduleJob(req, claims)
	if err != nil {
		t.Fatalf("create schedule job: %v", err)
	}
	if job.Status != domain.JobQueued || job.AttemptCount != 0 {
		t.Fatalf("unexpected queued job: %+v", job)
	}
	got, ok := store.GetScheduleJob(job.ID)
	if !ok || got.ID != job.ID || got.Status != domain.JobQueued {
		t.Fatalf("get queued job failed: %+v ok=%v", got, ok)
	}
	store.DeleteQueuedScheduleJob(job.ID)
	if _, ok := store.GetScheduleJob(job.ID); ok {
		t.Fatal("deleted queued job should not be found")
	}

	preview, err = store.PreviewSchedule(req, claims)
	if err != nil {
		t.Fatalf("preview schedule again: %v", err)
	}
	req.PreviewID = preview.PreviewID
	job, err = store.CreateScheduleJob(req, claims)
	if err != nil {
		t.Fatalf("create second schedule job: %v", err)
	}
	completed := store.ExecuteScheduleJob(job.ID)
	if completed.Status != domain.JobCompleted {
		t.Fatalf("expected completed memory execution, got %+v", completed)
	}
	persisted, ok := store.GetScheduleJob(job.ID)
	if !ok || persisted.Status != domain.JobCompleted {
		t.Fatalf("completed job should persist: %+v ok=%v", persisted, ok)
	}
	if persisted.AttemptCount != 0 {
		t.Fatalf("memory execution should not mutate DB attempt_count, got %d", persisted.AttemptCount)
	}
}

func TestPostgresScheduleCalendarPreviewConfirmAndHistory(t *testing.T) {
	withFixedNow(t, time.Date(2026, 5, 28, 4, 0, 0, 0, time.UTC))
	store := newIntegrationPostgresStore(t)
	insertIntegrationUser(t, store, "user-sales", "sales", domain.RoleSales, "", "", false)
	insertIntegrationUser(t, store, "user-scheduler-a", "scheduler-a", domain.RoleScheduler, "A", "", false)

	salesClaims := integrationClaims(domain.RoleSales, "user-sales", "")
	draftPreview, err := store.PreviewSchedule(scheduleRequest{
		LineID:      "A",
		StartDate:   "2026-06-01",
		CurrentDate: "2026-05-28",
		DraftOrder:  &createOrderRequest{Customer: "Draft", LineID: "A", Quantity: 100, Priority: domain.PriorityLow, DueDate: "2026-06-10"},
	}, salesClaims)
	if err != nil {
		t.Fatalf("draft preview: %v", err)
	}
	confirmedOrder, err := store.ConfirmPreviewOrder(draftPreview.PreviewID, salesClaims)
	if err != nil {
		t.Fatalf("confirm preview order: %v", err)
	}
	if confirmedOrder.Customer != "Draft" || confirmedOrder.Status != domain.StatusPending {
		t.Fatalf("unexpected confirmed draft order: %+v", confirmedOrder)
	}

	schedulerClaims := integrationClaims(domain.RoleScheduler, "user-scheduler-a", "A")
	req := scheduleRequest{LineID: "A", StartDate: "2026-06-01", CurrentDate: "2026-05-28", OrderIDs: []string{confirmedOrder.ID}}
	preview, err := store.PreviewSchedule(req, schedulerClaims)
	if err != nil {
		t.Fatalf("scheduler preview: %v", err)
	}
	if len(preview.Allocations) == 0 {
		t.Fatal("expected preview allocations")
	}
	req.PreviewID = preview.PreviewID
	job, err := store.CreateScheduleJob(req, schedulerClaims)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	completed := store.ExecuteScheduleJob(job.ID)
	if completed.Status != domain.JobCompleted {
		t.Fatalf("execute job: %+v", completed)
	}

	first, err := store.ScheduleCalendar("A", "2026-06", schedulerClaims)
	if err != nil {
		t.Fatalf("schedule calendar: %v", err)
	}
	second, err := store.ScheduleCalendar("A", "2026-06", schedulerClaims)
	if err != nil {
		t.Fatalf("schedule calendar second read: %v", err)
	}
	if fmt.Sprint(first.Allocations) != fmt.Sprint(second.Allocations) {
		t.Fatal("calendar output should be deterministic")
	}
	if len(first.Allocations) == 0 || first.Allocations[0].OrderID != confirmedOrder.ID {
		t.Fatalf("expected scheduled allocation in calendar, got %+v", first.Allocations)
	}
	history, err := store.ScheduleHistory("A", schedulerClaims)
	if err != nil {
		t.Fatalf("schedule history: %v", err)
	}
	if len(history) == 0 {
		t.Fatal("expected schedule history entries")
	}
	again, err := store.ScheduleHistory("A", schedulerClaims)
	if err != nil {
		t.Fatalf("schedule history second read: %v", err)
	}
	if fmt.Sprint(history) != fmt.Sprint(again) {
		t.Fatal("history output should be deterministic")
	}
}

func TestPostgresProductionTransitionsAndCompletedAllocations(t *testing.T) {
	store := newIntegrationPostgresStore(t)
	insertIntegrationUser(t, store, "user-sales", "sales", domain.RoleSales, "", "", false)
	insertIntegrationUser(t, store, "user-scheduler-a", "scheduler-a", domain.RoleScheduler, "A", "", false)
	insertIntegrationOrder(t, store, domain.Order{ID: "ORD-PROD-1", Customer: "Prod", LineID: "A", Quantity: 100, Priority: domain.PriorityLow, Status: domain.StatusScheduled, DueDate: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), CreatedBy: "user-sales"})
	_, err := store.db.Exec(`
		INSERT INTO schedule_allocations (order_id, line_id, allocation_date, quantity, priority, locked, status)
		VALUES ('ORD-PROD-1', 'A', '2026-06-01', 100, 'low', FALSE, '已排程')
	`)
	if err != nil {
		t.Fatalf("insert allocation: %v", err)
	}

	claims := integrationClaims(domain.RoleScheduler, "user-scheduler-a", "A")
	if _, err := store.ConfirmProduction(productionConfirmRequest{OrderID: "ORD-PROD-1", ProductionDate: "2026-06-01", ProducedQuantity: 100}, claims); err == nil {
		t.Fatal("confirm before start should fail")
	}
	started, err := store.StartProduction(productionStartRequest{OrderID: "ORD-PROD-1"}, claims)
	if err != nil {
		t.Fatalf("start production: %v", err)
	}
	if started.Status != domain.StatusInProgress {
		t.Fatalf("expected in-progress order, got %+v", started)
	}
	if _, err := store.ConfirmProduction(productionConfirmRequest{OrderID: "ORD-PROD-1", ProductionDate: "2026-06-01", ProducedQuantity: 101}, claims); err == nil {
		t.Fatal("over-production should fail")
	}
	confirmed, err := store.ConfirmProduction(productionConfirmRequest{OrderID: "ORD-PROD-1", ProductionDate: "2026-06-01", ProducedQuantity: 60}, claims)
	if err != nil {
		t.Fatalf("partial confirm production: %v", err)
	}
	if confirmed.Order.Status != domain.StatusCompleted || confirmed.Order.Quantity != 60 || confirmed.Remainder == nil || confirmed.Remainder.Quantity != 40 {
		t.Fatalf("unexpected confirmation response: %+v", confirmed)
	}
	var status domain.OrderStatus
	var locked bool
	if err := store.db.QueryRow("SELECT status, locked FROM schedule_allocations WHERE order_id = 'ORD-PROD-1' AND allocation_date = '2026-06-01'").Scan(&status, &locked); err != nil {
		t.Fatalf("read completed allocation: %v", err)
	}
	if status != domain.StatusCompleted || !locked {
		t.Fatalf("completed allocation should be locked and completed, status=%q locked=%v", status, locked)
	}
}

func TestPostgresHPAPeakDemoLifecycle(t *testing.T) {
	store := newIntegrationPostgresStore(t)
	insertIntegrationUser(t, store, "user-admin", "admin", domain.RoleAdmin, "", "", false)
	claims := integrationClaims(domain.RoleAdmin, "user-admin", "")

	summary, err := store.CreateHPAPeakDemo(claims)
	if err != nil {
		t.Fatalf("create hpa peak demo: %v", err)
	}
	if summary.LineCount != hpaDemoLastLine || summary.OrderCount != hpaDemoLastLine*hpaDemoOrdersPerLine || summary.Statuses[string(domain.JobQueued)] != hpaDemoLastLine*hpaDemoJobsPerLine {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	jobs := store.HPAPeakJobs()
	if len(jobs) != hpaDemoLastLine*hpaDemoJobsPerLine {
		t.Fatalf("unexpected job count %d", len(jobs))
	}
	if !sort.SliceIsSorted(jobs, func(i, j int) bool { return jobs[i].ID < jobs[j].ID }) {
		t.Fatal("HPA jobs should be sorted")
	}
	reset, err := store.CreateHPAPeakDemo(claims)
	if err != nil {
		t.Fatalf("reset hpa peak demo: %v", err)
	}
	if reset.OrderCount != summary.OrderCount || reset.JobCount != summary.JobCount {
		t.Fatalf("reset should recreate same counts: before=%+v after=%+v", summary, reset)
	}
	cleared, err := store.ClearHPAPeakDemo(claims)
	if err != nil {
		t.Fatalf("clear hpa peak demo: %v", err)
	}
	if cleared.OrderCount != 0 || cleared.Statuses[string(domain.JobQueued)] != 0 || cleared.Statuses[string(domain.JobCancelled)] != hpaDemoLastLine*hpaDemoJobsPerLine {
		t.Fatalf("unexpected cleared summary: %+v", cleared)
	}
}
