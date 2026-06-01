//go:build !windows

package lab

import "os/exec"

func configureNewConsole(_ *exec.Cmd) {
}
