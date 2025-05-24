package adf

import (
	"encoding/json"
	"testing"
)

func TestInlineCode(t *testing.T) {
	parser := NewAdfParser()

	markdown := "This has `inline code` in it."
	adf, err := parser.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
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
	parser := NewAdfParser()

	markdown := "```\nfunction hello() {\n    console.log(\"Hello\");\n}\n```"
	adf, err := parser.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
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
	parser := NewAdfParser()

	markdown := "```go\npackage main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	adf, err := parser.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
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
	parser := NewAdfParser()

	markdown := `# Code Features

This paragraph has ` + "`inline code`" + ` in it.

` + "```javascript\nconsole.log('hello');\n```" + `

Another paragraph with ` + "`more code`" + `.`

	adf, err := parser.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
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
	parser := NewAdfParser()

	markdown := "Contact `@jorres@nebius.com` or @admin@example.com for help."
	adf, err := parser.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
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

func TestValidADFOutput(t *testing.T) {
	parser := NewAdfParser()

	markdown := "# Test\n\n`code` and @user@example.com\n\n```go\nfmt.Println(\"test\")\n```"
	adf, err := parser.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
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

