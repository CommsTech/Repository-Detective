//go:build !windows

package scanners

import (
	"os/exec"
	"syscall"
)

// setProcessGroup gives the scanner its own process group so a timeout can
// terminate the tool and every child it spawned.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
