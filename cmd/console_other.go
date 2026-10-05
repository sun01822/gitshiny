//go:build !windows

package cmd

// ownsConsole is only meaningful on Windows; see console_windows.go.
func ownsConsole() bool { return false }
