//go:build linux

package input

import (
	"errors"

	"github.com/godbus/dbus/v5"
)

type windowEntry struct {
	id    uintptr
	title string
	pid   uint32
}

func dbusListApps() ([]AppInfo, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var apps []AppInfo

	kwinApps, kwinErr := dbusKWinList(conn)
	if kwinErr == nil && len(kwinApps) > 0 {
		apps = append(apps, kwinApps...)
	}

	gnomeApps, gnomeErr := dbusGnomeList(conn)
	if gnomeErr == nil && len(gnomeApps) > 0 {
		apps = append(apps, gnomeApps...)
	}

	if len(apps) == 0 {
		if kwinErr != nil && gnomeErr != nil {
			return nil, errors.New("no DE detected via D-Bus (KWin or GNOME)")
		}
	}

	return apps, nil
}

func dbusKWinList(conn *dbus.Conn) ([]AppInfo, error) {
	obj := conn.Object("org.kde.KWin", "/KWin")
	var windows []dbus.Variant
	err := obj.Call("org.kde.KWin.getWindowInfoList", 0).Store(&windows)
	if err != nil {
		return nil, err
	}

	var apps []AppInfo
	for _, v := range windows {
		m, ok := v.Value().(map[string]dbus.Variant)
		if !ok {
			continue
		}
		title := getVariantString(m, "caption")
		if title == "" {
			continue
		}
		pid := uint32(getVariantInt(m, "pid"))
		apps = append(apps, AppInfo{
			Hwnd:  uintptr(pid),
			PID:   pid,
			Title: title,
		})
	}
	return apps, nil
}

func dbusGnomeList(conn *dbus.Conn) ([]AppInfo, error) {
	obj := conn.Object("org.gnome.Shell", "/org/gnome/Shell")
	var windows []map[string]dbus.Variant
	err := obj.Call("org.gnome.Shell.Eval", 0,
		"global.get_window_actors().map(a => ({title: a.meta_window.get_title(), pid: a.meta_window.get_pid()}))",
	).Store(&windows)
	if err != nil {
		return nil, err
	}

	var apps []AppInfo
	for _, w := range windows {
		title := getVariantString(w, "title")
		if title == "" {
			continue
		}
		pid := uint32(getVariantInt(w, "pid"))
		apps = append(apps, AppInfo{
			Hwnd:  uintptr(pid),
			PID:   pid,
			Title: title,
		})
	}
	return apps, nil
}

func getVariantString(m map[string]dbus.Variant, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.Value().(string); ok {
			return s
		}
	}
	return ""
}

func getVariantInt(m map[string]dbus.Variant, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.Value().(type) {
		case int32:
			return int(val)
		case uint32:
			return int(val)
		case int:
			return val
		}
	}
	return 0
}
