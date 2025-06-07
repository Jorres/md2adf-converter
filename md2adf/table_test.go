package md2adf

import (
	"encoding/json"
	"github.com/jorres/md2adf-translator/adf"
	"github.com/jorres/md2adf-translator/adf2md"
	"testing"
)

func TestTableTranslation(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name     string
		markdown string
		expected func(*adf.ADFDocument) bool
	}{
		{
			name: "simple 2x2 table",
			markdown: `| **a** | **b** |
| ----- | ----- |
| c     | d     |`,
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != adf.NodeTable {
					return false
				}
				table := doc.Content[0]

				// Check table attributes
				if table.Attrs == nil {
					return false
				}
				if table.Attrs["isNumberColumnEnabled"] != false || table.Attrs["layout"] != "align-start" {
					return false
				}

				// Should have 2 rows
				if len(table.Content) != 2 {
					return false
				}

				// Header row
				headerRow := table.Content[0]
				if headerRow.Type != adf.ChildNodeTableRow || len(headerRow.Content) != 2 {
					return false
				}

				// Check first header cell
				headerCell1 := headerRow.Content[0]
				if headerCell1.Type != adf.ChildNodeTableHeader {
					return false
				}
				if len(headerCell1.Content) != 1 || headerCell1.Content[0].Type != adf.NodeParagraph {
					return false
				}
				paragraph1 := headerCell1.Content[0]
				if len(paragraph1.Content) != 1 || paragraph1.Content[0].Type != adf.ChildNodeText {
					return false
				}
				textNode1 := paragraph1.Content[0]
				if textNode1.Text != "a" || len(textNode1.Marks) != 1 || textNode1.Marks[0].Type != adf.MarkStrong {
					return false
				}

				// Check second header cell
				headerCell2 := headerRow.Content[1]
				if headerCell2.Type != adf.ChildNodeTableHeader {
					return false
				}
				if len(headerCell2.Content) != 1 || headerCell2.Content[0].Type != adf.NodeParagraph {
					return false
				}
				paragraph2 := headerCell2.Content[0]
				if len(paragraph2.Content) != 1 || paragraph2.Content[0].Type != adf.ChildNodeText {
					return false
				}
				textNode2 := paragraph2.Content[0]
				if textNode2.Text != "b" || len(textNode2.Marks) != 1 || textNode2.Marks[0].Type != adf.MarkStrong {
					return false
				}

				// Data row
				dataRow := table.Content[1]
				if dataRow.Type != adf.ChildNodeTableRow || len(dataRow.Content) != 2 {
					return false
				}

				// Check first data cell
				dataCell1 := dataRow.Content[0]
				if dataCell1.Type != adf.ChildNodeTableCell {
					return false
				}
				if len(dataCell1.Content) != 1 || dataCell1.Content[0].Type != adf.NodeParagraph {
					return false
				}
				dataParagraph1 := dataCell1.Content[0]
				if len(dataParagraph1.Content) != 1 || dataParagraph1.Content[0].Type != adf.ChildNodeText {
					return false
				}
				dataTextNode1 := dataParagraph1.Content[0]
				if dataTextNode1.Text != "c" || len(dataTextNode1.Marks) != 0 {
					return false
				}

				// Check second data cell
				dataCell2 := dataRow.Content[1]
				if dataCell2.Type != adf.ChildNodeTableCell {
					return false
				}
				if len(dataCell2.Content) != 1 || dataCell2.Content[0].Type != adf.NodeParagraph {
					return false
				}
				dataParagraph2 := dataCell2.Content[0]
				if len(dataParagraph2.Content) != 1 || dataParagraph2.Content[0].Type != adf.ChildNodeText {
					return false
				}
				dataTextNode2 := dataParagraph2.Content[0]
				return dataTextNode2.Text == "d" && len(dataTextNode2.Marks) == 0
			},
		},
		{
			name: "3x3 table with mixed content",
			markdown: `| **Header 1** | **Header 2** | **Header 3** |
| ------------ | ------------ | ------------ |
| Simple       | **Bold**     | Plain        |
| Text         | More         | Content      |`,
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != adf.NodeTable {
					return false
				}
				table := doc.Content[0]

				// Should have 3 rows (1 header + 2 data)
				if len(table.Content) != 3 {
					return false
				}

				// Header row should have 3 columns
				headerRow := table.Content[0]
				if headerRow.Type != adf.ChildNodeTableRow || len(headerRow.Content) != 3 {
					return false
				}

				// First data row should have 3 columns
				dataRow1 := table.Content[1]
				if dataRow1.Type != adf.ChildNodeTableRow || len(dataRow1.Content) != 3 {
					return false
				}

				// Check bold cell in data row
				boldCell := dataRow1.Content[1]
				if boldCell.Type != adf.ChildNodeTableCell {
					return false
				}
				if len(boldCell.Content) != 1 || boldCell.Content[0].Type != adf.NodeParagraph {
					return false
				}
				boldParagraph := boldCell.Content[0]
				if len(boldParagraph.Content) != 1 || boldParagraph.Content[0].Type != adf.ChildNodeText {
					return false
				}
				boldTextNode := boldParagraph.Content[0]
				if boldTextNode.Text != "Bold" || len(boldTextNode.Marks) != 1 || boldTextNode.Marks[0].Type != adf.MarkStrong {
					return false
				}

				// Second data row should have 3 columns
				dataRow2 := table.Content[2]
				return dataRow2.Type == adf.ChildNodeTableRow && len(dataRow2.Content) == 3
			},
		},
		{
			name: "table with empty cells",
			markdown: `| **A** | **B** |
| ----- | ----- |
|       | Data  |
| Text  |       |`,
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != adf.NodeTable {
					return false
				}
				table := doc.Content[0]

				// Should have 3 rows (1 header + 2 data)
				if len(table.Content) != 3 {
					return false
				}

				// Check first data row - first cell should be empty
				dataRow1 := table.Content[1]
				if dataRow1.Type != adf.ChildNodeTableRow || len(dataRow1.Content) != 2 {
					return false
				}

				emptyCell := dataRow1.Content[0]
				if emptyCell.Type != adf.ChildNodeTableCell {
					return false
				}
				// Empty cell should still have a paragraph, potentially with empty content
				if len(emptyCell.Content) != 1 || emptyCell.Content[0].Type != adf.NodeParagraph {
					return false
				}

				// Check second data row - second cell should be empty
				dataRow2 := table.Content[2]
				if dataRow2.Type != adf.ChildNodeTableRow || len(dataRow2.Content) != 2 {
					return false
				}

				emptyCell2 := dataRow2.Content[1]
				if emptyCell2.Type != adf.ChildNodeTableCell {
					return false
				}
				// Empty cell should still have a paragraph
				return len(emptyCell2.Content) == 1 && emptyCell2.Content[0].Type == adf.NodeParagraph
			},
		},
		{
			name: "single column table",
			markdown: `| **Single** |
| ---------- |
| Row 1      |
| Row 2      |`,
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != adf.NodeTable {
					return false
				}
				table := doc.Content[0]

				// Should have 3 rows (1 header + 2 data)
				if len(table.Content) != 3 {
					return false
				}

				// All rows should have exactly 1 column
				for i, row := range table.Content {
					if row.Type != adf.ChildNodeTableRow || len(row.Content) != 1 {
						return false
					}

					cell := row.Content[0]
					var expectedType adf.NodeType
					if i == 0 {
						expectedType = adf.ChildNodeTableHeader
					} else {
						expectedType = adf.ChildNodeTableCell
					}

					if cell.Type != expectedType {
						return false
					}
				}

				return true
			},
		},
		{
			name: "table with special characters",
			markdown: `| **Name** | **Value** |
| -------- | --------- |
| Test     | 100%      |
| Item     | $50       |`,
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != adf.NodeTable {
					return false
				}
				table := doc.Content[0]

				// Should have 3 rows
				if len(table.Content) != 3 {
					return false
				}

				// Check special characters are preserved
				dataRow1 := table.Content[1]
				percentCell := dataRow1.Content[1]
				if percentCell.Type != adf.ChildNodeTableCell {
					return false
				}
				percentParagraph := percentCell.Content[0]
				percentText := percentParagraph.Content[0]
				if percentText.Text != "100%" {
					return false
				}

				dataRow2 := table.Content[2]
				dollarCell := dataRow2.Content[1]
				if dollarCell.Type != adf.ChildNodeTableCell {
					return false
				}
				dollarParagraph := dollarCell.Content[0]
				dollarText := dollarParagraph.Content[0]
				return dollarText.Text == "$50"
			},
		},
		{
			name: "roundtrip test - markdown to ADF and back",
			markdown: `| **Name** | **Age** | **City** |
| -------- | ------- | -------- |
| Alice    | 25      | NYC      |
| Bob      | 30      | LA       |`,
			expected: func(doc *adf.ADFDocument) bool {
				// Basic structure validation
				if len(doc.Content) != 1 || doc.Content[0].Type != adf.NodeTable {
					return false
				}
				table := doc.Content[0]

				// Should have 3 rows (1 header + 2 data)
				if len(table.Content) != 3 {
					return false
				}

				// Each row should have 3 columns
				for _, row := range table.Content {
					if row.Type != adf.ChildNodeTableRow || len(row.Content) != 3 {
						return false
					}
				}

				// Verify some specific content
				headerRow := table.Content[0]
				nameHeader := headerRow.Content[0]
				if nameHeader.Type != adf.ChildNodeTableHeader {
					return false
				}
				nameHeaderText := nameHeader.Content[0].Content[0]
				if nameHeaderText.Text != "Name" || len(nameHeaderText.Marks) != 1 {
					return false
				}

				// Verify data content
				dataRow1 := table.Content[1]
				aliceCell := dataRow1.Content[0]
				if aliceCell.Type != adf.ChildNodeTableCell {
					return false
				}
				aliceText := aliceCell.Content[0].Content[0]
				return aliceText.Text == "Alice" && len(aliceText.Marks) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := translator.TranslateToADF([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("Translation failed: %v", err)
			}

			if !tt.expected(doc) {
				// Print the actual structure for debugging
				jsonBytes, _ := json.MarshalIndent(doc, "", "  ")
				t.Errorf("Test %s failed. Actual structure:\n%s", tt.name, string(jsonBytes))
			}
		})
	}
}

// TestTableRoundtrip tests that we can convert markdown table to ADF and back to markdown
func TestTableRoundtrip(t *testing.T) {
	md2adfTranslator := NewTranslator()
	adf2mdTranslator := adf2md.NewTranslator(adf2md.NewMarkdownTranslator())

	originalMarkdown := `| **Name** | **Age** | **City** |
| -------- | ------- | -------- |
| Alice    | 25      | NYC      |
| Bob      | 30      | LA       |`

	// Convert markdown to ADF
	adfDoc, err := md2adfTranslator.TranslateToADF([]byte(originalMarkdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown to ADF: %v", err)
	}

	// Verify ADF structure
	if len(adfDoc.Content) != 1 || adfDoc.Content[0].Type != adf.NodeTable {
		t.Fatal("Expected single table node in ADF")
	}

	// Convert ADF back to markdown - we need to create a document wrapper for the table
	docWrapper := &adf.ADFNode{
		Type:    "doc",
		Content: adfDoc.Content,
	}
	resultMarkdown := adf2mdTranslator.Translate(docWrapper)

	// The result should be a properly formatted table with boundaries
	if resultMarkdown == "" {
		// Debug: print the ADF structure
		jsonBytes, _ := json.MarshalIndent(adfDoc, "", "  ")
		t.Fatalf("Result markdown is empty. ADF structure:\n%s", string(jsonBytes))
	}

	// Test that we can parse the result again
	roundtripAdfDoc, err := md2adfTranslator.TranslateToADF([]byte(resultMarkdown))
	if err != nil {
		t.Fatalf("Failed to parse generated markdown: %v", err)
	}

	// Basic structure should be the same
	if len(roundtripAdfDoc.Content) != 1 || roundtripAdfDoc.Content[0].Type != adf.NodeTable {
		t.Fatal("Roundtrip failed: expected single table node")
	}

	roundtripTable := roundtripAdfDoc.Content[0]
	if len(roundtripTable.Content) != 3 { // 1 header + 2 data rows
		t.Fatalf("Roundtrip failed: expected 3 rows, got %d", len(roundtripTable.Content))
	}

	// Check header row
	headerRow := roundtripTable.Content[0]
	if len(headerRow.Content) != 3 {
		t.Fatalf("Roundtrip failed: expected 3 header columns, got %d", len(headerRow.Content))
	}

	// Check that headers are still bold
	nameHeader := headerRow.Content[0]
	if nameHeader.Type != adf.ChildNodeTableHeader {
		t.Fatal("Roundtrip failed: first column should be table header")
	}

	nameHeaderText := nameHeader.Content[0].Content[0]
	if nameHeaderText.Text != "Name" {
		t.Fatalf("Roundtrip failed: expected 'Name', got '%s'", nameHeaderText.Text)
	}
	if len(nameHeaderText.Marks) != 1 || nameHeaderText.Marks[0].Type != adf.MarkStrong {
		t.Fatal("Roundtrip failed: header should be bold")
	}

	// Check first data row content
	dataRow := roundtripTable.Content[1]
	if len(dataRow.Content) != 3 {
		t.Fatalf("Roundtrip failed: expected 3 data columns, got %d", len(dataRow.Content))
	}

	nameCell := dataRow.Content[0]
	if nameCell.Type != adf.ChildNodeTableCell {
		t.Fatal("Roundtrip failed: should be table cell")
	}

	nameCellText := nameCell.Content[0].Content[0]
	if nameCellText.Text != "Alice" {
		t.Fatalf("Roundtrip failed: expected 'Alice', got '%s'", nameCellText.Text)
	}

	t.Logf("Roundtrip test passed. Generated markdown:\n%s", resultMarkdown)
}

