package adf

import "encoding/json"

// ADF document structure
type ADFDocument struct {
	Version int       `json:"version"`
	Type    string    `json:"type"`
	Content []ADFNode `json:"content"`
}

// Generic ADF node
type ADFNode struct {
	Type    string         `json:"type"`
	Content []ADFNode      `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []ADFMark      `json:"marks,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// ADF mark for formatting
type ADFMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// Create a new ADF document
func NewADFDocument() *ADFDocument {
	return &ADFDocument{
		Version: 1,
		Type:    "doc",
		Content: []ADFNode{},
	}
}

// Create a paragraph node
func NewParagraphNode() *ADFNode {
	return &ADFNode{
		Type:    "paragraph",
		Content: []ADFNode{},
	}
}

// Create a text node
func NewTextNode(text string) *ADFNode {
	return &ADFNode{
		Type: "text",
		Text: text,
	}
}

// Create a text node with marks
func NewTextNodeWithMarks(text string, marks []ADFMark) *ADFNode {
	return &ADFNode{
		Type:  "text",
		Text:  text,
		Marks: marks,
	}
}

// Create a heading node
func NewHeadingNode(level int) *ADFNode {
	return &ADFNode{
		Type: "heading",
		Attrs: map[string]any{
			"level": level,
		},
		Content: []ADFNode{},
	}
}

// Create a link mark
func NewLinkMark(href string) ADFMark {
	return ADFMark{
		Type: "link",
		Attrs: map[string]any{
			"href": href,
		},
	}
}

// Create a people mention mark (custom ADF extension)
func NewPeopleMentionMark(email string) ADFMark {
	return ADFMark{
		Type: "mention",
		Attrs: map[string]any{
			"id":   email,
			"text": email,
		},
	}
}

// Create a code mark
func NewCodeMark() ADFMark {
	return ADFMark{
		Type: "code",
	}
}

// Create a mention node
func NewMentionNode(userID, displayText string) *ADFNode {
	return &ADFNode{
		Type: "mention",
		Attrs: map[string]any{
			"id":   userID,
			"text": displayText,
		},
	}
}

// Create a code block node
func NewCodeBlockNode(language string) *ADFNode {
	attrs := make(map[string]any)
	if language != "" {
		attrs["language"] = language
	}
	
	return &ADFNode{
		Type:    "codeBlock",
		Content: []ADFNode{},
		Attrs:   attrs,
	}
}

// Convert to JSON
func (doc *ADFDocument) ToJSON() ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}
