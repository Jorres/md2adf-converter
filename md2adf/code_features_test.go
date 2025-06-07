package md2adf

import (
	"encoding/json"
	"github.com/jorres/md2adf-translator/adf"
	"testing"
)

func TestInlineCode(t *testing.T) {
	translator := NewTranslator()

	markdown := "This has `inline code` in it."
	adf, err := translator.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Should have one paragraph
	if len(adf.Content) != 1 {
		t.Fatalf("Expected 1 content node, got %d", len(adf.Content))
	}

	paragraph := adf.Content[0]
	if paragraph.Type != "paragraph" {
		t.Fatalf("Expected paragraph, got %s", paragraph.Type)
	}

	// Should have 4 text nodes: "This has ", "inline code" (with code mark), " in it", "."
	if len(paragraph.Content) != 4 {
		t.Fatalf("Expected 4 text nodes, got %d", len(paragraph.Content))
	}

	// Check the code text node
	codeNode := paragraph.Content[1]
	if codeNode.Type != "text" || codeNode.Text != "inline code" {
		t.Fatalf("Expected code text 'inline code', got %s: %s", codeNode.Type, codeNode.Text)
	}

	// Check it has a code mark
	if len(codeNode.Marks) != 1 || codeNode.Marks[0].Type != "code" {
		t.Fatalf("Expected code mark, got %+v", codeNode.Marks)
	}
}

func TestCodeBlockWithoutLanguage(t *testing.T) {
	translator := NewTranslator()

	markdown := "```\nfunction hello() {\n    console.log(\"Hello\");\n}\n```"
	adf, err := translator.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Should have one code block
	if len(adf.Content) != 1 {
		t.Fatalf("Expected 1 content node, got %d", len(adf.Content))
	}

	codeBlock := adf.Content[0]
	if codeBlock.Type != "codeBlock" {
		t.Fatalf("Expected codeBlock, got %s", codeBlock.Type)
	}

	// Should have no language attribute (empty attrs map)
	if len(codeBlock.Attrs) != 0 {
		t.Fatalf("Expected no attributes, got %+v", codeBlock.Attrs)
	}

	// Should have one text node with the code
	if len(codeBlock.Content) != 1 {
		t.Fatalf("Expected 1 text node, got %d", len(codeBlock.Content))
	}

	textNode := codeBlock.Content[0]
	expectedCode := "function hello() {\n    console.log(\"Hello\");\n}"
	if textNode.Text != expectedCode {
		t.Fatalf("Expected code %q, got %q", expectedCode, textNode.Text)
	}
}

func TestCodeBlockWithLanguage(t *testing.T) {
	translator := NewTranslator()

	markdown := "```go\npackage main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	adf, err := translator.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Should have one code block
	if len(adf.Content) != 1 {
		t.Fatalf("Expected 1 content node, got %d", len(adf.Content))
	}

	codeBlock := adf.Content[0]
	if codeBlock.Type != "codeBlock" {
		t.Fatalf("Expected codeBlock, got %s", codeBlock.Type)
	}

	// Should have language attribute
	if codeBlock.Attrs["language"] != "go" {
		t.Fatalf("Expected language 'go', got %v", codeBlock.Attrs["language"])
	}

	// Check the code content
	textNode := codeBlock.Content[0]
	expectedCode := "package main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}"
	if textNode.Text != expectedCode {
		t.Fatalf("Expected code %q, got %q", expectedCode, textNode.Text)
	}
}

func TestMixedCodeContent(t *testing.T) {
	translator := NewTranslator()

	markdown := `# Code Features

This paragraph has ` + "`inline code`" + ` in it.

` + "```javascript\nconsole.log('hello');\n```" + `

Another paragraph with ` + "`more code`" + `.`

	adf, err := translator.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Should have: heading, paragraph, code block, paragraph
	if len(adf.Content) != 4 {
		t.Fatalf("Expected 4 content nodes, got %d", len(adf.Content))
	}

	// Check heading
	if adf.Content[0].Type != "heading" {
		t.Fatalf("Expected heading, got %s", adf.Content[0].Type)
	}

	// Check first paragraph with inline code
	para1 := adf.Content[1]
	if para1.Type != "paragraph" {
		t.Fatalf("Expected paragraph, got %s", para1.Type)
	}

	// Should find a text node with code mark
	foundCode := false
	for _, node := range para1.Content {
		if node.Text == "inline code" && len(node.Marks) == 1 && node.Marks[0].Type == "code" {
			foundCode = true
			break
		}
	}
	if !foundCode {
		t.Fatalf("Could not find inline code in first paragraph")
	}

	// Check code block
	codeBlock := adf.Content[2]
	if codeBlock.Type != "codeBlock" {
		t.Fatalf("Expected codeBlock, got %s", codeBlock.Type)
	}
	if codeBlock.Attrs["language"] != "javascript" {
		t.Fatalf("Expected javascript language, got %v", codeBlock.Attrs["language"])
	}

	// Check second paragraph with inline code
	para2 := adf.Content[3]
	foundCode2 := false
	for _, node := range para2.Content {
		if node.Text == "more code" && len(node.Marks) == 1 && node.Marks[0].Type == "code" {
			foundCode2 = true
			break
		}
	}
	if !foundCode2 {
		t.Fatalf("Could not find inline code in second paragraph")
	}
}

func TestCodeWithPeopleMentions(t *testing.T) {
	translator := NewTranslator()

	markdown := "Contact `@jorres@nebius.com` or @admin@example.com for help."
	adf, err := translator.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	paragraph := adf.Content[0]

	// Should find both a code mark and a mention node
	foundCodeMention := false
	foundRegularMention := false

	for _, node := range paragraph.Content {
		// Code mention should be a text node with code mark
		if node.Text == "@jorres@nebius.com" {
			if len(node.Marks) == 1 && node.Marks[0].Type == "code" {
				foundCodeMention = true
			}
		}
		// Regular mention should be a standalone mention node
		if node.Type == "mention" {
			if id, ok := node.Attrs["id"].(string); ok && id == "@admin@example.com" {
				if text, ok := node.Attrs["text"].(string); ok && text == "admin" {
					foundRegularMention = true
				}
			}
		}
	}

	if !foundCodeMention {
		t.Fatalf("Could not find code-marked mention")
	}
	if !foundRegularMention {
		t.Fatalf("Could not find regular mention")
	}
}

func TestSingleLineInlineCode(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "Single word code",
			markdown: "`code`",
			expected: "code",
		},
		{
			name:     "Code with spaces",
			markdown: "`hello world`",
			expected: "hello world",
		},
		{
			name:     "Code with special characters",
			markdown: "`var x = 'test';`",
			expected: "var x = 'test';",
		},
		{
			name:     "Code in middle of sentence",
			markdown: "Use the `console.log()` function to debug.",
			expected: "console.log()",
		},
		{
			name:     "Multiple code spans",
			markdown: "Run `npm install` then `npm start`.",
			expected: "npm install",
		},
		{
			name:     "Code at beginning",
			markdown: "`git status` shows current state",
			expected: "git status",
		},
		{
			name:     "Code at end",
			markdown: "Execute the command `exit`",
			expected: "exit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf, err := translator.TranslateToADF([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to translate markdown: %v", err)
			}

			// Should have one paragraph
			if len(adf.Content) != 1 {
				t.Fatalf("Expected 1 content node, got %d", len(adf.Content))
			}

			paragraph := adf.Content[0]
			if paragraph.Type != "paragraph" {
				t.Fatalf("Expected paragraph, got %s", paragraph.Type)
			}

			// Find the code text node
			foundCode := false
			for _, node := range paragraph.Content {
				if node.Text == tt.expected && len(node.Marks) == 1 && node.Marks[0].Type == "code" {
					foundCode = true
					break
				}
			}

			if !foundCode {
				t.Fatalf("Could not find code text '%s' with code mark in: %+v", tt.expected, paragraph.Content)
			}
		})
	}
}

func TestInlineCodeEdgeCases(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name     string
		markdown string
		hasCode  bool
	}{
		{
			name:     "Empty code spans",
			markdown: "Text with `` empty code",
			hasCode:  false, // Empty code spans shouldn't create code marks
		},
		{
			name:     "Unmatched backtick",
			markdown: "Text with ` unmatched backtick",
			hasCode:  false, // Should treat as plain text
		},
		{
			name:     "Code with newline inside",
			markdown: "This has `code\nwith newline` in it",
			hasCode:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf, err := translator.TranslateToADF([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to translate markdown: %v", err)
			}

			paragraph := adf.Content[0]
			foundCode := false

			for _, node := range paragraph.Content {
				if len(node.Marks) > 0 {
					for _, mark := range node.Marks {
						if mark.Type == "code" {
							foundCode = true
							break
						}
					}
				}
			}

			if foundCode != tt.hasCode {
				if tt.hasCode {
					t.Fatalf("Expected to find code mark but didn't in: %+v", paragraph.Content)
				} else {
					t.Fatalf("Found unexpected code mark in: %+v", paragraph.Content)
				}
			}
		})
	}
}

func TestValidADFOutput(t *testing.T) {
	translator := NewTranslator()

	markdown := "# Test\n\n`code` and @user@example.com\n\n```go\nfmt.Println(\"test\")\n```"
	translated, err := translator.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Test that the ADF can be marshaled to valid JSON
	jsonBytes, err := json.Marshal(translated)
	if err != nil {
		t.Fatalf("Failed to marshal ADF to JSON: %v", err)
	}

	// Test that it can be unmarshaled back
	var unmarshaled adf.ADFDocument
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ADF JSON: %v", err)
	}

	// Basic structure check
	if unmarshaled.Version != 1 || unmarshaled.Type != "doc" {
		t.Fatalf("Invalid ADF document structure")
	}
}
