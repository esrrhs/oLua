package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptPath(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "fold.lua")
	skipPath := filepath.Join(dir, "oLua-leftover.olua.tmp")
	nested := filepath.Join(dir, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	nestedPath := filepath.Join(nested, "nested.lua")

	original := []string{
		"function f()",
		"    local x = 1 + 2",
		"end",
	}
	unchanged := []string{
		"function g()",
		"    local y = x + z",
		"end",
	}
	if err := os.WriteFile(srcPath, []byte(strings.Join(original, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nestedPath, []byte(strings.Join(unchanged, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A stray temp name that must be ignored by the walk filter.
	if err := os.WriteFile(skipPath, []byte("should not be read as lua\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	withPassFlags(t, func() {
		*opt_const_fold = true
	}, func() {
		opt_path(dir)
	})

	got, err := readFileLines(srcPath)
	if err != nil {
		t.Fatalf("read optimized: %v", err)
	}
	if len(got) < 2 || !strings.Contains(got[1], "3") || !strings.Contains(got[1], "-- opt by oLua") {
		t.Fatalf("expected in-place const fold, got %#v", got)
	}

	gotNested, err := readFileLines(nestedPath)
	if err != nil {
		t.Fatalf("read nested: %v", err)
	}
	assertLinesEqual(t, nestedPath, gotNested, unchanged)

	// No optimizer temp files should remain after a successful run.
	leftovers, err := filepath.Glob(filepath.Join(dir, "*.olua.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	// The intentional skipPath fixture is allowed; any oLua-*.olua.tmp from CreateTemp is not.
	for _, p := range leftovers {
		base := filepath.Base(p)
		if strings.HasPrefix(base, "oLua-") && base != filepath.Base(skipPath) {
			t.Fatalf("temp file left behind: %s", p)
		}
	}
	// Nested temps
	nestedLeftovers, _ := filepath.Glob(filepath.Join(nested, "oLua-*.olua.tmp"))
	if len(nestedLeftovers) != 0 {
		t.Fatalf("nested temp left behind: %v", nestedLeftovers)
	}

	// Fixture temp must still exist (walk skipped it; we did not delete it).
	if _, err := os.Stat(skipPath); err != nil {
		t.Fatalf("walk should skip .olua.tmp without deleting it: %v", err)
	}
}

func TestOptPathSkipsTempNames(t *testing.T) {
	dir := t.TempDir()
	// Ends with .lua but also contains .olua.tmp marker in the basename filter path:
	// Create a file that looks like a temp and must not be processed.
	tmpLookalike := filepath.Join(dir, "x.olua.tmp")
	if err := os.WriteFile(tmpLookalike, []byte("function f()\n    local x = 1 + 2\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withPassFlags(t, func() {
		*opt_const_fold = true
	}, func() {
		opt_path(dir)
	})
	got, err := readFileLines(tmpLookalike)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range got {
		if strings.Contains(line, "-- opt by oLua") {
			t.Fatalf("temp-named file was incorrectly optimized: %#v", got)
		}
	}
}
