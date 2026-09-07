package materialparser

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

type SourceUnit struct {
	Text       string
	SourceType string
	Position   int
}
type Reader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}
type PDFOCR interface {
	Recognize(context.Context, Reader, int64, []int) (map[int]string, error)
}
type Parser struct {
	MaxPDFPages int
	OCR         PDFOCR
	OCRMinRunes int
}

func New(maxPDFPages int, ocr ...PDFOCR) *Parser {
	parser := &Parser{MaxPDFPages: maxPDFPages, OCRMinRunes: 20}
	if len(ocr) > 0 {
		parser.OCR = ocr[0]
	}
	return parser
}
func (p *Parser) Parse(ctx context.Context, filename string, size int64, reader Reader) ([]SourceUnit, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		return p.parsePDF(ctx, reader, size)
	case ".pptx":
		return parsePPTX(reader, size)
	case ".docx":
		return parseDOCX(reader, size)
	case ".txt", ".md":
		return parseText(reader)
	default:
		return nil, fmt.Errorf("unsupported material format")
	}
}
func (p *Parser) parsePDF(ctx context.Context, reader Reader, size int64) ([]SourceUnit, error) {
	document, err := pdf.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	pages := document.NumPage()
	if pages > p.MaxPDFPages {
		return nil, fmt.Errorf("pdf exceeds %d pages", p.MaxPDFPages)
	}
	pageTexts := make(map[int]string, pages)
	ocrPages := make([]int, 0)
	for pageNumber := 1; pageNumber <= pages; pageNumber++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		text, extractErr := document.Page(pageNumber).GetPlainText(nil)
		if extractErr != nil {
			return nil, fmt.Errorf("extract pdf page %d: %w", pageNumber, extractErr)
		}
		text = cleanText(text)
		pageTexts[pageNumber] = text
		if len([]rune(text)) < p.OCRMinRunes {
			ocrPages = append(ocrPages, pageNumber)
		}
	}
	if len(ocrPages) > 0 && p.OCR != nil {
		recognized, ocrErr := p.OCR.Recognize(ctx, reader, size, ocrPages)
		if ocrErr != nil {
			return nil, fmt.Errorf("ocr pdf: %w", ocrErr)
		}
		for pageNumber, text := range recognized {
			if text = cleanText(text); text != "" {
				pageTexts[pageNumber] = text
			}
		}
	}
	units := make([]SourceUnit, 0, pages)
	for pageNumber := 1; pageNumber <= pages; pageNumber++ {
		if text := pageTexts[pageNumber]; text != "" {
			units = append(units, SourceUnit{Text: text, SourceType: "page", Position: pageNumber})
		}
	}
	return requireContent(units)
}

var slidePattern = regexp.MustCompile(`^ppt/slides/slide([0-9]+)\.xml$`)

func parsePPTX(reader io.ReaderAt, size int64) ([]SourceUnit, error) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	type slide struct {
		number int
		file   *zip.File
	}
	slides := make([]slide, 0)
	for _, file := range archive.File {
		matches := slidePattern.FindStringSubmatch(file.Name)
		if len(matches) == 2 {
			number, _ := strconv.Atoi(matches[1])
			slides = append(slides, slide{number: number, file: file})
		}
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].number < slides[j].number })
	units := make([]SourceUnit, 0, len(slides))
	for _, item := range slides {
		text, parseErr := extractOfficeText(item.file, map[string]bool{"t": true})
		if parseErr != nil {
			return nil, fmt.Errorf("extract slide %d: %w", item.number, parseErr)
		}
		if text = cleanText(text); text != "" {
			units = append(units, SourceUnit{Text: text, SourceType: "slide", Position: item.number})
		}
	}
	return requireContent(units)
}
func parseDOCX(reader io.ReaderAt, size int64) ([]SourceUnit, error) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	for _, file := range archive.File {
		if file.Name != "word/document.xml" {
			continue
		}
		paragraphs, parseErr := extractWordParagraphs(file)
		if parseErr != nil {
			return nil, fmt.Errorf("extract docx: %w", parseErr)
		}
		units := make([]SourceUnit, 0, len(paragraphs))
		for index, text := range paragraphs {
			if text = cleanText(text); text != "" {
				units = append(units, SourceUnit{Text: text, SourceType: "document", Position: index + 1})
			}
		}
		return requireContent(units)
	}
	return nil, fmt.Errorf("docx main document missing")
}
func parseText(reader io.Reader) ([]SourceUnit, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read text: %w", err)
	}
	parts := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	units := make([]SourceUnit, 0, len(parts))
	for index, text := range parts {
		if text = cleanText(text); text != "" {
			units = append(units, SourceUnit{Text: text, SourceType: "text", Position: index + 1})
		}
	}
	return requireContent(units)
}
func extractOfficeText(file *zip.File, textElements map[string]bool) (string, error) {
	stream, err := file.Open()
	if err != nil {
		return "", err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	var parts []string
	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return "", tokenErr
		}
		start, ok := token.(xml.StartElement)
		if !ok || !textElements[start.Name.Local] {
			continue
		}
		var value string
		if err = decoder.DecodeElement(&value, &start); err != nil {
			return "", err
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, " "), nil
}
func extractWordParagraphs(file *zip.File) ([]string, error) {
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	var paragraphs []string
	var current []string
	depth := 0
	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return nil, tokenErr
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "p" {
				depth++
			}
			if value.Name.Local == "t" {
				var text string
				if err = decoder.DecodeElement(&text, &value); err != nil {
					return nil, err
				}
				current = append(current, text)
			}
		case xml.EndElement:
			if value.Name.Local == "p" && depth > 0 {
				depth--
				if depth == 0 {
					paragraphs = append(paragraphs, strings.Join(current, " "))
					current = nil
				}
			}
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, strings.Join(current, " "))
	}
	return paragraphs, nil
}
func cleanText(value string) string { return strings.Join(strings.Fields(value), " ") }
func requireContent(units []SourceUnit) ([]SourceUnit, error) {
	if len(units) == 0 {
		return nil, fmt.Errorf("material contains no extractable text")
	}
	return units, nil
}
