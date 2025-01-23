package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type FileContent struct {
	name    string
	content string
}

func createFilesContent(dirName string, contents []FileContent) error {
	for _, content := range contents {
		fileName := filepath.Join(dirName, content.name)
		// fmt.Println("writeFile>", fileName)
		if err := os.WriteFile(fileName, []byte(content.content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func createFileContent(dirName string, content FileContent) error {
	fileName := filepath.Join(dirName, content.name)
	return os.WriteFile(fileName, []byte(content.content), 0644)
}

func TestLoadDefs(t *testing.T) {
	assertions := require.New(t)
	assertions.Len(loadAllFacts(), 3)
}

func TestBuildDockerImage(t *testing.T) {
	variants := []struct {
		files      []FileContent
		force      bool
		dockerFile string
		result     int
	}{
		{
			files: []FileContent{
				{name: "v0.sdb.yaml", content: ""},
				{name: "Dockerfile", content: "FROM alpine:3.20.3\n"},
			},
			force:      true,
			dockerFile: "Dockerfile",
			result:     0,
		},
		{
			files: []FileContent{
				{name: "xxx.sdb.yaml", content: ""},
				{name: "Dockerfile.xxx", content: "FROM alpine:latest"},
			},
			force:      true,
			dockerFile: "Dockerfile.xxx",
			result:     0,
		},
	}
	assertions := require.New(t)
	for n, variant := range variants {
		workDir := filepath.Join(t.TempDir(), "v"+strconv.Itoa(n))
		assertions.NoError(os.Mkdir(workDir, 0755))
		assertions.NoError(createFilesContent(workDir, variant.files))
		assertions.Equal(variant.result, BuildDockerImage(workDir, variant.dockerFile, variant.force), n)
	}
}

func TestParseOptions(t *testing.T) {
	variants := []struct {
		args   []string
		result Options
	}{
		{
			args:   []string{"-help"},
			result: Options{isHelp: true},
		},
		{
			args:   []string{"-version"},
			result: Options{isVersion: true},
		},
		{
			args:   []string{"-force", "Dockerfile"},
			result: Options{isForce: true, DockerfileName: "Dockerfile"},
		},
		{
			args:   []string{"Dockerfile"},
			result: Options{DockerfileName: "Dockerfile"},
		},
	}
	for n, variant := range variants {
		assertions := require.New(t)
		options, err := parseOptions(variant.args)
		assertions.NoError(err, n)
		assertions.Equal(variant.result, options, n)
	}
}

func TestCreateDockerTag(t *testing.T) {
	assertions := require.New(t)
	assertions.Equal("docker image tag a:b a:c", strings.Join(createDockerTag("a", "b", "c").Args, " "))
}

func TestDockerImageList(t *testing.T) {
	assertions := require.New(t)
	cmd, _ := DockerImageList()
	assertions.Equal("docker image ls", strings.Join(cmd.Args, " "))
}

func TestLoadConfig(t *testing.T) {
	variants := []struct {
		content FileContent
		isError bool
	}{
		{
			content: FileContent{
				name: "a.sdb.yaml",
				content: "version: \"0.0.1\"\n" +
					"prefix: \"abatalev\"\n" +
					"facts:\n" +
					"  - name: os-name\n" +
					"    args: [\"cat\", \"/etc/os-release\", \"|\", \"awk\", \"-F=\", \"/^ID=/{ print $2 }\"]\n" +
					"  - name: \"os-version\"\n" +
					"    args:\n" +
					"      [\n" +
					"        \"cat\",\n" +
					"        \"/etc/os-release\",\n" +
					"        \"|\",\n" +
					"        \"awk\",\n" +
					"        \"-F=\",\n" +
					"        \"/^VERSION_ID=/{ print $2 }\",\n" +
					"      ]\n" +
					"tags:\n" +
					"  - \"$os-name|-|@os-version\"\n",
			},
			isError: false,
		},
		{
			content: FileContent{
				name:    "a.sdb.yaml",
				content: "hello",
			},
			isError: true,
		},
	}
	assertions := require.New(t)
	for n, variant := range variants {
		dirName := filepath.Join(t.TempDir(), "v"+strconv.Itoa(n))
		assertions.NoError(os.Mkdir(dirName, 0755))
		assertions.NoError(createFileContent(dirName, variant.content))
		fileName := filepath.Join(dirName, variant.content.name)
		_, err := loadConfig(fileName)
		if variant.isError {
			assertions.NotNil(err)
		} else {
			assertions.NoError(err)
		}
	}
}

func TestCheckOldImage(t *testing.T) {
	variants := []struct {
		isForce bool
		isNeed  bool
	}{
		{isForce: true, isNeed: true},
	}
	assertions := require.New(t)
	for n, variant := range variants {
		isNeed, err := checkOldBuild(variant.isForce, "a", "1")
		assertions.NoError(err, n)
		assertions.Equal(variant.isNeed, isNeed, n)
	}
}

func TestAssetDir(t *testing.T) {
	variants := []struct {
		name string
		err  bool
	}{
		{name: "", err: false},
		{name: "a", err: true},
	}
	assertions := require.New(t)
	for n, variant := range variants {
		_, err := AssetDir(variant.name)
		assertions.Equal(variant.err, err != nil, n)
	}
}

func TestRestoreAssets(t *testing.T) {
	assertions := require.New(t)
	assertions.NoError(RestoreAssets(t.TempDir(), ""))
}
