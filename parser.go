package main

import (
	"strings"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type MarkdownParser struct {
	parser *sitter.Parser
}

func NewMarkdownParser() *MarkdownParser {
	parser := sitter.NewParser()
	language := sitter.NewLanguage(tree_sitter_markdown.Language())
	parser.SetLanguage(language)

	return &MarkdownParser{
		parser: parser,
	}
}

func (mp *MarkdownParser) Parse(content []byte) (*sitter.Node, error) {
	tree := mp.parser.Parse(content, nil)
	return tree.RootNode(), nil
}

func (mp *MarkdownParser) ConvertToADF(content []byte) (*ADFDocument, error) {
	rootNode, err := mp.Parse(content)
	if err != nil {
		return nil, err
	}

	doc := NewADFDocument()
	mp.processNode(rootNode, content, doc)
	return doc, nil
}

func (mp *MarkdownParser) processNode(node *sitter.Node, content []byte, doc *ADFDocument) {
	nodeType := node.Kind()

	switch nodeType {
	case "document", "section":
		// Container nodes - process children
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil {
				mp.processNode(child, content, doc)
			}
		}
	case "atx_heading":
		heading := mp.convertHeading(node, content)
		if heading != nil {
			doc.Content = append(doc.Content, *heading)
		}
	case "paragraph":
		paragraph := mp.convertParagraph(node, content)
		if paragraph != nil {
			doc.Content = append(doc.Content, *paragraph)
		}
	}
}

func (mp *MarkdownParser) convertNode(node *sitter.Node, content []byte) *ADFNode {
	nodeType := node.Kind()

	switch nodeType {
	case "document":
		return mp.convertDocument(node, content)
	case "section":
		return mp.convertSection(node, content)
	case "atx_heading":
		return mp.convertHeading(node, content)
	case "paragraph":
		return mp.convertParagraph(node, content)
	case "inline":
		return mp.convertInline(node, content)
	case "link":
		return mp.convertLink(node, content)
	default:
		// For unknown node types, try to convert children
		if node.ChildCount() > 0 {
			return mp.convertChildren(node, content)
		}
		// If it's a leaf node, extract text
		text := string(content[node.StartByte():node.EndByte()])
		if strings.TrimSpace(text) != "" {
			return NewTextNode(strings.TrimSpace(text))
		}
		return nil
	}
}

func (mp *MarkdownParser) convertDocument(node *sitter.Node, content []byte) *ADFNode {
	// Document node is handled at the top level, just process children
	return nil
}

func (mp *MarkdownParser) convertSection(node *sitter.Node, content []byte) *ADFNode {
	// Section is a container, process its children directly
	return nil
}

func (mp *MarkdownParser) convertChildren(node *sitter.Node, content []byte) *ADFNode {
	// Helper to process children when we don't know the node type
	return nil
}

func (mp *MarkdownParser) convertHeading(node *sitter.Node, content []byte) *ADFNode {
	level := 1

	// Find the heading marker to determine level
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "atx_h1_marker" {
			level = 1
		} else if child.Kind() == "atx_h2_marker" {
			level = 2
		} else if child.Kind() == "atx_h3_marker" {
			level = 3
		} else if child.Kind() == "atx_h4_marker" {
			level = 4
		} else if child.Kind() == "atx_h5_marker" {
			level = 5
		} else if child.Kind() == "atx_h6_marker" {
			level = 6
		}
	}

	heading := NewHeadingNode(level)

	// Extract heading text content
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "inline" {
			textContent := mp.extractTextContent(child, content)
			if textContent != "" {
				heading.Content = append(heading.Content, *NewTextNode(textContent))
			}
		}
	}

	return heading
}

func (mp *MarkdownParser) convertParagraph(node *sitter.Node, content []byte) *ADFNode {
	paragraph := NewParagraphNode()

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "inline" {
			mp.convertInlineContent(child, content, paragraph)
		}
	}

	return paragraph
}

func (mp *MarkdownParser) convertInline(node *sitter.Node, content []byte) *ADFNode {
	// Extract text content from inline node, handling links
	text := string(content[node.StartByte():node.EndByte()])
	text = strings.TrimSpace(text)
	if text != "" {
		return NewTextNode(text)
	}
	return nil
}

func (mp *MarkdownParser) convertInlineContent(node *sitter.Node, content []byte, parent *ADFNode) {
	// For now, just extract the text content of the inline node
	text := string(content[node.StartByte():node.EndByte()])
	text = strings.TrimSpace(text)
	
	if text != "" {
		// Check if this inline content contains links
		if mp.containsLink(node, content) {
			mp.processLinksInInline(node, content, parent)
		} else {
			parent.Content = append(parent.Content, *NewTextNode(text))
		}
	}
}

func (mp *MarkdownParser) containsLink(node *sitter.Node, content []byte) bool {
	text := string(content[node.StartByte():node.EndByte()])
	return strings.Contains(text, "[") && strings.Contains(text, "](")
}

func (mp *MarkdownParser) processLinksInInline(node *sitter.Node, content []byte, parent *ADFNode) {
	text := string(content[node.StartByte():node.EndByte()])
	
	// Simple regex-like parsing for markdown links [text](url)
	// For the POC, let's handle the basic case
	linkStart := strings.Index(text, "[")
	if linkStart == -1 {
		parent.Content = append(parent.Content, *NewTextNode(text))
		return
	}
	
	// Add text before link
	if linkStart > 0 {
		beforeText := text[:linkStart]
		if strings.TrimSpace(beforeText) != "" {
			parent.Content = append(parent.Content, *NewTextNode(beforeText))
		}
	}
	
	// Find end of link text
	linkTextEnd := strings.Index(text[linkStart:], "]")
	if linkTextEnd == -1 {
		parent.Content = append(parent.Content, *NewTextNode(text[linkStart:]))
		return
	}
	linkTextEnd += linkStart
	
	// Find URL start
	urlStart := linkTextEnd + 2 // Skip "]("
	if urlStart >= len(text) || text[linkTextEnd+1] != '(' {
		parent.Content = append(parent.Content, *NewTextNode(text[linkStart:]))
		return
	}
	
	// Find URL end
	urlEnd := strings.Index(text[urlStart:], ")")
	if urlEnd == -1 {
		parent.Content = append(parent.Content, *NewTextNode(text[linkStart:]))
		return
	}
	urlEnd += urlStart
	
	// Extract link parts
	linkText := text[linkStart+1 : linkTextEnd]
	linkURL := text[urlStart:urlEnd]
	
	// Create link node
	linkMark := NewLinkMark(linkURL)
	linkNode := NewTextNodeWithMarks(linkText, []ADFMark{linkMark})
	parent.Content = append(parent.Content, *linkNode)
	
	// Add text after link
	if urlEnd+1 < len(text) {
		afterText := text[urlEnd+1:]
		if strings.TrimSpace(afterText) != "" {
			parent.Content = append(parent.Content, *NewTextNode(afterText))
		}
	}
}

func (mp *MarkdownParser) convertLink(node *sitter.Node, content []byte) *ADFNode {
	var linkText, linkURL string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "link_text" {
			linkText = mp.extractTextContent(child, content)
		} else if child.Kind() == "link_destination" {
			linkURL = string(content[child.StartByte():child.EndByte()])
		}
	}

	if linkText != "" && linkURL != "" {
		linkMark := NewLinkMark(linkURL)
		return NewTextNodeWithMarks(linkText, []ADFMark{linkMark})
	}

	return nil
}

func (mp *MarkdownParser) extractTextContent(node *sitter.Node, content []byte) string {
	if node.ChildCount() == 0 {
		return string(content[node.StartByte():node.EndByte()])
	}

	var text strings.Builder
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		text.WriteString(mp.extractTextContent(child, content))
	}

	return text.String()
}

