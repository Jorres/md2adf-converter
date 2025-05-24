package adf

import (
	"testing"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
)

func TestAdfMarkdownParserMentions(t *testing.T) {
	parser := tree_sitter_markdown.NewAdfMarkdownParser()

	testCases := []struct {
		name          string
		input         string
		expectedCount int
	}{
		{
			name:          "Single mention",
			input:         "Hello @user@domain.com world",
			expectedCount: 1,
		},
		{
			name:          "Multiple mentions",
			input:         "Contact @alice@company.com and @bob@example.org",
			expectedCount: 2,
		},
		{
			name:          "No mentions",
			input:         "Regular text with email@domain.com (no @ prefix)",
			expectedCount: 0,
		},
		{
			name: "Complex document",
			input: `# Header

Paragraph with @user@domain.com mentioned.

## Second Header

Another paragraph with @admin@system.local here.`,
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := parser.Parse([]byte(tc.input))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			mentions := parser.FindPeopleMentions(tree, []byte(tc.input))

			if len(mentions) != tc.expectedCount {
				t.Errorf("Expected %d mentions, got %d", tc.expectedCount, len(mentions))
			}

			// Verify mention format
			for _, mention := range mentions {
				if mention.Text == "" {
					t.Error("Mention text should not be empty")
				}
				if mention.StartByte >= mention.EndByte {
					t.Error("Invalid mention byte range")
				}
			}
		})
	}
}

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
