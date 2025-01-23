package osrunner

import (
	"io"
	"os/exec"
)

func Command(args ...string) *exec.Cmd {
	return exec.Command(args[0], args[1:]...)
}

func StartAndWait(cmds []*exec.Cmd, cmdOut io.ReadCloser) ([]byte, error) {
	for _, c := range cmds {
		if err := c.Start(); err != nil {
			return nil, err
		}
	}
	res, _ := io.ReadAll(cmdOut)
	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			return nil, err
		}
	}
	return res, nil
}
