package main

import (
	"bytes"
	"sync"
)

// LogHub is a small in-memory log aggregator for the UI.
// It implements HubIface and io.Writer.
type LogHub struct {
	mu    sync.RWMutex
	lines []string
	subs  map[chan string]struct{}
	max   int
	buf   bytes.Buffer
}

func NewLogHub(maxLines int) *LogHub {
	if maxLines <= 0 {
		maxLines = 5000
	}
	return &LogHub{
		lines: make([]string, 0, maxLines),
		subs:  make(map[chan string]struct{}),
		max:   maxLines,
	}
}

func (h *LogHub) Write(p []byte) (n int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Collect partial lines between writes.
	h.buf.Write(p)
	for {
		b := h.buf.Bytes()
		idx := bytes.IndexByte(b, '\n')
		if idx < 0 {
			break
		}
		line := string(bytes.TrimRight(b[:idx], "\r"))
		h.buf.Next(idx + 1)
		h.appendLineLocked(line)
	}
	return len(p), nil
}

func (h *LogHub) appendLineLocked(line string) {
	if line == "" {
		return
	}
	h.lines = append(h.lines, line)
	if len(h.lines) > h.max {
		h.lines = h.lines[len(h.lines)-h.max:]
	}
	for ch := range h.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (h *LogHub) Snapshot() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, len(h.lines))
	copy(out, h.lines)
	return out
}

func (h *LogHub) Subscribe(n int) (<-chan string, func()) {
	if n <= 0 {
		n = 1000
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
		delete(h.subs, ch)
		h.mu.Unlock()
		close(ch)
	}
	return ch, unsub
}

func (h *LogHub) Clear() {
	h.mu.Lock()
	h.lines = h.lines[:0]
	h.buf.Reset()
	h.mu.Unlock()
}
