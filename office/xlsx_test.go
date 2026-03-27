package office

import (
	"os"
	"path/filepath"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func buildTestXLSX(t *testing.T) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()

	// Sheet1 with employee data
	f.SetSheetName("Sheet1", "Employees")
	f.SetCellValue("Employees", "A1", "Name")
	f.SetCellValue("Employees", "B1", "Role")
	f.SetCellValue("Employees", "C1", "Salary")
	f.SetCellValue("Employees", "A2", "Alice")
	f.SetCellValue("Employees", "B2", "Engineer")
	f.SetCellValue("Employees", "C2", "120000")
	f.SetCellValue("Employees", "A3", "Bob")
	f.SetCellValue("Employees", "B3", "Designer")
	f.SetCellValue("Employees", "C3", "95000")

	// Sheet2 with department data
	idx, _ := f.NewSheet("Departments")
	_ = idx
	f.SetCellValue("Departments", "A1", "Department")
	f.SetCellValue("Departments", "B1", "Head")
	f.SetCellValue("Departments", "A2", "Engineering")
	f.SetCellValue("Departments", "B2", "Charlie")

	path := filepath.Join(t.TempDir(), "test.xlsx")
	require.NoError(t, f.SaveAs(path))
	return path
}

func TestXLSXParser(t *testing.T) {
	path := buildTestXLSX(t)

	p := NewXLSXParser()
	doc, err := p.ParseFile(path)
	require.NoError(t, err)

	assert.Equal(t, mq.FormatXLSX, doc.Format())
	assert.Contains(t, doc.Title(), "2 sheets")

	// Sheet names as H1
	h1s := doc.GetHeadings(1)
	require.Len(t, h1s, 2)
	assert.Equal(t, "Employees", h1s[0].Text)
	assert.Equal(t, "Departments", h1s[1].Text)

	// Column headers as H2
	h2s := doc.GetHeadings(2)
	assert.True(t, len(h2s) >= 5, "expected at least 5 column headers, got %d", len(h2s))

	// Tables
	tables := doc.GetTables()
	require.Len(t, tables, 2)
	assert.Equal(t, []string{"Name", "Role", "Salary"}, tables[0].Headers)
	assert.Len(t, tables[0].Rows, 2)
	assert.Equal(t, "Alice", tables[0].Rows[0][0])

	// Readable text
	readable := doc.ReadableText()
	assert.Contains(t, readable, "Employees")
	assert.Contains(t, readable, "Alice")
}

func TestXLSXParserFromBytes(t *testing.T) {
	path := buildTestXLSX(t)
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	p := NewXLSXParser()
	doc, err := p.Parse(content, "test.xlsx")
	require.NoError(t, err)

	h1s := doc.GetHeadings(1)
	require.Len(t, h1s, 2)
}
