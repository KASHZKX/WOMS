package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunHealthcheckSucceedsOn2xx(t *testing.T) {
	server := newHealthcheckTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var stderr bytes.Buffer
	code := runHealthcheck(lookup(map[string]string{"HEALTHCHECK_URL": server.URL}), server.Client(), &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunHealthcheckRejectsInvalidTimeout(t *testing.T) {
	var stderr bytes.Buffer
	code := runHealthcheck(lookup(map[string]string{"HEALTHCHECK_TIMEOUT": "fast"}), http.DefaultClient, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "invalid HEALTHCHECK_TIMEOUT") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunHealthcheckRejectsInvalidURL(t *testing.T) {
	var stderr bytes.Buffer
	code := runHealthcheck(lookup(map[string]string{"HEALTHCHECK_URL": "://bad"}), http.DefaultClient, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "invalid HEALTHCHECK_URL") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunHealthcheckReportsRequestError(t *testing.T) {
	expected := errors.New("dial failed")
	var stderr bytes.Buffer
	code := runHealthcheck(lookup(map[string]string{"HEALTHCHECK_URL": "http://example.test"}), errorClient{err: expected}, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "healthcheck request failed") || !strings.Contains(stderr.String(), expected.Error()) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunHealthcheckReportsNon2xxStatus(t *testing.T) {
	server := newHealthcheckTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	var stderr bytes.Buffer
	code := runHealthcheck(lookup(map[string]string{"HEALTHCHECK_URL": server.URL}), server.Client(), &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "healthcheck returned status 503") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func lookup(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

type errorClient struct {
	err error
}

func (c errorClient) Do(*http.Request) (*http.Response, error) {
	return nil, c.err
}

func newHealthcheckTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			if strings.Contains(fmt.Sprint(recovered), "operation not permitted") {
				t.Skipf("httptest server is not permitted in this sandbox: %v", recovered)
			}
			panic(recovered)
		}
	}()
	return httptest.NewServer(handler)
}
