package loghub

import (
	"bytes"
	"sync"
)

type Hub struct {
	mu       sync.Mutex
	buf      []string
	capacity int
	subs     map[chan string]struct{}
}

func New(capacity int) *Hub {
	if capacity <= 0 {
		capacity = 2000
	}
	return &Hub{
		capacity: capacity,
		subs:     make(map[chan string]struct{}),
	}
}

// Write implementa io.Writer para poder hacer log.SetOutput(hub)
func (h *Hub) Write(p []byte) (n int, err error) {
	// log package suele mandar líneas completas, pero por si vienen trozos:
	lines := bytes.Split(p, []byte{'\n'})
	for _, ln := range lines {
		if len(ln) == 0 {
			continue
		}
		h.add(string(ln))
	}
	return len(p), nil
}

func (h *Hub) add(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buf = append(h.buf, line)
	if len(h.buf) > h.capacity {
		h.buf = h.buf[len(h.buf)-h.capacity:]
	}

	for ch := range h.subs {
		select {
		case ch <- line:
		default:
			// si el UI está lento, no bloqueamos
		}
	}
}

func (h *Hub) Snapshot() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.buf))
	copy(out, h.buf)
	return out
}

func (h *Hub) Subscribe(buffer int) (<-chan string, func()) {
	if buffer <= 0 {
		buffer = 200
	}
	ch := make(chan string, buffer)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	unsub := func() {
		h.mu.Lock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
	return ch, unsub
}

func (h *Hub) Clear() {
	h.mu.Lock()
	h.buf = nil
	h.mu.Unlock()
}
