package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	strict := flag.Bool("strict", false, "reject any field other than to, from, weight, dims, service")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: shiplabel-lint [--strict] <file|dir> [file|dir ...]")
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	var files []string
	for _, arg := range flag.Args() {
		found, err := collectFiles(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "shiplabel-lint: %v\n", err)
			os.Exit(2)
		}
		files = append(files, found...)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "shiplabel-lint: no .labels files found")
		os.Exit(2)
	}

	var totalErrors int
	for _, filename := range files {
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "shiplabel-lint: %v\n", err)
			os.Exit(2)
		}

		labels, errs := ParseFile(filename, data, *strict)
		for _, e := range errs {
			fmt.Fprint(os.Stderr, e.Error())
			fmt.Fprintln(os.Stderr)
		}
		totalErrors += len(errs)

		if len(errs) == 0 {
			fmt.Printf("%s: %d label(s) OK\n", filename, len(labels))
		}
	}

	if totalErrors > 0 {
		fmt.Fprintf(os.Stderr, "%d error(s) found\n", totalErrors)
		os.Exit(1)
	}
}

// collectFiles resolves a command-line argument to a list of batch files
// to parse. A plain file is returned as-is; a directory is walked for
// *.labels files so a whole batch drop can be checked in one run without
// the caller having to glob it themselves.
func collectFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}

	var files []string
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != path && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".labels") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
