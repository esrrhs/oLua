package main

import (
	"testing"
)

// compareConstructorOptOutput 运行完整优化流程（只启用 constructor pass）并对比结果。
func compareConstructorOptOutput(t *testing.T, inputFile, expectedFile string) {
	t.Helper()

	oldConstructor := *opt_table_constructor
	oldAccess := *opt_table_access
	*opt_table_constructor = true
	*opt_table_access = false
	defer func() {
		*opt_table_constructor = oldConstructor
		*opt_table_access = oldAccess
	}()

	has_opt = false
	goptcount = 0
	gfilename = inputFile

	lines, err := readFileLines(inputFile)
	if err != nil {
		t.Fatalf("read %s: %v", inputFile, err)
	}
	gfilecontent = lines

	has_opt = true
	for has_opt {
		has_opt = false
		source := ""
		for _, line := range gfilecontent {
			source += line + "\n"
		}
		block, err := parseSource(source)
		if err != nil {
			t.Fatalf("parse %s: %v", inputFile, err)
		}
		gblock = block
		opt_lua()
	}

	actual := gfilecontent
	expected, err := readFileLines(expectedFile)
	if err != nil {
		t.Fatalf("read expected %s: %v", expectedFile, err)
	}

	if len(actual) != len(expected) {
		t.Errorf("line count mismatch: got %d want %d", len(actual), len(expected))
		minLen := len(actual)
		if len(expected) < minLen {
			minLen = len(expected)
		}
		for i := 0; i < minLen; i++ {
			if actual[i] != expected[i] {
				t.Errorf("  first diff at line %d:\n    got:  %q\n    want: %q", i+1, actual[i], expected[i])
				break
			}
		}
		return
	}

	for i := range actual {
		if actual[i] != expected[i] {
			t.Errorf("line %d:\n  got:  %q\n  want: %q", i+1, actual[i], expected[i])
		}
	}
}

func TestTableConstructorBasic(t *testing.T) {
	compareConstructorOptOutput(t,
		"input/table_constructor.lua",
		"output/table_constructor.lua")
}

func TestTableConstructorAdvanced(t *testing.T) {
	compareConstructorOptOutput(t,
		"input/table_constructor_advanced.lua",
		"output/table_constructor_advanced.lua")
}

func TestTableConstructorEdge(t *testing.T) {
	compareConstructorOptOutput(t,
		"input/table_constructor_edge.lua",
		"output/table_constructor_edge.lua")
}
