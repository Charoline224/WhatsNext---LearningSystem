package service

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type materialFile interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}
type materialFormat struct {
	Extension string
	MIMEType  string
}

var supportedMaterialFormats = map[string]materialFormat{
	".pdf":  {Extension: ".pdf", MIMEType: "application/pdf"},
	".pptx": {Extension: ".pptx", MIMEType: "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
	".docx": {Extension: ".docx", MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	".txt":  {Extension: ".txt", MIMEType: "text/plain; charset=utf-8"},
	".md":   {Extension: ".md", MIMEType: "text/markdown; charset=utf-8"},
}

func detectMaterialFormat(filename string, size int64, file materialFile) (materialFormat, error) {
	format, ok := supportedMaterialFormats[strings.ToLower(filepath.Ext(filename))]
	if !ok {
		return materialFormat{}, ErrUnsupportedFile
	}
	var err error
	switch format.Extension {
	case ".pdf":
		err = validatePDF(file)
	case ".pptx":
		err = validateOfficeZIP(file, size, "ppt/presentation.xml")
	case ".docx":
		err = validateOfficeZIP(file, size, "word/document.xml")
	case ".txt", ".md":
		err = validateText(file, size)
	}
	if err != nil {
		return materialFormat{}, ErrUnsupportedFile
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return materialFormat{}, err
	}
	return format, nil
}
func validatePDF(file materialFile) error {
	header := make([]byte, 5)
	if _, err := io.ReadFull(file, header); err != nil {
		return err
	}
	if string(header) != "%PDF-" {
		return ErrUnsupportedFile
	}
	return nil
}
func validateOfficeZIP(file materialFile, size int64, mainPart string) error {
	reader, err := zip.NewReader(file, size)
	if err != nil {
		return err
	}
	hasContentTypes, hasMain := false, false
	var expanded uint64
	for _, entry := range reader.File {
		expanded += entry.UncompressedSize64
		if expanded > 200*1024*1024 || len(reader.File) > 10000 {
			return ErrUnsupportedFile
		}
		name := strings.TrimPrefix(entry.Name, "/")
		if name == "[Content_Types].xml" {
			hasContentTypes = true
		}
		if name == mainPart {
			hasMain = true
		}
	}
	if !hasContentTypes || !hasMain {
		return ErrUnsupportedFile
	}
	return nil
}
func validateText(file materialFile, size int64) error {
	limit := size
	if limit > 8192 {
		limit = 8192
	}
	sample := make([]byte, limit)
	n, err := io.ReadFull(file, sample)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	sample = sample[:n]
	if bytes.IndexByte(sample, 0) >= 0 {
		return ErrUnsupportedFile
	}
	for trim := 0; trim <= 3 && trim <= len(sample); trim++ {
		if utf8.Valid(sample[:len(sample)-trim]) {
			return nil
		}
	}
	return ErrUnsupportedFile
}
