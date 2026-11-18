//go:build windows

package scanners

import "os/exec"

func setProcessGroup(*exec.Cmd) {}

// killProcessGroup is a no-op on Windows; CommandContext still terminates the
// direct child process.
func killProcessGroup(*exec.Cmd) {}
