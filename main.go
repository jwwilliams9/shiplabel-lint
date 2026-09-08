package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	strict := flag.Bool("strict", false, "reject any field other than to, from, weight, dims, service")
	jsonOutput := flag.Bool("json", false, "report results as JSON on stdout instead of compiler-style text")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: shiplabel-lint [--strict] [--json] <file|dir> [file|dir ...]")
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
	var results []fileResult
	for _, filename := range files {
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "shiplabel-lint: %v\n", err)
			os.Exit(2)
		}

		labels, errs := ParseFile(filename, data, *strict)
		totalErrors += len(errs)

		if *jsonOutput {
			results = append(results, newFileResult(filename, labels, errs))
			continue
		}

		for _, e := range errs {
			fmt.Fprint(os.Stderr, e.Error())
			fmt.Fprintln(os.Stderr)
		}
		if len(errs) == 0 {
			fmt.Printf("%s: %d label(s) OK\n", filename, len(labels))
		}
	}

	if *jsonOutput {
		printJSON(results, totalErrors)
	}

	if totalErrors > 0 {
		if !*jsonOutput {
			fmt.Fprintf(os.Stderr, "%d error(s) found\n", totalErrors)
		}
		os.Exit(1)
	}
}

// fileResult is one file's outcome, shaped for JSON encoding. Line and
// column are pulled out of each error's Position so a consumer doesn't
// have to know about LabelError's internal shape to read them.
type fileResult struct {
	File   string      `json:"file"`
	OK     bool        `json:"ok"`
	Labels int         `json:"labels"`
	Errors []jsonError `json:"errors"`
}

type jsonError struct {
	Line       int    `json:"line"`
	Col        int    `json:"col"`
	Message    string `json:"message"`
	SourceLine string `json:"source_line"`
}

func newFileResult(filename string, labels []Label, errs []*LabelError) fileResult {
	r := fileResult{File: filename, OK: len(errs) == 0, Labels: len(labels), Errors: []jsonError{}}
	for _, e := range errs {
		r.Errors = append(r.Errors, jsonError{
			Line:       e.Pos.Line,
			Col:        e.Pos.Col,
			Message:    e.Message,
			SourceLine: e.SourceLine,
		})
	}
	return r
}

func printJSON(results []fileResult, totalErrors int) {
	out := struct {
		Files      []fileResult `json:"files"`
		ErrorCount int          `json:"error_count"`
	}{Files: results, ErrorCount: totalErrors}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "shiplabel-lint: %v\n", err)
		os.Exit(2)
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
