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
			lowerName := strings.ToLower(f.Name())
			if strings.HasSuffix(lowerName, ".pdf") || strings.HasSuffix(lowerName, ".txt") {
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
			err = processPDFToChunks(task.Path, task.Name)
		case ".txt":
			err = processTXTToChunks(task.Path, task.Name)
		default:
			err = fmt.Errorf("unsupported file type: %s", ext)
		}
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
			chunkFilename := filepath.Join(pdfChunkDir, fmt.Sprintf("chunk_%03d.txt", ch.idx))
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

func processTXTToChunks(path, name string) error {
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	txtChunkDir := filepath.Join(outputDir, strings.TrimSuffix(name, filepath.Ext(name)))
	if err := os.MkdirAll(txtChunkDir, 0755); err != nil {
		return err
	}

	chunkIndex := 0
	for i := 0; i < len(content); i += chunkSize - chunkOverlap {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunkText := content[i:end]

		chunkFilename := filepath.Join(txtChunkDir, fmt.Sprintf("chunk_%03d.txt", chunkIndex))
		if err := os.WriteFile(chunkFilename, []byte(chunkText), 0644); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", chunkIndex, err)
		}
		chunkIndex++
	}

	return nil
}
