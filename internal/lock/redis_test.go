package lock

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
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

func startMockRedisServer(t *testing.T, handler func(cmd string) string) (string, func()) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock redis: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if !strings.HasPrefix(line, "*") {
						return
					}
					countStr := strings.TrimSpace(line[1:])
					count, err := strconv.Atoi(countStr)
					if err != nil {
						return
					}
					var args []string
					for i := 0; i < count; i++ {
						lenLine, err := reader.ReadString('\n')
						if err != nil {
							return
						}
						if !strings.HasPrefix(lenLine, "$") {
							return
						}
						length, _ := strconv.Atoi(strings.TrimSpace(lenLine[1:]))
						buf := make([]byte, length+2)
						if _, err := io.ReadFull(reader, buf); err != nil {
							return
						}
						args = append(args, string(buf[:length]))
					}
					if len(args) == 0 {
						return
					}
					cmd := strings.ToUpper(args[0])
					resp := handler(cmd)
					_, _ = c.Write([]byte(resp))
				}
			}(conn)
		}
	}()

	return l.Addr().String(), func() {
		cancel()
		_ = l.Close()
		wg.Wait()
	}
}

func TestMockRedisAcquireRefreshRelease(t *testing.T) {
	addr, cleanup := startMockRedisServer(t, func(cmd string) string {
		switch cmd {
		case "PING":
			return "+PONG\r\n"
		case "SET":
			return "+OK\r\n"
		case "EVAL":
			return ":1\r\n"
		default:
			return "-ERR unknown command\r\n"
		}
	})
	defer cleanup()

	p := NewRedisProvider(addr)
	ctx := context.Background()

	if err := p.Ping(ctx); err != nil {
		t.Fatalf("expected ping success, got %v", err)
	}

	lock, err := p.Acquire(ctx, "test-key", 10*time.Second)
	if err != nil {
		t.Fatalf("expected acquire success, got %v", err)
	}

	if err := lock.Refresh(ctx, 10*time.Second); err != nil {
		t.Fatalf("expected refresh success, got %v", err)
	}

	if err := lock.Release(ctx); err != nil {
		t.Fatalf("expected release success, got %v", err)
	}
}

func TestMockRedisAcquireAcquireLockErrors(t *testing.T) {
	// 1. Invalid TTL
	p := NewRedisProvider("localhost:6379")
	_, err := p.Acquire(context.Background(), "test-key", 0)
	if err == nil {
		t.Error("expected error for zero TTL, got nil")
	}

	// 2. Unexpected Ping response
	addr, cleanup := startMockRedisServer(t, func(cmd string) string {
		return "+NOT_PONG\r\n"
	})
	defer cleanup()

	pWrong := NewRedisProvider(addr)
	err = pWrong.Ping(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unexpected redis PING response") {
		t.Errorf("expected unexpected ping response error, got %v", err)
	}

	// 3. Acquire fails when SET returns ErrNotAcquired (nil bulk string)
	addr2, cleanup2 := startMockRedisServer(t, func(cmd string) string {
		return "$-1\r\n"
	})
	defer cleanup2()

	pFail := NewRedisProvider(addr2)
	_, err = pFail.Acquire(context.Background(), "test-key", time.Second)
	if !errors.Is(err, ErrNotAcquired) {
		t.Errorf("expected ErrNotAcquired, got %v", err)
	}

	// 4. SET returns something other than OK (e.g. +NOT_OK)
	addr3, cleanup3 := startMockRedisServer(t, func(cmd string) string {
		return "+NOT_OK\r\n"
	})
	defer cleanup3()

	pFail2 := NewRedisProvider(addr3)
	_, err = pFail2.Acquire(context.Background(), "test-key", time.Second)
	if !errors.Is(err, ErrNotAcquired) {
		t.Errorf("expected ErrNotAcquired when SET is not OK, got %v", err)
	}

	// 5. EVAL returns 0 on Refresh
	addr4, cleanup4 := startMockRedisServer(t, func(cmd string) string {
		if cmd == "SET" {
			return "+OK\r\n"
		}
		return ":0\r\n"
	})
	defer cleanup4()

	pFail3 := NewRedisProvider(addr4)
	lock, err := pFail3.Acquire(context.Background(), "test-key", time.Second)
	if err != nil {
		t.Fatalf("expected acquire success, got %v", err)
	}
	err = lock.Refresh(context.Background(), time.Second)
	if !errors.Is(err, ErrNotAcquired) {
		t.Errorf("expected ErrNotAcquired on Refresh returning 0, got %v", err)
	}
}

func TestReadValueRESPTypes(t *testing.T) {
	// Test error response prefix '-'
	_, err := readValue(bufio.NewReader(strings.NewReader("-ERR error message\r\n")))
	if err == nil || err.Error() != "ERR error message" {
		t.Errorf("expected error 'ERR error message', got %v", err)
	}

	// Test integer prefix ':'
	val, err := readValue(bufio.NewReader(strings.NewReader(":42\r\n")))
	if err != nil || val != "42" {
		t.Errorf("expected '42', got val=%q, err=%v", val, err)
	}

	// Test invalid prefix
	_, err = readValue(bufio.NewReader(strings.NewReader("*3\r\n")))
	if err == nil || !strings.Contains(err.Error(), "unsupported redis response prefix") {
		t.Errorf("expected unsupported prefix error, got %v", err)
	}

	// Test invalid bulk string length conversion
	_, err = readValue(bufio.NewReader(strings.NewReader("$abc\r\n")))
	if err == nil {
		t.Error("expected bulk string conversion error, got nil")
	}
}

func TestRandomValueHelper(t *testing.T) {
	val, err := randomValue()
	if err != nil {
		t.Fatalf("unexpected randomValue error: %v", err)
	}
	if len(val) == 0 {
		t.Error("expected non-empty random value string")
	}
}

