package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abatalev/smartdockerbuild/internal/logic"
	"github.com/abatalev/smartdockerbuild/internal/osrunner"
	"gopkg.in/yaml.v3"
)

//go:generate go-bindata -prefix "facts/" -pkg main -o bindata.go facts/...

type Def struct {
	Name    string   `yaml:"name"`
	CmdName string   `yaml:"cmd"`
	Args    []string `yaml:"args"`
}

type DefInternal struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}

type Defs struct {
	Facts []DefInternal `yaml:"facts"`
}

type Config struct {
	Name     string   `yaml:"name"`
	Prefixes []string `yaml:"prefixes"`
	Facts    []Def    `yaml:"facts"`
	Tags     []string `yaml:"tags"`
}

var gitHash = "development"
var p2hHash = ""

type Options struct {
	isVersion      bool
	isHelp         bool
	isForce        bool
	isPush         bool
	DockerfileName string
}

func main() {
	fmt.Println("smart docker build")
	args := os.Args[1:]

	options, err := parseOptions(args)
	if err != nil {
		panic(err)
	}

	if options.isVersion {
		fmt.Println("Version:")
		fmt.Println("     git", gitHash)
		if p2hHash != "" {
			fmt.Println("     p2h", p2hHash)
		}
		return
	}

	if options.isHelp {
		fmt.Println()
		flag.PrintDefaults()
		fmt.Println()
		return
	}

	os.Exit(BuildDockerImage(".", options.DockerfileName, options.isForce, options.isPush))
}

func parseOptions(args []string) (Options, error) {
	var options Options
	flags := flag.NewFlagSet("1", flag.ExitOnError)
	flags.BoolVar(&options.isVersion, "version", false, "Show version of application")
	flags.BoolVar(&options.isHelp, "help", false, "Show help")
	flags.BoolVar(&options.isForce, "force", false, "Ignore cached images")
	flags.BoolVar(&options.isPush, "push", false, "Push images")
	err := flags.Parse(args)
	if len(flags.Args()) > 0 {
		options.DockerfileName = flags.Args()[0]
	}
	return options, err
}

func BuildDockerImage(workDir, dockerFile string, isForce, isPush bool) int {
	fmt.Println(" -> file", dockerFile)
	fullDockerFile := filepath.Join(workDir, dockerFile)
	// fmt.Println(" -> workdir", workDir)
	// fmt.Println(" -> fullName", fullDockerFile)
	_, err := os.Lstat(fullDockerFile)
	if err != nil {
		fmt.Println(" -> ", err)
		return 1
	}
	dirName := filepath.Dir(fullDockerFile)
	imageName := logic.GetImageName(fullDockerFile)

	cfg, err := loadConfig(filepath.Join(dirName, imageName+".sdb.yaml"))
	if err != nil {
		return 1
	}

	hashName := imageName
	if cfg.Name != "" {
		hashName = cfg.Name
	}
	hashTag := logic.CalcHash(workDir, dockerFile) // TODO fix WorkDir
	isNeedBuild, err := checkOldBuild(isForce, hashName, hashTag)
	if err != nil {
		return 1
	}

	hash := hashName + ":" + hashTag
	if isNeedBuild {
		if exitCode := dockerBuild(workDir, dockerFile, hash); exitCode != 0 {
			return exitCode
		}
	} else {
		fmt.Println(" --> (" + hash + ") image exists. build skipped")
	}

	fmt.Println(" --> gathering facts")
	facts := cfg.GatheringFacts(hash)
	return cfg.DoRules(hashName, hashTag, facts, isPush, cfg.Prefixes)
}

func logStrings(name, content string) {
	for n, s := range strings.Split(content, "\n") {
		if n == 0 {
			fmt.Println(" ---> "+name+":", s)
		} else {
			fmt.Println("          ->:", s)
		}
	}
}

func dockerBuild(workDir, dockerFile, hash string) int {
	fmt.Println(" --> build", hash)
	cmd := exec.Command("docker", "build", "-t", hash, "-f", dockerFile, ".")
	cmd.Dir = workDir
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		logStrings("stderr", strings.TrimSpace(stderr.String()))
		logStrings("stdout", strings.TrimSpace(stdout.String()))
		fmt.Println(" ---> error:", err)
		fmt.Println(" ---> exit code:", cmd.ProcessState.ExitCode())
		fmt.Println(" -> aborted!")
		return cmd.ProcessState.ExitCode()
	}
	return 0
}

func checkOldBuild(isForce bool, hashName string, hashTag string) (bool, error) {
	if isForce {
		return true, nil
	}

	existImage, err := existsImage(hashName, hashTag)
	if err != nil {
		fmt.Println(" -> aborted. error", err)
		return false, err
	}

	return !existImage, nil
}

func loadConfig(configName string) (Config, error) {
	cfg := Config{}
	yamlFile, err := os.ReadFile(configName)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return Config{}, err
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		log.Printf("error: %v", err)
		return Config{}, err
	}
	return cfg, nil
}

func existsImage(hashName, hashTag string) (bool, error) {
	cmd, cmdOut := DockerImageList()
	res, err := osrunner.StartAndWait([]*exec.Cmd{cmd}, cmdOut)
	if err != nil {
		return false, err
	}
	return logic.FindImage(string(res), hashName, hashTag), nil
}

func DockerImageList() (*exec.Cmd, io.ReadCloser) {
	cmd := osrunner.Command("docker", "image", "list")
	cmdOut, _ := cmd.StdoutPipe()
	return cmd, cmdOut
}

func RunCmdChain(useEntryPoint bool, hash string, args []string) (string, error) {
	cmds, cmdOut := logic.GetCmdChain(useEntryPoint, hash, args)
	res, err := osrunner.StartAndWait(cmds, cmdOut)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res)), nil
}

func loadFactsYaml(yamlFile []byte) []DefInternal {
	defs := Defs{}
	err := yaml.Unmarshal(yamlFile, &defs)
	if err != nil {
		log.Printf("error: %v", err)
		return []DefInternal{}
	}
	return defs.Facts
}

func loadAllFacts() map[string][]string {
	globalFacts := make(map[string][]string, 0)
	fmt.Println(" ---> assets")
	for _, asset := range AssetNames() {
		fmt.Println(" ----> asset", asset)
		data, _ := Asset(asset)
		facts := loadFactsYaml(data)
		for _, fact := range facts {
			globalFacts[fact.Name] = fact.Args
		}
	}
	return globalFacts
}

func (cfg Config) GatheringFacts(hash string) map[string]string {
	globalFacts := loadAllFacts()

	facts := make(map[string]string)
	for _, def := range cfg.Facts {
		if def.CmdName != "" {
			facts = calcFact(facts, def.Name, hash, globalFacts[def.CmdName])
		} else {
			facts = calcFact(facts, def.Name, hash, def.Args)
		}
	}
	return facts
}

func calcFact(facts map[string]string, name, hash string, args []string) map[string]string {
	value, err := RunCmdChain(true, hash, args)
	if err != nil {
		fmt.Println(" ---> fact "+name+" skipped!", err)
		return facts
	}
	fmt.Println(" ---> fact:", name, "=", value)
	facts[name] = value
	return facts
}

func (cfg Config) DoRules(hashName, hashTag string,
	facts map[string]string, isPush bool, prefixes []string) int {
	fmt.Println(" --> create tags")
	for _, mask := range cfg.Tags {
		fmt.Println(" ---> mask", mask)
		if err := logic.TagsProcessing(mask, facts, func(tagName string) error {
			fmt.Println(" ----> tag", tagName)
			if err := createDockerTag(hashName+":"+hashTag, hashName+":"+tagName).Run(); err != nil {
				fmt.Println(" ----> tag: warning! ", err)
			}
			for _, prefix := range prefixes {
				hashNameWithPrefix := prefix + "/" + hashName
				if strings.HasSuffix(prefix, "/") {
					hashNameWithPrefix = prefix + hashName
				}
				if err := createDockerTag(hashName+":"+hashTag, hashNameWithPrefix+":"+tagName).Run(); err != nil {
					fmt.Println(" ----> tag: warning! ", err)
				}
				if isPush {
					if err := pushDockerImage(hashNameWithPrefix, tagName).Run(); err != nil {
						fmt.Println(" ----> push:  ", err)
						return err
					}
				}
			}
			return nil
		}); err != nil {
			fmt.Println(" --> aborted")
			return 1
		}
	}
	return 0
}

func pushDockerImage(imageName, imageTag string) *exec.Cmd {
	return osrunner.Command("docker", "image", imageName+":"+imageTag)
}

func createDockerTag(imageName1, imageName2 string) *exec.Cmd {
	return osrunner.Command("docker", "image", "tag", imageName1, imageName2)
}
