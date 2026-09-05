package main

import (
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
