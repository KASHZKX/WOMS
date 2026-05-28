package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/d11nn/woms/internal/api"
	"github.com/d11nn/woms/internal/startup"
)

func main() {
	config := parseAPIConfig(os.Getenv)
	var store api.Store
	if config.StoreMode == "postgres" {
		ctx, cancel := context.WithTimeout(context.Background(), config.DependencyTimeout)
		var postgresStore *api.PostgresStore
		err := startup.RetryDependency(ctx, "postgres store", config.DependencyInterval, log.Printf, func(ctx context.Context) error {
			var err error
			postgresStore, err = api.NewPostgresStoreContext(ctx, config.DatabaseURL, config.DemoSeedData)
			return err
		})
		cancel()
		if err != nil {
			log.Fatalf("postgres store failed: %v", err)
		}
		defer postgresStore.Close()
		store = postgresStore
	} else {
		memoryStore := api.NewMemoryStore()
		if config.DemoSeedData {
			memoryStore = api.NewDemoMemoryStore()
		}
		store = memoryStore
	}
	publisher := api.ScheduleJobPublisher(api.NoopScheduleJobPublisher{})
	if config.KafkaPublishEnabled {
		ctx, cancel := context.WithTimeout(context.Background(), config.DependencyTimeout)
		if err := startup.RetryDependency(ctx, "kafka broker", config.DependencyInterval, log.Printf, func(ctx context.Context) error {
			return startup.PingAnyTCP(ctx, config.KafkaBrokers)
		}); err != nil {
			cancel()
			log.Fatalf("kafka broker failed: %v", err)
		}
		cancel()
		publisher = api.NewKafkaScheduleJobPublisher(config.KafkaBrokers, config.KafkaScheduleTopic)
		defer publisher.Close()
	}
	tokenSessions := api.TokenSessionStore(api.NoopTokenSessionStore{})
	if config.AuthSessionStore == "redis" {
		if config.RedisAddr == "" {
			log.Fatal("AUTH_SESSION_STORE=redis requires REDIS_ADDR")
		}
		redisSessions := api.NewRedisTokenSessionStore(config.RedisAddr)
		ctx, cancel := context.WithTimeout(context.Background(), config.DependencyTimeout)
		if err := startup.RetryDependency(ctx, "redis auth session store", config.DependencyInterval, log.Printf, func(ctx context.Context) error {
			return redisSessions.Ping(ctx)
		}); err != nil {
			cancel()
			log.Fatalf("redis auth session store failed: %v", err)
		}
		cancel()
		tokenSessions = redisSessions
		defer tokenSessions.Close()
	}

	server := &http.Server{
		Addr: config.HTTPAddr,
		Handler: api.NewServerWithPublisherAndConfig(config.JWTSecret, store, publisher, api.ServerConfig{
			TokenSessions:     tokenSessions,
			CORSAllowedOrigin: config.CORSAllowedOrigin,
			AuthMode:          config.AuthMode,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("woms api listening on %s", config.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api server failed: %v", err)
	}
}

type apiConfig struct {
	HTTPAddr            string
	JWTSecret           string
	StoreMode           string
	DatabaseURL         string
	DemoSeedData        bool
	KafkaPublishEnabled bool
	KafkaBrokers        []string
	KafkaScheduleTopic  string
	AuthSessionStore    string
	RedisAddr           string
	CORSAllowedOrigin   string
	AuthMode            string
	DependencyTimeout   time.Duration
	DependencyInterval  time.Duration
}

func parseAPIConfig(lookup func(string) string) apiConfig {
	return apiConfig{
		HTTPAddr:            envLookup(lookup, "HTTP_ADDR", ":8080"),
		JWTSecret:           envLookup(lookup, "JWT_SECRET", "change-me-in-production"),
		StoreMode:           envLookup(lookup, "API_STORE", "memory"),
		DatabaseURL:         envLookup(lookup, "DATABASE_URL", ""),
		DemoSeedData:        envLookup(lookup, "DEMO_SEED_DATA", "true") != "false",
		KafkaPublishEnabled: envLookup(lookup, "KAFKA_PUBLISH_ENABLED", "true") != "false",
		KafkaBrokers:        startup.SplitCSV(envLookup(lookup, "KAFKA_BROKERS", "kafka:9092")),
		KafkaScheduleTopic:  envLookup(lookup, "KAFKA_SCHEDULE_TOPIC", "woms.schedule.jobs"),
		AuthSessionStore:    envLookup(lookup, "AUTH_SESSION_STORE", ""),
		RedisAddr:           envLookup(lookup, "REDIS_ADDR", ""),
		CORSAllowedOrigin:   envLookup(lookup, "CORS_ALLOWED_ORIGIN", "*"),
		AuthMode:            envLookup(lookup, "AUTH_MODE", "local"),
		DependencyTimeout:   envDurationLookup(lookup, "API_DEPENDENCY_RETRY_TIMEOUT_MS", 2*time.Minute),
		DependencyInterval:  envDurationLookup(lookup, "API_DEPENDENCY_RETRY_INTERVAL_MS", 2*time.Second),
	}
}

func env(key, fallback string) string {
	return envLookup(os.Getenv, key, fallback)
}

func envLookup(lookup func(string) string, key, fallback string) string {
	value := strings.TrimSpace(lookup(key))
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	return envDurationLookup(os.Getenv, key, fallback)
}

func envDurationLookup(lookup func(string) string, key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(lookup(key))
	if value == "" {
		return fallback
	}
	millis, err := strconv.Atoi(value)
	if err != nil || millis < 0 {
		return fallback
	}
	return time.Duration(millis) * time.Millisecond
}
