package md2adf

import (
	"encoding/json"
	"github.com/jorres/md2adf-translator/adf"
	"testing"
)

func TestPanelProcessing(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name     string
		markdown string
		expected func(*adf.ADFDocument) bool
	}{
		{
			name: "simple panel with default type",
			markdown: `{panel}
# Header

Paragraph content with inline code.

` + "```" + `
code block
` + "```" + `
{/panel}`,
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "panel" {
					return false
				}
				panel := doc.Content[0]

				// Check panel type attribute
				if panel.Attrs == nil || panel.Attrs["panelType"] != "info" {
					return false
				}

				// Should have 3 content items: heading, paragraph, code block
				return len(panel.Content) == 3 &&
					panel.Content[0].Type == "heading" &&
					panel.Content[1].Type == "paragraph" &&
					panel.Content[2].Type == "codeBlock"
			},
		},
		{
			name: "panel with custom type",
			markdown: `{panel:type=info}
TODO

{/panel}`,
			expected: func(doc *adf.ADFDocument) bool {
				if len(doc.Content) != 1 || doc.Content[0].Type != "panel" {
					return false
				}
				panel := doc.Content[0]

				// Check panel type attribute
				if panel.Attrs == nil || panel.Attrs["panelType"] != "info" {
					return false
				}

				// Should have 1 paragraph with "TODO" text
				return len(panel.Content) == 1 &&
					panel.Content[0].Type == "paragraph" &&
					len(panel.Content[0].Content) == 1 &&
					panel.Content[0].Content[0].Type == "text" &&
					panel.Content[0].Content[0].Text == "TODO"
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

