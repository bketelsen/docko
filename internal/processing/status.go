package processing

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// MaxSubscribers limits the number of concurrent SSE connections
const MaxSubscribers = 100

// StatusUpdate represents a processing status change
type StatusUpdate struct {
	DocumentID uuid.UUID
	Status     string // pending, processing, completed, failed
	Error      string // error message if failed
	QueueName  string // queue name for queue-level SSE events
}

// BulkProgressUpdate represents aggregate progress for bulk uploads
type BulkProgressUpdate struct {
	Processed int
	Total     int
}

// StatusBroadcaster manages SSE subscriptions for status updates
type StatusBroadcaster struct {
	subscribers map[chan StatusUpdate]struct{}
	mu          sync.RWMutex
}

// NewStatusBroadcaster creates a new StatusBroadcaster
func NewStatusBroadcaster() *StatusBroadcaster {
	return &StatusBroadcaster{
		subscribers: make(map[chan StatusUpdate]struct{}),
	}
}

// Subscribe returns a channel that receives status updates
// The channel is automatically removed when the context is cancelled
func (b *StatusBroadcaster) Subscribe(ctx context.Context) <-chan StatusUpdate {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check subscriber limit
	if len(b.subscribers) >= MaxSubscribers {
		slog.Warn("max subscribers reached, rejecting new subscription",
			"current", len(b.subscribers),
			"max", MaxSubscribers)
		return nil
	}

	ch := make(chan StatusUpdate, 16) // Buffer to prevent blocking
	b.subscribers[ch] = struct{}{}

	slog.Debug("subscriber added", "total", len(b.subscribers))

	// Start goroutine to clean up when context is cancelled
	go func() {
		<-ctx.Done()
		b.Unsubscribe(ch)
	}()

	return ch
}

// Unsubscribe removes a subscription
func (b *StatusBroadcaster) Unsubscribe(ch <-chan StatusUpdate) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Find and remove the channel (convert read-only back to bidirectional for map lookup)
	for subscriber := range b.subscribers {
		if subscriber == ch {
			delete(b.subscribers, subscriber)
			close(subscriber)
			slog.Debug("subscriber removed", "total", len(b.subscribers))
			return
		}
	}
}

// Broadcast sends a status update to all subscribers
// Uses non-blocking send to prevent slow subscribers from blocking others
func (b *StatusBroadcaster) Broadcast(update StatusUpdate) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.subscribers) == 0 {
		return
	}

	slog.Debug("broadcasting status update",
		"doc_id", update.DocumentID,
		"status", update.Status,
		"subscribers", len(b.subscribers))

	for ch := range b.subscribers {
		select {
		case ch <- update:
			// Message sent successfully
		default:
			// Channel buffer full, skip this subscriber
			slog.Warn("subscriber buffer full, dropping message",
				"doc_id", update.DocumentID)
		}
	}
}

// SubscriberCount returns the current number of active subscribers
func (b *StatusBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
