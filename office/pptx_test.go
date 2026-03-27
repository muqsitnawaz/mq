package office

import (
	"archive/zip"
	"bytes"
	"fmt"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestPPTX(slides []string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Content types
	ct, _ := w.Create("[Content_Types].xml")
	ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
</Types>`))

	// Relationships
	rels, _ := w.Create("_rels/.rels")
	rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`))

	for i, slideXML := range slides {
		sf, _ := w.Create(fmt.Sprintf("ppt/slides/slide%d.xml", i+1))
		sf.Write([]byte(slideXML))
	}

	w.Close()
	return buf.Bytes()
}

func TestPPTXParser(t *testing.T) {
	slide1 := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:nvPr><p:ph type="ctrTitle"/></p:nvPr></p:nvSpPr>
        <p:txBody>
          <a:p><a:r><a:t>Welcome to the Future</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
      <p:sp>
        <p:nvSpPr><p:nvPr><p:ph type="body"/></p:nvPr></p:nvSpPr>
        <p:txBody>
          <a:p><a:r><a:t>AI agents are changing everything.</a:t></a:r></a:p>
          <a:p><a:r><a:t>This presentation covers the key trends.</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`

	slide2 := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr>
        <p:txBody>
          <a:p><a:r><a:t>Market Overview</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
      <p:sp>
        <p:txBody>
          <a:p><a:r><a:t>Revenue grew 150% in Q4.</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`

	content := buildTestPPTX([]string{slide1, slide2})
	p := NewPPTXParser()
	doc, err := p.Parse(content, "deck.pptx")
	require.NoError(t, err)

	assert.Equal(t, mq.FormatPPTX, doc.Format())
	assert.Contains(t, doc.Title(), "2 slides")

	// Slide titles as H1
	h1s := doc.GetHeadings(1)
	require.Len(t, h1s, 2)
	assert.Contains(t, h1s[0].Text, "Welcome to the Future")
	assert.Contains(t, h1s[1].Text, "Market Overview")

	// Readable text
	readable := doc.ReadableText()
	assert.Contains(t, readable, "Welcome to the Future")
	assert.Contains(t, readable, "AI agents are changing everything")
	assert.Contains(t, readable, "Market Overview")
	assert.Contains(t, readable, "Revenue grew 150%")
}

func TestPPTXParserNoTitle(t *testing.T) {
	// Slide with no title placeholder
	slide := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p><a:r><a:t>Just some content.</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`

	content := buildTestPPTX([]string{slide})
	p := NewPPTXParser()
	doc, err := p.Parse(content, "simple.pptx")
	require.NoError(t, err)

	h1s := doc.GetHeadings(1)
	require.Len(t, h1s, 1)
	assert.Contains(t, h1s[0].Text, "Slide 1") // fallback title
}
