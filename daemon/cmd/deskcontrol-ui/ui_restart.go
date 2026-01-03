package main

import (
	"os"
	"os/exec"
	"strings"
)

// restartSelf restarts the current executable.
// On Windows we delay start ~1s to allow ports to be released.
func restartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// Preserve args
	args := os.Args[1:]

	// Windows delay using cmd timeout
	// cmd /C "timeout /T 1 /NOBREAK >NUL & "<exe>" <args...>"
	quotedExe := `"` + exe + `"`
	quotedArgs := ""
	if len(args) > 0 {
		// minimal quoting (good enough for typical flags)
		quotedArgs = " " + strings.Join(args, " ")
	}

	cmdline := `timeout /T 1 /NOBREAK >NUL & ` + quotedExe + quotedArgs
	cmd := exec.Command("cmd.exe", "/C", cmdline)
	// detach
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return err
	}

	os.Exit(0)
	return nil
}
