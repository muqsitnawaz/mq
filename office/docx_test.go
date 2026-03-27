package office

import (
	"archive/zip"
	"bytes"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTestDOCX creates a minimal valid DOCX file in memory.
func buildTestDOCX(documentXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Content types
	ct, _ := w.Create("[Content_Types].xml")
	ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	// Relationships
	rels, _ := w.Create("_rels/.rels")
	rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	// Document
	doc, _ := w.Create("word/document.xml")
	doc.Write([]byte(documentXML))

	w.Close()
	return buf.Bytes()
}

func TestDOCXParser(t *testing.T) {
	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Introduction</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>This is the introduction paragraph.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
      <w:r><w:t>Background</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Some background information here.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Conclusion</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Final thoughts.</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`

	content := buildTestDOCX(docXML)
	p := NewDOCXParser()
	doc, err := p.Parse(content, "report.docx")
	require.NoError(t, err)

	assert.Equal(t, mq.FormatDOCX, doc.Format())
	assert.Equal(t, "Introduction", doc.Title())

	// Headings
	h1s := doc.GetHeadings(1)
	require.Len(t, h1s, 2)
	assert.Equal(t, "Introduction", h1s[0].Text)
	assert.Equal(t, "Conclusion", h1s[1].Text)

	h2s := doc.GetHeadings(2)
	require.Len(t, h2s, 1)
	assert.Equal(t, "Background", h2s[0].Text)

	// Readable text
	readable := doc.ReadableText()
	assert.Contains(t, readable, "Introduction")
	assert.Contains(t, readable, "introduction paragraph")
	assert.Contains(t, readable, "Conclusion")
}

func TestDOCXParserWithTable(t *testing.T) {
	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Data</w:t></w:r>
    </w:p>
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Name</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Age</w:t></w:r></w:p></w:tc>
      </w:tr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Alice</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>30</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`

	content := buildTestDOCX(docXML)
	p := NewDOCXParser()
	doc, err := p.Parse(content, "data.docx")
	require.NoError(t, err)

	tables := doc.GetTables()
	require.Len(t, tables, 1)
	assert.Equal(t, []string{"Name", "Age"}, tables[0].Headers)
	assert.Len(t, tables[0].Rows, 1)
	assert.Equal(t, "Alice", tables[0].Rows[0][0])
}
