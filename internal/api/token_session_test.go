package api

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/d11nn/woms/internal/auth"
	"github.com/d11nn/woms/internal/domain"
)

func TestRedisCommandEncodesRESPArrays(t *testing.T) {
	got := string(redisCommand("SET", "woms:auth:token:test", "1", "EX", "60"))
	want := "*5\r\n$3\r\nSET\r\n$20\r\nwoms:auth:token:test\r\n$1\r\n1\r\n$2\r\nEX\r\n$2\r\n60\r\n"
	if got != want {
		t.Fatalf("unexpected RESP command:\nwant %q\ngot  %q", want, got)
	}
}

func TestReadRedisValueMapsNilBulkToMissingSession(t *testing.T) {
	_, err := readRedisValue(bufio.NewReader(strings.NewReader("$-1\r\n")))
	if err != ErrTokenSessionNotFound {
		t.Fatalf("expected ErrTokenSessionNotFound, got %v", err)
	}
}

func TestReadRedisValueParsesSupportedReplies(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple string", input: "+PONG\r\n", want: "PONG"},
		{name: "integer", input: ":42\r\n", want: "42"},
		{name: "bulk", input: "$5\r\nvalue\r\n", want: "value"},
		{name: "empty bulk", input: "$0\r\n\r\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readRedisValue(bufio.NewReader(strings.NewReader(tt.input)))
			if err != nil {
				t.Fatalf("readRedisValue returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected value: want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestReadRedisValueReturnsDeterministicErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
		wantIs    error
	}{
		{name: "protocol error", input: "-ERR invalid password\r\n", wantError: "ERR invalid password"},
		{name: "nil bulk", input: "$-1\r\n", wantIs: ErrTokenSessionNotFound},
		{name: "malformed length", input: "$abc\r\n", wantError: `strconv.Atoi: parsing "abc": invalid syntax`},
		{name: "partial bulk read", input: "$5\r\nva", wantIs: io.ErrUnexpectedEOF},
		{name: "malformed bulk terminator", input: "$5\r\nvaluexx", wantError: "malformed redis bulk string terminator"},
		{name: "unsupported prefix", input: "*1\r\n+OK\r\n", wantError: `unsupported redis response prefix '*'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readRedisValue(bufio.NewReader(strings.NewReader(tt.input)))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantIs != nil {
				if !errors.Is(err, tt.wantIs) {
					t.Fatalf("expected error matching %v, got %v", tt.wantIs, err)
				}
				return
			}
			if err.Error() != tt.wantError {
				t.Fatalf("unexpected error: want %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}

func TestRedisTokenSessionStoreIntegrationLifecycle(t *testing.T) {
	store := newRedisTokenSessionIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	if !store.TracksSessions() {
		t.Fatal("redis token session store should track sessions")
	}

	token := redisTokenSessionIntegrationToken(t)
	defer deleteRedisTokenSession(t, store, token)
	claims := redisTokenSessionClaims(time.Now().Add(5 * time.Minute))

	if err := store.Save(ctx, token, claims); err != nil {
		t.Fatalf("save token session: %v", err)
	}
	if err := store.Verify(ctx, token, claims); err != nil {
		t.Fatalf("verify token session: %v", err)
	}

	revoked, err := store.Revoke(ctx, token)
	if err != nil {
		t.Fatalf("revoke token session: %v", err)
	}
	if !revoked {
		t.Fatal("expected Revoke to report a deleted session")
	}
	if err := store.Verify(ctx, token, claims); !errors.Is(err, ErrTokenSessionNotFound) {
		t.Fatalf("expected revoked token to be missing, got %v", err)
	}

	revoked, err = store.Revoke(ctx, token)
	if err != nil {
		t.Fatalf("revoke missing token session: %v", err)
	}
	if revoked {
		t.Fatal("expected Revoke to report no deleted session after prior revoke")
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close token session store: %v", err)
	}
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("ping redis after close: %v", err)
	}
}

func TestRedisTokenSessionStoreIntegrationRejectsEmptyAndExpiredTokens(t *testing.T) {
	store := newRedisTokenSessionIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer store.Close()

	validClaims := redisTokenSessionClaims(time.Now().Add(time.Minute))
	if err := store.Save(ctx, "", validClaims); !errors.Is(err, ErrTokenSessionNotFound) {
		t.Fatalf("expected empty token save to fail as missing session, got %v", err)
	}
	if err := store.Verify(ctx, "", validClaims); !errors.Is(err, ErrTokenSessionNotFound) {
		t.Fatalf("expected empty token verify to fail as missing session, got %v", err)
	}

	expiredToken := redisTokenSessionIntegrationToken(t)
	if err := store.Save(ctx, expiredToken, redisTokenSessionClaims(time.Now().Add(-time.Second))); !errors.Is(err, ErrTokenSessionNotFound) {
		t.Fatalf("expected expired token save to fail as missing session, got %v", err)
	}
}

func TestRedisTokenSessionStoreIntegrationExpiresSavedToken(t *testing.T) {
	store := newRedisTokenSessionIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	defer store.Close()

	token := redisTokenSessionIntegrationToken(t)
	defer deleteRedisTokenSession(t, store, token)
	claims := redisTokenSessionClaims(time.Now().Add(1500 * time.Millisecond))

	if err := store.Save(ctx, token, claims); err != nil {
		t.Fatalf("save expiring token session: %v", err)
	}
	if err := store.Verify(ctx, token, claims); err != nil {
		t.Fatalf("verify expiring token session before ttl: %v", err)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		err := store.Verify(ctx, token, claims)
		if errors.Is(err, ErrTokenSessionNotFound) {
			return
		}
		if err != nil {
			t.Fatalf("verify expiring token session: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("token session did not expire before deadline")
}

func TestRedisTokenSessionStoreIntegrationReconnectsAfterCommandError(t *testing.T) {
	store := newRedisTokenSessionIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		t.Fatalf("initial ping redis: %v", err)
	}
	if _, err := store.command(ctx, "WOMS_UNKNOWN_COMMAND"); err == nil {
		t.Fatal("expected invalid Redis command to fail")
	}
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("ping redis after command error: %v", err)
	}
}

func newRedisTokenSessionIntegrationStore(t *testing.T) *RedisTokenSessionStore {
	t.Helper()
	addr := redisTokenSessionIntegrationAddr(t)
	store := NewRedisTokenSessionStore(addr)
	store.timeout = 500 * time.Millisecond
	return store
}

func redisTokenSessionIntegrationAddr(t *testing.T) string {
	t.Helper()
	requireRedisTokenSessionIntegration(t)
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		t.Skip("REDIS_ADDR is not set; skipping manual Redis token session integration test")
	}
	return addr
}

func requireRedisTokenSessionIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("WOMS_INTEGRATION_TESTS") != "1" {
		t.Skip("WOMS_INTEGRATION_TESTS=1 is not set; skipping manual Redis token session integration test")
	}
}

func redisTokenSessionIntegrationToken(t *testing.T) string {
	t.Helper()
	name := strings.NewReplacer("/", ":", " ", "_").Replace(t.Name())
	return fmt.Sprintf("woms-test-token-%s-%d", name, time.Now().UnixNano())
}

func redisTokenSessionClaims(expires time.Time) auth.Claims {
	return auth.Claims{
		Subject: "redis-session-test-user",
		Role:    domain.RoleSales,
		Expires: expires.Unix(),
	}
}

func deleteRedisTokenSession(t *testing.T, store *RedisTokenSessionStore, token string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := store.command(ctx, "DEL", tokenSessionKey(token)); err != nil {
		t.Logf("cleanup redis token session %q: %v", token, err)
	}
}
