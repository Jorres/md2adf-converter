package md2adf

import (
	"testing"
)

func TestOrderedList(t *testing.T) {
	converter := NewTranslator()
	markdown := `# Heading

1. first item
2. second item with code
3. third item`

	doc, err := converter.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	if len(doc.Content) != 2 {
		t.Fatalf("Expected 2 top-level elements, got %d", len(doc.Content))
	}

	// Check heading
	if doc.Content[0].Type != "heading" {
		t.Errorf("Expected first element to be heading, got %s", doc.Content[0].Type)
	}

	// Check ordered list
	if doc.Content[1].Type != "orderedList" {
		t.Errorf("Expected second element to be orderedList, got %s", doc.Content[1].Type)
	}

	orderedList := doc.Content[1]
	if len(orderedList.Content) != 3 {
		t.Errorf("Expected 3 list items, got %d", len(orderedList.Content))
	}

	// Check list items
	for i, item := range orderedList.Content {
		if item.Type != "listItem" {
			t.Errorf("Expected list item %d to be listItem, got %s", i, item.Type)
		}
		if len(item.Content) != 1 {
			t.Errorf("Expected list item %d to have 1 paragraph, got %d", i, len(item.Content))
		}
		if item.Content[0].Type != "paragraph" {
			t.Errorf("Expected list item %d content to be paragraph, got %s", i, item.Content[0].Type)
		}
	}
}

func TestUnorderedList(t *testing.T) {
	converter := NewTranslator()
	markdown := `# Heading

- first bullet
- second bullet  
- third bullet`

	doc, err := converter.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	if len(doc.Content) != 2 {
		t.Fatalf("Expected 2 top-level elements, got %d", len(doc.Content))
	}

	// Check bullet list
	if doc.Content[1].Type != "bulletList" {
		t.Errorf("Expected second element to be bulletList, got %s", doc.Content[1].Type)
	}

	bulletList := doc.Content[1]
	if len(bulletList.Content) != 3 {
		t.Errorf("Expected 3 list items, got %d", len(bulletList.Content))
	}

	// Check list items
	for i, item := range bulletList.Content {
		if item.Type != "listItem" {
			t.Errorf("Expected list item %d to be listItem, got %s", i, item.Type)
		}
	}
}

func TestListWithInlineCode(t *testing.T) {
	converter := NewTranslator()
	markdown := `1. first ` + "`element`" + ` with a codeblock
2. second element`

	doc, err := converter.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	if len(doc.Content) != 1 {
		t.Fatalf("Expected 1 top-level element, got %d", len(doc.Content))
	}

	orderedList := doc.Content[0]
	if orderedList.Type != "orderedList" {
		t.Errorf("Expected orderedList, got %s", orderedList.Type)
	}

	firstItem := orderedList.Content[0]
	firstParagraph := firstItem.Content[0]

	// Should have text + code + text
	if len(firstParagraph.Content) < 3 {
		t.Errorf("Expected at least 3 text nodes in first paragraph, got %d", len(firstParagraph.Content))
	}

	// Check that second text node has code mark
	codeTextNode := firstParagraph.Content[1]
	if len(codeTextNode.Marks) != 1 || codeTextNode.Marks[0].Type != "code" {
		t.Errorf("Expected second text node to have code mark, got %v", codeTextNode.Marks)
	}
	if codeTextNode.Text != "element" {
		t.Errorf("Expected code text to be 'element', got '%s'", codeTextNode.Text)
	}
}

func TestOrderedListStartingNumber(t *testing.T) {
	converter := NewTranslator()
	markdown := `5. fifth item
6. sixth item`

	doc, err := converter.TranslateToADF([]byte(markdown))
	if err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	orderedList := doc.Content[0]
	if orderedList.Type != "orderedList" {
		t.Errorf("Expected orderedList, got %s", orderedList.Type)
	}

	// Check that it has the order attribute for starting number
	if orderedList.Attrs == nil {
		t.Error("Expected orderedList to have attrs for starting number")
	} else if order, exists := orderedList.Attrs["order"]; !exists || order != 5 {
		t.Errorf("Expected order to be 5, got %v", order)
	}
}
