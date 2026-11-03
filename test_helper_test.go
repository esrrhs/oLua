package main

import (
	"bufio"
	"os"
	"testing"

	"github.com/milochristiansen/lua/ast"
)

// ============================================================================
// 测试公共工具函数
// 放在独立文件中，所有 *_test.go 文件都可以使用。
// ============================================================================

// TestMain 设置测试默认值。
// 注意：整个包只能有一个 TestMain，放在这里统一管理。
func TestMain(m *testing.M) {
	*opt_table_access_threshold = 2
	os.Exit(m.Run())
}

// readFileLines 按行读取文件内容（与优化器的 read_file 行为一致）。
func readFileLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, nil
}

// parseSource 解析 Lua 源码为 AST（不 fatal，返回 error）。
func parseSource(source string) ([]ast.Stmt, error) {
	block, err := ast.Parse(source, 1)
	return block, err
}

// passFlagSnapshot holds all optimizer pass toggles/thresholds for save/restore.
type passFlagSnapshot struct {
	tableAccess            bool
	tableAccessThreshold   int
	tableAccessGlobal      bool
	tableConstructor       bool
	constFold              bool
	localizeGlobals        bool
	localizeGlobalsThresh  int
	licm                   bool
	licmThreshold          int
}

func snapshotPassFlags() passFlagSnapshot {
	return passFlagSnapshot{
		tableAccess:           *opt_table_access,
		tableAccessThreshold:  *opt_table_access_threshold,
		tableAccessGlobal:     *opt_table_access_global,
		tableConstructor:      *opt_table_constructor,
		constFold:             *opt_const_fold,
		localizeGlobals:       *opt_localize_globals,
		localizeGlobalsThresh: *opt_localize_globals_threshold,
		licm:                  *opt_licm,
		licmThreshold:         *opt_licm_threshold,
	}
}

func restorePassFlags(s passFlagSnapshot) {
	*opt_table_access = s.tableAccess
	*opt_table_access_threshold = s.tableAccessThreshold
	*opt_table_access_global = s.tableAccessGlobal
	*opt_table_constructor = s.tableConstructor
	*opt_const_fold = s.constFold
	*opt_localize_globals = s.localizeGlobals
	*opt_localize_globals_threshold = s.localizeGlobalsThresh
	*opt_licm = s.licm
	*opt_licm_threshold = s.licmThreshold
}

func disableAllPasses() {
	*opt_table_access = false
	*opt_table_constructor = false
	*opt_const_fold = false
	*opt_localize_globals = false
	*opt_licm = false
}

// withPassFlags saves all pass flags, disables every pass, runs enable to turn on
// the desired pass(es), then runs body, and always restores the previous flags.
func withPassFlags(t *testing.T, enable func(), body func()) {
	t.Helper()
	snap := snapshotPassFlags()
	disableAllPasses()
	enable()
	defer restorePassFlags(snap)
	body()
}

// runOptLuaIterations runs the shared iterative opt_lua loop on gfilecontent.
func runOptLuaIterations() error {
	has_opt = true
	for has_opt {
		has_opt = false
		source := ""
		for _, line := range gfilecontent {
			source += line + "\n"
		}
		block, err := parseSource(source)
		if err != nil {
			return err
		}
		gblock = block
		opt_lua()
	}
	return nil
}

// assertLinesEqual compares actual vs expected line slices with helpful diffs.
func assertLinesEqual(t *testing.T, label string, actual, expected []string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Errorf("line count mismatch for %s: got %d, want %d", label, len(actual), len(expected))
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
		if len(actual) > minLen {
			t.Errorf("  extra line %d: %q", minLen+1, actual[minLen])
		} else if len(expected) > minLen {
			t.Errorf("  missing line %d: %q", minLen+1, expected[minLen])
		}
		if len(actual) > len(expected) {
			for i := len(expected); i < len(actual) && i < len(expected)+5; i++ {
				t.Errorf("  extra: %q", actual[i])
			}
		} else if len(expected) > len(actual) {
			for i := len(actual); i < len(expected) && i < len(actual)+5; i++ {
				t.Errorf("  missing: %q", expected[i])
			}
		}
		return
	}
	for i := range actual {
		if actual[i] != expected[i] {
			t.Errorf("line %d mismatch for %s:\n  got:  %q\n  want: %q", i+1, label, actual[i], expected[i])
		}
	}
}

// comparePassOutput runs the full opt_lua loop (flags already set) and compares
// the result to an expected output file.
func comparePassOutput(t *testing.T, inputFile, expectedFile string) {
	t.Helper()

	has_opt = false
	goptcount = 0
	gfilename = inputFile

	lines, err := readFileLines(inputFile)
	if err != nil {
		t.Fatalf("read %s: %v", inputFile, err)
	}
	gfilecontent = lines

	if err := runOptLuaIterations(); err != nil {
		t.Fatalf("parse/opt %s: %v", inputFile, err)
	}

	expected, err := readFileLines(expectedFile)
	if err != nil {
		t.Fatalf("read expected %s: %v", expectedFile, err)
	}
	assertLinesEqual(t, inputFile, gfilecontent, expected)
}

// runOptimizer 对输入文件执行表访问优化并返回结果行。
// 模拟 main.go 中 opt() 的迭代逻辑。
func runOptimizer(inputFile string) ([]string, error) {
	// 重置全局状态
	has_opt = false
	goptcount = 0
	gfilename = inputFile

	// 读取输入
	lines, err := readFileLines(inputFile)
	if err != nil {
		return nil, err
	}
	gfilecontent = lines

	// 迭代优化
	has_opt = true
	for has_opt {
		has_opt = false
		source := ""
		for _, line := range gfilecontent {
			source += line + "\n"
		}
		block, err := parseSource(source)
		if err != nil {
			return nil, err
		}
		gblock = block
		optTableAccess()
	}

	return gfilecontent, nil
}

// optTableAccess 执行一轮表访问优化遍历。
func optTableAccess() {
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if has_opt {
			*ok = false
			return
		}
		if n != nil {
			switch n.(type) {
			case *ast.FuncDecl:
				func_decl := n.(*ast.FuncDecl)
				opt_func_table_access(func_decl)
			}
		}
	}}
	for _, stmt := range gblock {
		ast.Walk(&f, stmt)
	}
}

// compareOptOutput 运行优化器并逐行对比结果与期望输出文件。
// 适用于表访问优化 pass 的集成测试（直接调用 opt_func_table_access）。
func compareOptOutput(t *testing.T, inputFile, expectedFile string) {
	t.Helper()

	*opt_table_access_threshold = 2

	actual, err := runOptimizer(inputFile)
	if err != nil {
		t.Fatalf("optimizer failed on %s: %v", inputFile, err)
	}

	expected, err := readFileLines(expectedFile)
	if err != nil {
		t.Fatalf("failed to read expected output %s: %v", expectedFile, err)
	}
	assertLinesEqual(t, inputFile, actual, expected)
}
