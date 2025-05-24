package adf

import (
	"encoding/json"
	"testing"
)

func TestInlineCode(t *testing.T) {
	converter := NewAdfConverter()

	markdown := "This has `inline code` in it."
	adf, err := converter.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
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
	converter := NewAdfConverter()

	markdown := "```\nfunction hello() {\n    console.log(\"Hello\");\n}\n```"
	adf, err := converter.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
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
	converter := NewAdfConverter()

	markdown := "```go\npackage main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	adf, err := converter.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
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
	converter := NewAdfConverter()

	markdown := `# Code Features

This paragraph has ` + "`inline code`" + ` in it.

` + "```javascript\nconsole.log('hello');\n```" + `

Another paragraph with ` + "`more code`" + `.`

	adf, err := converter.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
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
	converter := NewAdfConverter()

	markdown := "Contact `@jorres@nebius.com` or @admin@example.com for help."
	adf, err := converter.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	paragraph := adf.Content[0]

	// Should find both a code mark and a mention mark
	foundCodeMention := false
	foundRegularMention := false

	for _, node := range paragraph.Content {
		if node.Text == "@jorres@nebius.com" {
			if len(node.Marks) == 1 && node.Marks[0].Type == "code" {
				foundCodeMention = true
			}
		}
		if node.Text == "@admin@example.com" {
			if len(node.Marks) == 1 && node.Marks[0].Type == "mention" {
				foundRegularMention = true
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
	converter := NewAdfConverter()

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
			adf, err := converter.ConvertToADF([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to convert markdown: %v", err)
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
	converter := NewAdfConverter()

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
			adf, err := converter.ConvertToADF([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to convert markdown: %v", err)
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

// func TestMarkdownLinks(t *testing.T) {
// 	converter := NewAdfConverter()

// 	tests := []struct {
// 		name         string
// 		markdown     string
// 		expectedText string
// 		expectedURL  string
// 	}{
// 		{
// 			name:         "Simple link",
// 			markdown:     "[Google](https://google.com)",
// 			expectedText: "Google",
// 			expectedURL:  "https://google.com",
// 		},
// 		{
// 			name:         "Link with text context",
// 			markdown:     "Visit [Google](https://google.com) for search",
// 			expectedText: "Google",
// 			expectedURL:  "https://google.com",
// 		},
// 		{
// 			name:         "Link at beginning",
// 			markdown:     "[Click here](https://example.com) to continue",
// 			expectedText: "Click here",
// 			expectedURL:  "https://example.com",
// 		},
// 		{
// 			name:         "Link at end",
// 			markdown:     "Go to [this page](https://example.com)",
// 			expectedText: "this page",
// 			expectedURL:  "https://example.com",
// 		},
// 		{
// 			name:         "Multiple links",
// 			markdown:     "Check [Google](https://google.com) and [GitHub](https://github.com)",
// 			expectedText: "Google",
// 			expectedURL:  "https://google.com",
// 		},
// 		{
// 			name:         "Link with special characters",
// 			markdown:     "[API docs](https://api.example.com/v1/docs?format=json)",
// 			expectedText: "API docs",
// 			expectedURL:  "https://api.example.com/v1/docs?format=json",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			adf, err := converter.ConvertToADF([]byte(tt.markdown))
// 			if err != nil {
// 				t.Fatalf("Failed to convert markdown: %v", err)
// 			}

// 			// Should have one paragraph
// 			if len(adf.Content) != 1 {
// 				t.Fatalf("Expected 1 content node, got %d", len(adf.Content))
// 			}

// 			paragraph := adf.Content[0]
// 			if paragraph.Type != "paragraph" {
// 				t.Fatalf("Expected paragraph, got %s", paragraph.Type)
// 			}

// 			// Find the link text node
// 			foundLink := false
// 			for _, node := range paragraph.Content {
// 				if node.Text == tt.expectedText && len(node.Marks) == 1 && node.Marks[0].Type == "link" {
// 					if href, ok := node.Marks[0].Attrs["href"].(string); ok && href == tt.expectedURL {
// 						foundLink = true
// 						break
// 					}
// 				}
// 			}

// 			if !foundLink {
// 				t.Fatalf("Could not find link text '%s' with URL '%s' in: %+v", tt.expectedText, tt.expectedURL, paragraph.Content)
// 			}
// 		})
// 	}
// }

// func TestMarkdownLinksEdgeCases(t *testing.T) {
// 	converter := NewAdfConverter()

// 	tests := []struct {
// 		name     string
// 		markdown string
// 		hasLink  bool
// 	}{
// 		{
// 			name:     "Malformed link - missing closing bracket",
// 			markdown: "[Google(https://google.com)",
// 			hasLink:  false,
// 		},
// 		{
// 			name:     "Malformed link - missing closing paren",
// 			markdown: "[Google](https://google.com",
// 			hasLink:  false,
// 		},
// 		{
// 			name:     "Empty link text",
// 			markdown: "[](https://google.com)",
// 			hasLink:  false, // Empty text shouldn't create a link
// 		},
// 		{
// 			name:     "Empty URL",
// 			markdown: "[Google]()",
// 			hasLink:  false, // Empty URL shouldn't create a link
// 		},
// 		{
// 			name:     "Just brackets",
// 			markdown: "[not a link]",
// 			hasLink:  false,
// 		},
// 		{
// 			name:     "Nested brackets",
// 			markdown: "[Link with [nested] text](https://example.com)",
// 			hasLink:  true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			adf, err := converter.ConvertToADF([]byte(tt.markdown))
// 			if err != nil {
// 				t.Fatalf("Failed to convert markdown: %v", err)
// 			}

// 			paragraph := adf.Content[0]
// 			foundLink := false

// 			for _, node := range paragraph.Content {
// 				if len(node.Marks) > 0 {
// 					for _, mark := range node.Marks {
// 						if mark.Type == "link" {
// 							foundLink = true
// 							break
// 						}
// 					}
// 				}
// 			}

// 			if foundLink != tt.hasLink {
// 				if tt.hasLink {
// 					t.Fatalf("Expected to find link mark but didn't in: %+v", paragraph.Content)
// 				} else {
// 					t.Fatalf("Found unexpected link mark in: %+v", paragraph.Content)
// 				}
// 			}
// 		})
// 	}
// }

func TestValidADFOutput(t *testing.T) {
	converter := NewAdfConverter()

	markdown := "# Test\n\n`code` and @user@example.com\n\n```go\nfmt.Println(\"test\")\n```"
	adf, err := converter.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	// Test that the ADF can be marshaled to valid JSON
	jsonBytes, err := json.Marshal(adf)
	if err != nil {
		t.Fatalf("Failed to marshal ADF to JSON: %v", err)
	}

	// Test that it can be unmarshaled back
	var unmarshaled ADFDocument
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ADF JSON: %v", err)
	}

	// Basic structure check
	if unmarshaled.Version != 1 || unmarshaled.Type != "doc" {
		t.Fatalf("Invalid ADF document structure")
	}
}
