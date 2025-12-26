//go:build windows

package startup

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

func SetEnabled(appName string, enabled bool, args string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open run key: %w", err)
	}
	defer k.Close()

	if !enabled {
		_ = k.DeleteValue(appName)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable: %w", err)
	}
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return errors.New("empty executable path")
	}

	cmd := fmt.Sprintf(`"%s" %s`, exe, strings.TrimSpace(args))
	if err := k.SetStringValue(appName, cmd); err != nil {
		return fmt.Errorf("set run value: %w", err)
	}
	return nil
}

func IsEnabled(appName string) (bool, string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, "", fmt.Errorf("open run key: %w", err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue(appName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("get run value: %w", err)
	}
	return true, val, nil
}
