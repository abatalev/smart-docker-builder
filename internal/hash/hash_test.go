package hash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalcHashBytes(t *testing.T) {
	assertions := require.New(t)
	assertions.Equal("86f7e437faa5a7fce15d1ddcb9eaeaea377667b8", calcHashBytes([]byte("a")))
}

func TestCalcHashFile(t *testing.T) {
	assertions := require.New(t)
	fileName := filepath.Join(t.TempDir(), "a")
	assertions.NoError(os.WriteFile(fileName, []byte("a"), 0600))
	assertions.Equal("86f7e437faa5a7fce15d1ddcb9eaeaea377667b8", calcHashFile(fileName))
}
