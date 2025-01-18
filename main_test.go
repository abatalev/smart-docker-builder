package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetImageName(t *testing.T) {
	variants := []struct {
		value  string
		result string
	}{
		{value: "xxx/Dockerfile", result: "xxx"},
		{value: "yyy/xxx/Dockerfile", result: "xxx"},
		{value: "Dockerfile.xxx", result: "xxx"},
		{value: "yyy/Dockerfile.xxx", result: "xxx"},
		{value: "xxx.Dockerfile", result: "xxx"},
		{value: "yyy/xxx.Dockerfile", result: "xxx"},
	}
	for n, variant := range variants {
		name := getImageName(variant.value)
		require.Equal(t, variant.result, name, n)
	}
}
