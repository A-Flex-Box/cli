package agent

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EventBus provides a simple in-memory event bus for agent communication.
// It does not depend on external services like Redis.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]chan Event
	buffer      int
	logger      *zap.Logger
	running     bool
	stopCh      chan struct{}
}

// EventBusOption is a functional option for configuring EventBus.
type EventBusOption func(*EventBus)

// WithBuffer sets the buffer size for subscriber channels.
func WithBuffer(size int) EventBusOption {
	return func(b *EventBus) {
		b.buffer = size
	}
}

// WithLogger sets the logger for the event bus.
func WithLogger(logger *zap.Logger) EventBusOption {
	return func(b *EventBus) {
		b.logger = logger
	}
}

// NewEventBus creates a new EventBus instance.
func NewEventBus(opts ...EventBusOption) *EventBus {
	b := &EventBus{
		subscribers: make(map[EventType][]chan Event),
		buffer:      100,
		stopCh:      make(chan struct{}),
		logger:      zap.NewNop(),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Subscribe creates a subscription for events of a specific type.
// Returns a channel that will receive events.
func (b *EventBus) Subscribe(eventType EventType) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, b.buffer)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)

	if b.logger != nil {
		b.logger.Debug("event bus subscription created",
			zap.String("eventType", string(eventType)),
			zap.Int("totalSubscribers", len(b.subscribers[eventType])))
	}

	return ch
}

// SubscribeAll creates a subscription for all events.
func (b *EventBus) SubscribeAll() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, b.buffer)

	// Subscribe to all known event types
	for _, eventType := range []EventType{
		EventTaskCreated,
		EventTaskParsed,
		EventTaskReviewed,
		EventTaskExecuting,
		EventTaskCompleted,
		EventTaskFailed,
		EventAgentStatus,
		EventError,
	} {
		b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	}

	if b.logger != nil {
		b.logger.Debug("global subscription created")
	}

	return ch
}

// Publish sends an event to all subscribers.
func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	subscribers, ok := b.subscribers[event.Type]
	if !ok {
		if b.logger != nil {
			b.logger.Debug("no subscribers for event type",
				zap.String("eventType", string(event.Type)))
		}
		return
	}

	for _, ch := range subscribers {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel full, log warning
			if b.logger != nil {
				b.logger.Warn("subscriber channel full, event dropped",
					zap.String("eventType", string(event.Type)),
					zap.String("eventID", event.ID))
			}
		}
	}

	if b.logger != nil {
		b.logger.Debug("event published",
			zap.String("eventType", string(event.Type)),
			zap.String("eventID", event.ID),
			zap.Int("subscribers", len(subscribers)))
	}
}

// Unsubscribe removes a subscription.
func (b *EventBus) Unsubscribe(eventType EventType, ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers := b.subscribers[eventType]
	for i, subCh := range subscribers {
		// Compare the receive-only channel with the send channel
		if subCh == ch {
			b.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			close(subCh)
			break
		}
	}
}

// Start begins event processing.
func (b *EventBus) Start(ctx context.Context) error {
	b.mu.Lock()
	b.running = true
	b.mu.Unlock()

	if b.logger != nil {
		b.logger.Info("event bus started")
	}

	return nil
}

// Stop gracefully shuts down the event bus.
func (b *EventBus) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.running = false
	close(b.stopCh)

	// Close all subscriber channels
	for eventType, subscribers := range b.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
		delete(b.subscribers, eventType)
	}

	if b.logger != nil {
		b.logger.Info("event bus stopped")
	}

	return nil
}

// Emit creates and publishes an event.
func (b *EventBus) Emit(eventType EventType, source string, taskID string, payload map[string]interface{}) {
	event := Event{
		ID:        generateID(),
		Type:      eventType,
		Source:    source,
		TaskID:    taskID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	b.Publish(event)
}