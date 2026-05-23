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
// 适用于所有优化 pass 的集成测试。
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

	if len(actual) != len(expected) {
		t.Errorf("line count mismatch for %s: got %d, want %d", inputFile, len(actual), len(expected))
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
		return
	}

	for i := range actual {
		if actual[i] != expected[i] {
			t.Errorf("line %d mismatch for %s:\n  got:  %q\n  want: %q", i+1, inputFile, actual[i], expected[i])
		}
	}
}
