package main

import (
	"fmt"
	"github.com/milochristiansen/lua/ast"
	"log"
	"regexp"
	"sort"
	"strings"
)

// ============================================================================
// 辅助函数
// ============================================================================

// 内置纯函数白名单：这些函数保证不会修改传入的表参数。
var builtinPureFuncs = map[string]bool{
	// 输出/调试
	"print":    true,
	"error":    true,
	"warn":     true,
	"assert":   true,
	// 类型/元信息
	"type":     true,
	"typeof":   true,
	"tostring": true,
	"tonumber": true,
	"select":   true,
	// 迭代器（不修改表本身）
	"pairs":    true,
	"ipairs":   true,
	"next":     true,
	"unpack":   true,
	// 数学
	"math.abs":   true,
	"math.ceil":  true,
	"math.floor": true,
	"math.max":   true,
	"math.min":   true,
	"math.sqrt":  true,
	"math.sin":   true,
	"math.cos":   true,
	"math.tan":   true,
	"math.log":   true,
	"math.exp":   true,
	"math.random": true,
	// 字符串（不修改表）
	"string.format": true,
	"string.len":    true,
	"string.sub":    true,
	"string.find":   true,
	"string.match":  true,
	"string.gmatch": true,
	"string.rep":    true,
	"string.lower":  true,
	"string.upper":  true,
	"string.byte":   true,
	"string.char":   true,
	// 表查询（不修改）
	"table.concat": true,
	"#":            true,
	"rawget":       true,
	"rawlen":       true,
	"rawequal":     true,
	// OS/时间（只读）
	"os.clock": true,
	"os.time":  true,
	"os.date":  true,
}

// 用户自定义纯函数正则列表（在首次使用时编译）
var userPureFuncPatterns []*regexp.Regexp
var userPureFuncPatternsCompiled bool

// compilePureFuncPatterns 编译用户自定义的纯函数正则列表。
func compilePureFuncPatterns() {
	if userPureFuncPatternsCompiled {
		return
	}
	userPureFuncPatternsCompiled = true
	userPureFuncPatterns = nil

	if opt_table_access_pure_funcs == nil || *opt_table_access_pure_funcs == "" {
		return
	}
	parts := strings.Split(*opt_table_access_pure_funcs, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// 编译为完整匹配正则（自动加 ^...$）
		pattern := "^" + part + "$"
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("warning: invalid pure_funcs pattern %q: %v", part, err)
			continue
		}
		userPureFuncPatterns = append(userPureFuncPatterns, re)
	}
}

// isPureFunction 判断函数名是否在纯函数白名单中（不会修改参数）。
// funcName 是函数的完整路径（如 "print", "math.floor", "log_info"）。
func isPureFunction(funcName string) bool {
	// 检查内置白名单
	if builtinPureFuncs[funcName] {
		return true
	}

	// 检查用户自定义正则
	compilePureFuncPatterns()
	for _, re := range userPureFuncPatterns {
		if re.MatchString(funcName) {
			return true
		}
	}

	return false
}

// getFuncCallName 从函数调用中提取函数名。
// 返回函数名和是否成功。
// 支持：print(x) → "print", math.floor(x) → "math.floor", obj:method() → "method"
func getFuncCallName(call *ast.FuncCall) (string, bool) {
	if call.Receiver != nil {
		// 方法调用 a:method()，函数名是 method
		switch fn := call.Function.(type) {
		case *ast.ConstString:
			return fn.Value, true
		case *ast.ConstIdent:
			return fn.Value, true
		}
		return "", false
	}

	// 普通调用：提取 Function 表达式的路径
	path, ok := getExprPath(call.Function)
	if ok {
		return path, true
	}

	// Function 是 ConstIdent（简单函数名）
	switch fn := call.Function.(type) {
	case *ast.ConstIdent:
		return fn.Value, true
	case *ast.ConstString:
		return fn.Value, true
	}

	return "", false
}

// getExprPath 从 TableAccessor 链中提取点分路径。
// 对有效路径返回 ("a.b.c", true)，动态 key 返回 ("", false)。
// 支持点号访问 (a.b) 和常量字符串 key (a["key"])。
func getExprPath(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.ConstIdent:
		return e.Value, true
	case *ast.TableAccessor:
		objPath, ok := getExprPath(e.Obj)
		if !ok {
			return "", false
		}
		switch key := e.Key.(type) {
		case *ast.ConstString:
			return objPath + "." + key.Value, true
		case *ast.ConstIdent:
			// 点号访问 a.b 中 b 是 key 名
			return objPath + "." + key.Value, true
		default:
			// 动态 key 如 a[i]，不支持
			return "", false
		}
	default:
		return "", false
	}
}

// isPathPrefix 判断 prefix 是否是 path 的点分前缀。
// 例如 "a.b" 是 "a.b.c" 的前缀，但不是 "a.bc" 的前缀。
func isPathPrefix(prefix, path string) bool {
	if len(prefix) >= len(path) {
		return false
	}
	return strings.HasPrefix(path, prefix) && path[len(prefix)] == '.'
}

// pathsRelated 判断两个路径是否相关（一个是另一个的前缀，或相等）。
func pathsRelated(a, b string) bool {
	return a == b || isPathPrefix(a, b) || isPathPrefix(b, a)
}

// isWriteToTarget 判断对 writtenPath 的写操作是否会使 target 的缓存失效。
// 写 "a" 会使 "a.b.c" 失效（父级写）。
// 写 "a.b.c" 会使 "a.b.c" 失效（直接写）。
// 注意：写 "a.b.c.d" 不会使 "a.b.c" 失效（子级写不影响父级）。
func isWriteToTarget(writtenPath, target string) bool {
	return writtenPath == target || isPathPrefix(writtenPath, target)
}

// table_access_to_local_name 将表路径转换为合法的 local 变量名。
func table_access_to_local_name(name string) string {
	ret := strings.ReplaceAll(name, ".", "_")
	ret = strings.ReplaceAll(ret, ":", "_")
	ret = strings.ReplaceAll(ret, "[", "_")
	ret = strings.ReplaceAll(ret, "]", "_")
	ret = strings.ReplaceAll(ret, "\"", "_")
	ret = strings.ReplaceAll(ret, "'", "_")
	ret = strings.ReplaceAll(ret, "(", "_")
	ret = strings.ReplaceAll(ret, ")", "_")
	ret = strings.ReplaceAll(ret, "/", "_")
	return ret
}

// getUniqueLocalName 生成不与现有标识符冲突的 local 变量名。
// 如果 "a_b" 冲突，依次尝试 "a_b_1", "a_b_2" 等。
func getUniqueLocalName(block []ast.Stmt, baseName string) string {
	usedNames := collectIdentifiers(block)
	if !usedNames[baseName] {
		return baseName
	}
	// 尝试加后缀
	for i := 1; i <= 100; i++ {
		candidate := fmt.Sprintf("%s_%d", baseName, i)
		if !usedNames[candidate] {
			return candidate
		}
	}
	// 兜底：使用 _opt_ 前缀
	return "_opt_" + baseName
}

// isOluaGeneratedName 判断某个变量名是否由 oLua 在之前的 pass 中生成。
// 通过扫描 gfilecontent 查找 "local <name> = ... -- opt by oLua" 形式的行。
func isOluaGeneratedName(name string) bool {
	for _, line := range gfilecontent {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "-- opt by oLua") {
			if strings.HasPrefix(trimmed, "local "+name+" = ") || strings.HasPrefix(trimmed, name+" = ") {
				return true
			}
		}
	}
	return false
}

// collectIdentifiers 收集代码块中所有使用的标识符名称（用于冲突检测）。
func collectIdentifiers(block []ast.Stmt) map[string]bool {
	names := make(map[string]bool)
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			switch e := n.(type) {
			case *ast.ConstIdent:
				names[e.Value] = true
			case *ast.Assign:
				if e.LocalDecl {
					for _, t := range e.Targets {
						if ident, isIdent := t.(*ast.ConstIdent); isIdent {
							names[ident.Value] = true
						}
					}
				}
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}
	return names
}

// ============================================================================
// 读写分析
// ============================================================================

type AccessType int

const (
	AccessRead  AccessType = iota
	AccessWrite AccessType = iota
)

// AccessEvent 表示代码块中对某个表路径的一次读或写事件。
type AccessEvent struct {
	Type    AccessType
	Line    int // 语句起始行号（1-based）
	EndLine int // 语句结束行号（复合语句可能跨多行）
	StmtIdx int // 在父代码块中的语句索引
}

// ReadGroup 表示两次写操作之间的一组连续读事件。
type ReadGroup struct {
	Events   []AccessEvent
	StartIdx int // 在事件列表中的起始索引
	EndIdx   int // 在事件列表中的结束索引（含）
}

// exprContainsPathRead 检查表达式树中是否包含对 target 的"读"引用。
// 对于 target="a.b"：
//   - a.b.c 读取了 a.b（通过 a.b 访问 c）
//   - a.b 本身作为值也是读取
func exprContainsPathRead(expr ast.Expr, target string) bool {
	if expr == nil {
		return false
	}
	path, ok := getExprPath(expr)
	if ok {
		// 表达式路径等于 target 或以 target 为前缀，则是一次读
		if path == target || isPathPrefix(target, path) {
			return true
		}
	}

	// 递归子表达式
	switch e := expr.(type) {
	case *ast.TableAccessor:
		return exprContainsPathRead(e.Obj, target) || exprContainsPathRead(e.Key, target)
	case *ast.Operator:
		return exprContainsPathRead(e.Left, target) || exprContainsPathRead(e.Right, target)
	case *ast.FuncCall:
		if exprContainsPathRead(e.Receiver, target) {
			return true
		}
		if exprContainsPathRead(e.Function, target) {
			return true
		}
		for _, arg := range e.Args {
			if exprContainsPathRead(arg, target) {
				return true
			}
		}
	case *ast.TableConstructor:
		for i, key := range e.Keys {
			if exprContainsPathRead(key, target) {
				return true
			}
			if exprContainsPathRead(e.Vals[i], target) {
				return true
			}
		}
	case *ast.Parens:
		return exprContainsPathRead(e.Inner, target)
	}
	return false
}

// funcCallInvalidatesTarget 检查函数调用是否会使 target 缓存失效。
// 规则：
//   - 如果函数在纯函数白名单中，不失效（不修改参数）
//   - 接收者/参数路径是 target 的严格父级时 → 失效
//     （如 func1(a) 可以做 a.b=xxx → 使 a.b 缓存失效）
//   - 接收者/参数路径等于 target 或是 target 的子级时 → 不失效
//     （如 func1(a.b) 不能修改 a 表上 b 字段的绑定 → a.b 缓存仍有效）
//   - Function 表达式中的嵌套调用也需检查
func funcCallInvalidatesTarget(call *ast.FuncCall, target string) bool {
	// 纯函数白名单检查：如果函数名在白名单中，不使 target 失效
	funcName, nameOk := getFuncCallName(call)
	if nameOk && isPureFunction(funcName) {
		return false
	}

	// 检查接收者（方法调用：a.b:method() → self=a.b，可以修改 a.b 的字段）
	// 只有当 recvPath 是 target 的严格父级时才失效
	if call.Receiver != nil {
		recvPath, ok := getExprPath(call.Receiver)
		if ok && isPathPrefix(recvPath, target) {
			return true
		}
	}

	// 检查每个参数
	// 只有当 argPath 是 target 的严格父级时才失效
	for _, arg := range call.Args {
		argPath, ok := getExprPath(arg)
		if ok && isPathPrefix(argPath, target) {
			return true
		}
		// 同时检查参数中的嵌套函数调用
		if nestedCallInvalidates(arg, target) {
			return true
		}
	}

	// 检查 Function 表达式内的嵌套调用（如 getHandler(a.b)()）
	if nestedCallInvalidates(call.Function, target) {
		return true
	}

	return false
}

// nestedCallInvalidates 检查表达式中是否存在使 target 失效的嵌套函数调用。
func nestedCallInvalidates(expr ast.Expr, target string) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *ast.FuncCall:
		if funcCallInvalidatesTarget(e, target) {
			return true
		}
	case *ast.TableAccessor:
		return nestedCallInvalidates(e.Obj, target)
	case *ast.Operator:
		return nestedCallInvalidates(e.Left, target) || nestedCallInvalidates(e.Right, target)
	case *ast.TableConstructor:
		for i, key := range e.Keys {
			if nestedCallInvalidates(key, target) {
				return true
			}
			if nestedCallInvalidates(e.Vals[i], target) {
				return true
			}
		}
	case *ast.Parens:
		return nestedCallInvalidates(e.Inner, target)
	}
	return false
}

// exprContainsFuncCallInvalidating 检查表达式中是否有任何函数调用会使 target 失效。
func exprContainsFuncCallInvalidating(expr ast.Expr, target string) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *ast.FuncCall:
		if funcCallInvalidatesTarget(e, target) {
			return true
		}
		// 递归检查接收者、函数体、参数
		if exprContainsFuncCallInvalidating(e.Receiver, target) {
			return true
		}
		if exprContainsFuncCallInvalidating(e.Function, target) {
			return true
		}
		for _, arg := range e.Args {
			if exprContainsFuncCallInvalidating(arg, target) {
				return true
			}
		}
	case *ast.TableAccessor:
		return exprContainsFuncCallInvalidating(e.Obj, target) || exprContainsFuncCallInvalidating(e.Key, target)
	case *ast.Operator:
		return exprContainsFuncCallInvalidating(e.Left, target) || exprContainsFuncCallInvalidating(e.Right, target)
	case *ast.TableConstructor:
		for i, key := range e.Keys {
			if exprContainsFuncCallInvalidating(key, target) {
				return true
			}
			if exprContainsFuncCallInvalidating(e.Vals[i], target) {
				return true
			}
		}
	case *ast.Parens:
		return exprContainsFuncCallInvalidating(e.Inner, target)
	}
	return false
}

// blockContainsWrite 检查代码块中是否包含对 target 的写操作。
func blockContainsWrite(block []ast.Stmt, target string) bool {
	for _, stmt := range block {
		if stmtContainsWrite(stmt, target) {
			return true
		}
	}
	return false
}

// stmtContainsWrite 递归检查语句中是否包含对 target 的写操作。
func stmtContainsWrite(stmt ast.Stmt, target string) bool {
	switch s := stmt.(type) {
	case *ast.Assign:
		// 检查赋值左侧目标
		for _, t := range s.Targets {
			tPath, ok := getExprPath(t)
			if ok {
				// 写 tPath：如果 tPath == target 或 tPath 是 target 的父级，则使 target 失效
				if isWriteToTarget(tPath, target) {
					return true
				}
			}
		}
		// 检查右侧是否有使 target 失效的函数调用
		for _, v := range s.Values {
			if exprContainsFuncCallInvalidating(v, target) {
				return true
			}
		}
	case *ast.FuncCall:
		if funcCallInvalidatesTarget(s, target) {
			return true
		}
	case *ast.DoBlock:
		if blockContainsWrite(s.Block, target) {
			return true
		}
	case *ast.If:
		if exprContainsFuncCallInvalidating(s.Cond, target) {
			return true
		}
		if blockContainsWrite(s.Then, target) {
			return true
		}
		if blockContainsWrite(s.Else, target) {
			return true
		}
	case *ast.WhileLoop:
		if exprContainsFuncCallInvalidating(s.Cond, target) {
			return true
		}
		if blockContainsWrite(s.Block, target) {
			return true
		}
	case *ast.RepeatUntilLoop:
		if exprContainsFuncCallInvalidating(s.Cond, target) {
			return true
		}
		if blockContainsWrite(s.Block, target) {
			return true
		}
	case *ast.ForLoopNumeric:
		if exprContainsFuncCallInvalidating(s.Init, target) || exprContainsFuncCallInvalidating(s.Limit, target) || exprContainsFuncCallInvalidating(s.Step, target) {
			return true
		}
		if blockContainsWrite(s.Block, target) {
			return true
		}
	case *ast.ForLoopGeneric:
		for _, init := range s.Init {
			if exprContainsFuncCallInvalidating(init, target) {
				return true
			}
		}
		if blockContainsWrite(s.Block, target) {
			return true
		}
	}
	return false
}

// analyzeBlockAccess 分析代码块中各语句对 target 的读写事件。
// 不递归进入子块的内部事件——子块由外层单独处理。
// 复合语句作为整体：如果包含写则整条语句标记为写。
// 已被 oLua 优化过的行（含 "-- opt by oLua"）会被跳过。
func analyzeBlockAccess(block []ast.Stmt, target string) []AccessEvent {
	var events []AccessEvent

	for i, stmt := range block {
		stmtHasRead := false
		stmtHasWrite := false
		stmtLine := stmt.Line()

		// 跳过 oLua 生成的行（避免重复优化自己的输出）
		if stmtLine > 0 && stmtLine <= len(gfilecontent) {
			if strings.Contains(gfilecontent[stmtLine-1], "-- opt by oLua") {
				continue
			}
		}

		switch s := stmt.(type) {
		case *ast.Assign:
			// 检查左侧：判断写和读
			for _, t := range s.Targets {
				tPath, ok := getExprPath(t)
				if ok {
					// 写入 target 或 target 的父级 → 标记为写
					if isWriteToTarget(tPath, target) {
						stmtHasWrite = true
					}
					// target 是 tPath 的前缀 → 标记为读（如 a.b.c=1 读取了 a.b）
					if isPathPrefix(target, tPath) {
						stmtHasRead = true
					}
				}
			}

			// 检查右侧是否有读
			for _, v := range s.Values {
				if exprContainsPathRead(v, target) {
					stmtHasRead = true
				}
				// 检查右侧函数调用是否使 target 失效
				if exprContainsFuncCallInvalidating(v, target) {
					stmtHasWrite = true
				}
			}

			// 检查左侧表达式中的嵌套函数调用（少见但可能存在）
			for _, t := range s.Targets {
				if exprContainsFuncCallInvalidating(t, target) {
					stmtHasWrite = true
				}
			}

		case *ast.FuncCall:
			// 函数调用作为独立语句
			if funcCallInvalidatesTarget(s, target) {
				stmtHasWrite = true
			}
			// 函数表达式和参数可能包含读
			if exprContainsPathRead(s.Receiver, target) {
				stmtHasRead = true
			}
			if exprContainsPathRead(s.Function, target) {
				stmtHasRead = true
			}
			for _, arg := range s.Args {
				if exprContainsPathRead(arg, target) {
					stmtHasRead = true
				}
			}

		case *ast.DoBlock:
			if blockContainsWrite(s.Block, target) {
				stmtHasWrite = true
			}
			if blockContainsRead(s.Block, target) {
				stmtHasRead = true
			}

		case *ast.If:
			// 条件表达式的读（条件先于 body 执行，安全）
			condHasRead := exprContainsPathRead(s.Cond, target)
			condHasWrite := exprContainsFuncCallInvalidating(s.Cond, target)
			// body 中的写
			bodyHasWrite := blockContainsWrite(s.Then, target) || blockContainsWrite(s.Else, target)
			bodyHasRead := blockContainsRead(s.Then, target) || blockContainsRead(s.Else, target)

			if condHasRead {
				stmtHasRead = true
			}
			if condHasWrite {
				stmtHasWrite = true
			}
			if bodyHasWrite {
				stmtHasWrite = true
			}
			if bodyHasRead {
				stmtHasRead = true
			}

		case *ast.WhileLoop:
			if exprContainsPathRead(s.Cond, target) {
				stmtHasRead = true
			}
			if exprContainsFuncCallInvalidating(s.Cond, target) {
				stmtHasWrite = true
			}
			if blockContainsWrite(s.Block, target) {
				stmtHasWrite = true
			}
			if blockContainsRead(s.Block, target) {
				stmtHasRead = true
			}

		case *ast.RepeatUntilLoop:
			if exprContainsPathRead(s.Cond, target) {
				stmtHasRead = true
			}
			if exprContainsFuncCallInvalidating(s.Cond, target) {
				stmtHasWrite = true
			}
			if blockContainsWrite(s.Block, target) {
				stmtHasWrite = true
			}
			if blockContainsRead(s.Block, target) {
				stmtHasRead = true
			}

		case *ast.ForLoopNumeric:
			if exprContainsPathRead(s.Init, target) || exprContainsPathRead(s.Limit, target) || exprContainsPathRead(s.Step, target) {
				stmtHasRead = true
			}
			if blockContainsWrite(s.Block, target) {
				stmtHasWrite = true
			}
			if blockContainsRead(s.Block, target) {
				stmtHasRead = true
			}

		case *ast.ForLoopGeneric:
			for _, init := range s.Init {
				if exprContainsPathRead(init, target) {
					stmtHasRead = true
				}
			}
			if blockContainsWrite(s.Block, target) {
				stmtHasWrite = true
			}
			if blockContainsRead(s.Block, target) {
				stmtHasRead = true
			}

		case *ast.Return:
			for _, item := range s.Items {
				if exprContainsPathRead(item, target) {
					stmtHasRead = true
				}
				if exprContainsFuncCallInvalidating(item, target) {
					stmtHasWrite = true
				}
			}
		}

		// 生成事件——同一语句中写在读之后
		// （如 a.b.c = a.b.d → 先读 a.b，然后 a.b.c 被写但 a.b 仍有效）
		// 对于复合语句（if/while/for）：条件/头部表达式先于 body 执行，
		// 条件中的读是安全的，但 body 的写会使后续读失效。
		// 策略：条件中的读 emit READ（只覆盖条件行），body 的写 emit WRITE（覆盖全语句）
		// 计算复合语句的结束行
		stmtEndLine := stmtLine
		_, maxLine := find_stmt_line_range(stmt)
		if maxLine > stmtEndLine {
			stmtEndLine = maxLine
		}

		// 判断是否是复合语句
		isCompound := false
		condOnlyRead := false
		switch s := stmt.(type) {
		case *ast.DoBlock:
			isCompound = true
		case *ast.If:
			isCompound = true
			// if 条件只求值一次且先于 body 执行，条件中的读安全
			condOnlyRead = exprContainsPathRead(s.Cond, target) && !exprContainsFuncCallInvalidating(s.Cond, target)
		case *ast.WhileLoop:
			isCompound = true
			// while 条件每次迭代重新求值，如果 body 有写则条件不安全
			// 只有 body 无写时条件读才安全（但此时整体不会有 write，不会走到这个分支）
			// 所以 while 不设置 condOnlyRead
		case *ast.RepeatUntilLoop:
			isCompound = true
			// repeat 的条件在 body 之后执行，不适用 condOnlyRead
		case *ast.ForLoopNumeric:
			isCompound = true
			// for 的 init/limit/step 只在进入循环前求值一次，安全
			hasCondRead := exprContainsPathRead(s.Init, target) || exprContainsPathRead(s.Limit, target) || exprContainsPathRead(s.Step, target)
			hasCondWrite := exprContainsFuncCallInvalidating(s.Init, target) || exprContainsFuncCallInvalidating(s.Limit, target) || exprContainsFuncCallInvalidating(s.Step, target)
			condOnlyRead = hasCondRead && !hasCondWrite
		case *ast.ForLoopGeneric:
			isCompound = true
			// for-generic 的迭代器表达式只在进入循环前求值一次，安全
			hasRead := false
			hasWrite := false
			for _, init := range s.Init {
				if exprContainsPathRead(init, target) {
					hasRead = true
				}
				if exprContainsFuncCallInvalidating(init, target) {
					hasWrite = true
				}
			}
			condOnlyRead = hasRead && !hasWrite
		}

		if isCompound && stmtHasRead && stmtHasWrite {
			// 复合语句同时有读和写
			if condOnlyRead {
				// 条件中有读，条件先执行是安全的 → emit READ（只覆盖条件行）+ WRITE（覆盖全部）
				events = append(events, AccessEvent{Type: AccessRead, Line: stmtLine, EndLine: stmtLine, StmtIdx: i})
			}
			events = append(events, AccessEvent{Type: AccessWrite, Line: stmtLine, EndLine: stmtEndLine, StmtIdx: i})
		} else {
			if stmtHasRead {
				// 计算该语句中有多少个不同的子路径访问了 target
				// 这样单条语句中多次读取也能形成有效的 group
				readCount := countDistinctReads(stmt, target)
				if readCount < 1 {
					readCount = 1
				}
				for rc := 0; rc < readCount; rc++ {
					events = append(events, AccessEvent{Type: AccessRead, Line: stmtLine, EndLine: stmtEndLine, StmtIdx: i})
				}
			}
			if stmtHasWrite {
				events = append(events, AccessEvent{Type: AccessWrite, Line: stmtLine, EndLine: stmtEndLine, StmtIdx: i})
			}
		}
	}

	return events
}

// countDistinctReads 计算单条语句中有多少个不同的子路径访问了 target。
// 例如 target="a.b"，语句包含 a.b.c, a.b.d, a.b.e → 返回 3（各算一次读）。
func countDistinctReads(stmt ast.Stmt, target string) int {
	paths := make(map[string]bool)
	var walkExpr func(expr ast.Expr)
	walkExpr = func(expr ast.Expr) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.TableAccessor:
			path, ok := getExprPath(e)
			if ok {
				if path == target || isPathPrefix(target, path) {
					paths[path] = true
					return // 已匹配，不再递归子路径
				}
			}
			// 仅在路径提取失败时（动态 key）才递归
			walkExpr(e.Obj)
			walkExpr(e.Key)
		case *ast.Operator:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *ast.FuncCall:
			walkExpr(e.Receiver)
			walkExpr(e.Function)
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *ast.TableConstructor:
			for i, key := range e.Keys {
				walkExpr(key)
				walkExpr(e.Vals[i])
			}
		case *ast.Parens:
			walkExpr(e.Inner)
		}
	}

	switch s := stmt.(type) {
	case *ast.Assign:
		// 左侧：target 是 tPath 的前缀 → 算一次读
		for _, t := range s.Targets {
			tPath, ok := getExprPath(t)
			if ok && isPathPrefix(target, tPath) {
				paths[tPath] = true
			}
		}
		for _, v := range s.Values {
			walkExpr(v)
		}
	case *ast.FuncCall:
		walkExpr(s.Receiver)
		walkExpr(s.Function)
		for _, arg := range s.Args {
			walkExpr(arg)
		}
	case *ast.Return:
		for _, item := range s.Items {
			walkExpr(item)
		}
	}
	return len(paths)
}

// blockContainsRead 检查代码块中是否有语句读取了 target。
func blockContainsRead(block []ast.Stmt, target string) bool {
	for _, stmt := range block {
		if stmtContainsRead(stmt, target) {
			return true
		}
	}
	return false
}

// stmtContainsRead 检查语句中是否读取了 target。
func stmtContainsRead(stmt ast.Stmt, target string) bool {
	switch s := stmt.(type) {
	case *ast.Assign:
		for _, t := range s.Targets {
			tPath, ok := getExprPath(t)
			if ok && isPathPrefix(target, tPath) {
				return true
			}
		}
		for _, v := range s.Values {
			if exprContainsPathRead(v, target) {
				return true
			}
		}
	case *ast.FuncCall:
		if exprContainsPathRead(s.Receiver, target) || exprContainsPathRead(s.Function, target) {
			return true
		}
		for _, arg := range s.Args {
			if exprContainsPathRead(arg, target) {
				return true
			}
		}
	case *ast.DoBlock:
		return blockContainsRead(s.Block, target)
	case *ast.If:
		if exprContainsPathRead(s.Cond, target) {
			return true
		}
		return blockContainsRead(s.Then, target) || blockContainsRead(s.Else, target)
	case *ast.WhileLoop:
		if exprContainsPathRead(s.Cond, target) {
			return true
		}
		return blockContainsRead(s.Block, target)
	case *ast.RepeatUntilLoop:
		if exprContainsPathRead(s.Cond, target) {
			return true
		}
		return blockContainsRead(s.Block, target)
	case *ast.ForLoopNumeric:
		if exprContainsPathRead(s.Init, target) || exprContainsPathRead(s.Limit, target) || exprContainsPathRead(s.Step, target) {
			return true
		}
		return blockContainsRead(s.Block, target)
	case *ast.ForLoopGeneric:
		for _, init := range s.Init {
			if exprContainsPathRead(init, target) {
				return true
			}
		}
		return blockContainsRead(s.Block, target)
	case *ast.Return:
		for _, item := range s.Items {
			if exprContainsPathRead(item, target) {
				return true
			}
		}
	}
	return false
}

// ============================================================================
// 候选收集
// ============================================================================

// collectTableAccessCandidates 遍历 AST 统计表访问出现次数。
// 父路径 "a.b" 的计数 = 有多少个不同的子路径访问了它
// （例如 a.b.c 和 a.b.d 各使 a.b 计数+1，合计 count=2）。
// 路径计数必须 >= threshold 才算候选。
func collectTableAccessCandidates(block []ast.Stmt, threshold int) map[string]int {
	counts := make(map[string]int)

	// 每条语句收集叶路径，再统计父路径的贡献
	var walkExpr func(expr ast.Expr, leafPaths map[string]bool)
	walkExpr = func(expr ast.Expr, leafPaths map[string]bool) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.TableAccessor:
			path, ok := getExprPath(e)
			if ok && strings.Contains(path, ".") {
				// 记录完整路径作为叶节点，不递归 Obj/Key
				// 因为中间路径（如 a.b.c 中的 a.b）不是独立的读操作
				leafPaths[path] = true
				return
			}
			// 路径提取失败（动态 key），递归 Obj
			walkExpr(e.Obj, leafPaths)
			walkExpr(e.Key, leafPaths)
		case *ast.Operator:
			walkExpr(e.Left, leafPaths)
			walkExpr(e.Right, leafPaths)
		case *ast.FuncCall:
			walkExpr(e.Receiver, leafPaths)
			walkExpr(e.Function, leafPaths)
			for _, arg := range e.Args {
				walkExpr(arg, leafPaths)
			}
		case *ast.TableConstructor:
			for i, key := range e.Keys {
				walkExpr(key, leafPaths)
				walkExpr(e.Vals[i], leafPaths)
			}
		case *ast.Parens:
			walkExpr(e.Inner, leafPaths)
		}
	}

	var walkBlock func(block []ast.Stmt)
	walkBlock = func(block []ast.Stmt) {
		for _, stmt := range block {
			// 跳过 oLua 生成的行
			stmtLine := stmt.Line()
			if stmtLine > 0 && stmtLine <= len(gfilecontent) {
				if strings.Contains(gfilecontent[stmtLine-1], "-- opt by oLua") {
					continue
				}
			}
			leafPaths := make(map[string]bool)
			switch s := stmt.(type) {
			case *ast.Assign:
				for _, t := range s.Targets {
					walkExpr(t, leafPaths)
				}
				for _, v := range s.Values {
					walkExpr(v, leafPaths)
				}
			case *ast.FuncCall:
				walkExpr(s.Receiver, leafPaths)
				walkExpr(s.Function, leafPaths)
				for _, arg := range s.Args {
					walkExpr(arg, leafPaths)
				}
			case *ast.DoBlock:
				walkBlock(s.Block)
			case *ast.If:
				walkExpr(s.Cond, leafPaths)
				walkBlock(s.Then)
				walkBlock(s.Else)
			case *ast.WhileLoop:
				walkExpr(s.Cond, leafPaths)
				walkBlock(s.Block)
			case *ast.RepeatUntilLoop:
				walkExpr(s.Cond, leafPaths)
				walkBlock(s.Block)
			case *ast.ForLoopNumeric:
				walkExpr(s.Init, leafPaths)
				walkExpr(s.Limit, leafPaths)
				walkExpr(s.Step, leafPaths)
				walkBlock(s.Block)
			case *ast.ForLoopGeneric:
				for _, init := range s.Init {
					walkExpr(init, leafPaths)
				}
				walkBlock(s.Block)
			case *ast.Return:
				for _, item := range s.Items {
					walkExpr(item, leafPaths)
				}
			}

			// 对每个叶路径，统计它本身及其所有父级前缀
			// 去重：每个父级只从不同的叶路径各获得 +1
			parentsSeen := make(map[string]bool)
			for path := range leafPaths {
				// 统计叶路径本身
				counts[path]++
				// 统计所有父级前缀（每个叶路径贡献一次）
				parts := strings.Split(path, ".")
				for depth := 2; depth < len(parts); depth++ {
					parent := strings.Join(parts[:depth], ".")
					key := parent + "|" + path // 每个叶路径唯一
					if !parentsSeen[key] {
						parentsSeen[key] = true
						counts[parent]++
					}
				}
			}
		}
	}

	walkBlock(block)

	// 按阈值过滤
	result := make(map[string]int)
	for path, count := range counts {
		if count >= threshold {
			result[path] = count
		}
	}
	return result
}

// ============================================================================
// 优化逻辑
// ============================================================================

// findReadGroups 在事件列表中查找连续读组（被写操作分割）。
// 返回读次数 >= threshold 的组。
func findReadGroups(events []AccessEvent, threshold int) []ReadGroup {
	var groups []ReadGroup
	var currentReads []AccessEvent
	startIdx := 0

	for i, event := range events {
		if event.Type == AccessRead {
			if len(currentReads) == 0 {
				startIdx = i
			}
			currentReads = append(currentReads, event)
		} else {
			// 写操作：如果当前读组足够大，保存
			if len(currentReads) >= threshold {
				groups = append(groups, ReadGroup{
					Events:   append([]AccessEvent{}, currentReads...),
					StartIdx: startIdx,
					EndIdx:   i - 1,
				})
			}
			currentReads = nil
		}
	}
	// 刷新最后一个读组
	if len(currentReads) >= threshold {
		groups = append(groups, ReadGroup{
			Events:   append([]AccessEvent{}, currentReads...),
			StartIdx: startIdx,
			EndIdx:   len(events) - 1,
		})
	}

	return groups
}

// optimizeBlock 尝试优化代码块中的表访问。
// 返回 true 表示已应用优化。
// 策略：优先尝试当前层级（覆盖范围更广），
// 当前层无优化机会时才递归进入子块。
func optimizeBlock(block []ast.Stmt) bool {
	if has_opt {
		return true
	}

	// 先尝试当前层级（外层优先，覆盖更广）
	if optimizeBlockLevel(block) {
		return true
	}

	// 当前层无优化，递归子块
	for _, stmt := range block {
		if has_opt {
			return true
		}
		switch s := stmt.(type) {
		case *ast.DoBlock:
			if optimizeBlock(s.Block) {
				return true
			}
		case *ast.If:
			if optimizeBlock(s.Then) {
				return true
			}
			if optimizeBlock(s.Else) {
				return true
			}
		case *ast.WhileLoop:
			if optimizeBlock(s.Block) {
				return true
			}
		case *ast.RepeatUntilLoop:
			if optimizeBlock(s.Block) {
				return true
			}
		case *ast.ForLoopNumeric:
			if optimizeBlock(s.Block) {
				return true
			}
		case *ast.ForLoopGeneric:
			if optimizeBlock(s.Block) {
				return true
			}
		}
	}

	return false
}

// optimizeBlockLevel 尝试在当前代码块层级进行表访问优化（不递归）。
// 处理多个互不冲突的候选以减少重新解析轮次。
// 候选按最长路径优先排序。两个候选冲突的条件是路径存在前缀关系
// （如 "a.b" 和 "a.b.c" 冲突，因为优化 "a.b" 会改变 "a.b.c" 依赖的文本）。
func optimizeBlockLevel(block []ast.Stmt) bool {
	if has_opt {
		return true
	}

	threshold := *opt_table_access_threshold
	if threshold < 2 {
		threshold = 2
	}

	// 收集候选
	candidates := collectTableAccessCandidates(block, threshold)
	if len(candidates) == 0 {
		return false
	}

	// 排序候选：最长路径优先（优化收益更大），其次按出现次数
	type candidateInfo struct {
		path  string
		count int
	}
	var sortedCandidates []candidateInfo
	for path, count := range candidates {
		sortedCandidates = append(sortedCandidates, candidateInfo{path, count})
	}
	sort.Slice(sortedCandidates, func(i, j int) bool {
		depthI := strings.Count(sortedCandidates[i].path, ".")
		depthJ := strings.Count(sortedCandidates[j].path, ".")
		if depthI != depthJ {
			return depthI > depthJ
		}
		if sortedCandidates[i].count != sortedCandidates[j].count {
			return sortedCandidates[i].count > sortedCandidates[j].count
		}
		// 相同深度和计数时按字母序排列（保证稳定性）
		return sortedCandidates[i].path < sortedCandidates[j].path
	})

	// 计算代码块行范围（用于已优化检测）
	var blockStartLine, blockEndLine int
	if len(block) > 0 {
		blockStartLine = block[0].Line()
		blockEndLine = blockStartLine
		for _, stmt := range block {
			_, maxLine := find_stmt_line_range(stmt)
			if maxLine > blockEndLine {
				blockEndLine = maxLine
			}
		}
	}

	// 记录本轮已应用的候选路径（用于冲突检测）
	appliedPaths := make([]string, 0)
	appliedAny := false

	for _, cand := range sortedCandidates {
		target := cand.path

		// 跳过根标识符是 oLua 生成的路径
		rootIdent := strings.Split(target, ".")[0]
		if isOluaGeneratedName(rootIdent) {
			continue
		}

		// 默认跳过 _G.xxx 路径（可读性差，且通常是常量）
		if !*opt_table_access_global && rootIdent == "_G" {
			continue
		}

		// 跳过与本轮已应用候选冲突的路径
		conflicts := false
		for _, applied := range appliedPaths {
			if pathsRelated(target, applied) {
				conflicts = true
				break
			}
		}
		if conflicts {
			continue
		}

		// 检查是否已在该代码块中被优化过
		localName := getUniqueLocalName(block, table_access_to_local_name(target))
		alreadyOptimized := false
		if blockStartLine > 0 {
			for lineIdx := blockStartLine - 1; lineIdx < blockEndLine && lineIdx < len(gfilecontent); lineIdx++ {
				trimmed := strings.TrimSpace(gfilecontent[lineIdx])
				if strings.Contains(trimmed, "-- opt by oLua") {
					if strings.Contains(trimmed, " = "+target+" ") ||
						strings.HasSuffix(trimmed, " = "+target) {
						alreadyOptimized = true
						break
					}
				}
			}
		}
		if alreadyOptimized {
			continue
		}

		// 分析读写事件
		events := analyzeBlockAccess(block, target)
		if len(events) == 0 {
			continue
		}

		// 查找连续读组
		groups := findReadGroups(events, threshold)
		if len(groups) == 0 {
			continue
		}

		// 验证至少有一个组会实际改变代码
		hasValidGroup := false
		for _, group := range groups {
			for _, event := range group.Events {
				startLine := event.Line
				endLine := event.EndLine
				if endLine < startLine {
					endLine = startLine
				}
				for lineNum := startLine; lineNum <= endLine && lineNum <= len(gfilecontent); lineNum++ {
					line := gfilecontent[lineNum-1]
					if !strings.Contains(line, "-- opt by oLua") && contain_table_access(line, target) > 0 {
						hasValidGroup = true
						break
					}
				}
				if hasValidGroup {
					break
				}
			}
			if hasValidGroup {
				break
			}
		}
		if !hasValidGroup {
			continue
		}

		// 应用该候选的所有读组（从后往前以保持行号正确性）
		for gi := len(groups) - 1; gi >= 0; gi-- {
			group := groups[gi]
			isFirst := (gi == 0)
			applyTableAccessOptimization(target, localName, group, isFirst)
		}
		appliedPaths = append(appliedPaths, target)
		appliedAny = true

		// 插入行后，后续候选的行号已失效，需要重新解析。
		// 未来改进：对不需要插入行的操作可以批量处理。
		break
	}

	return appliedAny
}

// applyTableAccessOptimization 对单个读组应用优化。
func applyTableAccessOptimization(target string, localName string, group ReadGroup, isFirstDecl bool) {
	if len(group.Events) == 0 {
		return
	}

	firstReadLine := group.Events[0].Line

	// 获取首行缩进
	indent := get_content_space(gfilecontent[firstReadLine-1])

	// 构造插入行
	var insertLine string
	if isFirstDecl {
		insertLine = indent + "local " + localName + " = " + target + " -- opt by oLua"
	} else {
		insertLine = indent + localName + " = " + target + " -- opt by oLua"
	}

	// 替换读事件覆盖的所有行中的 target
	// 对于复合语句，替换整个语句行范围内的所有行
	replacedLines := make(map[int]bool)
	for _, event := range group.Events {
		startLine := event.Line
		endLine := event.EndLine
		if endLine < startLine {
			endLine = startLine
		}
		for lineNum := startLine; lineNum <= endLine && lineNum <= len(gfilecontent); lineNum++ {
			if !replacedLines[lineNum] {
				replacedLines[lineNum] = true
				// 跳过 oLua 生成的行
				if strings.Contains(gfilecontent[lineNum-1], "-- opt by oLua") {
					continue
				}
				gfilecontent[lineNum-1] = replace_table_access(gfilecontent[lineNum-1], target, localName)
			}
		}
	}

	// 在第一个读行之前插入 local 声明
	var filecontent []string
	filecontent = append(filecontent, gfilecontent[:firstReadLine-1]...)
	filecontent = append(filecontent, insertLine)
	filecontent = append(filecontent, gfilecontent[firstReadLine-1:]...)
	gfilecontent = filecontent

	log.Printf("opt table_access at: %s:%d target=%s", gfilename, firstReadLine, target)
	goptcount++
	has_opt = true
}

// ============================================================================
// 字符串替换（带单词边界检测）
// ============================================================================

// contain_table_access 统计 content 中 src 出现的次数（带单词边界检查）。
func contain_table_access(content string, src string) int {
	ret := 0
	tmp := content
	begin := 0
	for {
		idx := strings.Index(tmp[begin:], src)
		if idx == -1 {
			break
		}
		idx += begin
		if idx > 0 {
			// 前面不能是 . 或字母数字下划线
			if tmp[idx-1] == '.' || (tmp[idx-1] >= 'a' && tmp[idx-1] <= 'z') || (tmp[idx-1] >= 'A' && tmp[idx-1] <= 'Z') || (tmp[idx-1] >= '0' && tmp[idx-1] <= '9') || tmp[idx-1] == '_' {
				begin = idx + len(src)
				continue
			}
		}
		if idx+len(src) < len(tmp) {
			// 后面不能是字母数字下划线
			if (tmp[idx+len(src)] >= 'a' && tmp[idx+len(src)] <= 'z') || (tmp[idx+len(src)] >= 'A' && tmp[idx+len(src)] <= 'Z') || (tmp[idx+len(src)] >= '0' && tmp[idx+len(src)] <= '9') || tmp[idx+len(src)] == '_' {
				begin = idx + len(src)
				continue
			}
		}
		begin = idx + len(src)
		ret++
	}
	return ret
}

// replace_table_access 将 content 中的 src 替换为 dst（带单词边界检查）。
func replace_table_access(content string, src string, dst string) string {
	tmp := content
	begin := 0
	for {
		idx := strings.Index(tmp[begin:], src)
		if idx == -1 {
			break
		}
		idx += begin
		if idx > 0 {
			// 前面不能是 . 或字母数字下划线
			if tmp[idx-1] == '.' || (tmp[idx-1] >= 'a' && tmp[idx-1] <= 'z') || (tmp[idx-1] >= 'A' && tmp[idx-1] <= 'Z') || (tmp[idx-1] >= '0' && tmp[idx-1] <= '9') || tmp[idx-1] == '_' {
				begin = idx + len(src)
				continue
			}
		}
		if idx+len(src) < len(tmp) {
			// 后面不能是 . 或字母数字下划线
			if (tmp[idx+len(src)] >= 'a' && tmp[idx+len(src)] <= 'z') || (tmp[idx+len(src)] >= 'A' && tmp[idx+len(src)] <= 'Z') || (tmp[idx+len(src)] >= '0' && tmp[idx+len(src)] <= '9') || tmp[idx+len(src)] == '_' {
				begin = idx + len(src)
				continue
			}
		}
		tmp = tmp[:idx] + dst + tmp[idx+len(src):]
		begin = idx + len(dst)
	}
	return tmp
}

// ============================================================================
// 入口
// ============================================================================

// opt_func_table_access 对单个函数执行表访问优化。
func opt_func_table_access(func_decl *ast.FuncDecl) {
	optimizeBlock(func_decl.Block)
}
