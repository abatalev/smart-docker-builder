package osrunner

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	assertions := require.New(t)
	assertions.Equal("a b c", strings.Join(Command("a", "b", "c").Args, " "))
}

func TestStartAndWait(t *testing.T) {
	assertions := require.New(t)
	cmd := exec.Command("uname")
	out, _ := cmd.StdoutPipe()
	res, err := StartAndWait([]*exec.Cmd{cmd}, out)
	assertions.NoError(err)
	assertions.Equal("Linux", strings.TrimSpace(string(res)))
}
