package main

import (
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type logRing struct {
	mu    sync.RWMutex
	buf   []string
	start int
	count int
	limit int
	dirty bool
}

func newLogRing(limit int) *logRing {
	return &logRing{
		buf:   make([]string, limit),
		limit: limit,
	}
}

func (r *logRing) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.start = 0
	r.count = 0
	r.dirty = true
}

func (r *logRing) appendMany(lines []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, line := range lines {
		if r.limit == 0 {
			return
		}
		if r.count < r.limit {
			idx := (r.start + r.count) % r.limit
			r.buf[idx] = line
			r.count++
		} else {
			r.buf[r.start] = line
			r.start = (r.start + 1) % r.limit
		}
	}
	if len(lines) > 0 {
		r.dirty = true
	}
}

func (r *logRing) len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}

func (r *logRing) at(i int) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if i < 0 || i >= r.count {
		return ""
	}
	idx := (r.start + i) % r.limit
	return r.buf[idx]
}

func (r *logRing) snapshot() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, r.count)
	for i := 0; i < r.count; i++ {
		idx := (r.start + i) % r.limit
		out = append(out, r.buf[idx])
	}
	return out
}

func (r *logRing) consumeDirty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dirty {
		r.dirty = false
		return true
	}
	return false
}

func buildLogsTab(a fyne.App, w fyne.Window, hub HubIface, state *UIState, maxUILines int, tick time.Duration) fyne.CanvasObject {
	ring := newLogRing(maxUILines)

	initial := hub.Snapshot()
	if len(initial) > maxUILines {
		initial = initial[len(initial)-maxUILines:]
	}
	ring.appendMany(initial)
	_ = ring.consumeDirty()

	status := widget.NewLabel("Mostrando últimas " + itoa(maxUILines) + " líneas")

	logList := widget.NewList(
		func() int { return ring.len() },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(ring.at(i))
		},
	)

	btnClear := widget.NewButton("Clear", func() {
		hub.Clear()
		ring.clear()
		logList.Refresh()
	})

	btnCopyLast := widget.NewButton("Copy last 100", func() {
		lines := ring.snapshot()
		w.Clipboard().SetContent(strings.Join(lines, "\n"))
	})

	ch, unsub := hub.Subscribe(1000)
	_ = unsub

	pending := make(chan string, 5000)
	go func() {
		for ln := range ch {
			select {
			case pending <- ln:
			default:
			}
		}
	}()

	if tick <= 0 {
		tick = 250 * time.Millisecond
	}
	ticker := time.NewTicker(tick)

	go func() {
		for range ticker.C {
			a.Driver().DoFromGoroutine(func() {
				batch := make([]string, 0, 200)
				for i := 0; i < 200; i++ {
					select {
					case ln := <-pending:
						batch = append(batch, ln)
					default:
						goto APPLY
					}
				}
			APPLY:
				if len(batch) == 0 {
					return
				}

				// Siempre actualizamos el ring (para que al abrir veas lo último)
				ring.appendMany(batch)

				// Si la UI está “pausada” (tray/oculta), no refrescamos nada
				if !state.ShowUI {
					_ = ring.consumeDirty()
					return
				}

				if ring.consumeDirty() {
					logList.Refresh()

					// Siempre seguir el último
					last := ring.len() - 1
					if last >= 0 {
						logList.Select(last)
					}
				}
			}, false)
		}
	}()

	return container.NewBorder(
		container.NewHBox(btnClear, btnCopyLast),
		status, nil, nil,
		logList,
	)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}
	return sign + string(b[i:])
}
