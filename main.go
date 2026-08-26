package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	strict := flag.Bool("strict", false, "reject any field other than to, from, weight, dims, service")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: shiplabel-lint [--strict] <file>")
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	filename := flag.Arg(0)
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shiplabel-lint: %v\n", err)
		os.Exit(2)
	}

	labels, errs := ParseFile(filename, data, *strict)
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
