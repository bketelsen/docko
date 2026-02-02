package queue

import (
	"context"
	"testing"
	"time"

	"docko/internal/database/sqlc"
)

func TestNextRetryDelay(t *testing.T) {
	q := &Queue{
		config: Config{
			BaseRetryDelay: time.Second,
			MaxRetryDelay:  time.Hour,
		},
	}

	// Run multiple times to verify randomness stays within bounds
	for i := 0; i < 100; i++ {
		// Test attempt 0: should be between 0 and 1s
		delay0 := q.nextRetryDelay(0)
		if delay0 < 0 || delay0 > time.Second {
			t.Errorf("attempt 0: delay %v not in [0, 1s]", delay0)
		}

		// Test attempt 2: should be between 0 and 4s
		delay2 := q.nextRetryDelay(2)
		if delay2 < 0 || delay2 > 4*time.Second {
			t.Errorf("attempt 2: delay %v not in [0, 4s]", delay2)
		}

		// Test attempt 10: should cap at MaxRetryDelay
		delay10 := q.nextRetryDelay(10)
		if delay10 < 0 || delay10 > time.Hour {
			t.Errorf("attempt 10: delay %v not in [0, 1h]", delay10)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.PollInterval != time.Second {
		t.Errorf("expected PollInterval 1s, got %v", cfg.PollInterval)
	}
	if cfg.WorkerCount != 4 {
		t.Errorf("expected WorkerCount 4, got %d", cfg.WorkerCount)
	}
	if cfg.BaseRetryDelay != time.Second {
		t.Errorf("expected BaseRetryDelay 1s, got %v", cfg.BaseRetryDelay)
	}
	if cfg.MaxRetryDelay != time.Hour {
		t.Errorf("expected MaxRetryDelay 1h, got %v", cfg.MaxRetryDelay)
	}
	if cfg.VisibilityTimeout != 5*time.Minute {
		t.Errorf("expected VisibilityTimeout 5m, got %v", cfg.VisibilityTimeout)
	}
}

func TestNewWithDefaults(t *testing.T) {
	// Test that New() fills in zero values with defaults
	q := New(nil, Config{})

	if q.config.PollInterval != time.Second {
		t.Errorf("expected PollInterval 1s, got %v", q.config.PollInterval)
	}
	if q.config.WorkerCount != 4 {
		t.Errorf("expected WorkerCount 4, got %d", q.config.WorkerCount)
	}
	if q.config.BaseRetryDelay != time.Second {
		t.Errorf("expected BaseRetryDelay 1s, got %v", q.config.BaseRetryDelay)
	}
	if q.config.MaxRetryDelay != time.Hour {
		t.Errorf("expected MaxRetryDelay 1h, got %v", q.config.MaxRetryDelay)
	}
	if q.handlers == nil {
		t.Error("expected handlers map to be initialized")
	}
	if q.stop == nil {
		t.Error("expected stop channel to be initialized")
	}
}

func TestRegisterHandler(t *testing.T) {
	q := New(nil, Config{})

	q.RegisterHandler("test_job", func(_ context.Context, _ *sqlc.Job) error {
		return nil
	})

	if _, ok := q.handlers["test_job"]; !ok {
		t.Error("expected handler to be registered")
	}
}
