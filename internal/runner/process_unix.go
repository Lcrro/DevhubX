//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func command(line string) *exec.Cmd {
	c := exec.Command("/bin/sh", "-c", line)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return c
}
func contain(c *exec.Cmd) (func() error, error) {
	return func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }, nil
}
