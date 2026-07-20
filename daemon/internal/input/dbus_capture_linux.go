//go:build linux

package input

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

type dbusCapture struct {
	conn     *dbus.Conn
	a11yConn *dbus.Conn
	mu       sync.Mutex
}

func (d *dbusCapture) init() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	d.conn = conn

	var addr string
	addrCall := conn.Object("org.a11y.Bus", "/org/a11y/bus").Call("org.a11y.Bus.GetAddress", 0)
	select {
	case <-addrCall.Done:
		if err := addrCall.Store(&addr); err != nil {
			d.conn.Close()
			return err
		}
	case <-time.After(2 * time.Second):
		d.conn.Close()
		return errors.New("a11y bus GetAddress timeout")
	}
	if addr == "" {
		d.conn.Close()
		return errors.New("empty a11y bus address")
	}

	a11yConn, err := dbus.Dial(addr)
	if err != nil {
		d.conn.Close()
		return err
	}

	// Must call Hello on the a11y bus before using it
	helloCall := a11yConn.BusObject().Call("org.freedesktop.DBus.Hello", 0)
	select {
	case <-helloCall.Done:
		if helloCall.Err != nil {
			a11yConn.Close()
			d.conn.Close()
			return helloCall.Err
		}
	case <-time.After(2 * time.Second):
		a11yConn.Close()
		d.conn.Close()
		return errors.New("a11y bus Hello timeout")
	}
	d.a11yConn = a11yConn
	return nil
}

func (d *dbusCapture) close() {
	if d.a11yConn != nil {
		d.a11yConn.Close()
	}
	if d.conn != nil {
		d.conn.Close()
	}
}

func (d *dbusCapture) run(ch chan<- CaptureResult, stop <-chan struct{}) bool {
	matchCall := d.a11yConn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"interface='org.a11y.atspi.Event.Keyboard',member='KeyPressed'")
	select {
	case <-matchCall.Done:
		if matchCall.Err != nil {
			log.Printf("[capture] dbus AddMatch error: %v", matchCall.Err)
			return false
		}
	case <-time.After(2 * time.Second):
		log.Printf("[capture] dbus AddMatch timeout")
		return false
	case <-stop:
		return true
	}

	signals := make(chan *dbus.Signal, 100)
	d.a11yConn.Signal(signals)
	defer d.a11yConn.RemoveSignal(signals)

	for {
		select {
		case <-stop:
			return true
		case sig := <-signals:
			if sig == nil {
				return false
			}
			if sig.Name != "org.a11y.atspi.Event.Keyboard" {
				continue
			}
			if len(sig.Body) < 4 {
				continue
			}

			eventStr, ok := sig.Body[0].(string)
			if !ok || eventStr != "KeyPressed" {
				continue
			}

			kc, ok := sig.Body[2].(int32)
			if !ok || kc <= 0 {
				continue
			}

			modifiers, _ := sig.Body[3].(int32)

			mods := dbusModsToStrings(int32(modifiers))

			res := CaptureResult{
				Key: KeySpec{
					VK:   linuxCodeToVK(uint16(kc)),
					Scan: uint16(kc),
					Ext:  false,
				},
				Mods: mods,
			}

			select {
			case ch <- res:
				return true
			default:
			}
		}
	}
}

func linuxCodeToVK(lc uint16) uint16 {
	for vk, code := range vkToLinuxMap {
		if code == lc {
			return vk
		}
	}
	if lc >= 30 && lc <= 55 {
		return lc - 30 + 'A'
	}
	if lc >= 2 && lc <= 11 {
		return lc - 2 + '0'
	}
	if lc >= 59 && lc <= 68 {
		return 0x70 + (lc - 59)
	}
	if lc == 87 {
		return 0x7A
	}
	if lc == 88 {
		return 0x7B
	}
	return lc
}

func dbusModsToStrings(mods int32) []string {
	var result []string
	if mods&(1<<0) != 0 {
		result = append(result, "shift")
	}
	if mods&(1<<1) != 0 {
		result = append(result, "ctrl")
	}
	if mods&(1<<2) != 0 {
		result = append(result, "alt")
	}
	if mods&(1<<3) != 0 {
		result = append(result, "meta")
	}
	return result
}
