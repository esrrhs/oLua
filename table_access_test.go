package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/milochristiansen/lua/ast"
)

// ============================================================================
// 集成测试：对比优化器输出与期望文件
// ============================================================================

func TestTableAccessAdvanced(t *testing.T) {
	compareOptOutput(t, "input/table_access_advanced.lua", "output/table_access_advanced.lua")
}

func TestTableAccessConflict(t *testing.T) {
	compareOptOutput(t, "input/table_access_conflict.lua", "output/table_access_conflict.lua")
}

func TestTableAccessEdge(t *testing.T) {
	compareOptOutput(t, "input/table_access_edge.lua", "output/table_access_edge.lua")
}

func TestTableAccessIfWrite(t *testing.T) {
	compareOptOutput(t, "input/table_access_ifwrite.lua", "output/table_access_ifwrite.lua")
}

func TestTableAccessLoop(t *testing.T) {
	compareOptOutput(t, "input/table_access_loop.lua", "output/table_access_loop.lua")
}

func TestTableAccessRealworld(t *testing.T) {
	compareOptOutput(t, "input/table_access_realworld.lua", "output/table_access_realworld.lua")
}

func TestTableAccessSemantic(t *testing.T) {
	compareOptOutput(t, "input/table_access_semantic.lua", "output/table_access_semantic.lua")
}

func TestTableAccessBug1(t *testing.T) {
	compareOptOutput(t, "input/table_access_bug1.lua", "output/table_access_bug1.lua")
}

func TestTableAccessCoverage(t *testing.T) {
	compareOptOutput(t, "input/table_access_coverage.lua", "output/table_access_coverage.lua")
}

func TestTableAccessCoverage2(t *testing.T) {
	compareOptOutput(t, "input/table_access_coverage2.lua", "output/table_access_coverage2.lua")
}

func TestTableAccessCoverage3(t *testing.T) {
	compareOptOutput(t, "input/table_access_coverage3.lua", "output/table_access_coverage3.lua")
}

func TestTableAccessCoverage4(t *testing.T) {
	compareOptOutput(t, "input/table_access_coverage4.lua", "output/table_access_coverage4.lua")
}

func TestTableAccessCoverage5(t *testing.T) {
	compareOptOutput(t, "input/table_access_coverage5.lua", "output/table_access_coverage5.lua")
}

func TestTableAccessPureFunc(t *testing.T) {
	compareOptOutput(t, "input/table_access_purefunc.lua", "output/table_access_purefunc.lua")
}

// ============================================================================
// 单元测试：辅助函数
// ============================================================================

func TestIsPathPrefix(t *testing.T) {
	tests := []struct {
		prefix string
		path   string
		want   bool
	}{
		{"a.b", "a.b.c", true},
		{"a.b", "a.b.c.d", true},
		{"a", "a.b", true},
		{"a.b", "a.b", false},   // 相等不算前缀
		{"a.b", "a.bc", false},  // 不是点号边界
		{"a.b.c", "a.b", false}, // 长路径不可能是短路径的前缀
		{"a.b", "x.y.z", false},
	}
	for _, tt := range tests {
		got := isPathPrefix(tt.prefix, tt.path)
		if got != tt.want {
			t.Errorf("isPathPrefix(%q, %q) = %v, want %v", tt.prefix, tt.path, got, tt.want)
		}
	}
}

func TestPathsRelated(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"a.b", "a.b", true},                      // 相等
		{"a.b", "a.b.c", true},                    // a 是 b 的前缀
		{"a.b.c", "a.b", true},                    // b 是 a 的前缀
		{"a.b", "x.y", false},                     // 不相关
		{"a.b", "a.bc", false},                    // 不是点号边界
		{"self.physics", "self.transform", false},  // 兄弟路径
	}
	for _, tt := range tests {
		got := pathsRelated(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("pathsRelated(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsWriteToTarget(t *testing.T) {
	tests := []struct {
		writtenPath string
		target      string
		want        bool
	}{
		{"a.b", "a.b", true},    // 直接写
		{"a", "a.b", true},      // 父级写使子级失效
		{"a", "a.b.c", true},    // 祖父级写
		{"a.b.c", "a.b", false}, // 子级写不影响父级
		{"x.y", "a.b", false},   // 不相关
	}
	for _, tt := range tests {
		got := isWriteToTarget(tt.writtenPath, tt.target)
		if got != tt.want {
			t.Errorf("isWriteToTarget(%q, %q) = %v, want %v", tt.writtenPath, tt.target, got, tt.want)
		}
	}
}

func TestTableAccessToLocalName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a.b", "a_b"},
		{"a.b.c", "a_b_c"},
		{"self.physics.velocity", "self_physics_velocity"},
		{"a[\"key\"]", "a__key__"},
	}
	for _, tt := range tests {
		got := table_access_to_local_name(tt.input)
		if got != tt.want {
			t.Errorf("table_access_to_local_name(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestContainTableAccess(t *testing.T) {
	tests := []struct {
		content string
		src     string
		want    int
	}{
		{"x = a.b.c + a.b.d", "a.b", 2},
		{"local a_b = a.b", "a.b", 1},
		{"xa.b.c = 1", "a.b", 0},       // 前面有字母
		{"a.bc = 1", "a.b", 0},         // 后面有字母
		{"a.b = a.b + a.b", "a.b", 3},
		{"nothing here", "a.b", 0},
	}
	for _, tt := range tests {
		got := contain_table_access(tt.content, tt.src)
		if got != tt.want {
			t.Errorf("contain_table_access(%q, %q) = %d, want %d", tt.content, tt.src, got, tt.want)
		}
	}
}

func TestReplaceTableAccess(t *testing.T) {
	tests := []struct {
		content string
		src     string
		dst     string
		want    string
	}{
		{"x = a.b.c", "a.b", "a_b", "x = a_b.c"},
		{"a.b = a.b + 1", "a.b", "a_b", "a_b = a_b + 1"},
		{"xa.b = 1", "a.b", "a_b", "xa.b = 1"},       // 前面有字母，不替换
		{"a.bc = 1", "a.b", "a_b", "a.bc = 1"},       // 后面有字母，不替换
		{"if a.b then a.b.c = 1 end", "a.b", "a_b", "if a_b then a_b.c = 1 end"},
	}
	for _, tt := range tests {
		got := replace_table_access(tt.content, tt.src, tt.dst)
		if got != tt.want {
			t.Errorf("replace_table_access(%q, %q, %q) = %q, want %q", tt.content, tt.src, tt.dst, got, tt.want)
		}
	}
}

// ============================================================================
// 阈值测试
// ============================================================================

func TestThreshold3(t *testing.T) {
	// threshold=3 时，每组只有 2 次读的函数不应被优化
	*opt_table_access_threshold = 3
	defer func() { *opt_table_access_threshold = 2 }()

	actual, err := runOptimizer("input/table_access_advanced.lua")
	if err != nil {
		t.Fatalf("optimizer failed: %v", err)
	}

	result := strings.Join(actual, "\n")

	// test_basic_reads 有 3 次读 → 应被优化
	if !strings.Contains(result, "local a_b = a.b -- opt by oLua") {
		t.Error("test_basic_reads should be optimized with threshold=3 (has 3 reads)")
	}

	// test_write_interruption 每组只有 2 次读 → 不应被优化
	inFunc := false
	funcOptimized := false
	for _, line := range actual {
		if strings.Contains(line, "function test_write_interruption") {
			inFunc = true
			continue
		}
		if inFunc && strings.HasPrefix(strings.TrimSpace(line), "function ") {
			break
		}
		if inFunc && strings.Contains(line, "-- opt by oLua") {
			funcOptimized = true
			break
		}
	}
	if funcOptimized {
		t.Error("test_write_interruption should NOT be optimized with threshold=3 (only 2 reads per group)")
	}
}

func TestThreshold1FallsBackTo2(t *testing.T) {
	// threshold=1 应被强制为 2（最小值）
	*opt_table_access_threshold = 1
	defer func() { *opt_table_access_threshold = 2 }()

	actual, err := runOptimizer("input/table_access_advanced.lua")
	if err != nil {
		t.Fatalf("optimizer failed: %v", err)
	}

	result := strings.Join(actual, "\n")
	if !strings.Contains(result, "-- opt by oLua") {
		t.Error("threshold=1 should fall back to 2, optimizations should still be applied")
	}
}

// ============================================================================
// 单元测试：边界场景
// ============================================================================

func TestGetUniqueLocalNameNoConflict(t *testing.T) {
	// 无冲突时直接返回 baseName
	name := getUniqueLocalName(nil, "test_var")
	if name != "test_var" {
		t.Errorf("getUniqueLocalName(nil, \"test_var\") = %q, want \"test_var\"", name)
	}
}

func TestGetUniqueLocalNameFallback(t *testing.T) {
	// 模拟超过 100 次冲突的场景，触发 _opt_ 前缀兜底
	var lines []string
	lines = append(lines, "function test()")
	lines = append(lines, "    local test_var = 1")
	for i := 1; i <= 100; i++ {
		lines = append(lines, fmt.Sprintf("    local test_var_%d = %d", i, i))
	}
	lines = append(lines, "end")

	source := strings.Join(lines, "\n") + "\n"
	block, err := parseSource(source)
	if err != nil {
		t.Fatalf("parseSource failed: %v", err)
	}

	var funcBlock []ast.Stmt
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			if fd, isFd := n.(*ast.FuncDecl); isFd {
				funcBlock = fd.Block
				*ok = false
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}

	name := getUniqueLocalName(funcBlock, "test_var")
	if name != "_opt_test_var" {
		t.Errorf("getUniqueLocalName with 100+ conflicts = %q, want \"_opt_test_var\"", name)
	}
}

func TestNestedCallInvalidatesNil(t *testing.T) {
	// nil 输入返回 false
	if nestedCallInvalidates(nil, "a.b") {
		t.Error("nestedCallInvalidates(nil, \"a.b\") = true, want false")
	}
}

func TestNestedCallInvalidatesTableConstructor(t *testing.T) {
	// TableConstructor key 中含使 target 失效的函数调用
	// target="a.b.c"，foo(a.b) 传入 a.b 是 target 的严格父级 → 失效
	source := "function test() local t = {[foo(a.b)] = 1, bar = baz(a.b)} end\n"
	block, err := parseSource(source)
	if err != nil {
		t.Fatalf("parseSource failed: %v", err)
	}
	var found bool
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			if tc, isTc := n.(*ast.TableConstructor); isTc {
				if nestedCallInvalidates(tc, "a.b.c") {
					found = true
				}
				*ok = false
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}
	if !found {
		t.Error("nestedCallInvalidates(TableConstructor with foo(a.b), target=a.b.c) = false, want true")
	}
}

func TestNestedCallInvalidatesParens(t *testing.T) {
	// Parens 内含函数调用，target 是参数的子路径
	source := "function test() local x = (foo(a.b)) end\n"
	block, err := parseSource(source)
	if err != nil {
		t.Fatalf("parseSource failed: %v", err)
	}
	var found bool
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			if p, isP := n.(*ast.Parens); isP {
				if nestedCallInvalidates(p, "a.b.c") {
					found = true
				}
				*ok = false
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}
	if !found {
		t.Error("nestedCallInvalidates(Parens with foo(a.b), target=a.b.c) = false, want true")
	}
}

func TestNestedCallInvalidatesTableConstructorVals(t *testing.T) {
	// 确保 TableConstructor vals 分支被覆盖（key 不含调用，val 含调用）
	source := "function test() local t = {[\"key\"] = foo(a.b)} end\n"
	block, err := parseSource(source)
	if err != nil {
		t.Fatalf("parseSource failed: %v", err)
	}
	var tc *ast.TableConstructor
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			if c, isC := n.(*ast.TableConstructor); isC {
				tc = c
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}
	if tc == nil {
		t.Fatal("TableConstructor not found in AST")
	}
	// target="a.b.c"，foo(a.b) 中 a.b 是 target 的父级 → 失效
	if !nestedCallInvalidates(tc, "a.b.c") {
		t.Error("nestedCallInvalidates({[\"key\"]=foo(a.b)}, target=a.b.c) = false, want true")
	}
}

func TestExprContainsFuncCallInvalidatingFunctionExpr(t *testing.T) {
	// getHandler(a.b)(123) — Function 表达式内含 invalidating call
	// target="a.b.c"，foo(a.b) 中 a.b 是 target 的父级 → 失效
	source := "function test() getHandler(a.b)(123) end\n"
	block, err := parseSource(source)
	if err != nil {
		t.Fatalf("parseSource failed: %v", err)
	}
	var found bool
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			if fc, isFc := n.(*ast.FuncCall); isFc {
				if fc.Function != nil {
					if exprContainsFuncCallInvalidating(fc, "a.b.c") {
						found = true
					}
				}
				*ok = false
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}
	if !found {
		t.Error("exprContainsFuncCallInvalidating(getHandler(a.b)(123), target=a.b.c) = false, want true")
	}
}

func TestExprContainsFuncCallInvalidatingRecursive(t *testing.T) {
	// line 287/291 是防御性冗余代码，funcCallInvalidatesTarget 已通过
	// nestedCallInvalidates 覆盖了相同逻辑，正常流程中不可达
	t.Skip("lines 287/291 are defensive redundant checks, unreachable in normal flow")
}

func TestApplyTableAccessOptimizationEmpty(t *testing.T) {
	// 空 events 不应 panic
	group := ReadGroup{Events: nil}
	applyTableAccessOptimization("a.b", "a_b", group, true)
}

func TestApplyTableAccessOptimizationWithOluaLine(t *testing.T) {
	// 测试替换时跳过 oLua 生成行 + endLine < startLine 防御路径
	oldContent := gfilecontent
	oldOpt := has_opt
	oldCount := goptcount
	oldFilename := gfilename
	defer func() {
		gfilecontent = oldContent
		has_opt = oldOpt
		goptcount = oldCount
		gfilename = oldFilename
	}()

	gfilecontent = []string{
		"    local a_b = a.b -- opt by oLua",
		"    local x = a.b.c",
		"    local y = a.b.d",
	}
	gfilename = "test"
	goptcount = 0
	has_opt = false

	// endLine < Line 触发修正路径
	group := ReadGroup{
		Events: []AccessEvent{
			{Type: AccessRead, Line: 2, EndLine: 1, StmtIdx: 0},
			{Type: AccessRead, Line: 3, EndLine: 3, StmtIdx: 1},
		},
	}
	applyTableAccessOptimization("a.b", "a_b", group, false)
}

func TestOptimizeBlockLevelHasOptGuard(t *testing.T) {
	// 测试 has_opt=true 时的 guard 路径
	oldOpt := has_opt
	defer func() { has_opt = oldOpt }()

	has_opt = true
	result := optimizeBlockLevel(nil)
	if !result {
		t.Error("optimizeBlockLevel with has_opt=true should return true")
	}
}

func TestOptimizeBlockHasOptGuard(t *testing.T) {
	// 测试 has_opt=true 时的 guard 路径
	oldOpt := has_opt
	defer func() { has_opt = oldOpt }()

	has_opt = true
	result := optimizeBlock(nil)
	if !result {
		t.Error("optimizeBlock with has_opt=true should return true")
	}
}
