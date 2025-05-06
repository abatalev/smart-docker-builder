package docker

import (
	"fmt"
	"io"
	"strings"
)

type ProjectDependency struct { // from common
	Type  string
	Value string
}

// TODO from bnd
func ParseDockerFile(reader io.ReadCloser, dir string) ([]string, []ProjectDependency) {
	// TODO support dockerignore (https://docs.docker.com/build/building/context/#dockerignore-files)
	// TODO https://pkg.go.dev/github.com/moby/buildkit/frontend/dockerfile/parser
	// TODO variables in COPY
	list := make([]string, 0)
	dependencies := make([]ProjectDependency, 0)
	defer reader.Close()
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("ERROR", err)
			break
		}
		if n > 0 {
			ss := string(buf[:n]) // TODO work with buffer tail
			for _, s := range strings.Split(ss, "\n") {
				dependencies = parseFrom(s, dependencies)
				list = parseCopy(s, list)
			}
		}
	}
	return list, dependencies
}

func parseFrom(s string, list []ProjectDependency) []ProjectDependency {
	ss := strings.ToLower(s)
	if !strings.HasPrefix(ss, "from ") {
		return list
	}
	sss := strings.Split(s, " ")
	image := sss[1]
	return append(list, ProjectDependency{Type: "docker-image", Value: image})
}

func parseCopy(s string, list []string) []string {
	ss := strings.ToLower(s)
	if !(strings.HasPrefix(ss, "copy ") &&
		!strings.Contains(ss, "--from=")) {
		return list
	}

	sss := strings.Split(s, " ")
	path := sss[1]
	if path == "./" {
		path = "**/*"
	}

	path = strings.TrimPrefix(path, "./")
	if strings.HasSuffix(path, "/") {
		path += "**/*"
	}
	return append(list, path)
}
