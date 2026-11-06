//go:build unix

package git

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const commandWaitDelay = 2 * time.Second

func configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}

		return err
	}
	cmd.WaitDelay = commandWaitDelay
}
