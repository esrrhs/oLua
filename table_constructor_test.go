package main

import (
	"testing"
)

// compareConstructorOptOutput 运行完整优化流程（只启用 constructor pass）并对比结果。
func compareConstructorOptOutput(t *testing.T, inputFile, expectedFile string) {
	t.Helper()
	withPassFlags(t, func() {
		*opt_table_constructor = true
	}, func() {
		comparePassOutput(t, inputFile, expectedFile)
	})
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
