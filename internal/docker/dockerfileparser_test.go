package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFrom(t *testing.T) {
	assertions := require.New(t)
	assertions.Len(parseFrom("from a", []ProjectDependency{}), 1)
	assertions.Len(parseFrom("copy a b", []ProjectDependency{}), 0)
}

func TestParseCopy(t *testing.T) {
	variants := []struct {
		exists   bool
		result   string
		argument string
	}{
		{
			exists:   true,
			result:   "a",
			argument: "copy a b",
		},
		{
			exists:   false,
			result:   "",
			argument: "copy --from=x a b",
		},
		{
			exists:   true,
			result:   "**/*",
			argument: "copy ./ /opt",
		},
		{
			exists:   true,
			result:   "/opt/**/*",
			argument: "copy /opt/ /opt",
		},
	}

	assertions := require.New(t)
	for _, variant := range variants {
		list := parseCopy(variant.argument, []string{})
		if variant.exists {
			assertions.Len(list, 1)
			assertions.Equal(variant.result, list[0])
		} else {
			assertions.Len(list, 0)
		}
	}
}
