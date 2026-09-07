package service

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestDetectMaterialFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		mime string
	}{{"book.pdf", []byte("%PDF-1.4\n%%EOF"), "application/pdf"}, {"notes.txt", []byte("TCP 可靠传输"), "text/plain; charset=utf-8"}, {"notes.md", []byte("# TCP\n\n内容"), "text/markdown; charset=utf-8"}, {"slides.pptx", officeFixture(t, "ppt/presentation.xml"), "application/vnd.openxmlformats-officedocument.presentationml.presentation"}, {"handout.docx", officeFixture(t, "word/document.xml"), "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			format, err := detectMaterialFormat(tt.name, int64(len(tt.data)), reader)
			if err != nil {
				t.Fatalf("detect: %v", err)
			}
			if format.MIMEType != tt.mime {
				t.Fatalf("mime=%q", format.MIMEType)
			}
		})
	}
}
func TestDetectMaterialFormatRejectsSpoofedAndBinaryFiles(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{{"fake.pdf", []byte("not pdf")}, {"fake.pptx", officeFixture(t, "word/document.xml")}, {"binary.txt", []byte{'a', 0, 'b'}}, {"legacy.ppt", []byte("legacy")}, {"script.exe", []byte("MZ")}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			if _, err := detectMaterialFormat(tt.name, int64(len(tt.data)), reader); err == nil {
				t.Fatal("invalid material accepted")
			}
		})
	}
}
func officeFixture(t *testing.T, mainPart string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range []string{"[Content_Types].xml", mainPart} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write([]byte("<xml/>")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
