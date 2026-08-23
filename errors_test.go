package main

import (
	"strings"
	"testing"
)

func TestLabelErrorHeaderLine(t *testing.T) {
	e := &LabelError{
		File:       "t.labels",
		Pos:        Position{Line: 7, Col: 5},
		SourceLine: "to: 221B Baker St, London",
		Message:    "bad address",
	}

	got := e.Error()
	wantHeader := "t.labels:7:5: bad address\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("expected header %q, got %q", wantHeader, got)
	}
}

// TestLabelErrorCaretAlignment checks that the caret on the third line
// lines up under the exact source character named by Pos.Col, even
// when the line number's width changes how much the caret line is
// padded on the left.
func TestLabelErrorCaretAlignment(t *testing.T) {
	cases := []struct {
		name string
		pos  Position
		src  string
	}{
		{"single-digit line, early column", Position{Line: 1, Col: 5}, "to: 221B Baker St, London"},
		{"single-digit line, late column", Position{Line: 9, Col: 9}, "weight: 1.5lbs"},
		{"multi-digit line", Position{Line: 42, Col: 9}, "weight: 1.5lbs"},
		{"column at start of line", Position{Line: 3, Col: 1}, "carrier: fedex"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := &LabelError{File: "t.labels", Pos: c.pos, SourceLine: c.src, Message: "problem"}
			lines := strings.Split(strings.TrimRight(e.Error(), "\n"), "\n")
			if len(lines) != 3 {
				t.Fatalf("expected 3 lines, got %d: %q", len(lines), e.Error())
			}
			src, caret := lines[1], lines[2]

			pipeSrc := strings.IndexByte(src, '|')
			pipeCaret := strings.IndexByte(caret, '|')
			if pipeSrc == -1 || pipeCaret == -1 {
				t.Fatalf("expected both lines to contain '|': src=%q caret=%q", src, caret)
			}
			if pipeSrc != pipeCaret {
				t.Errorf("expected '|' to align between src and caret lines: src at %d, caret at %d", pipeSrc, pipeCaret)
			}

			caretPos := strings.IndexByte(caret, '^')
			if caretPos == -1 {
				t.Fatalf("expected a caret in %q", caret)
			}
			wantCaretPos := pipeSrc + 2 + (c.pos.Col - 1)
			if caretPos != wantCaretPos {
				t.Errorf("expected caret at column %d, got %d (src=%q caret=%q)", wantCaretPos, caretPos, src, caret)
			}

			gotChar := src[pipeSrc+2+(c.pos.Col-1)]
			wantChar := c.src[c.pos.Col-1]
			if gotChar != wantChar {
				t.Errorf("caret points at %q, want %q", gotChar, wantChar)
			}
		})
	}
}
