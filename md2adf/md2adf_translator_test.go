package md2adf

import (
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
