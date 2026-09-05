package scanners_test

import (
	"fmt"
	"runtime"
	"strings"
)

func testShell(script string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "powershell", []string{"-NoProfile", "-Command", script}
	}
	return "sh", []string{"-c", script}
}

func testWriteScript(output string, exitCode int) (string, []string) {
	if runtime.GOOS == "windows" {
		script := fmt.Sprintf("[Console]::Out.Write(@'%s'@); exit %d", output, exitCode)
		return testShell(script)
	}
	escaped := strings.ReplaceAll(output, `'`, `'\''`)
	script := fmt.Sprintf("printf '%%s' '%s'; exit %d", escaped, exitCode)
	return testShell(script)
}

func testSleepScript(seconds int) (string, []string) {
	if runtime.GOOS == "windows" {
		return testShell(fmt.Sprintf("Start-Sleep -Seconds %d", seconds))
	}
	return testShell(fmt.Sprintf("sleep %d", seconds))
}

