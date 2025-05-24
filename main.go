package main

import (
	"fmt"
	"io"
	"md-adf-exp/internal/adf"
	"os"
)

func main() {
	var input []byte
	var err error

	// Read from stdin or file argument
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

	// Parse markdown and convert to ADF using clean interface
	parser := adf.NewAdfParser()
	adfDoc, err := parser.ConvertToADF(input)
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
