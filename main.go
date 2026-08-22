package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: shiplabel-lint <file>")
		os.Exit(2)
	}

	filename := os.Args[1]
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shiplabel-lint: %v\n", err)
		os.Exit(2)
	}

	labels, errs := ParseFile(filename, data)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprint(os.Stderr, e.Error())
			fmt.Fprintln(os.Stderr)
		}
		fmt.Fprintf(os.Stderr, "%d error(s) found\n", len(errs))
		os.Exit(1)
	}

	fmt.Printf("%s: %d label(s) OK\n", filename, len(labels))
}
