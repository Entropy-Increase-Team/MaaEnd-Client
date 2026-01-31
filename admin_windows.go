//go:build windows
// +build windows

package main

import (
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func ensureAdmin() error {
	elevated, err := isWindowsElevated()
	if err != nil {
		return err
	}
	if elevated {
		return nil
	}
	if err := relaunchAsAdmin(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func isWindowsElevated() (bool, error) {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false, err
	}
	defer token.Close()

	type tokenElevation struct {
		TokenIsElevated uint32
	}
	var elevation tokenElevation
	var outLen uint32
	err := windows.GetTokenInformation(
		token,
		windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&outLen,
	)
	if err != nil {
		return false, err
	}
	return elevation.TokenIsElevated != 0, nil
}

func relaunchAsAdmin() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	args := os.Args[1:]
	argList := ""
	if len(args) > 0 {
		quoted := make([]string, len(args))
		for i, a := range args {
			quoted[i] = quotePowerShell(a)
		}
		argList = strings.Join(quoted, ", ")
	}

	cwd, _ := os.Getwd()

	parts := []string{
		"Start-Process",
		"-FilePath", quotePowerShell(exePath),
	}
	if argList != "" {
		parts = append(parts, "-ArgumentList", argList)
	}
	if cwd != "" {
		parts = append(parts, "-WorkingDirectory", quotePowerShell(cwd))
	}
	parts = append(parts, "-Verb", "RunAs")

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", strings.Join(parts, " "))
	return cmd.Start()
}

func quotePowerShell(value string) string {
	// PowerShell 单引号转义: ' -> ''
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
