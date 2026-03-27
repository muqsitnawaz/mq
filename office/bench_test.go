package office

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xuri/excelize/v2"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "testdata", name)
}

func BenchmarkParseCSV(b *testing.B) {
	content, err := os.ReadFile(testdataPath("sample.csv"))
	if err != nil {
		b.Fatal(err)
	}
	p := NewCSVParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "sample.csv")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseDOCX(b *testing.B) {
	content := buildTestDOCX(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Introduction</w:t></w:r></w:p>
    <w:p><w:r><w:t>This is the introduction paragraph with some content.</w:t></w:r></w:p>
    <w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr><w:r><w:t>Background</w:t></w:r></w:p>
    <w:p><w:r><w:t>Some background information here.</w:t></w:r></w:p>
    <w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr><w:r><w:t>Methods</w:t></w:r></w:p>
    <w:p><w:r><w:t>We used these methods to achieve results.</w:t></w:r></w:p>
    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Results</w:t></w:r></w:p>
    <w:p><w:r><w:t>The results were significant.</w:t></w:r></w:p>
    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Conclusion</w:t></w:r></w:p>
    <w:p><w:r><w:t>Final thoughts and future work.</w:t></w:r></w:p>
  </w:body>
</w:document>`)
	p := NewDOCXParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "report.docx")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseXLSX(b *testing.B) {
	// Build a test XLSX in memory
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Data")
	f.SetCellValue("Data", "A1", "Name")
	f.SetCellValue("Data", "B1", "Value")
	for i := 2; i <= 101; i++ {
		f.SetCellValue("Data", cellName("A", i), "item")
		f.SetCellValue("Data", cellName("B", i), i*100)
	}
	buf, _ := f.WriteToBuffer()
	f.Close()
	content := buf.Bytes()

	p := NewXLSXParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "data.xlsx")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func cellName(col string, row int) string {
	return col + string(rune('0'+row/100)) + string(rune('0'+(row%100)/10)) + string(rune('0'+row%10))
}

func BenchmarkParsePPTX(b *testing.B) {
	slide := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld><p:spTree>
    <p:sp><p:nvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr>
      <p:txBody><a:p><a:r><a:t>Slide Title</a:t></a:r></a:p></p:txBody></p:sp>
    <p:sp><p:txBody>
      <a:p><a:r><a:t>Bullet point one.</a:t></a:r></a:p>
      <a:p><a:r><a:t>Bullet point two.</a:t></a:r></a:p>
    </p:txBody></p:sp>
  </p:spTree></p:cSld>
</p:sld>`
	content := buildTestPPTX([]string{slide, slide, slide, slide, slide})
	p := NewPPTXParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "deck.pptx")
		if err != nil {
			b.Fatal(err)
		}
	}
}
