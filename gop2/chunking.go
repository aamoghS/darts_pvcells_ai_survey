package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ledongthuc/pdf"
)

const (
	inputDir     = "pdfs"
	outputDir    = "./chunks"
	chunkSize    = 500
	chunkOverlap = 50
)

type Task struct {
	Path string
	Name string
	Rel  string // Relative path to maintain folder structure
}

type Result struct {
	Name string
	Err  error
}

func main() {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	maxWorkers := runtime.NumCPU()
	taskCh := make(chan Task, maxWorkers*2)
	resultCh := make(chan Result, maxWorkers*2)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(taskCh, resultCh, &wg)
	}

	// Walk the directory recursively
	go func() {
		err := filepath.WalkDir(inputDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext == ".pdf" || ext == ".txt" {
				relPath, _ := filepath.Rel(inputDir, path)
				taskCh <- Task{
					Path: path,
					Name: d.Name(),
					Rel:  relPath, // used for output subdirectory structure
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("error walking input directory: %v\n", err)
		}
		close(taskCh)
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	for res := range resultCh {
		if res.Err != nil {
			fmt.Printf("error %s: %v\n", res.Name, res.Err)
		} else {
			fmt.Printf("done %s\n", res.Name)
		}
	}
}

func worker(tasks <-chan Task, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		ext := strings.ToLower(filepath.Ext(task.Name))
		var err error
		switch ext {
		case ".pdf":
			err = processPDFToChunks(task)
		case ".txt":
			err = processTXTToChunks(task)
		default:
			err = fmt.Errorf("unsupported file type: %s", ext)
		}
		results <- Result{Name: task.Rel, Err: err}
	}
}

func processPDFToChunks(task Task) error {
	f, r, err := pdf.Open(task.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	chunkDir := filepath.Join(outputDir, strings.TrimSuffix(task.Rel, filepath.Ext(task.Rel)))
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return err
	}

	var buffer strings.Builder
	chunkIndex := 0
	numPages := r.NumPage()

	type chunk struct {
		text string
		idx  int
	}
	chunkCh := make(chan chunk, 4)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for ch := range chunkCh {
			chunkFilename := filepath.Join(chunkDir, fmt.Sprintf("chunk_%03d.txt", ch.idx))
			if err := os.WriteFile(chunkFilename, []byte(ch.text), 0644); err != nil {
				fmt.Printf("failed %s: %v\n", chunkFilename, err)
			}
		}
	}()

	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			return fmt.Errorf("error extracting text from page %d: %w", i, err)
		}
		buffer.WriteString(content)

		for buffer.Len() >= chunkSize*2 {
			bufStr := buffer.String()
			chunkText := bufStr[:chunkSize]

			chunkCh <- chunk{text: chunkText, idx: chunkIndex}
			chunkIndex++

			remaining := bufStr[chunkSize-chunkOverlap:]
			buffer.Reset()
			buffer.WriteString(remaining)
		}
	}

	remainingText := buffer.String()
	for i := 0; i < len(remainingText); i += chunkSize - chunkOverlap {
		end := i + chunkSize
		if end > len(remainingText) {
			end = len(remainingText)
		}
		chunkCh <- chunk{text: remainingText[i:end], idx: chunkIndex}
		chunkIndex++
	}

	close(chunkCh)
	wg.Wait()

	return nil
}

func processTXTToChunks(task Task) error {
	contentBytes, err := ioutil.ReadFile(task.Path)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	chunkDir := filepath.Join(outputDir, strings.TrimSuffix(task.Rel, filepath.Ext(task.Rel)))
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return err
	}

	chunkIndex := 0
	for i := 0; i < len(content); i += chunkSize - chunkOverlap {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunkText := content[i:end]

		chunkFilename := filepath.Join(chunkDir, fmt.Sprintf("chunk_%03d.txt", chunkIndex))
		if err := os.WriteFile(chunkFilename, []byte(chunkText), 0644); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", chunkIndex, err)
		}
		chunkIndex++
	}

	return nil
}
