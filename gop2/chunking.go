package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ledongthuc/pdf"
)

const (
	inputDir     = "./pdfs"
	outputDir    = "./chunks"
	chunkSize    = 500
	chunkOverlap = 50
)

type Task struct {
	Path string
	Name string
}

type Result struct {
	Name string
	Err  error
}

func main() {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	files, err := os.ReadDir(inputDir)
	if err != nil {
		panic(err)
	}

	maxWorkers := runtime.NumCPU()
	taskCh := make(chan Task, maxWorkers*2)
	resultCh := make(chan Result, maxWorkers*2)
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(taskCh, resultCh, &wg)
	}

	go func() {
		for _, f := range files {
			if strings.HasSuffix(strings.ToLower(f.Name()), ".pdf") {
				taskCh <- Task{
					Path: filepath.Join(inputDir, f.Name()),
					Name: f.Name(),
				}
			}
		}
		close(taskCh)
	}()

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		if res.Err != nil {
			fmt.Printf("❌ Failed %s: %v\n", res.Name, res.Err)
		} else {
			fmt.Printf("✅ Processed %s\n", res.Name)
		}
	}
}

func worker(tasks <-chan Task, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		err := processPDFToChunks(task.Path, task.Name)
		results <- Result{Name: task.Name, Err: err}
	}
}

func processPDFToChunks(path, name string) error {
	f, r, err := pdf.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	pdfChunkDir := filepath.Join(outputDir, strings.TrimSuffix(name, filepath.Ext(name)))
	if err := os.MkdirAll(pdfChunkDir, 0755); err != nil {
		return err
	}

	var buffer strings.Builder
	chunkIndex := 0

	numPages := r.NumPage()

	// Channel to write chunks asynchronously
	type chunk struct {
		text string
		idx  int
	}
	chunkCh := make(chan chunk, 4)
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ch := range chunkCh {
			chunkFilename := filepath.Join(pdfChunkDir, fmt.Sprintf("chunk_%03d.txt", ch.idx))
			if err := os.WriteFile(chunkFilename, []byte(ch.text), 0644); err != nil {
				fmt.Printf("Failed writing chunk file %s: %v\n", chunkFilename, err)
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

			// Send to write channel
			chunkCh <- chunk{text: chunkText, idx: chunkIndex}
			chunkIndex++

			// Rebuild buffer with overlap preserved
			remaining := bufStr[chunkSize-chunkOverlap:]
			buffer.Reset()
			buffer.WriteString(remaining)
		}
	}

	// Write remaining text chunks
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
