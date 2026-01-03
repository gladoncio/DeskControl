package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// exeDir returns the directory where the current executable is located.
// If it fails, it falls back to current working directory.
func exeDir() string {
	p, err := os.Executable()
	if err != nil || p == "" {
		if wd, err2 := os.Getwd(); err2 == nil && wd != "" {
			return wd
		}
		return "."
	}
	return filepath.Dir(p)
}

func logsDir() (string, error) {
	ld := filepath.Join(exeDir(), "logs")
	if err := os.MkdirAll(ld, 0o755); err != nil {
		return "", err
	}
	return ld, nil
}

func currentLogFilePath() (string, error) {
	ld, err := logsDir()
	if err != nil {
		return "", err
	}
	// Un log por día
	name := "deskcontrol-" + time.Now().Format("2006-01-02") + ".log"
	return filepath.Join(ld, name), nil
}

// InitEarlyLogging must be called BEFORE starting UI.
// It wires standard log to: stdout + file + hub (if provided).
func InitEarlyLogging(hub io.Writer) (func(), error) {
	p, err := currentLogFilePath()
	if err != nil {
		return func() {}, err
	}

	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return func() {}, fmt.Errorf("no puedo abrir log file %s: %w", p, err)
	}

	// stdout puede no existir en GUI, pero no molesta. Lo importante: file + hub.
	out := io.MultiWriter(f)
	if hub != nil {
		out = io.MultiWriter(f, hub)
	}
	log.SetOutput(out)

	return func() { _ = f.Close() }, nil
}

// PurgeOldLogs deletes files in ./logs older than retentionDays.
// If retentionDays <= 0, it does nothing.
func PurgeOldLogs(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	ld, err := logsDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(ld)
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(ld, e.Name()))
		}
	}
	return nil
}

func DeleteAllLogs() error {
	ld, err := logsDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(ld)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		_ = os.Remove(filepath.Join(ld, e.Name()))
	}
	return nil
}
