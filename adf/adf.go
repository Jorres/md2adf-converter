package adf

import (
	"encoding/json"
	"slices"
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
	NodeMediaGroup  = NodeType("mediaGroup")
	NodeMediaSingle = NodeType("mediaSingle")

	ChildNodeText        = NodeType("text")
	ChildNodeListItem    = NodeType("listItem")
	ChildNodeTableRow    = NodeType("tableRow")
	ChildNodeTableHeader = NodeType("tableHeader")
	ChildNodeTableCell   = NodeType("tableCell")

	InlineNodeCard      = NodeType("inlineCard")
	InlineNodeEmoji     = NodeType("emoji")
	InlineNodeMention   = NodeType("mention")
	InlineNodeHardBreak = NodeType("hardBreak")

	MarkEm        = NodeType("em")
	MarkLink      = NodeType("link")
	MarkCode      = NodeType("code")
	MarkStrike    = NodeType("strike")
	MarkStrong    = NodeType("strong")
	MarkUnderline = NodeType("underline")
)

// ADF document structure (primary interface)
type ADFDocument struct {
	Version int        `json:"version"`
	Type    string     `json:"type"`
	Content []*ADFNode `json:"content"`
}

type ADFNode struct {
	Type    NodeType       `json:"type"`
	Content []*ADFNode     `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []*ADFMark     `json:"marks,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// ADF mark for formatting (primary interface)
type ADFMark struct {
	Type  NodeType       `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// ReplaceAll replaces all occurrences of an old string
// in a text node with a new one.
func (a *ADFNode) ReplaceAll(old, new string) {
	if a == nil || len(a.Content) == 0 {
		return
	}
	for _, parent := range a.Content {
		a.replace(parent, old, new)
	}
}

func (a *ADFNode) replace(n *ADFNode, old, new string) {
	for _, child := range n.Content {
		a.replace(child, old, new)
	}
	if n.Type == ChildNodeText {
		n.Text = strings.ReplaceAll(n.Text, old, new)
	}
}

// GetType gets node type.
func (n *ADFNode) GetType() NodeType { return n.Type }

// GetAttributes gets node attributes.
func (n *ADFNode) GetAttributes() any { return n.Attrs }

// GetType gets node type.
func (n *ADFMark) GetType() NodeType { return n.Type }

// GetAttributes gets node attributes.
func (n *ADFMark) GetAttributes() any { return n.Attrs }

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
	return slices.Contains(ParentNodes(), identifier)
}

// IsChildNode checks if the node is a child node.
func IsChildNode(identifier NodeType) bool {
	return slices.Contains(ChildNodes(), identifier)
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
		Content: []*ADFNode{},
	}
}

// Create a paragraph node
func NewParagraphNode() *ADFNode {
	return &ADFNode{
		Type:    "paragraph",
		Content: []*ADFNode{},
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
func NewTextNodeWithMarks(text string, marks []*ADFMark) *ADFNode {
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
		Content: []*ADFNode{},
	}
}

// Create a link mark
func NewLinkMark(href string) *ADFMark {
	return &ADFMark{
		Type: "link",
		Attrs: map[string]any{
			"href": href,
		},
	}
}

// Create a people mention mark (custom ADF extension)
func NewPeopleMentionMark(email string) *ADFMark {
	return &ADFMark{
		Type: "mention",
		Attrs: map[string]any{
			"id":   email,
			"text": email,
		},
	}
}

// Create a code mark
func NewCodeMark() *ADFMark {
	return &ADFMark{
		Type: "code",
	}
}

// Create a strong mark
func NewStrongMark() *ADFMark {
	return &ADFMark{
		Type: "strong",
	}
}

// Create an underline mark
func NewUnderlineMark() *ADFMark {
	return &ADFMark{
		Type: "underline",
	}
}

// Create a strikethrough mark
func NewStrikethroughMark() *ADFMark {
	return &ADFMark{
		Type: "strike",
	}
}

// Create an emphasis mark (italics)
func NewEmphasisMark() *ADFMark {
	return &ADFMark{
		Type: "em",
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
		Content: []*ADFNode{},
		Attrs:   attrs,
	}
}

// Create a bullet list node
func NewBulletListNode() *ADFNode {
	return &ADFNode{
		Type:    "bulletList",
		Content: []*ADFNode{},
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
		Content: []*ADFNode{},
		Attrs:   attrs,
	}
}

// Create a list item node
func NewListItemNode() *ADFNode {
	return &ADFNode{
		Type:    "listItem",
		Content: []*ADFNode{},
	}
}

// Create a panel node
func NewPanelNode(panelType string) *ADFNode {
	return &ADFNode{
		Type: "panel",
		Attrs: map[string]any{
			"panelType": panelType,
		},
		Content: []*ADFNode{},
	}
}

// Convert to JSON
func (doc *ADFDocument) ToJSON() ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}
