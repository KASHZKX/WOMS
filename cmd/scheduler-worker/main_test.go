package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/d11nn/woms/internal/domain"
	womslock "github.com/d11nn/woms/internal/lock"
	"github.com/segmentio/kafka-go"
)

func TestLoadWorkerConfigParsesDefaultsAndTrimmedValues(t *testing.T) {
	config, err := loadWorkerConfig(mapLookup(map[string]string{
		"KAFKA_BROKERS":                          " kafka-a:9092,kafka-b:9092 ",
		"KAFKA_SCHEDULE_TOPIC":                   " custom.topic ",
		"KAFKA_CONSUMER_GROUP":                   " custom-group ",
		"DATABASE_URL":                           " postgres://example ",
		"REDIS_ADDR":                             " redis:6379 ",
		"WORKER_MIN_JOB_DURATION_MS":             "125",
		"WORKER_MAX_RETRIES":                     "7",
		"WORKER_LOCK_TTL_MS":                     "20000",
		"WORKER_LOCK_RENEW_INTERVAL_MS":          "4000",
		"WORKER_LOCK_TIMEOUT_MS":                 "3000",
		"WORKER_BACKFILL_INTERVAL_MS":            "6000",
		"WORKER_DEPENDENCY_RETRY_TIMEOUT_MS":     "9000",
		"WORKER_DEPENDENCY_RETRY_INTERVAL_MS":    "250",
		"WORKER_START_OFFSET":                    " earliest ",
		"UNRELATED_EMPTY_VALUES_SHOULD_NOT_HURT": "",
	}))
	if err != nil {
		t.Fatalf("load worker config: %v", err)
	}
	if config.brokers != "kafka-a:9092,kafka-b:9092" {
		t.Fatalf("unexpected brokers %q", config.brokers)
	}
	if config.topic != "custom.topic" || config.group != "custom-group" {
		t.Fatalf("unexpected topic/group %q/%q", config.topic, config.group)
	}
	if config.databaseURL != "postgres://example" || config.redisAddr != "redis:6379" {
		t.Fatalf("unexpected database/redis config %q/%q", config.databaseURL, config.redisAddr)
	}
	if config.minJobDuration != 125*time.Millisecond || config.maxRetries != 7 {
		t.Fatalf("unexpected duration/retries %s/%d", config.minJobDuration, config.maxRetries)
	}
	if config.lockTTL != 20*time.Second || config.lockRenewInterval != 4*time.Second || config.lockTimeout != 3*time.Second {
		t.Fatalf("unexpected lock config %s/%s/%s", config.lockTTL, config.lockRenewInterval, config.lockTimeout)
	}
	if config.backfillInterval != 6*time.Second || config.dependencyTimeout != 9*time.Second || config.dependencyInterval != 250*time.Millisecond {
		t.Fatalf("unexpected worker intervals %s/%s/%s", config.backfillInterval, config.dependencyTimeout, config.dependencyInterval)
	}
	if config.startOffset != kafka.FirstOffset {
		t.Fatalf("expected first offset, got %d", config.startOffset)
	}
}

func TestLoadWorkerConfigUsesDefaultsForMissingOrEmptyValues(t *testing.T) {
	config, err := loadWorkerConfig(mapLookup(map[string]string{
		"KAFKA_BROKERS":              " ",
		"KAFKA_SCHEDULE_TOPIC":       "",
		"KAFKA_CONSUMER_GROUP":       " ",
		"WORKER_START_OFFSET":        "",
		"WORKER_MIN_JOB_DURATION_MS": " ",
		"WORKER_MAX_RETRIES":         "",
	}))
	if err != nil {
		t.Fatalf("load worker config: %v", err)
	}
	if config.brokers != "kafka:9092" {
		t.Fatalf("unexpected default brokers %q", config.brokers)
	}
	if config.topic != "woms.schedule.jobs" || config.group != "woms-scheduler-workers" {
		t.Fatalf("unexpected default topic/group %q/%q", config.topic, config.group)
	}
	if config.minJobDuration != 0 || config.maxRetries != 3 {
		t.Fatalf("unexpected default duration/retries %s/%d", config.minJobDuration, config.maxRetries)
	}
	if config.startOffset != kafka.LastOffset {
		t.Fatalf("expected last offset, got %d", config.startOffset)
	}
}

func TestConfigStartOffsetLabels(t *testing.T) {
	cases := []struct {
		label string
		want  int64
	}{
		{"earliest", kafka.FirstOffset},
		{"first", kafka.FirstOffset},
		{"oldest", kafka.FirstOffset},
		{" EARLIEST ", kafka.FirstOffset},
		{"latest", kafka.LastOffset},
		{"last", kafka.LastOffset},
		{"newest", kafka.LastOffset},
		{"", kafka.LastOffset},
	}
	for _, tc := range cases {
		got, err := configStartOffset(tc.label)
		if err != nil {
			t.Fatalf("%q: unexpected error %v", tc.label, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %d want %d", tc.label, got, tc.want)
		}
	}
	if _, err := configStartOffset("middle"); err == nil || !strings.Contains(err.Error(), "WORKER_START_OFFSET") {
		t.Fatalf("expected clear invalid offset error, got %v", err)
	}
}

func TestLoadWorkerConfigRejectsInvalidDurationsAndIntegers(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "malformed duration",
			env:  map[string]string{"WORKER_LOCK_TTL_MS": "abc"},
			want: "WORKER_LOCK_TTL_MS must be an integer number of milliseconds",
		},
		{
			name: "negative duration",
			env:  map[string]string{"WORKER_LOCK_TIMEOUT_MS": "-1"},
			want: "WORKER_LOCK_TIMEOUT_MS must be greater than or equal to zero",
		},
		{
			name: "malformed int",
			env:  map[string]string{"WORKER_MAX_RETRIES": "many"},
			want: "WORKER_MAX_RETRIES must be an integer",
		},
		{
			name: "negative int",
			env:  map[string]string{"WORKER_MAX_RETRIES": "-1"},
			want: "WORKER_MAX_RETRIES must be greater than or equal to zero",
		},
		{
			name: "invalid offset",
			env:  map[string]string{"WORKER_START_OFFSET": "middle"},
			want: "WORKER_START_OFFSET must be one of",
		},
	}
	for _, tc := range cases {
		_, err := loadWorkerConfig(mapLookup(tc.env))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: expected error containing %q, got %v", tc.name, tc.want, err)
		}
	}
}

func TestScheduleLineLockKeyScopesByProductionLine(t *testing.T) {
	if got := scheduleLineLockKey("A"); got != "woms:locks:schedule-line:A" {
		t.Fatalf("unexpected line A key %q", got)
	}
	if scheduleLineLockKey("A") == scheduleLineLockKey("B") {
		t.Fatal("different production lines must use different Redis lock keys")
	}
}

func TestAcquireLineLockRetriesContentionUntilAvailable(t *testing.T) {
	provider := &retryLockProvider{failures: 2}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	lineLock, err := acquireLineLock(ctx, provider, "woms:locks:schedule-line:A", time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if lineLock == nil {
		t.Fatal("expected lock")
	}
	if provider.attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", provider.attempts)
	}
}

func TestAcquireLineLockStopsOnNonContentionError(t *testing.T) {
	expected := errors.New("redis unavailable")
	provider := &retryLockProvider{err: expected}
	_, err := acquireLineLock(context.Background(), provider, "woms:locks:schedule-line:A", time.Second)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
	if provider.attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", provider.attempts)
	}
}

func TestProcessJobPayloadRejectsInvalidJSON(t *testing.T) {
	executor := &fakeJobExecutor{}
	err := processJobPayload(context.Background(), executor, nil, []byte("{"), 3, time.Second, 0, time.Second)
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if executor.failedJobs != 0 || executor.retryJobs != 0 || executor.lockedJobs != 0 {
		t.Fatalf("invalid JSON should not touch job state: %+v", executor)
	}
}

func TestProcessJobPayloadIgnoresMissingJobOrLineID(t *testing.T) {
	cases := []domain.ScheduleJob{
		{LineID: "A"},
		{ID: "JOB-1"},
	}
	for _, job := range cases {
		payload := mustMarshalJob(t, job)
		executor := &fakeJobExecutor{}
		if err := processJobPayload(context.Background(), executor, nil, payload, 3, time.Second, 0, time.Second); err != nil {
			t.Fatalf("missing ID/line should be ignored: %v", err)
		}
		if executor.failedJobs != 0 || executor.retryJobs != 0 || executor.lockedJobs != 0 {
			t.Fatalf("missing ID/line should not touch job state: %+v", executor)
		}
	}
}

func TestProcessJobPayloadMarksFailedWhenLockProviderMissing(t *testing.T) {
	executor := &fakeJobExecutor{}
	payload := mustMarshalJob(t, domain.ScheduleJob{ID: "JOB-1", LineID: "A"})
	if err := processJobPayload(context.Background(), executor, nil, payload, 3, time.Second, 0, time.Second); err != nil {
		t.Fatalf("process payload: %v", err)
	}
	if executor.failedJobs != 1 || executor.failedJobID != "JOB-1" {
		t.Fatalf("expected failed JOB-1, got %+v", executor)
	}
	if executor.failedMessage != "Redis 排程鎖未設定。" {
		t.Fatalf("unexpected failure message %q", executor.failedMessage)
	}
}

func TestProcessJobPayloadMarksRetryWhenLockAcquisitionTimesOut(t *testing.T) {
	executor := &fakeJobExecutor{}
	provider := &retryLockProvider{failures: 100}
	payload := mustMarshalJob(t, domain.ScheduleJob{ID: "JOB-2", LineID: "A"})
	if err := processJobPayload(context.Background(), executor, provider, payload, 3, time.Second, 0, time.Nanosecond); err != nil {
		t.Fatalf("process payload: %v", err)
	}
	if executor.retryJobs != 1 || executor.retryJobID != "JOB-2" {
		t.Fatalf("expected retry JOB-2, got %+v", executor)
	}
	if executor.retryMessage != "同產線排程鎖取得逾時，等待重試。" {
		t.Fatalf("unexpected retry message %q", executor.retryMessage)
	}
	if executor.lockedJobs != 0 {
		t.Fatalf("timed out job should not execute, got %d executions", executor.lockedJobs)
	}
}

func TestStartLockRenewalCancelsWorkWhenRefreshFails(t *testing.T) {
	refreshErr := errors.New("refresh failed")
	lineLock := &recordingLock{refreshErr: refreshErr}
	runCtx, stop := startLockRenewal(context.Background(), lineLock, time.Second, time.Millisecond)
	defer stop()

	select {
	case <-runCtx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected renewal failure to cancel run context")
	}
	if lineLock.refreshes == 0 {
		t.Fatal("expected at least one refresh attempt")
	}
}

func TestProcessJobPayloadStopsLockedWorkWhenRenewalFails(t *testing.T) {
	lineLock := &recordingLock{refreshErr: errors.New("refresh failed")}
	provider := &singleLockProvider{lock: lineLock}
	executor := &fakeJobExecutor{
		processFn: func(ctx context.Context, _ domain.ScheduleJob, _ int) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	payload := mustMarshalJob(t, domain.ScheduleJob{ID: "JOB-3", LineID: "A"})
	err := processJobPayload(context.Background(), executor, provider, payload, 3, time.Second, time.Millisecond, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled work after renewal failure, got %v", err)
	}
	if executor.lockedJobs != 1 {
		t.Fatalf("expected one locked execution, got %d", executor.lockedJobs)
	}
	if lineLock.releases != 1 {
		t.Fatalf("expected lock release, got %d", lineLock.releases)
	}
}

func TestRunLockedJobStateSkipsNonExecutableJobs(t *testing.T) {
	cases := []struct {
		name   string
		found  bool
		status domain.ScheduleJobStatus
	}{
		{name: "missing", found: false},
		{name: "cancelled", found: true, status: domain.JobCancelled},
		{name: "running", found: true, status: domain.JobRunning},
		{name: "completed", found: true, status: domain.JobCompleted},
	}
	for _, tc := range cases {
		store := &fakeLockedJobStore{found: tc.found, status: tc.status}
		err, commit := runLockedJobState(context.Background(), store, domain.ScheduleJob{ID: "JOB-STATE", LineID: "A"}, 3)
		if err != nil || !commit {
			t.Fatalf("%s: expected clean commit, got err=%v commit=%v", tc.name, err, commit)
		}
		if store.runningCalls != 0 || store.persistCalls != 0 || store.completedCalls != 0 {
			t.Fatalf("%s: should not execute job, got store %+v", tc.name, store)
		}
	}
}

func TestRunLockedJobStateCompletesQueuedJob(t *testing.T) {
	store := &fakeLockedJobStore{found: true, status: domain.JobQueued}
	err, commit := runLockedJobState(context.Background(), store, domain.ScheduleJob{ID: "JOB-QUEUED", LineID: "A"}, 3)
	if err != nil || !commit {
		t.Fatalf("expected success commit, got err=%v commit=%v", err, commit)
	}
	if store.status != domain.JobCompleted {
		t.Fatalf("expected completed status, got %q", store.status)
	}
	if store.attempt != 1 {
		t.Fatalf("expected attempt count 1, got %d", store.attempt)
	}
	if store.persistCalls != 1 || store.completedCalls != 1 {
		t.Fatalf("expected persist and complete calls, got %+v", store)
	}
}

func TestRunLockedJobStateRetriesTransientPersistFailureBelowMaxRetries(t *testing.T) {
	persistErr := errors.New("temporary database problem")
	store := &fakeLockedJobStore{found: true, status: domain.JobQueued, persistErr: persistErr}
	err, commit := runLockedJobState(context.Background(), store, domain.ScheduleJob{ID: "JOB-RETRY", LineID: "A"}, 3)
	if !errors.Is(err, persistErr) || !commit {
		t.Fatalf("expected retryable persist error with commit, got err=%v commit=%v", err, commit)
	}
	if store.status != domain.JobQueued || store.retryCalls != 1 {
		t.Fatalf("expected queued retry state, got %+v", store)
	}
	if store.retryMessage != "排程任務暫時失敗，等待重試。" {
		t.Fatalf("unexpected retry message %q", store.retryMessage)
	}
}

func TestRunLockedJobStateFailsPersistFailureAtMaxRetries(t *testing.T) {
	persistErr := errors.New("permanent database problem")
	store := &fakeLockedJobStore{found: true, status: domain.JobQueued, attempt: 2, persistErr: persistErr}
	err, commit := runLockedJobState(context.Background(), store, domain.ScheduleJob{ID: "JOB-FAIL", LineID: "A"}, 3)
	if err != nil || !commit {
		t.Fatalf("expected failed job commit without returned error, got err=%v commit=%v", err, commit)
	}
	if store.status != domain.JobFailed || store.failedCalls != 1 {
		t.Fatalf("expected failed state, got %+v", store)
	}
	if store.failedMessage != "排程任務失敗："+persistErr.Error() || store.failedReason != persistErr.Error() {
		t.Fatalf("unexpected failed message/reason %q/%q", store.failedMessage, store.failedReason)
	}
}

func TestRunLockedJobStateFailsStaleScheduleDataWithoutRetry(t *testing.T) {
	store := &fakeLockedJobStore{found: true, status: domain.JobQueued, persistErr: errStaleScheduleData{}}
	err, commit := runLockedJobState(context.Background(), store, domain.ScheduleJob{ID: "JOB-STALE", LineID: "A"}, 3)
	if err != nil || !commit {
		t.Fatalf("expected stale data to fail cleanly, got err=%v commit=%v", err, commit)
	}
	if store.status != domain.JobFailed || store.retryCalls != 0 {
		t.Fatalf("expected failed stale-data state without retry, got %+v", store)
	}
}

func TestValidateLockConfigRejectsInvalidDurations(t *testing.T) {
	validTTL := 15 * time.Second
	validRenew := 5 * time.Second
	validTimeout := 10 * time.Second
	if err := validateLockConfig(validTTL, validRenew, validTimeout); err != nil {
		t.Fatalf("expected valid lock config, got %v", err)
	}
	cases := []struct {
		name    string
		ttl     time.Duration
		renew   time.Duration
		timeout time.Duration
	}{
		{"zero ttl", 0, validRenew, validTimeout},
		{"zero timeout", validTTL, validRenew, 0},
		{"zero renew", validTTL, 0, validTimeout},
		{"renew equals ttl", validTTL, validTTL, validTimeout},
	}
	for _, tc := range cases {
		if err := validateLockConfig(tc.ttl, tc.renew, tc.timeout); err == nil {
			t.Fatalf("%s: expected invalid lock config", tc.name)
		}
	}
}

type retryLockProvider struct {
	failures int
	attempts int
	err      error
}

func (p *retryLockProvider) Acquire(context.Context, string, time.Duration) (womslock.Lock, error) {
	p.attempts++
	if p.err != nil {
		return nil, p.err
	}
	if p.attempts <= p.failures {
		return nil, womslock.ErrNotAcquired
	}
	return noopLock{}, nil
}

type noopLock struct{}

func (noopLock) Refresh(context.Context, time.Duration) error { return nil }
func (noopLock) Release(context.Context) error                { return nil }

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func mustMarshalJob(t *testing.T, job domain.ScheduleJob) []byte {
	t.Helper()
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	return payload
}

type fakeJobExecutor struct {
	failedJobs    int
	failedJobID   string
	failedMessage string
	retryJobs     int
	retryJobID    string
	retryMessage  string
	lockedJobs    int
	processFn     func(context.Context, domain.ScheduleJob, int) error
}

func (e *fakeJobExecutor) markJobFailed(_ context.Context, jobID, message string) error {
	e.failedJobs++
	e.failedJobID = jobID
	e.failedMessage = message
	return nil
}

func (e *fakeJobExecutor) markJobRetry(_ context.Context, jobID, message string) error {
	e.retryJobs++
	e.retryJobID = jobID
	e.retryMessage = message
	return nil
}

func (e *fakeJobExecutor) processJobLocked(ctx context.Context, job domain.ScheduleJob, maxRetries int) error {
	e.lockedJobs++
	if e.processFn != nil {
		return e.processFn(ctx, job, maxRetries)
	}
	return nil
}

type singleLockProvider struct {
	lock     womslock.Lock
	attempts int
}

func (p *singleLockProvider) Acquire(context.Context, string, time.Duration) (womslock.Lock, error) {
	p.attempts++
	return p.lock, nil
}

type recordingLock struct {
	mu         sync.Mutex
	refreshes  int
	releases   int
	refreshErr error
}

func (l *recordingLock) Refresh(context.Context, time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refreshes++
	return l.refreshErr
}

func (l *recordingLock) Release(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.releases++
	return nil
}

type fakeLockedJobStore struct {
	found          bool
	status         domain.ScheduleJobStatus
	attempt        int
	persistErr     error
	runningCalls   int
	persistCalls   int
	retryCalls     int
	retryMessage   string
	failedCalls    int
	failedMessage  string
	failedReason   string
	completedCalls int
}

func (s *fakeLockedJobStore) jobStatus(context.Context, string) (domain.ScheduleJobStatus, bool, error) {
	return s.status, s.found, nil
}

func (s *fakeLockedJobStore) markRunning(context.Context, string) (int, error) {
	s.runningCalls++
	s.status = domain.JobRunning
	s.attempt++
	return s.attempt, nil
}

func (s *fakeLockedJobStore) persist(context.Context, domain.ScheduleJob) error {
	s.persistCalls++
	return s.persistErr
}

func (s *fakeLockedJobStore) markRetryAfterRun(_ context.Context, _ string, message string) error {
	s.retryCalls++
	s.status = domain.JobQueued
	s.retryMessage = message
	return nil
}

func (s *fakeLockedJobStore) markFailedAfterRun(_ context.Context, _ string, message, reason string) error {
	s.failedCalls++
	s.status = domain.JobFailed
	s.failedMessage = message
	s.failedReason = reason
	return nil
}

func (s *fakeLockedJobStore) markCompleted(context.Context, domain.ScheduleJob) error {
	s.completedCalls++
	s.status = domain.JobCompleted
	return nil
}
