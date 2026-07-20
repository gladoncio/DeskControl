//go:build linux

package input

import (
	"errors"
	"log"
	"sync"
	"time"
)

var (
	captureMu     sync.Mutex
	captureCh     chan CaptureResult
	captureActive bool
	captureStop   chan struct{}
)

func captureNextKey(timeoutMs int) (CaptureResult, error) {
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}

	captureMu.Lock()
	if captureActive {
		captureMu.Unlock()
		return CaptureResult{}, errors.New("capture already active")
	}
	captureActive = true
	captureCh = make(chan CaptureResult, 1)
	captureStop = make(chan struct{})
	captureMu.Unlock()

	defer func() {
		captureMu.Lock()
		captureActive = false
		captureCh = nil
		close(captureStop)
		captureStop = nil
		captureMu.Unlock()
	}()

	started := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	go func() {
		close(started)
		runCaptureLoop(captureCh, captureStop, errCh)
	}()

	<-started

	timeout := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timeout.Stop()

	select {
	case err := <-errCh:
		return CaptureResult{}, err
	case res := <-captureCh:
		return res, nil
	case <-timeout.C:
		return CaptureResult{}, errors.New("capture timeout")
	}
}

func runCaptureLoop(ch chan<- CaptureResult, stop <-chan struct{}, errCh chan<- error) {
	if tryEvdevCapture(ch, stop) {
		return
	}
	if tryDbusCapture(ch, stop) {
		return
	}
	if tryX11Capture(ch, stop) {
		return
	}
	select {
	case errCh <- errors.New("capture not supported on this desktop (no evdev, D-Bus/At-SPI nor X11 available)"):
	default:
	}
}

func tryDbusCapture(ch chan<- CaptureResult, stop <-chan struct{}) bool {
	impl := &dbusCapture{}
	if err := impl.init(); err != nil {
		log.Printf("[capture] D-Bus capture not available: %v", err)
		return false
	}
	defer impl.close()
	return impl.run(ch, stop)
}

func tryX11Capture(ch chan<- CaptureResult, stop <-chan struct{}) bool {
	impl := &x11Capture{}
	if err := impl.init(); err != nil {
		log.Printf("[capture] X11 capture not available: %v", err)
		return false
	}
	defer impl.close()
	return impl.run(ch, stop)
}
