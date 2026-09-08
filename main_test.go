package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCollectFilesSingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.labels")
	if err := os.WriteFile(path, []byte("to: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := collectFiles(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != path {
		t.Fatalf("expected [%s], got %v", path, got)
	}
}

func TestCollectFilesDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(rel, contents string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("a.labels", "to: x\n")
	writeFile("notes.txt", "not a batch file\n")
	writeFile("nested/b.labels", "to: y\n")
	writeFile(".hidden/c.labels", "to: z\n")

	got, err := collectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(got)

	want := []string{
		filepath.Join(dir, "a.labels"),
		filepath.Join(dir, "nested/b.labels"),
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected %v, got %v", want, got)
			break
		}
	}
}

func TestCollectFilesMissingPath(t *testing.T) {
	_, err := collectFiles(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a missing path")
	}
}

func TestNewFileResultClean(t *testing.T) {
	labels := []Label{{}, {}}
	r := newFileResult("clean.labels", labels, nil)

	if !r.OK {
		t.Error("expected OK to be true with no errors")
	}
	if r.Labels != 2 {
		t.Errorf("expected 2 labels, got %d", r.Labels)
	}
	if r.Errors == nil || len(r.Errors) != 0 {
		t.Errorf("expected an empty (not nil) error slice, got %v", r.Errors)
	}

	// An empty slice must marshal to [] rather than null, since a
	// consumer of --json shouldn't have to special-case a missing key.
	data, err := json.Marshal(r.Errors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("expected errors to marshal to [], got %s", data)
	}
}

func TestNewFileResultWithErrors(t *testing.T) {
	errs := []*LabelError{
		{
			Pos:        Position{Line: 7, Col: 5},
			SourceLine: "to: 221B Baker St, London",
			Message:    "bad address",
		},
	}
	r := newFileResult("bad.labels", nil, errs)

	if r.OK {
		t.Error("expected OK to be false with errors present")
	}
	if len(r.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(r.Errors))
	}

	got := r.Errors[0]
	want := jsonError{Line: 7, Col: 5, Message: "bad address", SourceLine: "to: 221B Baker St, London"}
	if got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}
