//go:build !unix

package git

import (
	"os/exec"
	"time"
)

const commandWaitDelay = 2 * time.Second

func configureCommand(cmd *exec.Cmd) {
	cmd.WaitDelay = commandWaitDelay
}
