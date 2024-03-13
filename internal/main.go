package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	sitterVersion = "0.22.1"
	sitterURL     = "https://github.com/tree-sitter/tree-sitter/archive/refs/tags/v" + sitterVersion + ".tar.gz"
)

func copyAndReportFiles(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		dstFilePath := filepath.Join(dstDir, relPath)

		if _, err := os.Stat(dstFilePath); err == nil {
			fmt.Printf("%-39s %s\n", filepath.Base(dstFilePath), "[replaced]")
		} else if os.IsNotExist(err) {
			fmt.Printf("%-39s %s\n", filepath.Base(dstFilePath), "[new file]")
		}

		return copyFile(path, dstFilePath)
	})
}

func copyFiles(srcDir, dstDir, pattern string) {
	files, err := os.ReadDir(srcDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if matched, _ := filepath.Match(pattern, file.Name()); matched {
			srcFilePath := filepath.Join(srcDir, file.Name())
			dstFilePath := filepath.Join(dstDir, file.Name())

			copyFile(srcFilePath, dstFilePath)
		}
	}
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func modifyIncludePaths(path string) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || (filepath.Ext(filePath) != ".c" && filepath.Ext(filePath) != ".h") {
			return nil
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		modifiedContent := strings.ReplaceAll(string(content), `"tree_sitter/`, `"`)
		modifiedContent = strings.ReplaceAll(modifiedContent, `"unicode/`, `"`)

		return os.WriteFile(filePath, []byte(modifiedContent), info.Mode())
	})
}

func downloadAndExtractSitter(url, version string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !strings.HasPrefix(header.Name, "tree-sitter-"+version+"/lib/src") && !strings.HasPrefix(header.Name, "tree-sitter-"+version+"/lib/include") {
			continue
		}

		relPath := strings.TrimPrefix(header.Name, version+"/")
		target := filepath.Join("tmpts", relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}

			outFile.Close()
		}
	}

	return nil
}

func cleanup(path string) {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".h" && filepath.Ext(path) != ".c" || filepath.Base(path) == "lib.c" {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		fmt.Println("cleanup failed:", err)
	}
}

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	treeSitterDir := "tree-sitter-" + sitterVersion
	parentPath := filepath.Join(currentDir, "tmpts", treeSitterDir)

	if err := downloadAndExtractSitter(sitterURL, sitterVersion); err != nil {
		log.Fatalf("Error: %v", err)
	}

	copyFiles(filepath.Join(parentPath, "lib", "include", "tree_sitter"), filepath.Join(currentDir, "tmpts"), "*.h")
	copyFiles(filepath.Join(parentPath, "lib", "src"), filepath.Join(currentDir, "tmpts"), "*.c")
	copyFiles(filepath.Join(parentPath, "lib", "src"), filepath.Join(currentDir, "tmpts"), "*.h")
	copyFiles(filepath.Join(parentPath, "lib", "src", "unicode"), filepath.Join(currentDir, "tmpts"), "*.h")

	err = os.RemoveAll(parentPath)
	if err != nil {
		log.Fatalf("Error removing extracted treesitter directory: %v", err)
	}

	if err := modifyIncludePaths(filepath.Join(currentDir, "tmpts")); err != nil {
		log.Fatalf("Error modifying include paths: %v", err)
	}

	cleanup(filepath.Join(currentDir, "tmpts"))

	err = copyAndReportFiles(filepath.Join(currentDir, "tmpts"), filepath.Join(currentDir, "..", ".."))
	if err != nil {
		log.Fatalf("Error copying and reporting files: %v", err)
	}

	err = os.RemoveAll(filepath.Join(currentDir, "tmpts"))
	if err != nil {
		log.Fatalf("Error removing tmpts directory: %v", err)
	}

	fmt.Printf("\n\nDone!\n")
}
