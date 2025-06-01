package adf2md

import (
	"encoding/json"
	"md-adf-exp/adf"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestADF(t *testing.T) {
	data, err := os.ReadFile("./testdata/md.json")
	assert.NoError(t, err)

	var adf adf.ADFNode
	err = json.Unmarshal(data, &adf)
	assert.NoError(t, err)

	tr := NewTranslator(&adf, NewMarkdownTranslator())

	expected := `# H1
## H2
1. Some text

2. Some more text



> Blockquote text


Inline Node 📍 https://antiklabs.atlassian.net/wiki/spaces/ANK/pages/124234/hello-world 

Implement epic browser

---
Panel paragraph

---
 @Person A 

---
**Strong** Paragraph 1

Paragraph 2

---
**Bold Text**

_Italic Text_

Prefix: Underlined Text

` + "`" + `Prefix: Inline Code Block` + "`" + `

-Prefix: Strikethrough text-

[Link](https://ankit.pl) 

- Prefix: Unordered list item 1
    - Next
        - Another
            - New level
- Unordered list item 2
- Unordered list item 3
1. Ordered list item 1
2. Ordered list item 2
3. Ordered list item 3
    1. nested
        1. second level
            1. third level
                1. fourth level

**Table Header 1** | **Table Header 2** | **Table Header 3**
--- | --- | ---
Table row 1 column 1 | Table row 1 column 2 | Table row 1 column 3
Table row 2 column 1 | Table row 2 column 2 | Table row 2 column 3
` + "```" + `go
package main

import (
    "fmt"
)

func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

**Table Header 1** | **Table Header 2** | **Table Header 3** | **Table Header 4** | **Table Header 5**
--- | --- | --- | --- | ---
Table row 1 column 1 | Table row 2 column 1 | Table row 3 column 1 | Table row 4 column 1 | Table row 5 column 1
Table row 1 column 2 | Table row 2 column 2 | Table row 3 column 2 | Table row 4 column 2 | Table row 5 column 2
Table row 1 column 2 | Table row 2 column 3 | Table row 3 column 3 | Table row 4 column 3 | Table row 5 column 3
`

	assert.Equal(t, expected, tr.Translate())
}

func TestADFReplaceAll(t *testing.T) {
	data, err := os.ReadFile("./testdata/md.json")
	assert.NoError(t, err)

	var adf adf.ADFNode
	err = json.Unmarshal(data, &adf)
	assert.NoError(t, err)

	adf.ReplaceAll("Prefix:", "Replaced:")

	dump, err := json.Marshal(adf)
	assert.NoError(t, err)

	assert.False(t, strings.Contains(string(dump), "Prefix:"))
	assert.True(t, strings.Contains(string(dump), "Replaced:"))
}
