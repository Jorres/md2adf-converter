package adf

import (
	"strings"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type AdfConverter struct {
	markdownParser *tree_sitter_markdown.AdfMarkdownParser
}

// NewAdfParser creates a parser using the clean bindings interface
func NewAdfParser() *AdfConverter {
	return &AdfConverter{
		markdownParser: tree_sitter_markdown.NewAdfMarkdownParser(),
	}
}

// ConvertToADF converts markdown to ADF using the clean interface
func (p *AdfConverter) ConvertToADF(content []byte) (*ADFDocument, error) {
	// Parse using the clean bindings interface
	tree, err := p.markdownParser.Parse(content)
	if err != nil {
		return nil, err
	}

	doc := NewADFDocument()
	p.processNode(tree.RootNode(), content, doc)
	return doc, nil
}

// processNode processes a tree-sitter node and converts it to ADF
func (p *AdfConverter) processNode(node *sitter.Node, content []byte, doc *ADFDocument) {
	nodeType := node.Kind()

	switch nodeType {
	case "document", "section":
		// Container nodes - process children
		p.processChildren(node, content, doc)

	case "atx_heading":
		heading := p.convertHeading(node, content)
		if heading != nil {
			doc.Content = append(doc.Content, *heading)
		}

	case "paragraph":
		paragraph := p.convertParagraph(node, content)
		if paragraph != nil {
			doc.Content = append(doc.Content, *paragraph)
		}

	case "fenced_code_block":
		codeBlock := p.convertCodeBlock(node, content)
		if codeBlock != nil {
			doc.Content = append(doc.Content, *codeBlock)
		}
	}
}

// processChildren processes all children of a node
func (p *AdfConverter) processChildren(node *sitter.Node, content []byte, doc *ADFDocument) {
	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		if child != nil {
			p.processNode(child, content, doc)
		}
	}
}

// convertHeading converts a heading node to ADF
func (p *AdfConverter) convertHeading(node *sitter.Node, content []byte) *ADFNode {
	level := 1
	var inlineNode *sitter.Node

	// Find the heading level and inline content
	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		switch child.Kind() {
		case "atx_h1_marker":
			level = 1
		case "atx_h2_marker":
			level = 2
		case "atx_h3_marker":
			level = 3
		case "atx_h4_marker":
			level = 4
		case "atx_h5_marker":
			level = 5
		case "atx_h6_marker":
			level = 6
		case "inline":
			inlineNode = child
		}
	}

	heading := NewHeadingNode(level)
	if inlineNode != nil {
		p.processInlineContent(inlineNode, content, heading)
	}

	return heading
}

// convertParagraph converts a paragraph node to ADF
func (p *AdfConverter) convertParagraph(node *sitter.Node, content []byte) *ADFNode {
	paragraph := NewParagraphNode()

	// Find inline content
	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		if child.Kind() == "inline" {
			p.processInlineContent(child, content, paragraph)
		}
	}

	return paragraph
}

// convertCodeBlock converts a fenced code block to ADF
func (p *AdfConverter) convertCodeBlock(node *sitter.Node, content []byte) *ADFNode {
	var language string
	var codeContent string

	// Process children to find language and code content
	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		switch child.Kind() {
		case "info_string":
			// Extract language from info string
			languageText := string(content[child.StartByte():child.EndByte()])
			language = strings.TrimSpace(languageText)
		case "code_fence_content":
			// Extract code content
			rawContent := string(content[child.StartByte():child.EndByte()])
			// Remove any trailing closing fence (``` at the end)
			if strings.HasSuffix(rawContent, "\n```") {
				codeContent = strings.TrimSuffix(rawContent, "\n```")
			} else if strings.HasSuffix(rawContent, "```") {
				codeContent = strings.TrimSuffix(rawContent, "```")
			} else {
				codeContent = rawContent
			}
		}
	}

	// If we didn't find code_fence_content, try to extract manually
	if codeContent == "" {
		fullText := string(content[node.StartByte():node.EndByte()])
		
		// Simple approach: remove opening and closing fence markers
		// Look for first newline after opening ```
		firstNewline := strings.Index(fullText, "\n")
		if firstNewline == -1 {
			return NewCodeBlockNode(language)
		}
		
		// Find the last occurrence of ``` 
		lastFence := strings.LastIndex(fullText, "```")
		if lastFence <= firstNewline {
			return NewCodeBlockNode(language)
		}
		
		// Extract content between first newline and last fence
		startPos := firstNewline + 1
		endPos := lastFence
		
		// If there's a newline right before the closing fence, exclude it
		if endPos > 0 && fullText[endPos-1] == '\n' {
			endPos--
		}
		
		if startPos < endPos {
			codeContent = fullText[startPos:endPos]
		}
	}

	codeBlock := NewCodeBlockNode(language)
	if codeContent != "" {
		codeBlock.Content = append(codeBlock.Content, *NewTextNode(codeContent))
	}

	return codeBlock
}

// processInlineContent processes inline content using the clean interface
func (p *AdfConverter) processInlineContent(inlineNode *sitter.Node, content []byte, parent *ADFNode) {
	// Get the inline tree for this node using the clean bindings interface
	inlineTree := p.markdownParser.GetInlineTree(inlineNode, content)
	if inlineTree == nil {
		// No inline tree, treat as plain text
		text := string(content[inlineNode.StartByte():inlineNode.EndByte()])
		if strings.TrimSpace(text) != "" {
			parent.Content = append(parent.Content, *NewTextNode(text))
		}
		return
	}

	// Extract the inline content for correct byte offset calculations
	inlineContent := content[inlineNode.StartByte():inlineNode.EndByte()]

	// Process the inline tree with gap filling
	p.processInlineTreeWithGaps(inlineTree.RootNode(), inlineContent, parent)
}

// processInlineTreeWithGaps processes inline tree nodes and fills text gaps
func (p *AdfConverter) processInlineTreeWithGaps(inlineRoot *sitter.Node, inlineContent []byte, parent *ADFNode) {
	// Track position for gap filling
	currentPos := uint(0)

	// Process all direct children of the inline root
	childCount := int(inlineRoot.ChildCount())
	for i := range childCount {
		child := inlineRoot.Child(uint(i))

		// Add gap before this node
		if child.StartByte() > currentPos {
			gapText := string(inlineContent[currentPos:child.StartByte()])
			if strings.TrimSpace(gapText) != "" {
				parent.Content = append(parent.Content, *NewTextNode(gapText))
			}
		}

		// Process this node
		switch child.Kind() {
		case "people_mention":
			text := string(inlineContent[child.StartByte():child.EndByte()])
			email := strings.TrimSpace(text)

			mentionMark := NewPeopleMentionMark(email)
			textNode := NewTextNodeWithMarks(email, []ADFMark{mentionMark})
			parent.Content = append(parent.Content, *textNode)

		case "code_span":
			p.processCodeSpan(child, inlineContent, parent)

		case "text":
			text := string(inlineContent[child.StartByte():child.EndByte()])
			if strings.TrimSpace(text) != "" {
				parent.Content = append(parent.Content, *NewTextNode(text))
			}

		default:
			// For other elements (punctuation, etc.), include as plain text
			text := string(inlineContent[child.StartByte():child.EndByte()])
			if strings.TrimSpace(text) != "" {
				parent.Content = append(parent.Content, *NewTextNode(text))
			}
		}

		currentPos = child.EndByte()
	}

	// Add any remaining text after the last node
	if currentPos < uint(len(inlineContent)) {
		remainingText := string(inlineContent[currentPos:])
		if strings.TrimSpace(remainingText) != "" {
			parent.Content = append(parent.Content, *NewTextNode(remainingText))
		}
	}
}

// processCodeSpan processes a code span node (inline code)
func (p *AdfConverter) processCodeSpan(codeNode *sitter.Node, inlineContent []byte, parent *ADFNode) {
	// Find the actual code content within the code span
	// Code spans have structure: code_span -> code_span_delimiter + text + code_span_delimiter
	var codeText string
	
	childCount := int(codeNode.ChildCount())
	for i := range childCount {
		child := codeNode.Child(uint(i))
		if child.Kind() == "text" {
			codeText = string(inlineContent[child.StartByte():child.EndByte()])
			break
		}
	}
	
	// If we didn't find a text child, extract the whole content and strip backticks
	if codeText == "" {
		fullText := string(inlineContent[codeNode.StartByte():codeNode.EndByte()])
		// Remove surrounding backticks
		codeText = strings.Trim(fullText, "`")
	}
	
	if codeText != "" {
		codeMark := NewCodeMark()
		textNode := NewTextNodeWithMarks(codeText, []ADFMark{codeMark})
		parent.Content = append(parent.Content, *textNode)
	}
}
