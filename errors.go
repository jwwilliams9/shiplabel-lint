package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Position is a 1-based line and column, matching what an editor shows
// in its status bar, so it can be pasted straight into "go to line".
type Position struct {
	Line int
	Col  int
}

// LabelError ties a problem to an exact spot in the source file.
// Error() renders it as a message plus a source snippet with a caret,
// the same shape as a compiler diagnostic, because "line 9" alone
// means re-opening the file and counting.
type LabelError struct {
	File       string
	Pos        Position
	SourceLine string
	Message    string
}

func (e *LabelError) Error() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s:%d:%d: %s\n", e.File, e.Pos.Line, e.Pos.Col, e.Message)

	lineNumStr := strconv.Itoa(e.Pos.Line)
	pad := strings.Repeat(" ", len(lineNumStr))

	fmt.Fprintf(&b, "  %s | %s\n", lineNumStr, e.SourceLine)

	col := e.Pos.Col - 1
	if col < 0 {
		col = 0
	}
	fmt.Fprintf(&b, "  %s | %s^\n", pad, strings.Repeat(" ", col))

	return b.String()
}
