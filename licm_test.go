package main

import (
	"strings"
	"testing"

	"github.com/milochristiansen/lua/ast"
)

func compareLicmOptOutput(t *testing.T, inputFile, expectedFile string) {
	t.Helper()
	withPassFlags(t, func() {
		*opt_licm = true
		*opt_licm_threshold = 2
	}, func() {
		comparePassOutput(t, inputFile, expectedFile)
	})
}

func TestLicmBasic(t *testing.T) {
	compareLicmOptOutput(t, "input/licm_basic.lua", "output/licm_basic.lua")
}

func TestLicmSkip(t *testing.T) {
	compareLicmOptOutput(t, "input/licm_skip.lua", "output/licm_skip.lua")
}

func TestLicmThresholdClamp(t *testing.T) {
	withPassFlags(t, func() {
		*opt_licm = true
		*opt_licm_threshold = 1 // should clamp to 2
	}, func() {
		gfilecontent = []string{
			"function f()",
			"    for i = 1, 10 do",
			"        local x = a.b.c + i",
			"    end",
			"end",
		}
		gfilename = "licm_threshold.lua"
		goptcount = 0
		if err := runOptLuaIterations(); err != nil {
			t.Fatalf("parse: %v", err)
		}

		for _, line := range gfilecontent {
			if strings.Contains(line, "-- opt by oLua") {
				t.Fatalf("threshold clamp to 2 should not hoist single read, got %q", line)
			}
		}
	})
}

func TestLicmHasOptGuard(t *testing.T) {
	old := has_opt
	defer func() { has_opt = old }()
	has_opt = true

	src := "function f()\n    for i = 1, 10 do\n        local x = a.b.c\n        local y = a.b.d\n    end\nend\n"
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
	opt_func_licm(fd)
	if has_opt != true {
		t.Error("has_opt should remain true")
	}
}
