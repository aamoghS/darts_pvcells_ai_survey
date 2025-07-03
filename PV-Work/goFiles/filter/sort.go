package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	folderPath := "./pdfs"

	var pdfFiles []string

	err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".pdf") {
			pdfFiles = append(pdfFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Println("error with talking:", err)
		return
	}

	// sort the files
	sort.Slice(pdfFiles, func(i, j int) bool {
		return strings.ToLower(filepath.Base(pdfFiles[i])) < strings.ToLower(filepath.Base(pdfFiles[j]))
	})

	for i, oldPath := range pdfFiles {
		dir := filepath.Dir(oldPath)
		oldName := filepath.Base(oldPath)
		newName := fmt.Sprintf("%03d_%s", i+1, oldName)
		newPath := filepath.Join(dir, newName)

		err := os.Rename(oldPath, newPath)
		if err != nil {
			fmt.Printf("issue %s: %v\n", oldName, err)
		} else {
			fmt.Printf("fixed: %s → %s\n", oldName, newName)
		}
	}
}
