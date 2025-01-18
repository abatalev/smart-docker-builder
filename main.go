package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Def struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}

type Config struct {
	Prefix string   `yaml:"prefix"`
	Facts  []Def    `yaml:"facts"`
	Tags   []string `yaml:"tags"`
}

var gitHash = "development"
var p2hHash = ""

func main() {
	fmt.Println("smart docker build")

	isVersion := flag.Bool("version", false, "Show version of application")
	isHelp := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *isVersion {
		fmt.Println("Version:")
		fmt.Println("     git", gitHash)
		if p2hHash != "" {
			fmt.Println("     p2h", p2hHash)
		}
		return
	}

	if *isHelp {
		fmt.Println()
		flag.PrintDefaults()
		fmt.Println()
		return
	}

	os.Exit(BuildDockerImage(os.Args[1]))
}

func BuildDockerImage(dockerFile string) int {
	fmt.Println(" -> file", dockerFile)
	dirName := filepath.Dir(dockerFile)
	imageName := getImageName(dockerFile)

	// load Config
	cfg := Config{}
	yamlFile, err := os.ReadFile(filepath.Join(dirName, imageName+".sdb.yaml"))
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	hashName := cfg.Prefix + "/" + imageName
	hashTag := calcHash(dockerFile)
	hash := hashName + ":" + hashTag
	flag, err := existsImage(hashName, hashTag)
	if err != nil {
		fmt.Println(" -> aborted. error", err)
		return 1
	}

	if !flag {
		baseNameDockerFile := filepath.Base(dockerFile)
		dirDockerFile := filepath.Dir(dockerFile)
		fmt.Println(" --> build", hash)
		cmd := exec.Command("docker", "build", "-t", hash, "-f", baseNameDockerFile, ".")
		cmd.Dir = dirDockerFile
		var stderr, stdout bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			fmt.Println(" ---> stderr:", strings.TrimSpace(stderr.String()))
			fmt.Println(" ---> stdout:", strings.TrimSpace(stdout.String()))
			fmt.Println(" ---> error:", err)
			fmt.Println(" ---> exit code:", cmd.ProcessState.ExitCode())
			fmt.Println(" -> aborted!")
			return cmd.ProcessState.ExitCode()
		}
	} else {
		fmt.Println(" --> (" + hash + ") image exists. build skipped")
	}
	fmt.Println(" --> gathering facts")
	facts := cfg.GatheringFacts(hash)
	return cfg.DoRules(hashName, hashTag, facts)
}

func getImageName(dockerFile string) string {
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

func calcHash(dockerFile string) string {
	// TODO fix calcHash
	hash := calcHashFile(dockerFile)
	return hash[:8]
}

func calcHashBytes(buf []byte) string {
	// TODO bnd (hash.go)
	h := sha1.New()
	h.Write(buf)
	return hex.EncodeToString(h.Sum(nil))
}

func calcHashFile(path string) string {
	// TODO bnd (hash.go)
	buf, _ := os.ReadFile(path)
	return calcHashBytes(buf)
}

func existsImage(hashName, hashTag string) (bool, error) {
	cmd := exec.Command("docker", "image", "ls")
	cmdOut, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return false, err
	}
	res, _ := io.ReadAll(cmdOut)
	if err := cmd.Wait(); err != nil {
		return false, err
	}
	return findImage(string(res), hashName, hashTag), nil
}

func findImage(stdout string, project string, hash string) bool {
	// TODO bnd (dockerproject.go)
	for _, s := range strings.Split(stdout, "\n") {
		ss := strings.ReplaceAll(s, "\t", " ")
		for strings.Contains(ss, "  ") {
			ss = strings.ReplaceAll(ss, "  ", " ")
		}
		if strings.Contains(ss, project+" "+hash) {
			return true
		}
	}
	return false
}

func calcFact(facts map[string]string, name, hash string, args []string) map[string]string {
	useEntryPoint := true
	cmds, cmdOut := GetCmdChain(useEntryPoint, hash, args)
	//cmd.Stderr = os.Stderr
	for _, c := range cmds {
		if err := c.Start(); err != nil {
			fmt.Println(" ---> fact "+name+" skipped!", err)
			return facts
		}
	}
	res, _ := io.ReadAll(cmdOut)
	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			fmt.Println(" ---> fact "+name+" skipped!", err)
			return facts
		}
	}
	value := strings.TrimSpace(string(res))
	fmt.Println(" ---> fact:", name, "=", value)
	facts[name] = value
	return facts
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
			cmdIdx = cmdIdx + 1
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

func (cfg Config) GatheringFacts(hash string) map[string]string {
	facts := make(map[string]string)

	for _, def := range cfg.Facts {
		facts = calcFact(facts, def.Name, hash, def.Args)
	}

	//x("apk", hash, []string{"which", "apk"})
	//x("apk-list", hash, []string{"apk", "list", "-I"})
	//x("java-version", hash, []string{ pandoc --version | awk '{ print $2 }'})
	//x("java-version", hash, []string{apk info texlive | awk '/description:/{ print $1 }'})
	return facts
}

func (cfg Config) DoRules(hashName, hashTag string, facts map[string]string) int {
	fmt.Println(" --> create tags")
	for _, mask := range cfg.Tags {
		createTag(hashName, hashTag, mask, facts)
	}
	return 0
}

type Token struct {
	Index int
	Value []string
}

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

func createTag(hashName, hashTag, mask string, facts map[string]string) {
	tokens := make([]Token, 0)

	tt := strings.Split(mask, "|")
	for _, ttt := range tt {
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

	fmt.Println(" ---> mask", mask)
	for {
		x := ""
		for _, token := range tokens {
			x = x + token.Value[token.Index]
		}

		fmt.Println(" ----> tag", x)
		if err := exec.Command("docker", "image", "tag", hashName+":"+hashTag, hashName+":"+x).Run(); err != nil {
			fmt.Println(" ----> tag: warning! ", err)
		}

		flag := false
		for i, token := range tokens {
			if token.Index < len(token.Value)-1 {
				for v := range tokens {
					if v < i {
						tokens[v].Index = 0
					}
				}
				tokens[i].Index = tokens[i].Index + 1
				flag = true
				break
			}
		}
		if !flag {
			break
		}
	}
}
