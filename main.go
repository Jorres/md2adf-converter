package main

import (
	"fmt"
	"io"
	"md-adf-exp/md2adf"
	"os"
)

func main() {
	var input []byte
	var err error

	if len(os.Args) > 1 {
		filename := os.Args[1]
		input, err = os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
			os.Exit(1)
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
	}

	// Sample user mapping for testing
	userMapping := map[string]string{
		"@jorres@nebius.com": "6acd447c-fd28-4da8-b7cb-5b95d4405540",
	}

	translator := md2adf.NewTranslator(
		md2adf.WithUserEmailMapping(userMapping),
	)

	adfDoc, err := translator.TranslateToADF(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing markdown: %v\n", err)
		os.Exit(1)
	}

	// Output ADF JSON
	jsonOutput, err := adfDoc.ToJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonOutput))
}
