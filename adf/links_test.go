package adf

import (
	"testing"
)

func TestSimpleInlineLink(t *testing.T) {
	converter := NewAdfConverter()
	markdown := "[link](https://example.com)"

	doc, err := converter.ConvertToADF([]byte(markdown), nil)
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	if len(doc.Content) != 1 {
		t.Fatalf("Expected 1 top-level element, got %d", len(doc.Content))
	}

	// Check paragraph
	paragraph := doc.Content[0]
	if paragraph.Type != "paragraph" {
		t.Errorf("Expected paragraph, got %s", paragraph.Type)
	}

	// Check text node with link mark
	if len(paragraph.Content) != 1 {
		t.Fatalf("Expected 1 text node, got %d", len(paragraph.Content))
	}

	textNode := paragraph.Content[0]
	if textNode.Type != "text" {
		t.Errorf("Expected text node, got %s", textNode.Type)
	}
	if textNode.Text != "link" {
		t.Errorf("Expected text 'link', got '%s'", textNode.Text)
	}

	// Check link mark
	if len(textNode.Marks) != 1 {
		t.Fatalf("Expected 1 mark, got %d", len(textNode.Marks))
	}
	
	linkMark := textNode.Marks[0]
	if linkMark.Type != "link" {
		t.Errorf("Expected link mark, got %s", linkMark.Type)
	}
	if href, exists := linkMark.Attrs["href"]; !exists || href != "https://example.com" {
		t.Errorf("Expected href 'https://example.com', got %v", href)
	}
}

func TestLinkInListItem(t *testing.T) {
	converter := NewAdfConverter()
	markdown := "1. Item with [link](https://test.com)"

	doc, err := converter.ConvertToADF([]byte(markdown), nil)
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	if len(doc.Content) != 1 {
		t.Fatalf("Expected 1 top-level element, got %d", len(doc.Content))
	}

	// Navigate to the paragraph inside list item
	orderedList := doc.Content[0]
	if orderedList.Type != "orderedList" {
		t.Errorf("Expected orderedList, got %s", orderedList.Type)
	}

	listItem := orderedList.Content[0]
	paragraph := listItem.Content[0]

	// Check text nodes: "Item with " + "link" (with mark)
	if len(paragraph.Content) != 2 {
		t.Fatalf("Expected 2 text nodes, got %d", len(paragraph.Content))
	}

	// Check first text node
	firstText := paragraph.Content[0]
	if firstText.Text != "Item with " {
		t.Errorf("Expected first text 'Item with ', got '%s'", firstText.Text)
	}

	// Check link text node
	linkText := paragraph.Content[1]
	if linkText.Text != "link" {
		t.Errorf("Expected link text 'link', got '%s'", linkText.Text)
	}
	if len(linkText.Marks) != 1 || linkText.Marks[0].Type != "link" {
		t.Errorf("Expected link mark on second text node")
	}
	if href, exists := linkText.Marks[0].Attrs["href"]; !exists || href != "https://test.com" {
		t.Errorf("Expected href 'https://test.com', got %v", href)
	}
}

func TestMultipleLinksInParagraph(t *testing.T) {
	converter := NewAdfConverter()
	markdown := "Check [Google](https://google.com) and [GitHub](https://github.com) for more info."

	doc, err := converter.ConvertToADF([]byte(markdown), nil)
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	paragraph := doc.Content[0]
	
	// Should have 6 text nodes: "Check ", "Google" (with mark), " and ", "GitHub" (with mark), " for more info", "."
	if len(paragraph.Content) != 6 {
		t.Fatalf("Expected 6 text nodes, got %d", len(paragraph.Content))
	}

	// Check first Google link
	googleText := paragraph.Content[1]
	if googleText.Text != "Google" {
		t.Errorf("Expected Google text, got '%s'", googleText.Text)
	}
	if len(googleText.Marks) != 1 || googleText.Marks[0].Type != "link" {
		t.Errorf("Expected link mark on Google text")
	}
	if href, exists := googleText.Marks[0].Attrs["href"]; !exists || href != "https://google.com" {
		t.Errorf("Expected Google href, got %v", href)
	}

	// Check GitHub link
	githubText := paragraph.Content[3]
	if githubText.Text != "GitHub" {
		t.Errorf("Expected GitHub text, got '%s'", githubText.Text)
	}
	if len(githubText.Marks) != 1 || githubText.Marks[0].Type != "link" {
		t.Errorf("Expected link mark on GitHub text")
	}
	if href, exists := githubText.Marks[0].Attrs["href"]; !exists || href != "https://github.com" {
		t.Errorf("Expected GitHub href, got %v", href)
	}
}