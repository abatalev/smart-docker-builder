package logic

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abatalev/smartdockerbuild/internal/docker"
	"github.com/abatalev/smartdockerbuild/internal/hash"
)

func SemanticVersion(v string) []string {
	a := strings.Split(v, ".")
	versions := make([]string, 0)
	x := ""
	for _, v := range a {
		if x == "" {
			x = v
		} else {
			x = x + "." + v
		}
		versions = append(versions, x)
	}
	return versions
}

type Token struct {
	Index int
	Value []string
}

func TagsProcessing(mask string, facts map[string]string, doTag func(tag string) error) error {
	tokens := ParseMask(mask, facts)
	for {
		if err := doTag(GetTag(tokens)); err != nil {
			return err
		}
		if !NextTag(tokens) {
			break
		}
	}
	return nil
}

func ParseMask(mask string, facts map[string]string) []Token {
	tokens := make([]Token, 0)
	for _, ttt := range strings.Split(mask, "|") {
		if strings.HasPrefix(ttt, "@") {
			k := strings.TrimPrefix(ttt, "@")
			tokens = append(tokens, Token{Value: SemanticVersion(facts[k])})
			continue
		}
		if strings.HasPrefix(ttt, "$") {
			k := strings.TrimPrefix(ttt, "$")
			tokens = append(tokens, Token{Value: []string{facts[k]}})
			continue
		}
		tokens = append(tokens, Token{Value: []string{ttt}})
	}
	return tokens
}

func NextTag(tokens []Token) bool {
	flag := false
	for i, token := range tokens {
		if token.Index < len(token.Value)-1 {
			for v := range tokens {
				if v < i {
					tokens[v].Index = 0
				}
			}
			tokens[i].Index += 1
			flag = true
			break
		}
	}
	return flag
}

func GetTag(tokens []Token) string {
	x := ""
	for _, token := range tokens {
		x += token.Value[token.Index]
	}
	return x
}

func quote(a string) string {
	if !strings.Contains(a, " ") {
		return a
	}
	return "\"" + a + "\""
}
func merge(a, b string) string {
	return a + " " + quote(b)
}

func GetCmdChain(useEntryPoint bool, hash string, args []string) ([]*exec.Cmd, io.ReadCloser) {
	cmdIdx := 0
	cmds := make([]*exec.Cmd, 0)
	var v []string
	if useEntryPoint {
		v = []string{"docker", "run", "--rm", "--entrypoint", "/bin/sh", hash, "-c"}
	} else {
		v = []string{"docker", "run", "--rm", hash}
	}
	lv := len(v)
	var prvCmd *exec.Cmd = nil
	var prvPipe string = ""
	for _, arg := range args {
		if arg == "|" || arg == "|&" {
			curCmd := exec.Command(v[0], v[1:]...)
			if prvCmd != nil {
				if prvPipe == "|" {
					curCmd.Stdin, _ = prvCmd.StdoutPipe()
				} else {
					curCmd.Stdin, _ = prvCmd.StderrPipe()
				}
			}
			prvCmd = curCmd
			prvPipe = arg
			cmds = append(cmds, curCmd)
			cmdIdx += 1
			v = make([]string, 0)
		} else {
			if useEntryPoint && cmdIdx == 0 {
				cnt := len(v)
				if lv == cnt {
					v = append(v, arg)
				} else {
					v[cnt-1] = merge(v[cnt-1], arg)
				}
			} else {
				v = append(v, arg)
			}
		}
	}
	cmd := exec.Command(v[0], v[1:]...)
	if prvCmd != nil {
		if prvPipe == "|" {
			cmd.Stdin, _ = prvCmd.StdoutPipe()
		} else {
			cmd.Stdin, _ = prvCmd.StderrPipe()
		}
	}
	cmds = append(cmds, cmd)
	cmdOut, _ := cmd.StdoutPipe()
	return cmds, cmdOut
}

func FindImage(stdout string, project string, hash string) bool {
	// TODO bnd (dockerproject.go)
	for _, s := range strings.Split(stdout, "\n") {
		ss := strings.ReplaceAll(s, "\t", " ")
		for strings.Contains(ss, "  ") {
			ss = strings.ReplaceAll(ss, "  ", " ")
		}
		if strings.HasPrefix(ss, project+" "+hash+" ") {
			return true
		}
	}
	return false
}

func GetImageName(dockerFile string) string {
	// TODO bnd (project.go:CheckFile)
	baseName := filepath.Base(dockerFile)
	if baseName == "Dockerfile" {
		return filepath.Base(filepath.Dir(dockerFile))
	}
	if strings.HasPrefix(baseName, "Dockerfile.") {
		return strings.TrimPrefix(baseName, "Dockerfile.")
	}
	if strings.HasSuffix(baseName, ".Dockerfile") {
		return strings.TrimSuffix(baseName, ".Dockerfile")
	}

	panic("unknown pattern '" + dockerFile + "'") // TODO remove panic
}

func CalcHash(workDir, dockerFile string) string {
	hash := hash.CalcHashFiles(hash.CalcHashes(workDir, GetFilesForDockerFile(workDir, dockerFile)))
	return hash[:8]
}

func GetFilesForDockerFile(workDir, dockerFile string) []string {
	f, _ := os.Open(filepath.Join(workDir, dockerFile))
	// if err != nil {
	// 	return []string{}, []docker.ProjectDependency{}, err
	// }
	files := []string{dockerFile}
	patterns, _ := docker.ParseDockerFile(f, workDir)
	files = append(files, patterns...)
	return hash.WalkDirWithPatterns(workDir, files)
}
