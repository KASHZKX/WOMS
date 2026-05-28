package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseAPIConfigDefaults(t *testing.T) {
	config := parseAPIConfig(func(string) string { return "" })

	if config.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want default", config.HTTPAddr)
	}
	if config.JWTSecret != "change-me-in-production" {
		t.Fatalf("JWTSecret = %q, want default", config.JWTSecret)
	}
	if config.StoreMode != "memory" {
		t.Fatalf("StoreMode = %q, want memory", config.StoreMode)
	}
	if !config.DemoSeedData {
		t.Fatal("DemoSeedData = false, want true")
	}
	if !config.KafkaPublishEnabled {
		t.Fatal("KafkaPublishEnabled = false, want true")
	}
	if !reflect.DeepEqual(config.KafkaBrokers, []string{"kafka:9092"}) {
		t.Fatalf("KafkaBrokers = %#v, want default broker", config.KafkaBrokers)
	}
	if config.DependencyTimeout != 2*time.Minute {
		t.Fatalf("DependencyTimeout = %s, want 2m", config.DependencyTimeout)
	}
	if config.DependencyInterval != 2*time.Second {
		t.Fatalf("DependencyInterval = %s, want 2s", config.DependencyInterval)
	}
}

func TestParseAPIConfigTrimsAndParsesValues(t *testing.T) {
	values := map[string]string{
		"HTTP_ADDR":                        " :9090 ",
		"JWT_SECRET":                       " secret ",
		"API_STORE":                        " postgres ",
		"DATABASE_URL":                     " postgres://woms ",
		"DEMO_SEED_DATA":                   " false ",
		"KAFKA_PUBLISH_ENABLED":            " false ",
		"KAFKA_BROKERS":                    " kafka-0:9092, , kafka-1:9092 ",
		"KAFKA_SCHEDULE_TOPIC":             " jobs ",
		"AUTH_SESSION_STORE":               " redis ",
		"REDIS_ADDR":                       " redis:6379 ",
		"CORS_ALLOWED_ORIGIN":              " https://woms.example ",
		"AUTH_MODE":                        " ingress ",
		"API_DEPENDENCY_RETRY_TIMEOUT_MS":  "1500",
		"API_DEPENDENCY_RETRY_INTERVAL_MS": "250",
	}
	config := parseAPIConfig(func(key string) string { return values[key] })

	if config.HTTPAddr != ":9090" || config.JWTSecret != "secret" || config.StoreMode != "postgres" {
		t.Fatalf("string values were not trimmed: %+v", config)
	}
	if config.DatabaseURL != "postgres://woms" || config.AuthSessionStore != "redis" || config.RedisAddr != "redis:6379" {
		t.Fatalf("connection values were not parsed: %+v", config)
	}
	if config.DemoSeedData || config.KafkaPublishEnabled {
		t.Fatalf("boolean toggles were not parsed: demo=%t kafka=%t", config.DemoSeedData, config.KafkaPublishEnabled)
	}
	if !reflect.DeepEqual(config.KafkaBrokers, []string{"kafka-0:9092", "kafka-1:9092"}) {
		t.Fatalf("KafkaBrokers = %#v", config.KafkaBrokers)
	}
	if config.KafkaScheduleTopic != "jobs" || config.CORSAllowedOrigin != "https://woms.example" || config.AuthMode != "ingress" {
		t.Fatalf("server options were not parsed: %+v", config)
	}
	if config.DependencyTimeout != 1500*time.Millisecond || config.DependencyInterval != 250*time.Millisecond {
		t.Fatalf("durations = %s/%s, want 1500ms/250ms", config.DependencyTimeout, config.DependencyInterval)
	}
}

func TestParseAPIConfigFallsBackForMalformedAndNegativeDurations(t *testing.T) {
	values := map[string]string{
		"API_DEPENDENCY_RETRY_TIMEOUT_MS":  "not-an-int",
		"API_DEPENDENCY_RETRY_INTERVAL_MS": "-1",
	}
	config := parseAPIConfig(func(key string) string { return values[key] })

	if config.DependencyTimeout != 2*time.Minute {
		t.Fatalf("DependencyTimeout = %s, want fallback", config.DependencyTimeout)
	}
	if config.DependencyInterval != 2*time.Second {
		t.Fatalf("DependencyInterval = %s, want fallback", config.DependencyInterval)
	}
}
