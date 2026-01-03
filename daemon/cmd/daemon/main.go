package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func logsDir() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	ld := filepath.Join(dir, "logs")
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
	name := "deskcontrol-" + time.Now().Format("2006-01-02") + ".log"
	return filepath.Join(ld, name), nil
}

func InitFileLogging(hub io.Writer) (func(), error) {
	p, err := currentLogFilePath()
	if err != nil {
		return func() {}, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return func() {}, err
	}

	mw := io.MultiWriter(os.Stdout, f)
	if hub != nil {
		mw = io.MultiWriter(os.Stdout, f, hub)
	}
	log.SetOutput(mw)

	return func() { _ = f.Close() }, nil
}

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
