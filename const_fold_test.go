package main

import (
	"strings"
	"testing"

	"github.com/milochristiansen/lua/ast"
)

func compareConstFoldOptOutput(t *testing.T, inputFile, expectedFile string) {
	t.Helper()
	withPassFlags(t, func() {
		*opt_const_fold = true
	}, func() {
		comparePassOutput(t, inputFile, expectedFile)
	})
}

func TestConstFoldBasic(t *testing.T) {
	compareConstFoldOptOutput(t, "input/const_fold_basic.lua", "output/const_fold_basic.lua")
}

func TestConstFoldSkip(t *testing.T) {
	compareConstFoldOptOutput(t, "input/const_fold_skip.lua", "output/const_fold_skip.lua")
}

func TestConstFoldHasOptGuard(t *testing.T) {
	old := has_opt
	defer func() { has_opt = old }()
	has_opt = true

	src := "function f()\n    local x = 1 + 2\nend\n"
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
	gfilecontent = strings.Split(strings.TrimRight(src, "\n"), "\n")
	opt_func_const_fold(fd)
	if has_opt != true {
		t.Error("has_opt should remain true")
	}
}

func TestConstFoldDivZeroSkipped(t *testing.T) {
	withPassFlags(t, func() {
		*opt_const_fold = true
	}, func() {
		gfilecontent = []string{
			"function f()",
			"    local x = 1 / 0",
			"    local y = 2 + 2",
			"end",
		}
		gfilename = "div0.lua"
		goptcount = 0
		if err := runOptLuaIterations(); err != nil {
			t.Fatalf("parse: %v", err)
		}

		if strings.Contains(gfilecontent[1], "-- opt by oLua") {
			t.Fatalf("div0 must not fold, got %q", gfilecontent[1])
		}
		if !strings.Contains(gfilecontent[2], "4") {
			t.Fatalf("2+2 should fold, got %q", gfilecontent[2])
		}
	})
}
