package api

import (
	"context"
	"testing"

	"github.com/d11nn/woms/internal/domain"
)

func TestNoopScheduleJobPublisher(t *testing.T) {
	pub := NoopScheduleJobPublisher{}
	if err := pub.PublishScheduleJob(context.Background(), domain.ScheduleJob{ID: "job-noop"}); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if err := pub.Close(); err != nil {
		t.Errorf("expected no error on close, got %v", err)
	}
}

func TestKafkaScheduleJobPublisher_ClosedWriter(t *testing.T) {
	pub := NewKafkaScheduleJobPublisher([]string{"localhost:9092"}, "test-topic")
	
	// Close it immediately
	if err := pub.Close(); err != nil {
		t.Errorf("expected no error closing uninitialized writer, got %v", err)
	}

	// Try to publish - should fail because it is closed
	err := pub.PublishScheduleJob(context.Background(), domain.ScheduleJob{ID: "job-1"})
	if err == nil {
		t.Error("expected error publishing to a closed publisher, got nil")
	}
}
