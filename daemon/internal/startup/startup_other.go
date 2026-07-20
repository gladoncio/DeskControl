//go:build unix

package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var xdgAutostartDir = func() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "autostart")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autostart")
}()

func SetEnabled(appName string, enabled bool, args string) error {
	if err := os.MkdirAll(xdgAutostartDir, 0755); err != nil {
		return err
	}

	desktopPath := filepath.Join(xdgAutostartDir, appName+".desktop")

	if !enabled {
		if err := os.Remove(desktopPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable: %w", err)
	}
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return fmt.Errorf("empty executable path")
	}

	args = strings.TrimSpace(args)

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s %s
Terminal=false
X-GNOME-Autostart-enabled=true
`, appName, exe, args)

	if err := os.WriteFile(desktopPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write desktop file: %w", err)
	}
	return nil
}

func IsEnabled(appName string) (bool, string, error) {
	desktopPath := filepath.Join(xdgAutostartDir, appName+".desktop")
	data, err := os.ReadFile(desktopPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, strings.TrimSpace(string(data)), nil
}


