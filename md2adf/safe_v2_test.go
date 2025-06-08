package md2adf

import (
	"strings"
	"testing"
)

func TestCheckSafeForV2(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name                string
		markdown            string
		expectError         bool
		expectedUnsafeTypes []string
	}{
		{
			name:        "safe markdown - basic text",
			markdown:    "# Header\n\nThis is a paragraph with **bold** and _italic_ text.",
			expectError: false,
		},
		{
			name:        "safe markdown - code blocks and lists",
			markdown:    "```go\nfunc main() {\n  fmt.Println(\"Hello\")\n}\n```\n\n- Item 1\n- Item 2",
			expectError: false,
		},
		{
			name:                "unsafe markdown - underline",
			markdown:            "This has <u>underlined</u> text.",
			expectError:         true,
			expectedUnsafeTypes: []string{"underline"},
		},
		{
			name:                "unsafe markdown - mention",
			markdown:            "Hello @user@example.com",
			expectError:         true,
			expectedUnsafeTypes: []string{"mention"},
		},
		{
			name:                "unsafe markdown - panel",
			markdown:            "{panel}\nThis is an info panel\n\n{/panel}",
			expectError:         true,
			expectedUnsafeTypes: []string{"panel"},
		},
		{
			name:                "unsafe markdown - multiple unsafe types",
			markdown:            "{panel:type=warning}\nThis panel mentions @user@example.com with <u>underlined</u> text\n\n{/panel}",
			expectError:         true,
			expectedUnsafeTypes: []string{"panel", "mention", "underline"},
		},
		{
			name:        "safe markdown - table",
			markdown:    "| Header 1 | Header 2 |\n| -------- | -------- |\n| Cell 1   | Cell 2   |",
			expectError: false,
		},
		{
			name:        "safe markdown - links",
			markdown:    "Check out [this link](https://example.com)",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := translator.CheckSafeForV2(tt.markdown)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				// Check that the error message contains the expected unsafe types
				errorMsg := err.Error()
				for _, expectedType := range tt.expectedUnsafeTypes {
					if !strings.Contains(errorMsg, expectedType) {
						t.Errorf("Expected error message to contain '%s', but got: %s", expectedType, errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
