package materialparser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"testing"

	"whatsnext/backend/internal/model"
)

func TestParseSupportedFormats(t *testing.T) {
	parser := New(500)
	tests := []struct {
		name   string
		data   []byte
		source string
		want   string
	}{{"notes.txt", []byte("first line\nsecond line"), "text", "first line"}, {"notes.md", []byte("# Heading\nbody"), "text", "# Heading"}, {"slides.pptx", officeArchive(t, map[string]string{"[Content_Types].xml": "<Types/>", "ppt/presentation.xml": "<p:presentation xmlns:p=\"p\"/>", "ppt/slides/slide1.xml": "<p:sld xmlns:p=\"p\" xmlns:a=\"a\"><a:t>TCP slide</a:t></p:sld>"}), "slide", "TCP slide"}, {"handout.docx", officeArchive(t, map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<w:document xmlns:w=\"w\"><w:body><w:p><w:r><w:t>DOCX paragraph</w:t></w:r></w:p></w:body></w:document>"}), "document", "DOCX paragraph"}, {"book.pdf", minimalPDF(), "page", "Hello PDF"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			units, err := parser.Parse(context.Background(), tt.name, int64(len(tt.data)), reader)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(units) == 0 || units[0].SourceType != tt.source || !bytes.Contains([]byte(units[0].Text), []byte(tt.want)) {
				t.Fatalf("units=%+v", units)
			}
		})
	}
}
func TestChunkerSplitsLongContentWithOverlap(t *testing.T) {
	material := model.LearningMaterial{ID: "m1", UserID: "u1", LearningSpaceID: "s1"}
	text := string(bytes.Repeat([]byte("a"), 25))
	chunks := NewChunker(10, 2).Build(material, []SourceUnit{{Text: text, SourceType: "text", Position: 1}})
	if len(chunks) != 3 {
		t.Fatalf("chunks=%d", len(chunks))
	}
	if chunks[0].CharCount != 10 || chunks[1].CharCount != 10 || chunks[2].CharCount != 9 {
		t.Fatalf("sizes=%d,%d,%d", chunks[0].CharCount, chunks[1].CharCount, chunks[2].CharCount)
	}
	if chunks[0].TokenEstimate < 1 {
		t.Fatal("missing token estimate")
	}
}

type fakeOCR struct {
	pages []int
}

func (f *fakeOCR) Recognize(_ context.Context, _ Reader, _ int64, pages []int) (map[int]string, error) {
	f.pages = append([]int(nil), pages...)
	return map[int]string{1: "扫描识别出的计算机网络课程内容，包含 TCP 可靠传输与滑动窗口。"}, nil
}

func TestParsePDFUsesOCRForPagesWithTooLittleText(t *testing.T) {
	ocr := &fakeOCR{}
	parser := New(500, ocr)
	data := minimalPDF()
	units, err := parser.Parse(context.Background(), "scanned.pdf", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(ocr.pages) != 1 || ocr.pages[0] != 1 {
		t.Fatalf("OCR pages = %v", ocr.pages)
	}
	if len(units) != 1 || !bytes.Contains([]byte(units[0].Text), []byte("扫描识别")) {
		t.Fatalf("units = %+v", units)
	}
}
func officeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
func minimalPDF() []byte {
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 300] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>", "<< /Length 42 >>\nstream\nBT /F1 12 Tf 72 200 Td (Hello PDF) Tj ET\nendstream", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"}
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return output.Bytes()
}
