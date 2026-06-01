//go:build windows

package lab

import (
	"os/exec"
	"syscall"
)

const createNewConsole = 0x00000010

func configureNewConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewConsole,
	}
}
