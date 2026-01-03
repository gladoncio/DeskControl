package loghub

import (
	"bytes"
	"sync"
)

// Hub guarda líneas de log en memoria y permite suscribirse a nuevas líneas.
// Además implementa io.Writer para poder conectar el package log.
type Hub struct {
	mu       sync.Mutex
	buf      []string
	capacity int
	subs     map[chan string]struct{}

	// para juntar fragmentos hasta encontrar '\n'
	partial bytes.Buffer
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

// Add agrega una línea al buffer y la emite a subs.
func (h *Hub) Add(line string) {
	if line == "" {
		return
	}
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
			// si el consumidor se atrasa, no bloqueamos
		}
	}
}

// Snapshot retorna copia de las líneas actuales.
func (h *Hub) Snapshot() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.buf))
	copy(out, h.buf)
	return out
}

// Subscribe devuelve un canal con líneas nuevas + función para desuscribir.
func (h *Hub) Subscribe(n int) (<-chan string, func()) {
	if n <= 0 {
		n = 2000
	}
	ch := make(chan string, n)

	h.mu.Lock()
	if h.subs == nil {
		h.subs = make(map[chan string]struct{})
	}
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
	h.partial.Reset()
	h.mu.Unlock()
}

// Write implementa io.Writer: convierte bytes a líneas (por \n) y las guarda en el hub.
func (h *Hub) Write(p []byte) (int, error) {
	h.mu.Lock()
	h.partial.Write(p)

	for {
		b := h.partial.Bytes()
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			break
		}
		line := string(bytes.TrimRight(b[:i], "\r"))
		h.partial.Next(i + 1)

		// Add() requiere lock, pero ya estamos con lock:
		if line != "" {
			h.buf = append(h.buf, line)
			if len(h.buf) > h.capacity {
				h.buf = h.buf[len(h.buf)-h.capacity:]
			}
			for ch := range h.subs {
				select {
				case ch <- line:
				default:
				}
			}
		}
	}

	h.mu.Unlock()
	return len(p), nil
}
