package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	os.Exit(runHealthcheck(os.Getenv, http.DefaultClient, os.Stderr))
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func runHealthcheck(lookup func(string) string, client httpDoer, stderr io.Writer) int {
	url := envLookup(lookup, "HEALTHCHECK_URL", "http://127.0.0.1:8080/readyz")
	timeout, err := time.ParseDuration(envLookup(lookup, "HEALTHCHECK_TIMEOUT", "2s"))
	if err != nil {
		fmt.Fprintf(stderr, "invalid HEALTHCHECK_TIMEOUT: %v\n", err)
		return 2
	}

	err = checkHealth(context.Background(), client, url, timeout)
	switch {
	case err == nil:
		return 0
	case errors.Is(err, errInvalidHealthcheckURL):
		fmt.Fprintf(stderr, "invalid HEALTHCHECK_URL: %v\n", err)
		return 2
	case errors.Is(err, errUnhealthyStatus):
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	default:
		fmt.Fprintf(stderr, "healthcheck request failed: %v\n", err)
		return 1
	}
}

var (
	errInvalidHealthcheckURL = errors.New("invalid healthcheck url")
	errUnhealthyStatus       = errors.New("healthcheck unhealthy status")
)

func checkHealth(parent context.Context, client httpDoer, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", errInvalidHealthcheckURL, err)
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: healthcheck returned status %d", errUnhealthyStatus, res.StatusCode)
	}
	return nil
}

func env(key, fallback string) string {
	return envLookup(os.Getenv, key, fallback)
}

func envLookup(lookup func(string) string, key, fallback string) string {
	value := lookup(key)
	if value == "" {
		return fallback
	}
	return value
}
