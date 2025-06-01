package adf2md

import (
	"encoding/json"
	"fmt"
	"log"
	"md-adf-exp/adf"
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

// Translator transforms ADF to a new format.
type Translator struct {
	doc          *adf.ADFNode
	tsl          TagOpenerCloser
	buf          *strings.Builder
	mediaMapping map[string]*adf.ADFNode
}

// NewTranslator constructs an ADF translator.
func NewTranslator(tr TagOpenerCloser) *Translator {
	return &Translator{
		doc:          nil,
		tsl:          tr,
		buf:          nil,
		mediaMapping: make(map[string]*adf.ADFNode),
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

func (a *Translator) walk() {
	if a.doc == nil || len(a.doc.Content) == 0 {
		return
	}
	for _, parent := range a.doc.Content {
		a.visit(parent, a.doc, 0)
	}
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

		tag.WriteString(sanitize(n.Text))

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
		rows int
		cols int
		ccol int // current column count
		sep  bool
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
			if tr.table.cols != 0 {
				tag.WriteString(" | ")
			}
			tr.table.cols++
		case adf.ChildNodeTableCell:
			if tr.table.ccol != 0 {
				tag.WriteString(" | ")
			}
			tr.table.ccol++
		case adf.ChildNodeTableRow:
			tr.table.rows++
			if tr.table.rows == 1 && !tr.table.sep {
				tr.table.sep = true
			}
			tr.table.ccol = 0
		case adf.InlineNodeHardBreak:
			tag.WriteString("\n\n")
		case adf.InlineNodeMention:
			tag.WriteString(" @")
			tag.WriteString(tr.setOpenTagAttributesForMention(attrs))
			return tag.String() // Return early to avoid double processing
		case adf.InlineNodeCard:
			tag.WriteString(" 📍 ")
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
			tr.table.rows = 0
			tr.table.cols = 0
			tr.table.sep = false
		case adf.ChildNodeTableRow:
			tag.WriteString("\n")
			if tr.table.sep {
				for i := 0; i < tr.table.cols; i++ {
					tag.WriteString("---")
					if i != tr.table.cols-1 {
						tag.WriteString(" | ")
					}
				}
				tr.table.sep = false
				tag.WriteString("\n")
			}
		case adf.InlineNodeMention:
			tag.WriteString(" ")
		case adf.InlineNodeEmoji:
			tag.WriteString(" ")
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
	} else if h, ok := attrs["url"]; ok {
		tag.WriteString(fmt.Sprintf("%s ", h))
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

// extractMediaID extracts the media ID from attributes using proper struct unmarshalling
func (*MarkdownTranslator) extractMediaID(attrs interface{}) string {
	if attrs == nil {
		return ""
	}

	// Convert attributes to JSON and unmarshal into MediaAttributes struct
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

const (
	bgColorInfo    = "#deebff"
	bgColorNote    = "#eae6ff"
	bgColorError   = "#ffebe6"
	bgColorSuccess = "#e3fcef"
	bgColorWarning = "#fffae6"

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
		a := attrs.(map[string]interface{})
		if len(a) > 0 {
			tag.WriteString(":")
		}
		for k, v := range a {
			if k == "panelType" {
				switch v {
				case panelTypeInfo:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorInfo))
				case panelTypeNote:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorNote))
				case panelTypeError:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorError))
				case panelTypeSuccess:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorSuccess))
				case panelTypeWarning:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorWarning))
				}
			} else {
				tag.WriteString(fmt.Sprintf("|%s=%s", k, v))
			}
		}
	}
	tag.WriteString("}\n")

	return tag.String()
}

func nodePanelCloseHook(Connector) string {
	return "{panel}\n"
}
