package adf2md

import (
	"encoding/json"
	"fmt"
	"github.com/jorres/md2adf-translator/adf"
	"log"
	"strings"
)

// TagOpener is a tag opener.
type TagOpener interface {
	Open(c Connector, depth int) string
}

// TagCloser is a tag closer.
type TagCloser interface {
	Close(Connector) string
}

// TagOpenerCloser wraps tag opener and closer.
type TagOpenerCloser interface {
	TagOpener
	TagCloser
}

// Connector is a connector interface.
type Connector interface {
	GetType() adf.NodeType
	GetAttributes() interface{}
}

// MediaAttributes represents the attributes of a media node in ADF
type MediaAttributes struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Collection string `json:"collection"`
	Alt        string `json:"alt,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
}

// InlineCardAttributes represents the attributes of an inline card node in ADF
type InlineCardAttributes struct {
	Data string `json:"data"`
	URL  string `json:"url"`
}

// Translator transforms ADF to a new format.
type Translator struct {
	doc               *adf.ADFNode
	tsl               TagOpenerCloser
	buf               *strings.Builder
	mediaMapping      map[string]*adf.ADFNode
	inlineCardMapping map[string]*adf.ADFNode
}

// NewTranslator constructs an ADF translator.
func NewTranslator(tr TagOpenerCloser) *Translator {
	return &Translator{
		doc:               nil,
		tsl:               tr,
		buf:               nil,
		mediaMapping:      make(map[string]*adf.ADFNode),
		inlineCardMapping: make(map[string]*adf.ADFNode),
	}
}

// Translate translates ADF to a new format.
func (a *Translator) Translate(doc *adf.ADFNode) string {
	a.doc = doc
	a.buf = new(strings.Builder)

	a.walk()
	return a.buf.String()
}

// GetMediaMapping returns the mapping of media IDs to their ADF nodes.
func (a *Translator) GetMediaMapping() map[string]*adf.ADFNode {
	return a.mediaMapping
}

// GetInlineCardMapping returns the mapping of inline card URLs to their ADF nodes.
func (a *Translator) GetInlineCardMapping() map[string]*adf.ADFNode {
	return a.inlineCardMapping
}

func (a *Translator) walk() {
	if a.doc == nil || len(a.doc.Content) == 0 {
		return
	}
	for _, parent := range a.doc.Content {
		a.visit(parent, a.doc, 0)
	}
}

func (a *Translator) CheckSupport(n *adf.ADFNode) map[adf.NodeType]bool {
	forbidden := make(map[adf.NodeType]bool)

	if n.Type == adf.NodeBlockquote {
		forbidden[n.Type] = true
	}

	for _, child := range n.Content {
		for k, _ := range a.CheckSupport(child) {
			forbidden[k] = true
		}
	}

	return forbidden
}

func (a *Translator) visit(n *adf.ADFNode, parent *adf.ADFNode, depth int) {
	if n.Type == adf.NodeMediaGroup || n.Type == adf.NodeMediaSingle {
		// We currently don't distinguish between group \ single, just preserve them
		// fully and resend them back to jira on update
		var firstChildMediaAttrs MediaAttributes
		firstChildNode := n.Content[0]
		jsonBytes, err := json.Marshal(firstChildNode.Attrs)
		if err != nil {
			panic("NodeMedia node is supposed to have children")
		}

		_ = json.Unmarshal(jsonBytes, &firstChildMediaAttrs)
		if firstChildMediaAttrs.ID != "" {
			a.mediaMapping[firstChildMediaAttrs.ID] = n
		}
	}

	if n.Type == adf.InlineNodeCard {
		var attrs InlineCardAttributes
		jsonBytes, _ := json.Marshal(n.Attrs)
		_ = json.Unmarshal(jsonBytes, &attrs)
		if attrs.URL != "" {
			a.inlineCardMapping[attrs.URL] = n
		}
	}

	a.buf.WriteString(a.tsl.Open(n, depth))

	for _, child := range n.Content {
		a.visit(child, n, depth+1)
	}

	if adf.GetADFNodeType(n.Type) == adf.NodeTypeChild {
		var tag strings.Builder

		opened := make([]*adf.ADFMark, 0, len(n.Marks))
		if n.Type == adf.ChildNodeText {
			for _, m := range n.Marks {
				opened = append(opened, m)
				tag.WriteString(a.tsl.Open(m, depth))
			}
		}

		textContent := sanitize(n.Text)
		
		// If we're inside a table cell, accumulate content in the translator
		var mdTranslator *MarkdownTranslator
		if mt, ok := a.tsl.(*MarkdownTranslator); ok {
			mdTranslator = mt
		} else if jmt, ok := a.tsl.(*JiraMarkdownTranslator); ok {
			mdTranslator = jmt.MarkdownTranslator
		}
		
		if mdTranslator != nil && mdTranslator.isInTableCell() {
			// Add opening marks
			for _, m := range opened {
				mdTranslator.addCellContent(a.tsl.Open(m, depth))
			}
			mdTranslator.addCellContent(textContent)
			// Add closing marks
			for i := len(opened) - 1; i >= 0; i-- {
				m := opened[i]
				mdTranslator.addCellContent(a.tsl.Close(m))
			}
			return
		}

		tag.WriteString(textContent)

		// Close tags in reverse order.
		for i := len(opened) - 1; i >= 0; i-- {
			m := opened[i]
			tag.WriteString(a.tsl.Close(m))
		}

		a.buf.WriteString(tag.String())
	}

	a.buf.WriteString(a.tsl.Close(n))
}

func sanitize(s string) string {
	s = strings.TrimRight(s, "\n")
	s = strings.ReplaceAll(s, "<", "❬")
	s = strings.ReplaceAll(s, ">", "❭")
	return s
}

type nodeTypeHook map[adf.NodeType]func(Connector) string

// UserEmailResolver is a function type for resolving user IDs to emails
type UserEmailResolver func(userID string) string

// MarkdownTranslator is a markdown translator.
type MarkdownTranslator struct {
	table struct {
		rows        int
		cols        int
		ccol        int        // current column count
		sep         bool
		content     [][]string // store table content for width calculation
		widths      []int      // column widths
		inTable     bool       // whether we're currently inside a table
		inTableCell bool       // whether we're currently inside a table cell/header
	}
	list struct {
		ol, ul  map[int]bool
		depthO  int
		depthU  int
		counter map[int]int // each level starts with same numeric counter at the moment.
	}
	openHooks  nodeTypeHook
	closeHooks nodeTypeHook

	emailResolver UserEmailResolver
}

// MarkdownTranslatorOption is a functional option for MarkdownTranslator.
type MarkdownTranslatorOption func(*MarkdownTranslator)

// NewMarkdownTranslator constructs markdown translator.
func NewMarkdownTranslator(opts ...MarkdownTranslatorOption) *MarkdownTranslator {
	tr := MarkdownTranslator{
		list: struct {
			ol, ul  map[int]bool
			depthO  int
			depthU  int
			counter map[int]int
		}{
			ol:      make(map[int]bool),
			ul:      make(map[int]bool),
			counter: make(map[int]int),
		},
	}

	for _, opt := range opts {
		opt(&tr)
	}

	return &tr
}

// WithMarkdownOpenHooks sets open hooks of a markdown translator.
func WithMarkdownOpenHooks(hooks nodeTypeHook) MarkdownTranslatorOption {
	return func(tr *MarkdownTranslator) {
		tr.openHooks = hooks
	}
}

// WithMarkdownCloseHooks sets close hooks of a markdown translator.
func WithMarkdownCloseHooks(hooks nodeTypeHook) MarkdownTranslatorOption {
	return func(tr *MarkdownTranslator) {
		tr.closeHooks = hooks
	}
}

// WithUserEmailResolver sets a user email resolver function
func WithUserEmailResolver(resolver UserEmailResolver) MarkdownTranslatorOption {
	return func(tr *MarkdownTranslator) {
		tr.emailResolver = resolver
	}
}

// Open implements TagOpener interface.
//
//nolint:gocyclo
// renderTable renders the complete table with proper formatting
func (tr *MarkdownTranslator) renderTable() string {
	if len(tr.table.content) == 0 {
		return ""
	}

	var result strings.Builder

	// Calculate column widths
	tr.calculateColumnWidths()

	// Render each row
	for rowIdx, row := range tr.table.content {
		result.WriteString("|")
		for colIdx, cell := range row {
			width := tr.table.widths[colIdx]
			padded := fmt.Sprintf(" %-*s ", width, cell)
			result.WriteString(padded)
			result.WriteString("|")
		}
		result.WriteString("\n")

		// Add separator after header row
		if rowIdx == 0 {
			result.WriteString("|")
			for colIdx := range row {
				width := tr.table.widths[colIdx]
				separator := strings.Repeat("-", width+2) // +2 for spaces around content
				result.WriteString(separator)
				result.WriteString("|")
			}
			result.WriteString("\n")
		}
	}

	return result.String()
}

// calculateColumnWidths calculates the maximum width for each column
func (tr *MarkdownTranslator) calculateColumnWidths() {
	if len(tr.table.content) == 0 {
		return
	}

	maxCols := 0
	for _, row := range tr.table.content {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	tr.table.widths = make([]int, maxCols)

	// Find maximum width for each column
	for _, row := range tr.table.content {
		for colIdx, cell := range row {
			if len(cell) > tr.table.widths[colIdx] {
				tr.table.widths[colIdx] = len(cell)
			}
		}
	}

	// Ensure minimum width of 5 for readability
	for i := range tr.table.widths {
		if tr.table.widths[i] < 5 {
			tr.table.widths[i] = 5
		}
	}
}

// addCellContent adds content to the current table cell
func (tr *MarkdownTranslator) addCellContent(content string) {
	if tr.table.rows == 0 || len(tr.table.content) < tr.table.rows {
		return
	}
	
	currentRow := &tr.table.content[tr.table.rows-1]
	// Use cols for headers and ccol for regular cells
	currentCol := tr.table.cols - 1
	if tr.table.ccol > 0 {
		currentCol = tr.table.ccol - 1
	}
	
	// Ensure we have enough cells in the current row
	for len(*currentRow) <= currentCol {
		*currentRow = append(*currentRow, "")
	}
	
	// Append content to the current cell
	(*currentRow)[currentCol] += content
}

// isInTableCell returns true if we're currently inside a table cell
func (tr *MarkdownTranslator) isInTableCell() bool {
	return tr.table.inTableCell
}

func (tr *MarkdownTranslator) Open(n Connector, _ int) string {
	var tag strings.Builder

	nt, attrs := n.GetType(), n.GetAttributes()

	if hook, ok := tr.openHooks[nt]; ok {
		tag.WriteString(hook(n))
	} else {
		switch nt {
		case adf.NodeBlockquote:
			tag.WriteString("> ")
		case adf.NodeCodeBlock:
			tag.WriteString("```")

			nl := true
			if attrs != nil {
				a := attrs.(map[string]interface{})
				for k := range a {
					if k == "language" {
						nl = false
						break
					}
				}
			}
			if nl {
				tag.WriteString("\n")
			}
		case adf.NodePanel:
			tag.WriteString("---\n")
		case adf.NodeTable:
			tag.WriteString("\n")
			tr.table.inTable = true
		case adf.NodeMedia:
			mediaID := tr.extractMediaID(attrs)
			if mediaID != "" {
				tag.WriteString(fmt.Sprintf("\n{attachment:%s}", mediaID))
			} else {
				tag.WriteString("\n[attachment]")
			}
		case adf.NodeBulletList:
			tr.list.depthU++
			tr.list.ul[tr.list.depthU] = true
		case adf.NodeOrderedList:
			tr.list.depthO++
			tr.list.ol[tr.list.depthO] = true
		case adf.ChildNodeListItem:
			if tr.list.ol[tr.list.depthO] {
				for i := 0; i < tr.list.depthO-1; i++ {
					tag.WriteString("    ")
				}
				tr.list.counter[tr.list.depthO]++
				tag.WriteString(fmt.Sprintf("%d. ", tr.list.counter[tr.list.depthO]))
			} else {
				for i := 0; i < tr.list.depthU-1; i++ {
					tag.WriteString("    ")
				}
				tag.WriteString("- ")
			}
		case adf.ChildNodeTableHeader:
			tr.table.cols++
			tr.table.inTableCell = true
			// Don't output anything, content will be captured later
		case adf.ChildNodeTableCell:
			tr.table.ccol++
			tr.table.inTableCell = true
			// Don't output anything, content will be captured later
		case adf.ChildNodeTableRow:
			tr.table.rows++
			if tr.table.rows == 1 && !tr.table.sep {
				tr.table.sep = true
			}
			// Initialize row in content if needed
			if len(tr.table.content) < tr.table.rows {
				tr.table.content = append(tr.table.content, make([]string, 0))
			}
			tr.table.ccol = 0
		case adf.InlineNodeHardBreak:
			tag.WriteString("\n\n")
		case adf.InlineNodeMention:
			tag.WriteString(" @")
			tag.WriteString(tr.setOpenTagAttributesForMention(attrs))
			return tag.String() // Return early to avoid double processing
		case adf.InlineNodeCard:
			cardURL := tr.extractCardURL(attrs)
			if cardURL != "" {
				tag.WriteString(fmt.Sprintf("[link](%s)", cardURL))
			} else {
				tag.WriteString(" 📍 ")
			}
		case adf.MarkUnderline:
			tag.WriteString("<u>")
		case adf.MarkStrong:
			tag.WriteString("**")
		case adf.MarkEm:
			tag.WriteString("_")
		case adf.MarkCode:
			tag.WriteString("`")
		case adf.MarkStrike:
			tag.WriteString("-")
		case adf.MarkLink:
			tag.WriteString("[")
		}
	}

	tag.WriteString(tr.setOpenTagAttributes(attrs))

	return tag.String()
}

// Close implements TagCloser interface.
//
//nolint:gocyclo
func (tr *MarkdownTranslator) Close(n Connector) string {
	var tag strings.Builder

	nt := n.GetType()

	if hook, ok := tr.closeHooks[nt]; ok {
		tag.WriteString(hook(n))
	} else {
		switch nt {
		case adf.NodeBlockquote:
			tag.WriteString("\n")
		case adf.NodeCodeBlock:
			tag.WriteString("\n```\n")
		case adf.NodePanel:
			tag.WriteString("---\n")
		case adf.NodeHeading:
			tag.WriteString("\n")
		case adf.NodeBulletList:
			tr.list.ul[tr.list.depthU] = false
			tr.list.depthU--
		case adf.NodeOrderedList:
			tr.list.ol[tr.list.depthO] = false
			tr.list.depthO--
		case adf.NodeParagraph:
			if tr.list.ul[tr.list.depthU] || tr.list.ol[tr.list.depthO] {
				tag.WriteString("\n")
			} else if tr.table.rows == 0 {
				tag.WriteString("\n\n")
			}
		case adf.NodeTable:
			// Render the complete table with proper formatting
			tag.WriteString(tr.renderTable())
			// Reset table state
			tr.table.rows = 0
			tr.table.cols = 0
			tr.table.sep = false
			tr.table.content = nil
			tr.table.widths = nil
			tr.table.inTable = false
			tr.table.inTableCell = false
		case adf.ChildNodeTableHeader:
			tr.table.inTableCell = false
		case adf.ChildNodeTableCell:
			tr.table.inTableCell = false
		case adf.ChildNodeTableRow:
			// Table rows are handled in renderTable()
		case adf.InlineNodeMention:
			tag.WriteString(" ")
		case adf.InlineNodeEmoji:
			tag.WriteString(" ")
		case adf.MarkUnderline:
			tag.WriteString("</u>")
		case adf.MarkStrong:
			tag.WriteString("**")
		case adf.MarkEm:
			tag.WriteString("_")
		case adf.MarkCode:
			tag.WriteString("`")
		case adf.MarkStrike:
			tag.WriteString("-")
		case adf.MarkLink:
			tag.WriteString("]")
		}
	}

	tag.WriteString(tr.setCloseTagAttributes(n.GetAttributes()))

	return tag.String()
}

func (tr *MarkdownTranslator) setOpenTagAttributes(a interface{}) string {
	if a == nil {
		return ""
	}

	var (
		tag strings.Builder
		nl  bool
	)

	attrs := a.(map[string]interface{})
	for k, v := range attrs {
		if tr.isValidAttr(k) {
			switch k {
			case "language":
				tag.WriteString(fmt.Sprintf("%s", v))
				nl = true
			case "level":
				for i := 0; i < int(v.(float64)); i++ {
					tag.WriteString("#")
				}
				tag.WriteString(" ")
			case "text":
				tag.WriteString(fmt.Sprintf("%s", v))
				nl = false
			}
		}
		if nl {
			tag.WriteString("\n")
		}
	}

	return tag.String()
}

func (tr *MarkdownTranslator) setOpenTagAttributesForMention(a interface{}) string {
	if a == nil {
		return ""
	}

	attrs := a.(map[string]interface{})

	// For mentions, we want to render as @email instead of @displayName
	if userID, ok := attrs["id"]; ok {
		if email := tr.resolveUserEmail(userID.(string)); email != "" {
			return email
		}
	}

	// Fallback to display name if email resolution fails
	if text, ok := attrs["text"]; ok {
		textStr := text.(string)
		if tr.emailResolver != nil {
			log.Printf("DEBUG: Using fallback text: %s", textStr)
		}
		// Remove leading @ if present since we already add @ in the Open function
		if strings.HasPrefix(textStr, "@") {
			return textStr[1:]
		}
		return textStr
	}

	return ""
}

// resolveUserEmail attempts to resolve a user ID to email
func (tr *MarkdownTranslator) resolveUserEmail(userID string) string {
	if tr.emailResolver != nil {
		return tr.emailResolver(userID)
	}
	return ""
}

func (*MarkdownTranslator) setCloseTagAttributes(a interface{}) string {
	if a == nil {
		return ""
	}

	var tag strings.Builder

	attrs := a.(map[string]interface{})
	if h, ok := attrs["href"]; ok {
		tag.WriteString(fmt.Sprintf("(%s) ", h))
	}

	return tag.String()
}

func (*MarkdownTranslator) isValidAttr(attr string) bool {
	known := []string{"language", "level", "text"}
	for _, k := range known {
		if k == attr {
			return true
		}
	}
	return false
}

// extractMediaID extracts the media ID from attributes
func (*MarkdownTranslator) extractMediaID(attrs interface{}) string {
	if attrs == nil {
		return ""
	}

	jsonBytes, err := json.Marshal(attrs)
	if err != nil {
		return ""
	}

	var mediaAttrs MediaAttributes
	if err := json.Unmarshal(jsonBytes, &mediaAttrs); err != nil {
		return ""
	}

	return mediaAttrs.ID
}

// extractCardURL extracts the inline card URL from attributes
func (*MarkdownTranslator) extractCardURL(attrs interface{}) string {
	if attrs == nil {
		return ""
	}

	jsonBytes, err := json.Marshal(attrs)
	if err != nil {
		return ""
	}

	var inlineCardAttrs InlineCardAttributes
	if err := json.Unmarshal(jsonBytes, &inlineCardAttrs); err != nil {
		return ""
	}

	return inlineCardAttrs.URL
}

const (
	panelTypeInfo    = "info"
	panelTypeNote    = "note"
	panelTypeError   = "error"
	panelTypeSuccess = "success"
	panelTypeWarning = "warning"
)

// JiraMarkdownTranslator is a jira markdown translator.
type JiraMarkdownTranslator struct {
	*MarkdownTranslator
}

// NewJiraMarkdownTranslator constructs jira markdown translator.
func NewJiraMarkdownTranslator(opts ...MarkdownTranslatorOption) *JiraMarkdownTranslator {
	openHooks := nodeTypeHook{
		adf.NodePanel: nodePanelOpenHook,
	}

	closeHooks := nodeTypeHook{
		adf.NodePanel: nodePanelCloseHook,
	}

	// Combine built-in hooks with any additional options
	allOpts := []MarkdownTranslatorOption{
		WithMarkdownOpenHooks(openHooks),
		WithMarkdownCloseHooks(closeHooks),
	}
	allOpts = append(allOpts, opts...)

	return &JiraMarkdownTranslator{
		MarkdownTranslator: NewMarkdownTranslator(allOpts...),
	}
}

// Open implements TagOpener interface.
func (tr *JiraMarkdownTranslator) Open(n Connector, d int) string {
	return tr.MarkdownTranslator.Open(n, d)
}

// Close implements TagCloser interface.
func (tr *JiraMarkdownTranslator) Close(n Connector) string {
	return tr.MarkdownTranslator.Close(n)
}

func nodePanelOpenHook(n Connector) string {
	attrs := n.GetAttributes()

	var tag strings.Builder

	tag.WriteString("\n{panel")
	if attrs != nil {
		a := attrs.(map[string]any)
		if len(a) > 0 {
			tag.WriteString(":")
		}
		for k, v := range a {
			if k == "panelType" {
				tag.WriteString(fmt.Sprintf("type=%s", v.(string)))
			} else {
				tag.WriteString(fmt.Sprintf("|%s=%s", k, v))
			}
		}
	}
	tag.WriteString("}\n")

	return tag.String()
}

func nodePanelCloseHook(Connector) string {
	return "{/panel}\n"
}
