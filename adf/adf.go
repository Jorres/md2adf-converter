package adf

import (
	"encoding/json"
	"strings"
)

// NodeType is an Atlassian document node type.
type NodeType string

// Node types.
const (
	NodeTypeParent  = NodeType("parent")
	NodeTypeChild   = NodeType("child")
	NodeTypeUnknown = NodeType("unknown")

	NodeBlockquote  = NodeType("blockquote")
	NodeBulletList  = NodeType("bulletList")
	NodeCodeBlock   = NodeType("codeBlock")
	NodeHeading     = NodeType("heading")
	NodeOrderedList = NodeType("orderedList")
	NodePanel       = NodeType("panel")
	NodeParagraph   = NodeType("paragraph")
	NodeTable       = NodeType("table")
	NodeMedia       = NodeType("media")

	ChildNodeText        = NodeType("text")
	ChildNodeListItem    = NodeType("listItem")
	ChildNodeTableRow    = NodeType("tableRow")
	ChildNodeTableHeader = NodeType("tableHeader")
	ChildNodeTableCell   = NodeType("tableCell")

	InlineNodeCard      = NodeType("inlineCard")
	InlineNodeEmoji     = NodeType("emoji")
	InlineNodeMention   = NodeType("mention")
	InlineNodeHardBreak = NodeType("hardBreak")

	MarkEm     = NodeType("em")
	MarkLink   = NodeType("link")
	MarkCode   = NodeType("code")
	MarkStrike = NodeType("strike")
	MarkStrong = NodeType("strong")
)

// ADF document structure (primary interface)
type ADFDocument struct {
	Version int       `json:"version"`
	Type    string    `json:"type"`
	Content []ADFNode `json:"content"`
}

// ADF is an Atlassian document format object (legacy interface).
type ADF struct {
	Version int     `json:"version"`
	DocType string  `json:"type"`
	Content []*Node `json:"content"`
}

// Generic ADF node (primary interface)
type ADFNode struct {
	Type    string         `json:"type"`
	Content []ADFNode      `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []ADFMark      `json:"marks,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// Node is an ADF content node (legacy interface).
type Node struct {
	NodeType   NodeType    `json:"type"`
	Content    []*Node     `json:"content,omitempty"`
	Attributes interface{} `json:"attrs,omitempty"`
	NodeValue
}

// ADF mark for formatting (primary interface)
type ADFMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// MarkNode is a mark node type (legacy interface).
type MarkNode struct {
	MarkType   NodeType    `json:"type,omitempty"`
	Attributes interface{} `json:"attrs,omitempty"`
}

// NodeValue is an actual ADF node content.
type NodeValue struct {
	Text  string     `json:"text,omitempty"`
	Marks []MarkNode `json:"marks,omitempty"`
}

// ReplaceAll replaces all occurrences of an old string
// in a text node with a new one.
func (a *ADF) ReplaceAll(old, new string) {
	if a == nil || len(a.Content) == 0 {
		return
	}
	for _, parent := range a.Content {
		a.replace(parent, old, new)
	}
}

func (a *ADF) replace(n *Node, old, new string) {
	for _, child := range n.Content {
		a.replace(child, old, new)
	}
	if n.NodeType == ChildNodeText {
		n.Text = strings.ReplaceAll(n.Text, old, new)
	}
}

// GetType gets node type.
func (n Node) GetType() NodeType { return n.NodeType }

// GetAttributes gets node attributes.
func (n Node) GetAttributes() interface{} { return n.Attributes }

// GetType gets node type.
func (n MarkNode) GetType() NodeType { return n.MarkType }

// GetAttributes gets node attributes.
func (n MarkNode) GetAttributes() interface{} { return n.Attributes }

// ParentNodes returns supported ADF parent nodes.
func ParentNodes() []NodeType {
	return []NodeType{
		NodeBlockquote,
		NodeBulletList,
		NodeCodeBlock,
		NodeHeading,
		NodeOrderedList,
		NodePanel,
		NodeParagraph,
		NodeTable,
		NodeMedia,
	}
}

// ChildNodes returns supported ADF child nodes.
func ChildNodes() []NodeType {
	return []NodeType{
		ChildNodeText,
		ChildNodeListItem,
		ChildNodeTableRow,
		ChildNodeTableHeader,
		ChildNodeTableCell,
	}
}

// IsParentNode checks if the node is a parent node.
func IsParentNode(identifier NodeType) bool {
	for _, n := range ParentNodes() {
		if n == identifier {
			return true
		}
	}
	return false
}

// IsChildNode checks if the node is a child node.
func IsChildNode(identifier NodeType) bool {
	for _, n := range ChildNodes() {
		if n == identifier {
			return true
		}
	}
	return false
}

// GetADFNodeType returns the type of ADF node.
func GetADFNodeType(identifier NodeType) NodeType {
	if IsParentNode(identifier) {
		return NodeTypeParent
	}
	if IsChildNode(identifier) {
		return NodeTypeChild
	}
	return NodeTypeUnknown
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

// Create a bullet list node
func NewBulletListNode() *ADFNode {
	return &ADFNode{
		Type:    "bulletList",
		Content: []ADFNode{},
	}
}

// Create an ordered list node
func NewOrderedListNode(order int) *ADFNode {
	attrs := make(map[string]any)
	if order > 1 {
		attrs["order"] = order
	}

	return &ADFNode{
		Type:    "orderedList",
		Content: []ADFNode{},
		Attrs:   attrs,
	}
}

// Create a list item node
func NewListItemNode() *ADFNode {
	return &ADFNode{
		Type:    "listItem",
		Content: []ADFNode{},
	}
}

// Convert to JSON
func (doc *ADFDocument) ToJSON() ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}
