package pdf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserFormat(t *testing.T) {
	p := NewParser()
	assert.Equal(t, mq.FormatPDF, p.Format())
}

func TestParseValidPDF(t *testing.T) {
	// Create a minimal valid PDF for testing
	// This is a tiny but valid PDF file
	minimalPDF := []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 44 >>
stream
BT /F1 12 Tf 100 700 Td (Hello PDF) Tj ET
endstream
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000214 00000 n
trailer
<< /Size 5 /Root 1 0 R >>
startxref
312
%%EOF`)

	parser := NewParser()
	doc, err := parser.Parse(minimalPDF, "test.pdf")
	require.NoError(t, err)

	// Check format
	assert.Equal(t, mq.FormatPDF, doc.Format())

	// Check source preserved
	assert.Equal(t, minimalPDF, doc.Source())

	// Check path preserved
	assert.Equal(t, "test.pdf", doc.Path())
}

func TestParseInvalidPDF(t *testing.T) {
	// Not a PDF - should still parse but with empty content
	parser := NewParser()
	doc, err := parser.Parse([]byte("not a pdf"), "test.pdf")
	require.NoError(t, err)

	// ReadableText should be empty for non-PDF
	assert.Empty(t, doc.ReadableText())
}

func TestParsePDFFile(t *testing.T) {
	testFile := filepath.Join("testdata", "attention.pdf")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("attention.pdf test file not found")
	}

	parser := NewParser()
	doc, err := parser.ParseFile(testFile)
	require.NoError(t, err)

	// Should be PDF format
	assert.Equal(t, mq.FormatPDF, doc.Format())

	// Should have content (at minimum the placeholder)
	readable := doc.ReadableText()
	t.Logf("Attention paper: %d chars readable", len(readable))
	assert.NotEmpty(t, readable)

	// Check source size
	source := doc.Source()
	t.Logf("Attention paper: %d bytes source", len(source))
	assert.Greater(t, len(source), 1000000) // Should be >1MB
}

func TestParseLargePDF(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	testFile := filepath.Join("testdata", "gpt3.pdf")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("gpt3.pdf test file not found")
	}

	parser := NewParser()
	doc, err := parser.ParseFile(testFile)
	require.NoError(t, err)

	// Should have content
	readable := doc.ReadableText()
	t.Logf("GPT-3 paper: %d chars readable", len(readable))
	assert.NotEmpty(t, readable)

	// Check source size
	source := doc.Source()
	t.Logf("GPT-3 paper: %d bytes source", len(source))
	assert.Greater(t, len(source), 5000000) // Should be >5MB
}

func TestParseFileNotFound(t *testing.T) {
	parser := NewParser()
	_, err := parser.ParseFile("nonexistent.pdf")
	require.Error(t, err)

	// Should be a ParseError
	var parseErr *mq.ParseError
	assert.ErrorAs(t, err, &parseErr)
	assert.Equal(t, mq.FormatPDF, parseErr.Format)
}

func TestParserOptions(t *testing.T) {
	// Test heading inference option
	parser := NewParser(WithHeadingInference(true))
	assert.NotNil(t, parser)

	// Test table detection option
	parser = NewParser(WithTableDetection(false))
	assert.NotNil(t, parser)

	// Test heading ratio option
	parser = NewParser(WithHeadingRatio(1.3))
	assert.NotNil(t, parser)

	// Test combining options
	parser = NewParser(
		WithHeadingInference(true),
		WithTableDetection(true),
		WithHeadingRatio(1.2),
	)
	assert.NotNil(t, parser)
}

func TestConvenienceFunctions(t *testing.T) {
	minimalPDF := []byte(`%PDF-1.0
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/Parent 2 0 R>>endobj xref 0 4 0000000000 65535 f 0000000009 00000 n 0000000052 00000 n 0000000101 00000 n trailer<</Size 4/Root 1 0 R>>startxref 150 %%EOF`)

	// Test ParsePDF
	doc, err := ParsePDF(minimalPDF, "test.pdf")
	require.NoError(t, err)
	assert.Equal(t, mq.FormatPDF, doc.Format())
}

func TestParsePDFFileConvenience(t *testing.T) {
	testFile := filepath.Join("testdata", "attention.pdf")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("attention.pdf test file not found")
	}

	doc, err := ParsePDFFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, mq.FormatPDF, doc.Format())
}

func TestNormalizeExtractedTextTracksPages(t *testing.T) {
	raw := "Cover Page\n\fSection One\nLine Two"

	text, linePages := normalizeExtractedText(raw, 2)

	assert.Equal(t, "Cover Page\n\nSection One\nLine Two", text)
	assert.Equal(t, []int{1, 1, 2, 2}, linePages)
}

func TestInferStructuredHeadingsFromTextFallback(t *testing.T) {
	text := strings.Join([]string{
		"Application Form",
		"",
		"Applicant Details:",
		"- Legal Name",
		"- Mailing Address",
		"",
		"Contact Information:",
		"- Email Address",
		"- Phone Number",
	}, "\n")
	linePages := []int{1, 1, 1, 1, 1, 1, 2, 2, 2}

	headings := inferStructuredHeadings(text, linePages)
	require.Len(t, headings, 7)

	assert.Equal(t, "Application Form", headings[0].Text)
	assert.Equal(t, 1, headings[0].Level)
	assert.Equal(t, 1, headings[0].Page)

	assert.Equal(t, "Applicant Details", headings[1].Text)
	assert.Equal(t, 2, headings[1].Level)
	assert.Equal(t, "Legal Name", headings[2].Text)
	assert.Equal(t, 3, headings[2].Level)
	assert.Equal(t, "Mailing Address", headings[3].Text)
	assert.Equal(t, 3, headings[3].Level)
	assert.Equal(t, "Contact Information", headings[4].Text)
	assert.Equal(t, 2, headings[4].Level)
	assert.Equal(t, 2, headings[4].Page)
	assert.Equal(t, "Email Address", headings[5].Text)
	assert.Equal(t, 3, headings[5].Level)
	assert.Equal(t, "Phone Number", headings[6].Text)
}

func TestFallbackTextStructureBuildsPDFTree(t *testing.T) {
	text := strings.Join([]string{
		"Application Form",
		"",
		"Applicant Details:",
		"- Legal Name",
		"- Mailing Address",
		"",
		"Contact Information:",
		"- Email Address",
		"- Phone Number",
	}, "\n")
	linePages := []int{1, 1, 1, 1, 1, 1, 2, 2, 2}

	headings := inferStructuredHeadings(text, linePages)
	sections := (&extractor{}).buildSections(headings, text)
	doc := mq.NewDocument(
		[]byte("%PDF-1.4"), "fallback.pdf", mq.FormatPDF, "",
		headings, sections, nil, nil, nil, nil, nil,
		text,
	)
	doc.SetPageCount(2)

	tree := doc.BuildTree()
	require.NotEmpty(t, tree.Root)

	output := tree.String()
	assert.Contains(t, output, "Application Form")
	assert.Contains(t, output, "Applicant Details")
	assert.Contains(t, output, "Legal Name")
	assert.Contains(t, output, "(2 pages)")
	assert.Contains(t, output, "(p. 2)")
}
