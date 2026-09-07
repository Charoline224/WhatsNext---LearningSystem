package materialparser

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type TesseractOCR struct {
	Languages string
	DPI       int
	Timeout   time.Duration
	Workers   int
}

func NewTesseractOCR(languages string, dpi, workers int, timeout time.Duration) (*TesseractOCR, error) {
	languages = strings.TrimSpace(languages)
	if languages == "" || dpi <= 0 || workers <= 0 || timeout <= 0 {
		return nil, fmt.Errorf("OCR languages, DPI, workers, and timeout must be configured")
	}
	return &TesseractOCR{Languages: languages, DPI: dpi, Workers: workers, Timeout: timeout}, nil
}

func (o *TesseractOCR) Recognize(ctx context.Context, reader Reader, size int64, pages []int) (map[int]string, error) {
	result := make(map[int]string, len(pages))
	if len(pages) == 0 {
		return result, nil
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil, fmt.Errorf("pdftoppm is not installed: %w", err)
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return nil, fmt.Errorf("tesseract is not installed: %w", err)
	}
	directory, err := os.MkdirTemp("", "whatsnext-ocr-*")
	if err != nil {
		return nil, fmt.Errorf("create OCR temporary directory: %w", err)
	}
	defer os.RemoveAll(directory)
	inputPath := filepath.Join(directory, "input.pdf")
	input, err := os.OpenFile(inputPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create OCR input: %w", err)
	}
	_, err = io.Copy(input, io.NewSectionReader(reader, 0, size))
	closeErr := input.Close()
	if err != nil {
		return nil, fmt.Errorf("copy OCR input: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close OCR input: %w", closeErr)
	}
	type pageResult struct {
		page int
		text string
		err  error
	}
	jobs := make(chan int)
	results := make(chan pageResult)
	workerCount := o.Workers
	if workerCount > len(pages) {
		workerCount = len(pages)
	}
	workCtx, cancelWork := context.WithCancel(ctx)
	defer cancelWork()
	for range workerCount {
		go func() {
			for page := range jobs {
				text, pageErr := o.recognizePage(workCtx, inputPath, directory, page)
				results <- pageResult{page: page, text: text, err: pageErr}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, page := range pages {
			jobs <- page
		}
	}()
	var firstErr error
	for range pages {
		item := <-results
		if item.err != nil && firstErr == nil {
			firstErr = item.err
			cancelWork()
		}
		if item.err == nil {
			result[item.page] = item.text
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return result, nil
}

func (o *TesseractOCR) recognizePage(ctx context.Context, inputPath, directory string, page int) (string, error) {
	pageCtx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	prefix := filepath.Join(directory, "page-"+strconv.Itoa(page))
	render := exec.CommandContext(pageCtx, "pdftoppm", "-f", strconv.Itoa(page), "-l", strconv.Itoa(page), "-r", strconv.Itoa(o.DPI), "-png", "-singlefile", inputPath, prefix)
	if output, err := render.CombinedOutput(); err != nil {
		return "", fmt.Errorf("render page %d: %w: %s", page, err, truncateCommandOutput(output))
	}
	imagePath := prefix + ".png"
	defer os.Remove(imagePath)
	recognize := exec.CommandContext(pageCtx, "tesseract", imagePath, "stdout", "-l", o.Languages, "--psm", "6")
	output, err := recognize.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("recognize page %d: %w: %s", page, err, truncateCommandOutput(output))
	}
	return string(output), nil
}

func truncateCommandOutput(value []byte) string {
	text := strings.TrimSpace(string(value))
	if len(text) > 1000 {
		return text[:1000]
	}
	return text
}
