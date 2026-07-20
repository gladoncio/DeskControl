package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// restartSelf restarts the current executable with a brief delay.
func restartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	args := os.Args[1:]

	if runtime.GOOS == "windows" {
		quotedExe := `"` + exe + `"`
		quotedArgs := ""
		if len(args) > 0 {
			quotedArgs = " " + strings.Join(args, " ")
		}
		cmdline := `timeout /T 1 /NOBREAK >NUL & ` + quotedExe + quotedArgs
		cmd := exec.Command("cmd.exe", "/C", cmdline)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
		if err := cmd.Start(); err != nil {
			return err
		}
	} else {
		allArgs := append([]string{"-c", "sleep 1 && exec $@", "--"}, exe)
		allArgs = append(allArgs, args...)
		cmd := exec.Command("sh", allArgs...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
		if err := cmd.Start(); err != nil {
			return err
		}
	}

	os.Exit(0)
	return nil
}
