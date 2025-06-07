package md2adf

import (
	"encoding/json"
	"github.com/jorres/md2adf-converter/adf"
	"testing"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
)

func TestCleanInterfaceStructure(t *testing.T) {
	parser := tree_sitter_markdown.NewAdfMarkdownParser()
	content := []byte("# Header\n\nParagraph with @user@domain.com")

	tree, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if tree == nil {
		t.Fatal("Tree should not be nil")
	}

	root := tree.RootNode()
	if root.Kind() != "document" {
		t.Errorf("Expected document root, got %s", root.Kind())
	}

	// Test that we can access the tree structure normally
	if root.ChildCount() == 0 {
		t.Error("Document should have children")
	}
}

func TestTextMarksProcessing(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name     string
		markdown string
		expected func(*adf.ADFDocument) bool
	}{
		{
			name:     "bold text",
			markdown: "**bold**",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				return textNode.Text == "bold" &&
					len(textNode.Marks) == 1 &&
					textNode.Marks[0].Type == "strong"
			},
		},
		{
			name:     "underlined text",
			markdown: "<u>underlined</u>",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				return textNode.Text == "underlined" &&
					len(textNode.Marks) == 1 &&
					textNode.Marks[0].Type == "underline"
			},
		},
		{
			name:     "strikethrough text",
			markdown: "~strikethrough~",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				return textNode.Text == "strikethrough" &&
					len(textNode.Marks) == 1 &&
					textNode.Marks[0].Type == "strike"
			},
		},
		{
			name:     "italic text",
			markdown: "_italic_",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				return textNode.Text == "italic" &&
					len(textNode.Marks) == 1 &&
					textNode.Marks[0].Type == "em"
			},
		},
		{
			name:     "combined marks - bold underlined text",
			markdown: "**<u>text</u>**",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				if textNode.Text != "text" || len(textNode.Marks) != 2 {
					return false
				}
				// Check for both strong and underline marks
				hasStrong := false
				hasUnderline := false
				for _, mark := range textNode.Marks {
					if mark.Type == "strong" {
						hasStrong = true
					}
					if mark.Type == "underline" {
						hasUnderline = true
					}
				}
				return hasStrong && hasUnderline
			},
		},
		{
			name:     "triple combined marks - strikethrough bold underlined text",
			markdown: "~**<u>text</u>**~",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				if textNode.Text != "text" || len(textNode.Marks) != 3 {
					return false
				}
				// Check for all three marks
				hasStrong := false
				hasUnderline := false
				hasStrike := false
				for _, mark := range textNode.Marks {
					switch mark.Type {
					case "strong":
						hasStrong = true
					case "underline":
						hasUnderline = true
					case "strike":
						hasStrike = true
					}
				}
				return hasStrong && hasUnderline && hasStrike
			},
		},
		{
			name:     "underline with nested content",
			markdown: "<u>**~text~**</u>",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				// The parser treats this as underlined content with the raw text, not nested formatting
				return textNode.Text == "**~text~**" &&
					len(textNode.Marks) == 1 &&
					textNode.Marks[0].Type == "underline"
			},
		},
		{
			name:     "partial combinations - strikethrough and underline",
			markdown: "~<u>text</u>~",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				if textNode.Text != "text" || len(textNode.Marks) != 2 {
					return false
				}
				// Check for strike and underline marks
				hasUnderline := false
				hasStrike := false
				for _, mark := range textNode.Marks {
					switch mark.Type {
					case "underline":
						hasUnderline = true
					case "strike":
						hasStrike = true
					}
				}
				return hasUnderline && hasStrike
			},
		},
		{
			name:     "bold and italic combination",
			markdown: "**_text_**",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 1 || paragraph.Content[0].Type != "text" {
					return false
				}
				textNode := paragraph.Content[0]
				if textNode.Text != "text" || len(textNode.Marks) != 2 {
					return false
				}
				// Check for strong and em marks
				hasStrong := false
				hasEm := false
				for _, mark := range textNode.Marks {
					switch mark.Type {
					case "strong":
						hasStrong = true
					case "em":
						hasEm = true
					}
				}
				return hasStrong && hasEm
			},
		},
		{
			name:     "text with marks in paragraph context",
			markdown: "This is **bold** and this is <u>underlined</u> text.",
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
					return false
				}
				paragraph := doc.Content[0]
				if len(paragraph.Content) != 6 {
					return false
				}
				// Check structure: text + bold + text + underlined + text + text
				return paragraph.Content[0].Type == "text" && paragraph.Content[0].Text == "This is " &&
					paragraph.Content[1].Type == "text" && paragraph.Content[1].Text == "bold" && len(paragraph.Content[1].Marks) == 1 && paragraph.Content[1].Marks[0].Type == "strong" &&
					paragraph.Content[2].Type == "text" && paragraph.Content[2].Text == " and this is " &&
					paragraph.Content[3].Type == "text" && paragraph.Content[3].Text == "underlined" && len(paragraph.Content[3].Marks) == 1 && paragraph.Content[3].Marks[0].Type == "underline" &&
					paragraph.Content[4].Type == "text" && paragraph.Content[4].Text == " text" &&
					paragraph.Content[5].Type == "text" && paragraph.Content[5].Text == "."
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
