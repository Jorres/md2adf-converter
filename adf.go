package main

import "encoding/json"

// ADF document structure
type ADFDocument struct {
	Version int       `json:"version"`
	Type    string    `json:"type"`
	Content []ADFNode `json:"content"`
}

// Generic ADF node
type ADFNode struct {
	Type    string                 `json:"type"`
	Content []ADFNode              `json:"content,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Marks   []ADFMark              `json:"marks,omitempty"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
}

// ADF mark for formatting
type ADFMark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
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
		Attrs: map[string]interface{}{
			"level": level,
		},
		Content: []ADFNode{},
	}
}

// Create a link mark
func NewLinkMark(href string) ADFMark {
	return ADFMark{
		Type: "link",
		Attrs: map[string]interface{}{
			"href": href,
		},
	}
}

// Convert to JSON
func (doc *ADFDocument) ToJSON() ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}