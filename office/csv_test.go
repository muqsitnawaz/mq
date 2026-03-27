package office

import (
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVParser(t *testing.T) {
	csv := []byte("name,age,city\nAlice,30,NYC\nBob,25,SF\nCharlie,35,LA\n")
	p := NewCSVParser()
	doc, err := p.Parse(csv, "people.csv")
	require.NoError(t, err)

	assert.Equal(t, mq.FormatCSV, doc.Format())
	assert.Contains(t, doc.Title(), "3 rows")
	assert.Contains(t, doc.Title(), "3 columns")

	// Column headers as H2
	headings := doc.GetHeadings(2)
	require.Len(t, headings, 3)
	assert.Equal(t, "name", headings[0].Text)
	assert.Equal(t, "age", headings[1].Text)
	assert.Equal(t, "city", headings[2].Text)

	// Table
	tables := doc.GetTables()
	require.Len(t, tables, 1)
	assert.Equal(t, []string{"name", "age", "city"}, tables[0].Headers)
	assert.Len(t, tables[0].Rows, 3)
	assert.Equal(t, "Alice", tables[0].Rows[0][0])

	// Readable text
	readable := doc.ReadableText()
	assert.Contains(t, readable, "name | age | city")
	assert.Contains(t, readable, "Alice | 30 | NYC")
}

func TestCSVParserEmpty(t *testing.T) {
	p := NewCSVParser()
	doc, err := p.Parse([]byte(""), "empty.csv")
	require.NoError(t, err)
	assert.Contains(t, doc.Title(), "empty")
}

func TestTSVParser(t *testing.T) {
	tsv := []byte("name\tage\nAlice\t30\nBob\t25\n")
	p := NewCSVParser()
	doc, err := p.Parse(tsv, "data.tsv")
	require.NoError(t, err)

	tables := doc.GetTables()
	require.Len(t, tables, 1)
	assert.Equal(t, []string{"name", "age"}, tables[0].Headers)
	assert.Len(t, tables[0].Rows, 2)
}
