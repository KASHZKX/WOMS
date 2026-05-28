package lock

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCommandEncodesRESPArrays(t *testing.T) {
	got := string(command("SET", "woms:locks:schedule-line:A", "value", "NX", "PX", "15000"))
	want := "*6\r\n$3\r\nSET\r\n$26\r\nwoms:locks:schedule-line:A\r\n$5\r\nvalue\r\n$2\r\nNX\r\n$2\r\nPX\r\n$5\r\n15000\r\n"
	if got != want {
		t.Fatalf("unexpected command:\nwant %q\ngot  %q", want, got)
	}
}

func TestReadValueMapsNilBulkToNotAcquired(t *testing.T) {
	_, err := readValue(bufio.NewReader(strings.NewReader("$-1\r\n")))
	if err != ErrNotAcquired {
		t.Fatalf("expected ErrNotAcquired, got %v", err)
	}
}

func TestRedisProviderIntegrationPingAcquireRefreshRelease(t *testing.T) {
	p := newRedisIntegrationProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Ping(ctx); err != nil {
		t.Fatalf("ping redis: %v", err)
	}

	key := redisIntegrationKey(t)
	lock, err := p.Acquire(ctx, key, 400*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	defer lock.Release(context.Background())

	time.Sleep(100 * time.Millisecond)
	if err := lock.Refresh(ctx, 1500*time.Millisecond); err != nil {
		t.Fatalf("refresh lock: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	if _, err := p.Acquire(ctx, key, time.Second); !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("expected refreshed lock to block acquisition, got %v", err)
	}

	if err := lock.Release(ctx); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	reacquired, err := p.Acquire(ctx, key, time.Second)
	if err != nil {
		t.Fatalf("acquire lock after release: %v", err)
	}
	if err := reacquired.Release(ctx); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestRedisProviderIntegrationAcquireSucceedsAfterExpiry(t *testing.T) {
	p := newRedisIntegrationProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := redisIntegrationKey(t)
	lock, err := p.Acquire(ctx, key, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	defer lock.Release(context.Background())

	if _, err := p.Acquire(ctx, key, time.Second); !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("expected locked key acquisition to fail, got %v", err)
	}

	reacquired := acquireAfterExpiry(t, p, key)
	if err := reacquired.Release(ctx); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestRedisProviderIntegrationOwnerMismatchDoesNotRefreshOrRelease(t *testing.T) {
	p := newRedisIntegrationProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := redisIntegrationKey(t)
	lock, err := p.Acquire(ctx, key, 1500*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	defer lock.Release(context.Background())

	wrongOwner := &redisLock{provider: p, key: key, value: "not-the-owner"}
	if err := wrongOwner.Refresh(ctx, time.Second); !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("expected wrong owner refresh to fail, got %v", err)
	}

	if err := wrongOwner.Release(ctx); err != nil {
		t.Fatalf("wrong owner release should be a no-op, got %v", err)
	}
	if _, err := p.Acquire(ctx, key, time.Second); !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("expected wrong owner release to keep lock held, got %v", err)
	}

	if err := lock.Release(ctx); err != nil {
		t.Fatalf("release original lock: %v", err)
	}
	reacquired, err := p.Acquire(ctx, key, time.Second)
	if err != nil {
		t.Fatalf("acquire lock after owner release: %v", err)
	}
	if err := reacquired.Release(ctx); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestRedisProviderIntegrationEmptyAddressErrors(t *testing.T) {
	requireRedisIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	p := NewRedisProvider(" ")
	if err := p.Ping(ctx); err == nil || !strings.Contains(err.Error(), "REDIS_ADDR") {
		t.Fatalf("expected REDIS_ADDR error from Ping, got %v", err)
	}
	if _, err := p.Acquire(ctx, redisIntegrationKey(t), time.Second); err == nil || !strings.Contains(err.Error(), "REDIS_ADDR") {
		t.Fatalf("expected REDIS_ADDR error from Acquire, got %v", err)
	}
}

func newRedisIntegrationProvider(t *testing.T) *RedisProvider {
	t.Helper()
	addr := redisIntegrationAddr(t)
	p := NewRedisProvider(addr)
	p.timeout = 500 * time.Millisecond
	return p
}

func redisIntegrationAddr(t *testing.T) string {
	t.Helper()
	requireRedisIntegration(t)
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		t.Skip("REDIS_ADDR is not set; skipping manual Redis lock integration test")
	}
	return addr
}

func requireRedisIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("WOMS_INTEGRATION_TESTS") != "1" {
		t.Skip("WOMS_INTEGRATION_TESTS=1 is not set; skipping manual Redis lock integration test")
	}
}

func redisIntegrationKey(t *testing.T) string {
	t.Helper()
	name := strings.NewReplacer("/", ":", " ", "_").Replace(t.Name())
	return fmt.Sprintf("woms:test:lock:%s:%d", name, time.Now().UnixNano())
}

func acquireAfterExpiry(t *testing.T, p *RedisProvider, key string) Lock {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		lock, err := p.Acquire(ctx, key, time.Second)
		cancel()
		if err == nil {
			return lock
		}
		lastErr = err
		if !errors.Is(err, ErrNotAcquired) {
			t.Fatalf("acquire after expiry failed unexpectedly: %v", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("lock did not expire before deadline: %v", lastErr)
	return nil
}
