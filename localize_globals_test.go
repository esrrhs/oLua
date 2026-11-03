package main

import (
	"strings"
	"testing"

	"github.com/milochristiansen/lua/ast"
)

// compareLocalizeOptOutput runs the full opt_lua loop with only localize_globals enabled.
func compareLocalizeOptOutput(t *testing.T, inputFile, expectedFile string) {
	t.Helper()
	withPassFlags(t, func() {
		*opt_localize_globals = true
		*opt_localize_globals_threshold = 2
	}, func() {
		comparePassOutput(t, inputFile, expectedFile)
	})
}

func TestLocalizeGlobalsBasic(t *testing.T) {
	compareLocalizeOptOutput(t,
		"input/localize_globals_basic.lua",
		"output/localize_globals_basic.lua")
}

func TestLocalizeGlobalsSkip(t *testing.T) {
	compareLocalizeOptOutput(t,
		"input/localize_globals_skip.lua",
		"output/localize_globals_skip.lua")
}

func TestLocalizeGlobalsMath(t *testing.T) {
	compareLocalizeOptOutput(t,
		"input/localize_globals_math.lua",
		"output/localize_globals_math.lua")
}

func TestLocalizeGlobalsThreshold(t *testing.T) {
	withPassFlags(t, func() {
		*opt_localize_globals = true
		*opt_localize_globals_threshold = 3
	}, func() {
		gfilecontent = []string{
			"function f()",
			"    print(1)",
			"    print(2)",
			"end",
		}
		gfilename = "threshold.lua"
		has_opt = false
		goptcount = 0

		if err := runOptLuaIterations(); err != nil {
			t.Fatalf("parse: %v", err)
		}

		for _, line := range gfilecontent {
			if strings.Contains(line, "-- opt by oLua") {
				t.Fatalf("threshold=3 should not optimize 2 prints, got %q", line)
			}
		}
	})
}

func TestLocalizeGlobalsHasOptGuard(t *testing.T) {
	old := has_opt
	defer func() { has_opt = old }()
	has_opt = true

	src := "function f()\n    print(1)\n    print(2)\nend\n"
	block, err := parseSource(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var fd *ast.FuncDecl
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n == nil {
			return
		}
		if d, is := n.(*ast.FuncDecl); is && fd == nil {
			fd = d
		}
	}}
	for _, s := range block {
		ast.Walk(&f, s)
	}
	if fd == nil {
		t.Fatal("no FuncDecl")
	}
	before := append([]string{}, gfilecontent...)
	gfilecontent = strings.Split(strings.TrimRight(src, "\n"), "\n")
	opt_func_localize_globals(fd)
	if has_opt != true {
		t.Error("has_opt should remain true")
	}
	_ = before
}
