package md2adf

import (
	"fmt"
	"github.com/jorres/md2adf-translator/adf"
	"github.com/jorres/md2adf-translator/adf2md"
	"strings"

	tree_sitter_markdown "github.com/jorres/tree-sitter-jira-markdown/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Translator struct {
	markdownParser *tree_sitter_markdown.AdfMarkdownParser

	userMapping       map[string]string // email -> user ID
	reverseTranslator *adf2md.Translator
}

type TranslatorOption func(*Translator)

// WithUserEmailMapping sets a user email mapping to render emails to user IDs
func WithUserEmailMapping(mapping map[string]string) TranslatorOption {
	return func(tr *Translator) {
		tr.userMapping = mapping
	}
}

func WithAdf2MdTranslator(translator *adf2md.Translator) TranslatorOption {
	return func(tr *Translator) {
		tr.reverseTranslator = translator
	}
}

func NewTranslator(opts ...TranslatorOption) *Translator {
	tr := &Translator{
		markdownParser: tree_sitter_markdown.NewAdfMarkdownParser(),
	}

	for _, opt := range opts {
		opt(tr)
	}

	// if no one supplied a translator with info about links and attachments,
	// assume we do just one-off parsing and default to empty knowledge about the
	// document
	if tr.reverseTranslator == nil {
		tr.reverseTranslator = adf2md.NewTranslator(adf2md.NewJiraMarkdownTranslator())
	}

	return tr
}

func (p *Translator) TranslateToADF(content []byte) (*adf.ADFDocument, error) {
	tree, err := p.markdownParser.Parse(content)
	if err != nil {
		return nil, err
	}

	doc := adf.NewADFDocument()
	p.processNode(tree.RootNode(), content, doc)
	return doc, nil
}

// CheckSafeForV2 parses the markdown content into an ADF tree and checks if it contains
// any node types that are not safe for V2 processing. Returns an error if unsafe nodes are found.
func (p *Translator) CheckSafeForV2(body string) error {
	doc, err := p.TranslateToADF([]byte(body))
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Define the unsafe node types
	unsafeTypes := map[adf.NodeType]bool{
		adf.NodePanel:           true,
		adf.NodeMedia:           true,
		adf.NodeMediaGroup:      true,
		adf.NodeMediaSingle:     true,
		adf.InlineNodeCard:      true,
		adf.InlineNodeEmoji:     true,
		adf.InlineNodeMention:   true,
		adf.InlineNodeHardBreak: true,
		adf.MarkUnderline:       true,
	}

	// Traverse the ADF tree and collect unsafe node types
	var foundUnsafeTypes []adf.NodeType
	p.traverseADFTree(doc, unsafeTypes, &foundUnsafeTypes)

	if len(foundUnsafeTypes) > 0 {
		return fmt.Errorf("unsafe node types found: %v", foundUnsafeTypes)
	}

	return nil
}

// traverseADFTree recursively traverses the ADF tree and collects unsafe node types
func (p *Translator) traverseADFTree(doc *adf.ADFDocument, unsafeTypes map[adf.NodeType]bool, foundUnsafeTypes *[]adf.NodeType) {
	for _, node := range doc.Content {
		p.traverseADFNode(node, unsafeTypes, foundUnsafeTypes)
	}
}

// traverseADFNode recursively traverses an ADF node and its children
func (p *Translator) traverseADFNode(node *adf.ADFNode, unsafeTypes map[adf.NodeType]bool, foundUnsafeTypes *[]adf.NodeType) {
	// Check if this node type is unsafe
	if unsafeTypes[node.Type] {
		// Add to found list if not already present
		alreadyFound := false
		for _, existingType := range *foundUnsafeTypes {
			if existingType == node.Type {
				alreadyFound = true
				break
			}
		}
		if !alreadyFound {
			*foundUnsafeTypes = append(*foundUnsafeTypes, node.Type)
		}
	}

	// Check marks for unsafe types (like underline)
	for _, mark := range node.Marks {
		if unsafeTypes[mark.Type] {
			// Add to found list if not already present
			alreadyFound := false
			for _, existingType := range *foundUnsafeTypes {
				if existingType == mark.Type {
					alreadyFound = true
					break
				}
			}
			if !alreadyFound {
				*foundUnsafeTypes = append(*foundUnsafeTypes, mark.Type)
			}
		}
	}

	// Recursively traverse child nodes
	for _, child := range node.Content {
		p.traverseADFNode(child, unsafeTypes, foundUnsafeTypes)
	}
}

// processNode processes a tree-sitter node and converts it to ADF
func (p *Translator) processNode(node *sitter.Node, content []byte, doc *adf.ADFDocument) {
	nodeType := node.Kind()

	switch nodeType {
	case "document", "section":
		// Container nodes - process children
		p.processChildren(node, content, doc)

	case "atx_heading":
		heading := p.convertHeading(node, content)
		if heading != nil {
			doc.Content = append(doc.Content, heading)
		}

	case "attachment":
		for i := range int(node.ChildCount()) {
			child := node.Child(uint(i))
			if child.Kind() == "attachment_path" {
				attachmentMap := p.reverseTranslator.GetMediaMapping()
				attachmentId := string(content[child.StartByte():child.EndByte()])
				if mediaNode, exists := attachmentMap[attachmentId]; exists {
					doc.Content = append(doc.Content, mediaNode)
				}
			}
		}

	case "paragraph":
		paragraph := p.convertParagraph(node, content)
		if paragraph != nil {
			doc.Content = append(doc.Content, paragraph)
		}

	case "fenced_code_block":
		codeBlock := p.convertCodeBlock(node, content)
		if codeBlock != nil {
			doc.Content = append(doc.Content, codeBlock)
		}

	case "list":
		list := p.convertList(node, content)
		if list != nil {
			doc.Content = append(doc.Content, list)
		}

	case "panel":
		panel := p.convertPanel(node, content)
		if panel != nil {
			doc.Content = append(doc.Content, panel)
		}

	case "pipe_table":
		table := p.convertPipeTable(node, content)
		if table != nil {
			doc.Content = append(doc.Content, table)
		}
	}
}

// processChildren processes all children of a node
func (p *Translator) processChildren(node *sitter.Node, content []byte, doc *adf.ADFDocument) {
	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		if child != nil {
			p.processNode(child, content, doc)
		}
	}
}

// convertHeading converts a heading node to ADF
func (p *Translator) convertHeading(node *sitter.Node, content []byte) *adf.ADFNode {
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

	heading := adf.NewHeadingNode(level)
	if inlineNode != nil {
		p.processInlineContent(inlineNode, content, heading)
	}

	return heading
}

// convertParagraph converts a paragraph node to ADF
func (p *Translator) convertParagraph(node *sitter.Node, content []byte) *adf.ADFNode {
	paragraph := adf.NewParagraphNode()

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
func (p *Translator) convertCodeBlock(node *sitter.Node, content []byte) *adf.ADFNode {
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

	codeBlock := adf.NewCodeBlockNode(language)
	if codeContent != "" {
		codeBlock.Content = append(codeBlock.Content, adf.NewTextNode(codeContent))
	}

	return codeBlock
}

func (p *Translator) processInlineContent(inlineNode *sitter.Node, content []byte, parent *adf.ADFNode) {
	inlineTree := p.markdownParser.GetInlineTree(inlineNode, content)
	if inlineTree == nil {
		// No inline tree, treat as plain text
		text := string(content[inlineNode.StartByte():inlineNode.EndByte()])
		if strings.TrimSpace(text) != "" {
			parent.Content = append(parent.Content, adf.NewTextNode(text))
		}
		return
	}

	// Extract the inline content for correct byte offset calculations
	inlineContent := content[inlineNode.StartByte():inlineNode.EndByte()]

	// Process the inline tree with gap filling
	p.processInlineTreeWithGaps(inlineTree.RootNode(), inlineContent, parent)
}

// processInlineTreeWithGaps processes inline tree nodes and fills text gaps
func (p *Translator) processInlineTreeWithGaps(inlineRoot *sitter.Node, inlineContent []byte, parent *adf.ADFNode) {
	// Track position for gap filling
	currentPos := uint(0)

	// Process all direct children of the inline root
	childCount := int(inlineRoot.ChildCount())
	for i := range childCount {
		child := inlineRoot.Child(uint(i))

		// Add gap before this node
		if child.StartByte() > currentPos {
			gapText := string(inlineContent[currentPos:child.StartByte()])
			parent.Content = append(parent.Content, adf.NewTextNode(gapText))
		}

		// Process this node
		switch child.Kind() {
		case "people_mention":
			text := string(inlineContent[child.StartByte():child.EndByte()])
			email := strings.TrimSpace(text)

			// Look up user ID from mapping
			userID := email // fallback to email if not found
			if id, exists := p.userMapping[email]; exists {
				userID = id
			}

			// Strip company domain from display text and the @ prefix
			displayText := email
			if strings.HasPrefix(displayText, "@") {
				displayText = displayText[1:] // Remove @ prefix
			}
			if atIndex := strings.Index(displayText, "@"); atIndex != -1 {
				displayText = displayText[:atIndex] // Remove domain part
			}

			mentionNode := adf.NewMentionNode(userID, displayText)
			parent.Content = append(parent.Content, mentionNode)

		case "code_span":
			p.processCodeSpan(child, inlineContent, parent)

		case "inline_link":
			p.processLink(child, inlineContent, parent)

		case "strong_emphasis":
			p.processTextWithMarks(child, inlineContent, parent)

		case "underline":
			p.processTextWithMarks(child, inlineContent, parent)

		case "strikethrough":
			p.processTextWithMarks(child, inlineContent, parent)

		case "emphasis":
			p.processTextWithMarks(child, inlineContent, parent)

		case "text":
			text := string(inlineContent[child.StartByte():child.EndByte()])
			if strings.TrimSpace(text) != "" {
				parent.Content = append(parent.Content, adf.NewTextNode(text))
			}

		default:
			// For other elements (punctuation, etc.), include as plain text
			text := string(inlineContent[child.StartByte():child.EndByte()])
			if strings.TrimSpace(text) != "" {
				parent.Content = append(parent.Content, adf.NewTextNode(text))
			}
		}

		currentPos = child.EndByte()
	}

	// Add any remaining text after the last node
	if currentPos < uint(len(inlineContent)) {
		remainingText := string(inlineContent[currentPos:])
		if strings.TrimSpace(remainingText) != "" {
			parent.Content = append(parent.Content, adf.NewTextNode(remainingText))
		}
	}
}

// processCodeSpan processes a code span node (inline code)
func (p *Translator) processCodeSpan(codeNode *sitter.Node, inlineContent []byte, parent *adf.ADFNode) {
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
		codeMark := adf.NewCodeMark()
		textNode := adf.NewTextNodeWithMarks(codeText, []*adf.ADFMark{codeMark})
		parent.Content = append(parent.Content, textNode)
	}
}

// processLink processes an inline_link node to create ADF link marks
func (p *Translator) processLink(linkNode *sitter.Node, inlineContent []byte, parent *adf.ADFNode) {
	var linkText string
	var linkURL string

	// Process children to find link text and URL
	childCount := int(linkNode.ChildCount())
	for i := range childCount {
		child := linkNode.Child(uint(i))
		switch child.Kind() {
		case "link_text":
			// Extract the text content from inside the brackets
			linkText = string(inlineContent[child.StartByte():child.EndByte()])
			// Remove the surrounding brackets
			if strings.HasPrefix(linkText, "[") && strings.HasSuffix(linkText, "]") {
				linkText = linkText[1 : len(linkText)-1]
			}
		case "link_destination":
			// Extract the URL from inside the parentheses
			linkURL = string(inlineContent[child.StartByte():child.EndByte()])
			// Remove the surrounding parentheses
			if strings.HasPrefix(linkURL, "(") && strings.HasSuffix(linkURL, ")") {
				linkURL = linkURL[1 : len(linkURL)-1]
			}
		}
	}

	if inlineCardNode, exists := p.reverseTranslator.GetInlineCardMapping()[linkURL]; exists {
		parent.Content = append(parent.Content, inlineCardNode)
		return
	}

	if linkText != "" && linkURL != "" {
		linkMark := adf.NewLinkMark(linkURL)
		textNode := adf.NewTextNodeWithMarks(linkText, []*adf.ADFMark{linkMark})
		parent.Content = append(parent.Content, textNode)
	}
}

// convertList converts a list node to ADF
func (p *Translator) convertList(node *sitter.Node, content []byte) *adf.ADFNode {
	// Determine if this is an ordered or unordered list by checking the first list item's marker
	var isOrdered bool
	var startingOrder int = 1

	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		if child.Kind() == "list_item" {
			// Check the list marker in the first list item
			markerType := p.getListItemMarkerType(child, content)
			if markerType == "ordered" {
				isOrdered = true
				startingOrder = p.extractOrderFromListItem(child, content)
				break
			} else if markerType == "unordered" {
				isOrdered = false
				break
			}
		}
	}

	// Create the appropriate list node
	var listNode *adf.ADFNode
	if isOrdered {
		listNode = adf.NewOrderedListNode(startingOrder)
	} else {
		listNode = adf.NewBulletListNode()
	}

	// Convert all list items
	for i := range childCount {
		child := node.Child(uint(i))
		if child.Kind() == "list_item" {
			listItem := p.convertListItem(child, content)
			if listItem != nil {
				listNode.Content = append(listNode.Content, listItem)
			}
		}
	}

	return listNode
}

// convertListItem converts a list_item node to ADF
func (p *Translator) convertListItem(node *sitter.Node, content []byte) *adf.ADFNode {
	listItem := adf.NewListItemNode()

	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		switch child.Kind() {
		case "paragraph":
			// Convert the paragraph content of the list item
			paragraph := p.convertParagraph(child, content)
			if paragraph != nil {
				listItem.Content = append(listItem.Content, paragraph)
			}
		case "list":
			// Handle nested lists
			nestedList := p.convertList(child, content)
			if nestedList != nil {
				listItem.Content = append(listItem.Content, nestedList)
			}
		}
		// Ignore list markers and other elements
	}

	return listItem
}

// getListItemMarkerType determines if a list item has an ordered or unordered marker
func (p *Translator) getListItemMarkerType(listItemNode *sitter.Node, content []byte) string {
	childCount := int(listItemNode.ChildCount())
	for i := range childCount {
		child := listItemNode.Child(uint(i))
		switch child.Kind() {
		case "list_marker_dot":
			return "ordered"
		case "list_marker_minus", "list_marker_plus", "list_marker_star":
			return "unordered"
		}
	}
	return "unknown"
}

// extractOrderFromListItem extracts the starting number from an ordered list item
func (p *Translator) extractOrderFromListItem(listItemNode *sitter.Node, content []byte) int {
	childCount := int(listItemNode.ChildCount())
	for i := range childCount {
		child := listItemNode.Child(uint(i))
		if child.Kind() == "list_marker_dot" {
			markerText := string(content[child.StartByte():child.EndByte()])
			// Extract number from marker like "1. " or "42. "
			numberStr := strings.TrimSuffix(strings.TrimSpace(markerText), ".")
			var num int
			if n, err := fmt.Sscanf(numberStr, "%d", &num); n == 1 && err == nil {
				return num
			}
		}
	}
	return 1 // Default to 1 if we can't parse
}

// processTextWithMarks processes nodes with text formatting marks (strong, underline, strikethrough, emphasis)
func (p *Translator) processTextWithMarks(node *sitter.Node, inlineContent []byte, parent *adf.ADFNode) {
	text, marks := p.extractTextContentWithMarks(node, inlineContent)

	if strings.TrimSpace(text) != "" {
		textNode := adf.NewTextNodeWithMarks(text, marks)
		parent.Content = append(parent.Content, textNode)
	}
}

// extractTextContentWithMarks recursively extracts text content and collects marks
func (p *Translator) extractTextContentWithMarks(node *sitter.Node, inlineContent []byte) (string, []*adf.ADFMark) {
	nodeType := node.Kind()
	marks := []*adf.ADFMark{}

	// Add mark based on node type
	switch nodeType {
	case "strong_emphasis":
		marks = append(marks, adf.NewStrongMark())
	case "underline":
		marks = append(marks, adf.NewUnderlineMark())
	case "strikethrough":
		marks = append(marks, adf.NewStrikethroughMark())
	case "emphasis":
		marks = append(marks, adf.NewEmphasisMark())
	}

	childCount := int(node.ChildCount())

	// Handle different formatting node types
	switch nodeType {
	case "strong_emphasis":
		// Find first and last delimiter positions for **text**
		var firstDelimiterEnd, lastDelimiterStart uint
		delimiterCount := 0

		for i := range childCount {
			child := node.Child(uint(i))
			if child.Kind() == "emphasis_delimiter" {
				delimiterCount++
				if delimiterCount == 2 { // After second delimiter (opening pair)
					firstDelimiterEnd = child.EndByte()
				}
				if delimiterCount == 3 { // Third delimiter (start of closing pair)
					lastDelimiterStart = child.StartByte()
				}
			}
		}

		// Extract text between the delimiters or process nested formatting
		if delimiterCount >= 4 && lastDelimiterStart > firstDelimiterEnd {
			// Check for nested formatting within this content first
			for i := range childCount {
				child := node.Child(uint(i))
				childType := child.Kind()

				if childType == "underline" || childType == "strikethrough" || childType == "emphasis" {
					nestedText, nestedMarks := p.extractTextContentWithMarks(child, inlineContent)
					// Combine marks: current marks + nested marks
					allMarks := append(marks, nestedMarks...)
					return nestedText, allMarks
				}
			}

			// No nested formatting, return text between delimiters
			return string(inlineContent[firstDelimiterEnd:lastDelimiterStart]), marks
		}

	case "strikethrough", "emphasis":
		// Find first and last delimiter positions for ~text~ or _text_
		var firstDelimiterEnd, lastDelimiterStart uint
		delimiterCount := 0

		for i := range childCount {
			child := node.Child(uint(i))
			if child.Kind() == "emphasis_delimiter" {
				delimiterCount++
				if delimiterCount == 1 { // After first delimiter
					firstDelimiterEnd = child.EndByte()
				}
				if delimiterCount == 2 { // Second delimiter
					lastDelimiterStart = child.StartByte()
				}
			}
		}

		// Extract text between the delimiters or process nested formatting
		if delimiterCount >= 2 && lastDelimiterStart > firstDelimiterEnd {
			// Check for nested formatting within this content first
			for i := range childCount {
				child := node.Child(uint(i))
				childType := child.Kind()

				if childType == "strong_emphasis" || childType == "underline" || childType == "emphasis" || childType == "strikethrough" {
					// Skip self-reference to avoid infinite recursion
					if childType != nodeType {
						nestedText, nestedMarks := p.extractTextContentWithMarks(child, inlineContent)
						// Combine marks: current marks + nested marks
						allMarks := append(marks, nestedMarks...)
						return nestedText, allMarks
					}
				}
			}

			// No nested formatting, return text between delimiters
			return string(inlineContent[firstDelimiterEnd:lastDelimiterStart]), marks
		}

	case "underline":
		// For underline, look for underline_content directly
		for i := range childCount {
			child := node.Child(uint(i))
			if child.Kind() == "underline_content" {
				return string(inlineContent[child.StartByte():child.EndByte()]), marks
			}
		}
	}

	// Look for text content in children (fallback for other node types)
	var textContent strings.Builder
	for i := range childCount {
		child := node.Child(uint(i))
		childType := child.Kind()

		switch childType {
		case "underline_content":
			// Direct text content from underline
			text := string(inlineContent[child.StartByte():child.EndByte()])
			textContent.WriteString(text)

		case "strong_emphasis", "underline", "strikethrough", "emphasis":
			// Nested formatting - recurse
			nestedText, nestedMarks := p.extractTextContentWithMarks(child, inlineContent)
			marks = append(marks, nestedMarks...)
			textContent.WriteString(nestedText)

		case "emphasis_delimiter", "underline_open", "underline_close":
			// Skip delimiters and markup
			continue

		default:
			// For text content that's not a delimiter, include it
			if !strings.Contains(childType, "delimiter") &&
				!strings.Contains(childType, "_open") &&
				!strings.Contains(childType, "_close") {
				text := string(inlineContent[child.StartByte():child.EndByte()])
				textContent.WriteString(text)
			}
		}
	}

	return textContent.String(), marks
}

// convertPanel converts a panel node to ADF
func (p *Translator) convertPanel(node *sitter.Node, content []byte) *adf.ADFNode {
	var panelType string = "info" // default panel type

	// Create the panel node
	panel := adf.NewPanelNode(panelType)

	// Process children to find panel_start and content
	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		switch child.Kind() {
		case "panel_start":
			// Extract panel type from panel_start
			panelType = p.extractPanelType(child, content)
			// Update the panel type attribute
			panel.Attrs["panelType"] = panelType
		case "section":
			// This is a content section within the panel
			tempDoc := adf.NewADFDocument()
			p.processChildren(child, content, tempDoc)
			panel.Content = append(panel.Content, tempDoc.Content...)
		case "paragraph", "atx_heading", "fenced_code_block", "list":
			// Direct content nodes within the panel
			tempDoc := adf.NewADFDocument()
			p.processNode(child, content, tempDoc)
			panel.Content = append(panel.Content, tempDoc.Content...)
		case "panel_end_mark":
			// Ignore panel end mark
			continue
		}
	}

	return panel
}

// extractPanelType extracts the panel type from a panel_start node
func (p *Translator) extractPanelType(panelStartNode *sitter.Node, content []byte) string {
	childCount := int(panelStartNode.ChildCount())
	for i := range childCount {
		child := panelStartNode.Child(uint(i))
		if child.Kind() == "panel_type" {
			// Look for the type child within panel_type
			typeChildCount := int(child.ChildCount())
			for j := range typeChildCount {
				typeChild := child.Child(uint(j))
				if typeChild.Kind() == "type" {
					typeText := string(content[typeChild.StartByte():typeChild.EndByte()])
					// Remove the # prefix if present
					if strings.HasPrefix(typeText, "#") {
						return strings.TrimPrefix(typeText, "#")
					}
					return typeText
				}
			}
		}
	}
	return "info" // default fallback
}

// convertPipeTable converts a pipe table to ADF table
func (p *Translator) convertPipeTable(node *sitter.Node, content []byte) *adf.ADFNode {
	table := adf.NewTableNode()

	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		switch child.Kind() {
		case "pipe_table_header":
			headerRow := p.convertPipeTableRow(child, content, true)
			if headerRow != nil {
				table.Content = append(table.Content, headerRow)
			}
		case "pipe_table_row":
			dataRow := p.convertPipeTableRow(child, content, false)
			if dataRow != nil {
				table.Content = append(table.Content, dataRow)
			}
		case "pipe_table_delimiter_row":
			// Skip delimiter rows - they're just formatting
			continue
		}
	}

	return table
}

// convertPipeTableRow converts a pipe table row to ADF table row
func (p *Translator) convertPipeTableRow(node *sitter.Node, content []byte, isHeader bool) *adf.ADFNode {
	row := adf.NewTableRowNode()

	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		if child.Kind() == "pipe_table_cell" {
			var cell *adf.ADFNode
			if isHeader {
				cell = adf.NewTableHeaderNode()
			} else {
				cell = adf.NewTableCellNode()
			}

			// Get cell content and convert it
			cellText := strings.TrimSpace(string(content[child.StartByte():child.EndByte()]))
			if cellText != "" {
				paragraph := adf.NewParagraphNode()

				// Parse formatting within the cell
				p.parseCellContent(cellText, paragraph, isHeader)

				cell.Content = append(cell.Content, paragraph)
			} else {
				// Empty cell gets empty paragraph
				cell.Content = append(cell.Content, adf.NewParagraphNode())
			}

			row.Content = append(row.Content, cell)
		}
	}

	return row
}

// parseCellContent parses the content of a table cell and handles formatting
func (p *Translator) parseCellContent(cellText string, paragraph *adf.ADFNode, isHeader bool) {
	// Simple parsing for bold text marked with **text**
	if strings.HasPrefix(cellText, "**") && strings.HasSuffix(cellText, "**") && len(cellText) > 4 {
		// Bold text
		innerText := cellText[2 : len(cellText)-2]
		textNode := adf.NewTextNode(innerText)
		textNode.Marks = append(textNode.Marks, &adf.ADFMark{Type: adf.MarkStrong})
		paragraph.Content = append(paragraph.Content, textNode)
	} else {
		// Plain text
		textNode := adf.NewTextNode(cellText)
		// Headers are automatically bold in ADF, but we can add explicit bold mark if needed
		if isHeader {
			textNode.Marks = append(textNode.Marks, &adf.ADFMark{Type: adf.MarkStrong})
		}
		paragraph.Content = append(paragraph.Content, textNode)
	}
}
