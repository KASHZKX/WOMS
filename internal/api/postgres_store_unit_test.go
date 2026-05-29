package api

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/d11nn/woms/internal/domain"
)

type scanFunc func(dest ...any) error

func (fn scanFunc) Scan(dest ...any) error {
	return fn(dest...)
}

func TestPostgresStorePureHelpersNormalizeValues(t *testing.T) {
	ids := uniqueOrderIDs([]string{" ORD-2 ", "", "ORD-1", "ORD-2", "ORD-1 ", "ORD-3"})
	if got, want := strings.Join(ids, ","), "ORD-2,ORD-1,ORD-3"; got != want {
		t.Fatalf("uniqueOrderIDs kept wrong order: got %q want %q", got, want)
	}

	if value := nullableTime(time.Time{}); value != nil {
		t.Fatalf("zero nullable time should be nil, got %#v", value)
	}
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	if value := nullableTime(now); value != now {
		t.Fatalf("non-zero nullable time should be preserved, got %#v", value)
	}

	id := auditID("AUD-schedule.job/create:ORD-1")
	if !strings.HasPrefix(id, "AUD-schedule-job-create-ORD-1-") {
		t.Fatalf("auditID should sanitize separators, got %q", id)
	}
}

func TestScanScheduleJobSortsOrderIDsAndPropagatesScannerErrors(t *testing.T) {
	created := time.Date(2026, 5, 29, 1, 2, 3, 0, time.UTC)
	updated := created.Add(time.Minute)
	job, err := scanScheduleJob(scanFunc(func(dest ...any) error {
		*(dest[0].(*string)) = "JOB-1"
		*(dest[1].(*string)) = "A"
		*(dest[2].(*domain.ScheduleJobStatus)) = domain.JobRunning
		*(dest[3].(*string)) = "running"
		*(dest[4].(*string)) = "scheduler"
		*(dest[5].(*string)) = "PREVIEW-1"
		*(dest[6].(*string)) = "hash"
		*(dest[7].(*int64)) = 7
		*(dest[8].(*int)) = 2
		*(dest[9].(*[]byte)) = []byte(`["ORD-2","ORD-1"]`)
		*(dest[10].(*time.Time)) = created
		*(dest[11].(*time.Time)) = updated
		return nil
	}))
	if err != nil {
		t.Fatalf("scanScheduleJob returned error: %v", err)
	}
	if job.ID != "JOB-1" || job.LineID != "A" || job.Status != domain.JobRunning || job.AttemptCount != 2 {
		t.Fatalf("unexpected job fields: %+v", job)
	}
	if got, want := strings.Join(job.OrderIDs, ","), "ORD-1,ORD-2"; got != want {
		t.Fatalf("order ids should be sorted: got %q want %q", got, want)
	}

	boom := errors.New("scan failed")
	if _, err := scanScheduleJob(scanFunc(func(dest ...any) error { return boom })); !errors.Is(err, boom) {
		t.Fatalf("scanScheduleJob should propagate scanner errors, got %v", err)
	}
}

func TestScanOrderMapsNullableRejectionTimestamp(t *testing.T) {
	rejectedAt := time.Date(2026, 5, 29, 2, 0, 0, 0, time.UTC)
	order, err := scanOrder(scanFunc(func(dest ...any) error {
		*(dest[0].(*string)) = "ORD-1"
		*(dest[1].(*string)) = "ACME"
		*(dest[2].(*string)) = "A"
		*(dest[3].(*int)) = 250
		*(dest[4].(*domain.Priority)) = domain.PriorityHigh
		*(dest[5].(*domain.OrderStatus)) = domain.StatusRejected
		*(dest[6].(*time.Time)) = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		*(dest[7].(*string)) = "note"
		*(dest[8].(*string)) = "user-sales"
		*(dest[9].(*string)) = ""
		*(dest[10].(*string)) = "capacity"
		*(dest[11].(*string)) = "scheduler-a"
		*(dest[12].(*sql.NullTime)) = sql.NullTime{Time: rejectedAt, Valid: true}
		*(dest[13].(*time.Time)) = rejectedAt.Add(-time.Hour)
		*(dest[14].(*time.Time)) = rejectedAt
		return nil
	}))
	if err != nil {
		t.Fatalf("scanOrder returned error: %v", err)
	}
	if order.ID != "ORD-1" || order.Status != domain.StatusRejected || !order.RejectedAt.Equal(rejectedAt) {
		t.Fatalf("unexpected rejected order: %+v", order)
	}

	order, err = scanOrder(scanFunc(func(dest ...any) error {
		*(dest[12].(*sql.NullTime)) = sql.NullTime{}
		return nil
	}))
	if err != nil {
		t.Fatalf("scanOrder with null rejected_at returned error: %v", err)
	}
	if !order.RejectedAt.IsZero() {
		t.Fatalf("null rejected_at should leave zero time, got %v", order.RejectedAt)
	}
}
