package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Directory to process
const chunksFolder = "chunkTest"

// Delimiters to remove
var delimitersToRemove = []string{",", "*", "#", "[", "]", "(", ")", "{", "}", "`", "~", "^", "=", "|"}

func cleanText(input string) string {
	cleaned := input
	for _, delim := range delimitersToRemove {
		cleaned = strings.ReplaceAll(cleaned, delim, "")
	}
	return cleaned
}

func processFile(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() || !strings.HasSuffix(info.Name(), ".txt") {
		return nil
	}

	content, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return fmt.Errorf("failed to read file %s: %v", path, readErr)
	}

	cleaned := cleanText(string(content))

	writeErr := ioutil.WriteFile(path, []byte(cleaned), 0644)
	if writeErr != nil {
		return fmt.Errorf("failed to write cleaned content to %s: %v", path, writeErr)
	}

	fmt.Printf("✅ Cleaned: %s\n", path)
	return nil
}

func main() {
	err := filepath.Walk(chunksFolder, processFile)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ All files cleaned successfully.")
	}
}
