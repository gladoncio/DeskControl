//go:build !windows

package startup

func SetEnabled(appName string, enabled bool, args string) error {
	return nil
}

func IsEnabled(appName string) (bool, string, error) {
	return false, "", nil
}
