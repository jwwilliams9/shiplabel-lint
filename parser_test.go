package main

import (
	"strings"
	"testing"
)

func TestParseValidLabel(t *testing.T) {
	src := "to: 742 Evergreen Terrace, Springfield, IL 62704\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 3.2 lb\n" +
		"dims: 14x10x6 in\n" +
		"service: priority\n"

	labels, errs := ParseFile("t.labels", []byte(src), false)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(labels))
	}

	to := labels[0].Fields["to"]
	if to.Value != "742 Evergreen Terrace, Springfield, IL 62704" {
		t.Errorf("unexpected to value: %q", to.Value)
	}
	if to.Pos != (Position{Line: 1, Col: 5}) {
		t.Errorf("unexpected to position: %+v", to.Pos)
	}

	weight := labels[0].Fields["weight"]
	if weight.Pos != (Position{Line: 3, Col: 9}) {
		t.Errorf("unexpected weight position: %+v", weight.Pos)
	}
}

func TestParseMultipleLabelsSeparatedByBlankLine(t *testing.T) {
	src := "to: 742 Evergreen Terrace, Springfield, IL 62704\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 3.2 lb\n" +
		"dims: 14x10x6 in\n" +
		"\n" +
		"to: 1 Infinite Loop, Cupertino, CA 95014\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 1 lb\n" +
		"dims: 2x2x2 in\n"

	labels, errs := ParseFile("t.labels", []byte(src), false)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	if labels[0].StartLine != 1 {
		t.Errorf("expected first label to start at line 1, got %d", labels[0].StartLine)
	}
	if labels[1].StartLine != 6 {
		t.Errorf("expected second label to start at line 6, got %d", labels[1].StartLine)
	}
}

func TestParseIgnoresCommentLines(t *testing.T) {
	src := "# a batch of one\n" +
		"to: 742 Evergreen Terrace, Springfield, IL 62704\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 3.2 lb\n" +
		"dims: 14x10x6 in\n"

	labels, errs := ParseFile("t.labels", []byte(src), false)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(labels))
	}
	if labels[0].StartLine != 2 {
		t.Errorf("expected label to start at line 2, got %d", labels[0].StartLine)
	}
}

func TestParseHandlesCRLF(t *testing.T) {
	src := "to: 742 Evergreen Terrace, Springfield, IL 62704\r\n" +
		"from: 1 Amazon Way, Reno, NV 89501\r\n" +
		"weight: 3.2 lb\r\n" +
		"dims: 14x10x6 in\r\n"

	labels, errs := ParseFile("t.labels", []byte(src), false)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if v := labels[0].Fields["weight"].Value; v != "3.2 lb" {
		t.Errorf("expected trailing \\r to be stripped, got %q", v)
	}
}

func TestParseMissingColon(t *testing.T) {
	labels, errs := ParseFile("t.labels", []byte("hello world\n"), false)
	if len(labels) != 0 {
		t.Fatalf("expected no labels, got %d", len(labels))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	want := Position{Line: 1, Col: 12}
	if errs[0].Pos != want {
		t.Errorf("expected position %+v, got %+v", want, errs[0].Pos)
	}
}

func TestParseEmptyFieldName(t *testing.T) {
	_, errs := ParseFile("t.labels", []byte(" : value\n"), false)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	want := Position{Line: 1, Col: 2}
	if errs[0].Pos != want {
		t.Errorf("expected position %+v, got %+v", want, errs[0].Pos)
	}
	if !strings.Contains(errs[0].Message, "field name is empty") {
		t.Errorf("unexpected message: %q", errs[0].Message)
	}
}

func TestParseUnknownFieldAllowedByDefault(t *testing.T) {
	src := "to: 742 Evergreen Terrace, Springfield, IL 62704\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 3.2 lb\n" +
		"dims: 14x10x6 in\n" +
		"carrier: fedex\n"

	labels, errs := ParseFile("t.labels", []byte(src), false)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(labels))
	}
	if v := labels[0].Fields["carrier"].Value; v != "fedex" {
		t.Errorf("expected unknown field to be kept as-is, got %q", v)
	}
}

func TestParseUnknownFieldRejectedByStrict(t *testing.T) {
	src := "to: 742 Evergreen Terrace, Springfield, IL 62704\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 3.2 lb\n" +
		"dims: 14x10x6 in\n" +
		"carrier: fedex\n"

	_, errs := ParseFile("t.labels", []byte(src), true)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	want := Position{Line: 5, Col: 1}
	if errs[0].Pos != want {
		t.Errorf("expected position %+v, got %+v", want, errs[0].Pos)
	}
	if !strings.Contains(errs[0].Message, `unknown field "carrier"`) {
		t.Errorf("unexpected message: %q", errs[0].Message)
	}
}

func TestParseDuplicateField(t *testing.T) {
	src := "to: 742 Evergreen Terrace, Springfield, IL 62704\n" +
		"to: 1 Main St, Reno, NV 89501\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 3.2 lb\n" +
		"dims: 14x10x6 in\n"

	_, errs := ParseFile("t.labels", []byte(src), false)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	want := Position{Line: 2, Col: 1}
	if errs[0].Pos != want {
		t.Errorf("expected position %+v, got %+v", want, errs[0].Pos)
	}
	if !strings.Contains(errs[0].Message, `"to" is already set`) {
		t.Errorf("unexpected message: %q", errs[0].Message)
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	_, errs := ParseFile("t.labels", []byte("to: 742 Evergreen Terrace, Springfield, IL 62704\n"), false)
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors (from, weight, dims), got %d: %v", len(errs), errs)
	}
	for _, e := range errs {
		if e.Pos != (Position{Line: 1, Col: 1}) {
			t.Errorf("expected missing-field error to point at label start, got %+v", e.Pos)
		}
	}
}

func TestParseEmptyValueColumn(t *testing.T) {
	// with no value after the colon, the caret should land right after
	// the colon rather than past the end of the line.
	_, errs := ParseFile("t.labels", []byte("to:\nfrom: 1 Amazon Way, Reno, NV 89501\nweight: 1 lb\ndims: 1x1x1 in\n"), false)
	var toErr *LabelError
	for _, e := range errs {
		if strings.Contains(e.Message, "address") {
			toErr = e
		}
	}
	if toErr == nil {
		t.Fatalf("expected an address validation error, got %v", errs)
	}
	if toErr.Pos != (Position{Line: 1, Col: 4}) {
		t.Errorf("expected position {1 4}, got %+v", toErr.Pos)
	}
}

func TestValidateAddress(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"742 Evergreen Terrace, Springfield, IL 62704", true},
		{"1 Amazon Way, Reno, NV 89501-1234", true},
		{"221B Baker St, London", false},
		{"", false},
	}
	for _, c := range cases {
		err := validateAddress(c.value)
		if (err == nil) != c.ok {
			t.Errorf("validateAddress(%q): got err=%v, want ok=%v", c.value, err, c.ok)
		}
	}
}

func TestValidateWeight(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"3.2 lb", true},
		{"4 oz", true},
		{"1.5kg", true},
		{"1.5lbs", false},
		{"0 lb", false},
		{"-1 lb", false},
		{"lb", false},
	}
	for _, c := range cases {
		err := validateWeight(c.value)
		if (err == nil) != c.ok {
			t.Errorf("validateWeight(%q): got err=%v, want ok=%v", c.value, err, c.ok)
		}
	}
}

func TestValidateDims(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"14x10x6 in", true},
		{"1.5x2x3 cm", true},
		{"14x10 in", false},
		{"14x10x6", false},
		{"14x10x6 ft", false},
	}
	for _, c := range cases {
		err := validateDims(c.value)
		if (err == nil) != c.ok {
			t.Errorf("validateDims(%q): got err=%v, want ok=%v", c.value, err, c.ok)
		}
	}
}

func TestValidateService(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"ground", true},
		{"PRIORITY", true},
		{"overnight", true},
		{"same-day", false},
	}
	for _, c := range cases {
		err := validateService(c.value)
		if (err == nil) != c.ok {
			t.Errorf("validateService(%q): got err=%v, want ok=%v", c.value, err, c.ok)
		}
	}
}

func TestParseSampleFile(t *testing.T) {
	// mirrors testdata/sample.labels: one good label, one with a bad
	// address and a bad weight.
	src := "to: 742 Evergreen Terrace, Springfield, IL 62704\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 3.2 lb\n" +
		"dims: 14x10x6 in\n" +
		"service: priority\n" +
		"\n" +
		"to: 221B Baker St, London\n" +
		"from: 1 Amazon Way, Reno, NV 89501\n" +
		"weight: 1.5lbs\n" +
		"dims: 9x9x4 in\n"

	labels, errs := ParseFile("sample.labels", []byte(src), false)
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
	if errs[0].Pos != (Position{Line: 7, Col: 5}) {
		t.Errorf("expected address error at {7 5}, got %+v", errs[0].Pos)
	}
	if errs[1].Pos != (Position{Line: 9, Col: 9}) {
		t.Errorf("expected weight error at {9 9}, got %+v", errs[1].Pos)
	}
}
