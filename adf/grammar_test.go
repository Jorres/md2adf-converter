package adf

import (
	"testing"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
	"slices"
)

// TestCase represents a test case for people mention parsing
type TestCase struct {
	name                 string
	input                string
	expectedMentions     int
	expectedMentionTexts []string
}

func TestPeopleMentionGrammar(t *testing.T) {
	parser := tree_sitter_markdown.NewAdfMarkdownParser()

	testCases := []TestCase{
		{
			name:                 "Single mention",
			input:                "Hello @user@domain.com world",
			expectedMentions:     1,
			expectedMentionTexts: []string{"@user@domain.com"},
		},
		{
			name:                 "Multiple mentions",
			input:                "Contact @alice@company.com and @bob@example.org",
			expectedMentions:     2,
			expectedMentionTexts: []string{"@alice@company.com", "@bob@example.org"},
		},
		{
			name:                 "No mentions",
			input:                "Regular text with email@domain.com (no @ prefix)",
			expectedMentions:     0,
			expectedMentionTexts: []string{},
		},
		{
			name:                 "Edge case - double at",
			input:                "Invalid @@double@test.com should be parsed",
			expectedMentions:     1,
			expectedMentionTexts: []string{"@double@test.com"},
		},
		{
			name:                 "Underscore and dash support",
			input:                "Users @user_name@site-name.co.uk and @test-user@sub.domain.org",
			expectedMentions:     2,
			expectedMentionTexts: []string{"@user_name@site-name.co.uk", "@test-user@sub.domain.org"},
		},
		{
			name: "Multi-paragraph document",
			input: `# Header

First paragraph with @user1@domain.com mentioned.

Second paragraph with @user2@company.org here.`,
			expectedMentions:     2,
			expectedMentionTexts: []string{"@user1@domain.com", "@user2@company.org"},
		},
		{
			name: "Complex document with mixed content",
			input: `# Project Update

This paragraph has no mentions.

Contact @admin@system.local for issues.

## Follow-up

Regular email@address.com shouldn't match, but @dev@team.io should.`,
			expectedMentions:     2,
			expectedMentionTexts: []string{"@admin@system.local", "@dev@team.io"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte(tc.input)

			tree, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			mentions := parser.FindPeopleMentions(tree, content)
			actualCount := len(mentions)
			actualMentions := make([]string, len(mentions))
			for i, mention := range mentions {
				actualMentions[i] = mention.Text
			}

			if actualCount != tc.expectedMentions {
				t.Errorf("Expected %d people mentions, got %d. Actual mentions: %v", tc.expectedMentions, actualCount, actualMentions)
			}

			if len(actualMentions) != len(tc.expectedMentionTexts) {
				t.Errorf("Expected %d mention texts, got %d", len(tc.expectedMentionTexts), len(actualMentions))
			}

			// Check that all expected mentions are present
			for _, expectedMention := range tc.expectedMentionTexts {
				found := slices.Contains(actualMentions, expectedMention)
				if !found {
					t.Errorf("Expected mention %q not found in actual mentions %v", expectedMention, actualMentions)
				}
			}
		})
	}
}

func TestGrammarStructure(t *testing.T) {
	parser := tree_sitter_markdown.NewAdfMarkdownParser()
	content := []byte(`# Test Header

Paragraph with @user@domain.com mention.

## Second Header

Another paragraph here.`)

	tree, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Verify we have a valid tree structure
	if tree == nil {
		t.Error("Tree should not be nil")
	}

	root := tree.RootNode()
	if root.Kind() != "document" {
		t.Errorf("Expected document root, got %s", root.Kind())
	}

	// Verify people mention is found using the clean interface
	mentions := parser.FindPeopleMentions(tree, content)

	if len(mentions) != 1 {
		t.Errorf("Expected 1 people mention, got %d", len(mentions))
	}

	if len(mentions) > 0 && mentions[0].Text != "@user@domain.com" {
		t.Errorf("Expected @user@domain.com, got %s", mentions[0].Text)
	}
}

func TestEmptyAndEdgeCases(t *testing.T) {
	parser := tree_sitter_markdown.NewAdfMarkdownParser()

	edgeCases := []TestCase{
		{
			name:                 "Empty string",
			input:                "",
			expectedMentions:     0,
			expectedMentionTexts: []string{},
		},
		{
			name:                 "Only whitespace",
			input:                "   \n\n  \t  ",
			expectedMentions:     0,
			expectedMentionTexts: []string{},
		},
		{
			name:                 "Just header",
			input:                "# Header Only",
			expectedMentions:     0,
			expectedMentionTexts: []string{},
		},
		{
			name:                 "Mention at start",
			input:                "@user@domain.com is mentioned first",
			expectedMentions:     1,
			expectedMentionTexts: []string{"@user@domain.com"},
		},
		{
			name:                 "Mention at end",
			input:                "Contact @user@domain.com",
			expectedMentions:     1,
			expectedMentionTexts: []string{"@user@domain.com"},
		},
		{
			name:                 "Multiple mentions same line",
			input:                "@a@b.com @c@d.com @e@f.com",
			expectedMentions:     3,
			expectedMentionTexts: []string{"@a@b.com", "@c@d.com", "@e@f.com"},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte(tc.input)

			tree, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			mentions := parser.FindPeopleMentions(tree, content)
			actualCount := len(mentions)
			actualMentions := make([]string, len(mentions))
			for i, mention := range mentions {
				actualMentions[i] = mention.Text
			}

			if actualCount != tc.expectedMentions {
				t.Errorf("Expected %d people mentions, got %d. Actual mentions: %v", tc.expectedMentions, actualCount, actualMentions)
			}

			if len(actualMentions) != len(tc.expectedMentionTexts) {
				t.Errorf("Expected %d mention texts, got %d", len(tc.expectedMentionTexts), len(actualMentions))
			}

			// Check that all expected mentions are present
			for _, expectedMention := range tc.expectedMentionTexts {
				found := slices.Contains(actualMentions, expectedMention)
				if !found {
					t.Errorf("Expected mention %q not found in actual mentions %v", expectedMention, actualMentions)
				}
			}
		})
	}
}
