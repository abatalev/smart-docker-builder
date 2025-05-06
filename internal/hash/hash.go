package hash

import (
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
)

func WalkDirWithPatterns(workDir string, patters []string) []string {
	if _, err := os.Lstat(workDir); err != nil {
		return []string{}
	}

	if !strings.HasSuffix(workDir, "/") {
		workDir += "/"
	}
	files := make([]string, 0)
	err := filepath.Walk(workDir, func(path string, info fs.FileInfo, err error) error {
		// log.Println("@@@@", path, workDir)
		filename := strings.TrimPrefix(path, workDir)
		if info.IsDir() {
			return nil
		}
		if walkFunc(filename, patters) {
			files = append(files, filename)
		}
		return nil
	})
	if err != nil {
		// panic("!!!") // TODO remove panic
		log.Println("ERROR", err)
		return []string{}
	}
	return files
}

func walkFunc(filename string, patters []string) bool {
	for _, p := range patters {
		if x, _ := doublestar.Match(p, filename); x {
			return true
		}
	}
	return false
}

func CalcHashFiles(files []string) string {
	s := ""
	for _, file := range files {
		s += file + "\n"
	}
	return calcHashBytes([]byte(s))
}

func CalcHashes(workDir string, files []string) []string {
	filesWithHashes := make([]string, 0)
	for _, f := range files {
		filesWithHashes = appendFileAndHash(filesWithHashes, f, calcHashFile(filepath.Join(workDir, f)))
	}
	return filesWithHashes
}

func appendFileAndHash(filesWithHashes []string, f, hash string) []string {
	return append(filesWithHashes, f+" "+hash)
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
