package main

import "testing"

func TestBasicConversion(t *testing.T) {
	parser := NewMarkdownParser()
	markdown := "# Header\n\nParagraph with [link](https://example.com).\n"
	
	adfDoc, err := parser.ConvertToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}
	
	if len(adfDoc.Content) != 2 {
		t.Fatalf("Expected 2 content items, got %d", len(adfDoc.Content))
	}
	
	// Check heading
	heading := adfDoc.Content[0]
	if heading.Type != "heading" || heading.Attrs["level"] != 1 {
		t.Errorf("Expected h1 heading, got %v", heading)
	}
	
	// Check paragraph with link
	paragraph := adfDoc.Content[1]
	if paragraph.Type != "paragraph" || len(paragraph.Content) != 3 {
		t.Errorf("Expected paragraph with 3 parts, got %v", paragraph)
	}
	
	// Check link mark
	linkNode := paragraph.Content[1]
	if len(linkNode.Marks) != 1 || linkNode.Marks[0].Type != "link" {
		t.Errorf("Expected link mark, got %v", linkNode.Marks)
	}
}