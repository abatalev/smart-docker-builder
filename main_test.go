package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/abatalev/smartdockerbuild/internal/hash"
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
	assertions := require.New(t)
	for n, variant := range variants {
		assertions.Equal(variant.result, getImageName(variant.value), n)
	}
}

type FileInfo struct {
	FileName string
	Content  string
}

func TestCalcHash(t *testing.T) {
	variants := []struct {
		dockerFile       string
		files            []FileInfo
		resultFiles      []string
		resultFileHashes []string
		result           string
	}{
		{
			dockerFile: "Dockerfile",
			files: []FileInfo{
				{FileName: "Dockerfile", Content: "FROM alpine:latest"},
			},
			resultFiles:      []string{"Dockerfile"},
			resultFileHashes: []string{"Dockerfile 47e00eaac9bbcb2764b1608a7e17ceba481cdcbb"},
			result:           "3ec05fad",
		},
		{
			dockerFile: "Dockerfile",
			files: []FileInfo{
				{FileName: "Dockerfile", Content: "FROM alpine:latest\nCOPY app.sh /opt/app/app.sh"},
				{FileName: "app.sh", Content: "#!/bin/sh\necho \"app.sh\""},
			},
			resultFiles: []string{"Dockerfile", "app.sh"},
			resultFileHashes: []string{
				"Dockerfile b0653e09926cf5a68a555c872b6bcf7a989c30ed",
				"app.sh 9af03f3e531f0be9ae7c8379767f41089f474da4"},
			result: "084fb41f",
		},
		{
			dockerFile: "Dockerfile",
			files: []FileInfo{
				{FileName: "Dockerfile", Content: "FROM alpine:latest\nCOPY app.sh /opt/app/app.sh"},
				{FileName: "app.sh", Content: "#!/bin/bash\necho \"app.sh\""},
			},
			resultFiles: []string{"Dockerfile", "app.sh"},
			resultFileHashes: []string{
				"Dockerfile b0653e09926cf5a68a555c872b6bcf7a989c30ed",
				"app.sh ed889a228dbaf4aa6febb6e2a8b72e1de11ce17f"},
			result: "8ce8f799",
		},
		{
			dockerFile: "Dockerfile",
			files: []FileInfo{
				{FileName: "Dockerfile", Content: "FROM alpine:latest\nCOPY *.go /opt/app/"},
				{FileName: "file1.go", Content: "aaa"},
				{FileName: "file2.go", Content: "bbb"},
			},
			resultFiles: []string{"Dockerfile", "file1.go", "file2.go"},
			resultFileHashes: []string{
				"Dockerfile 39ba751caaafe740a9cf96e5d5a2f9a793fe98a0",
				"file1.go 7e240de74fb1ed08fa08d38063f6a6a91462a815",
				"file2.go 5cb138284d431abd6a053a56625ec088bfb88912"},
			result: "3f9974ce",
		},
		// {
		// 	dockerFile: "Dockerfile",
		// 	files: []FileInfo{
		// 		{FileName: "Dockerfile", Content: "FROM alpine:latest\nCOPY *.go /opt/app/"},
		// 		{FileName: "file1.go", Content: "aaa"},
		// 		{FileName: "file2.go", Content: "bbb"},
		// 		{FileName: ".dockerignore", Content: "file2.go"},
		// 	},
		// 	resultFiles: []string{"Dockerfile", "file1.go"},
		// 	resultFileHashes: []string{
		// 		"Dockerfile 39ba751caaafe740a9cf96e5d5a2f9a793fe98a0",
		// 		"file1.go 7e240de74fb1ed08fa08d38063f6a6a91462a815"},
		// 	result: "111",
		// },
		// {
		// 	dockerFile: "Dockerfile",
		// 	files: []FileInfo{
		// 		{FileName: "Dockerfile", Content: "FROM alpine:latest\nCOPY *.go /opt/app/"},
		// 		{FileName: "file1.go", Content: "aaa"},
		// 		{FileName: "file2.go", Content: "bbb"},
		// 		{FileName: "Dockerfile.dockerignore", Content: "file2.go"},
		// 	},
		// 	resultFiles: []string{"Dockerfile", "file1.go"},
		// 	resultFileHashes: []string{
		// 		"Dockerfile 39ba751caaafe740a9cf96e5d5a2f9a793fe98a0",
		// 		"file1.go 7e240de74fb1ed08fa08d38063f6a6a91462a815"},
		// 	result: "111",
		// },
	}
	assertions := require.New(t)
	for n, variant := range variants {
		dirName := filepath.Join(t.TempDir(), "v"+strconv.Itoa(n))
		assertions.NoError(os.Mkdir(dirName, 0755))
		for _, f := range variant.files {
			fileName := filepath.Join(dirName, f.FileName)
			assertions.NoError(os.WriteFile(fileName, []byte(f.Content), 0644))
		}
		files := GetFilesForDockerFile(dirName, variant.dockerFile)
		assertions.ElementsMatch(variant.resultFiles, files, n)
		assertions.ElementsMatch(variant.resultFileHashes, hash.CalcHashes(dirName, files), n)
		assertions.Equal(variant.result, calcHash(dirName, variant.dockerFile), n)
	}
}

func TestQuote(t *testing.T) {
	variants := []struct {
		value  string
		result string
	}{
		{value: "a", result: "a"},
		{value: "a a", result: "\"a a\""},
	}
	assertions := require.New(t)
	for n, variant := range variants {
		assertions.Equal(variant.result, quote(variant.value), n)
	}
}

func TestMerge(t *testing.T) {
	variants := []struct {
		value1 string
		value2 string
		result string
	}{
		{value1: "a", value2: "b", result: "a b"},
		{value1: "b", value2: "a a", result: "b \"a a\""},
	}
	assertions := require.New(t)
	for n, variant := range variants {
		assertions.Equal(variant.result, merge(variant.value1, variant.value2), n)
	}
}

func TestSemanticVersion(t *testing.T) {
	variants := []struct {
		value  string
		result []string
	}{
		{value: "1", result: []string{"1"}},
		{value: "1.0.2", result: []string{"1", "1.0", "1.0.2"}},
	}
	assertions := require.New(t)
	for n, variant := range variants {
		assertions.ElementsMatch(variant.result, SemanticVersion(variant.value), n)
	}
}
