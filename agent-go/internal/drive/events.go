package drive

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type Event struct {
	Type      string `json:"type"`
	Payload   any    `json:"payload"`
	Timestamp int64  `json:"timestamp"`
}

type EventBus struct {
	mu          sync.Mutex
	subscribers map[chan Event]struct{}
}

func NewEventBus() *EventBus {
	return &EventBus{subscribers: map[chan Event]struct{}{}}
}

func (b *EventBus) Subscribe(ctx context.Context) <-chan Event {
	ch := make(chan Event, 16)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subscribers, ch)
		close(ch)
		b.mu.Unlock()
	}()
	return ch
}

func (b *EventBus) Publish(eventType string, payload any) {
	if b == nil {
		return
	}
	event := Event{Type: eventType, Payload: payload, Timestamp: time.Now().Unix()}
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (e Event) JSON() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		return []byte(`{"type":"error","timestamp":0}`)
	}
	return data
}
