package main

import (
	"log"
	"strings"

	"github.com/milochristiansen/lua/ast"
)

// ============================================================================
// 树表示：合并后的构造器
// ============================================================================

// pathStep 是表访问链中的一个静态段。
type pathStep struct {
	keyStr  string   // 规范化 key：".name" 或 "[N]"
	keyExpr ast.Expr // 原始 key 表达式（用于代码生成）
}

// ctorField 表示构造器中的一个字段。
type ctorField struct {
	keyStr  string    // ".name" / "[N]" / "" (数组位)
	keyExpr ast.Expr  // 原始 key（数组位为 nil）
	valExpr ast.Expr  // 叶子值（sub != nil 时为 nil）
	sub     *ctorTree // 嵌套子树（叶子时为 nil）
}

// ctorTree 表示一个（可能嵌套的）table 构造器。
type ctorTree struct {
	fields []*ctorField
}

func (t *ctorTree) find(keyStr string) *ctorField {
	if keyStr == "" {
		return nil
	}
	for _, f := range t.fields {
		if f.keyStr == keyStr {
			return f
		}
	}
	return nil
}

// buildTreeFromConstructor 从 AST 的 TableConstructor 建立初始 ctorTree。
// 数组位（k == nil）保留为无 key 字段。
func buildTreeFromConstructor(cons *ast.TableConstructor) *ctorTree {
	t := &ctorTree{}
	for i, k := range cons.Keys {
		v := cons.Vals[i]
		f := &ctorField{}
		if k == nil {
			// 数组位
			f.keyStr = ""
			f.keyExpr = nil
		} else {
			ks, ok := canonicalKeyString(k)
			if !ok {
				// 不支持的 key（如动态变量）：作为普通字段保留，用 expr_to_string 输出
				f.keyStr = "__opaque_" + expr_to_string(k)
			} else {
				f.keyStr = ks
			}
			f.keyExpr = k
		}
		if sub, isCons := v.(*ast.TableConstructor); isCons {
			f.sub = buildTreeFromConstructor(sub)
		} else {
			f.valExpr = v
		}
		t.fields = append(t.fields, f)
	}
	return t
}

// canonicalKeyString 把 key 表达式规范成字符串标识（用于 tree 查找）。
// 只接受编译期常量 key：ConstString（如 a.name / a["name"]）和 ConstInt（如 a[3]）。
// 拒绝 ConstIdent（如 a[b]，b 是变量）和其他动态 key。
func canonicalKeyString(k ast.Expr) (string, bool) {
	switch e := k.(type) {
	case *ast.ConstString:
		return "." + e.Value, true
	case *ast.ConstInt:
		return "[" + e.Value + "]", true
	}
	return "", false
}

// ============================================================================
// 路径提取
// ============================================================================

// extractStaticPath 从 target 表达式提取根名称和静态路径段。
// 例：a.b[3].c → root="a", steps=[.b, [3], .c]
// 遇到动态 key 返回 ok=false。
func extractStaticPath(expr ast.Expr) (root string, steps []pathStep, ok bool) {
	switch e := expr.(type) {
	case *ast.ConstIdent:
		return e.Value, nil, true
	case *ast.TableAccessor:
		r, prev, ok := extractStaticPath(e.Obj)
		if !ok {
			return "", nil, false
		}
		ks, ok := canonicalKeyString(e.Key)
		if !ok {
			return "", nil, false
		}
		return r, append(prev, pathStep{keyStr: ks, keyExpr: e.Key}), true
	}
	return "", nil, false
}

// ============================================================================
// 吸收单条赋值
// ============================================================================

// absorbAssign 尝试把 assign 吸收进 tree。
// rootPath 是初始构造器的目标路径（如 "a" 或 "s.o"）。
// 返回 true 表示已吸收（tree 被修改）。
func (t *ctorTree) absorbAssign(rootName string, rootSteps []pathStep, assign *ast.Assign) bool {
	if len(assign.Targets) != 1 || len(assign.Values) != 1 {
		return false
	}
	value := assign.Values[0]
	if !can_expr_to_string(value) {
		return false
	}

	targetName, targetSteps, ok := extractStaticPath(assign.Targets[0])
	if !ok {
		return false
	}
	if targetName != rootName {
		return false
	}

	// targetSteps 必须以 rootSteps 为前缀，且严格更长
	if len(targetSteps) <= len(rootSteps) {
		return false
	}
	for i, s := range rootSteps {
		if targetSteps[i].keyStr != s.keyStr {
			return false
		}
	}
	remainder := targetSteps[len(rootSteps):]
	if len(remainder) == 0 {
		return false
	}

	// 沿 tree 走到最后一级的父节点
	cur := t
	for i := 0; i < len(remainder)-1; i++ {
		f := cur.find(remainder[i].keyStr)
		if f == nil || f.sub == nil {
			// 中间节点缺失或不是子表 → 不能折叠
			return false
		}
		cur = f.sub
	}

	// Numeric keys have equivalent spellings (1, 01, 0x1). The legacy
	// constructor tree keys are textual, so only named/string keys are proven.
	for _, step := range remainder {
		if _, ok := step.keyExpr.(*ast.ConstString); !ok {
			return false
		}
	}
	lastStep := remainder[len(remainder)-1]
	newField := &ctorField{keyStr: lastStep.keyStr, keyExpr: lastStep.keyExpr}
	if cons, isCons := value.(*ast.TableConstructor); isCons {
		newField.sub = buildTreeFromConstructor(cons)
	} else {
		newField.valExpr = value
	}

	// 同键覆盖：找到已有字段则原地替换（保持位置），否则追加
	if existing := cur.find(lastStep.keyStr); existing != nil {
		existing.keyExpr = newField.keyExpr
		existing.valExpr = newField.valExpr
		existing.sub = newField.sub
	} else {
		cur.fields = append(cur.fields, newField)
	}
	return true
}

// ============================================================================
// 候选查找
// ============================================================================

// mergeCandidate 描述一次可执行的合并优化。
type mergeCandidate struct {
	block       []ast.Stmt // 语句所在 block（同层）
	ctorStmt    ast.Stmt   // 初始的构造器赋值语句
	rootName    string     // 目标根名（如 "a"）
	rootSteps   []pathStep // 目标其余段（如 "s.o" → [.o]）
	tree        *ctorTree  // 合并后的树
	absorbed    []ast.Stmt // 被吸收的后续赋值语句
	ctorLine    int        // 原构造器所在起始行
	ctorEndLine int        // 原构造器所在结束行（多行时的最后一行）
}

// findMergeCandidate 在 block 中查找一个可合并的构造器（不递归子块）。
// 策略：从第一个构造器赋值开始，向后扫描：
//   - 可静态折叠的 rootPath 后续赋值 → 吸收
//   - 读 rootName 或写 rootName / 可能修改 rootName 的调用 → 停止
//   - 其他无关语句 → 跳过继续扫描
//
// 返回至少吸收了一条时才算成功。
func findMergeCandidate(block []ast.Stmt) *mergeCandidate {
	for i, stmt := range block {
		assign, isAssign := stmt.(*ast.Assign)
		if !isAssign {
			continue
		}
		if len(assign.Targets) != 1 || len(assign.Values) != 1 {
			continue
		}
		// Only fresh locals and literal-only, standalone source statements.
		// Merging arbitrary RHS expressions can remove calls, observe a later
		// variable value too early, or reorder errors/global __newindex effects.
		if !assign.LocalDecl || !standaloneConstructorStmt(assign) {
			continue
		}
		if _, ok := assign.Targets[0].(*ast.ConstIdent); !ok {
			continue
		}
		if !safeConstructorLiteral(assign.Values[0]) {
			continue
		}
		cons, isCons := assign.Values[0].(*ast.TableConstructor)
		if !isCons {
			continue
		}
		rootName, rootSteps, ok := extractStaticPath(assign.Targets[0])
		if !ok {
			continue
		}

		// 跳过已优化的行
		stmtLine := stmt.Line()
		if stmtLine > 0 && stmtLine <= len(gfilecontent) &&
			strings.Contains(gfilecontent[stmtLine-1], "-- opt by oLua") {
			continue
		}

		// 构造器所有初始字段值必须可以字符串化
		initOk := true
		for j, v := range cons.Vals {
			if !can_expr_to_string(v) {
				initOk = false
				break
			}
			_ = j
		}
		if !initOk {
			continue
		}
		// key 也需要可以字符串化（数组位 k==nil 除外）
		for _, k := range cons.Keys {
			if k == nil {
				continue
			}
			if _, ok := canonicalKeyString(k); !ok {
				initOk = false
				break
			}
		}
		if !initOk {
			continue
		}

		tree := buildTreeFromConstructor(cons)
		var absorbed []ast.Stmt

		// 用完整路径作为读写分析的 target
		fullPath := rootName
		for _, s := range rootSteps {
			// 只用 . 形式作为 access.go 里的 path（它不支持 [N]）
			if strings.HasPrefix(s.keyStr, ".") {
				fullPath += s.keyStr
			} else {
				// 遇到 [N] 段：access.go 的路径工具不支持，退化为根名安全检查
				fullPath = rootName
				break
			}
		}

		stopped := false
		for k := i + 1; k < len(block) && !stopped; k++ {
			nxt := block[k]
			if !standaloneConstructorStmt(nxt) {
				break
			}

			// Stop at generated statements too: a generated do block can have effects.
			nLine := nxt.Line()
			if nLine > 0 && nLine <= len(gfilecontent) &&
				strings.Contains(gfilecontent[nLine-1], "-- opt by oLua") {
				break
			}

			// 尝试吸收
			if a, isA := nxt.(*ast.Assign); isA {
				if a.LocalDecl || len(a.Values) != 1 || !safeConstructorLiteral(a.Values[0]) {
					break
				}
				// 尝试折叠
				snapshot := deepCopyTree(tree)
				if tree.absorbAssign(rootName, rootSteps, a) {
					// 吸收成功前还需要确认：RHS 不读 fullPath、不失效 fullPath
					rhs := a.Values[0]
					if exprContainsPathRead(rhs, fullPath) || exprContainsFuncCallInvalidating(rhs, fullPath) {
						// 回滚：这条不能吸收，也不能穿插（RHS 读/改了目标）
						tree = snapshot
						stopped = true
						break
					}
					absorbed = append(absorbed, nxt)
					continue
				}
				// 吸收失败：当作非候选语句判断能否穿插
			}

			// 非吸收语句：检查是否会破坏目标语义
			if stmtTouchesRoot(nxt, rootName) {
				stopped = true
				break
			}
			if stmtContainsRead(nxt, fullPath) || stmtContainsWrite(nxt, fullPath) {
				stopped = true
				break
			}
			// Do not move a write across even apparently unrelated statements.
			break
		}

		if len(absorbed) == 0 {
			continue
		}

		// 计算原构造器的多行范围
		startLine := stmt.Line()
		endLine := startLine
		if assign, isA := stmt.(*ast.Assign); isA && len(assign.Values) > 0 {
			if _, e := stmtLineRangeIncludingRHS(assign); e > endLine {
				endLine = e
			}
		}

		return &mergeCandidate{
			block:       block,
			ctorStmt:    stmt,
			rootName:    rootName,
			rootSteps:   rootSteps,
			tree:        tree,
			absorbed:    absorbed,
			ctorLine:    startLine,
			ctorEndLine: endLine,
		}
	}
	return nil
}

// findMergeCandidateRecursive 递归查找可合并候选：优先当前层，再进入子块。
func findMergeCandidateRecursive(block []ast.Stmt) *mergeCandidate {
	if c := findMergeCandidate(block); c != nil {
		return c
	}
	for _, stmt := range block {
		switch nn := stmt.(type) {
		case *ast.DoBlock:
			if c := findMergeCandidateRecursive(nn.Block); c != nil {
				return c
			}
		case *ast.If:
			if c := findMergeCandidateRecursive(nn.Then); c != nil {
				return c
			}
			if c := findMergeCandidateRecursive(nn.Else); c != nil {
				return c
			}
		case *ast.WhileLoop:
			if c := findMergeCandidateRecursive(nn.Block); c != nil {
				return c
			}
		case *ast.RepeatUntilLoop:
			if c := findMergeCandidateRecursive(nn.Block); c != nil {
				return c
			}
		case *ast.ForLoopNumeric:
			if c := findMergeCandidateRecursive(nn.Block); c != nil {
				return c
			}
		case *ast.ForLoopGeneric:
			if c := findMergeCandidateRecursive(nn.Block); c != nil {
				return c
			}
		}
	}
	return nil
}

// deepCopyTree 浅拷贝 tree 的字段列表（用于回滚）。字段本身也需要重建，因为 absorbAssign
// 可能就地覆盖 existing 字段的 valExpr/sub。
func deepCopyTree(t *ctorTree) *ctorTree {
	if t == nil {
		return nil
	}
	nt := &ctorTree{fields: make([]*ctorField, len(t.fields))}
	for i, f := range t.fields {
		nf := *f
		nf.sub = deepCopyTree(f.sub)
		nt.fields[i] = &nf
	}
	return nt
}

// stmtTouchesRoot 保守判断：语句是否可能读或写以 rootName 为根的表访问链。
// 用于 access.go 的路径工具无法覆盖的动态 key 场景（如 a[var] = 5、y = a[i]）。
func stmtTouchesRoot(stmt ast.Stmt, rootName string) bool {
	var walk func(expr ast.Expr) bool
	walk = func(expr ast.Expr) bool {
		if expr == nil {
			return false
		}
		switch e := expr.(type) {
		case *ast.ConstIdent:
			return e.Value == rootName
		case *ast.TableAccessor:
			if walk(e.Obj) {
				return true
			}
			return walk(e.Key)
		case *ast.Operator:
			return walk(e.Left) || walk(e.Right)
		case *ast.FuncCall:
			if walk(e.Receiver) || walk(e.Function) {
				return true
			}
			for _, arg := range e.Args {
				if walk(arg) {
					return true
				}
			}
		case *ast.TableConstructor:
			for i, k := range e.Keys {
				if walk(k) || walk(e.Vals[i]) {
					return true
				}
			}
		case *ast.Parens:
			return walk(e.Inner)
		case *ast.FuncDecl:
			// 匿名函数体不递归——它不立即执行；如果被立即调用会走 FuncCall 分支。
			return false
		}
		return false
	}

	switch s := stmt.(type) {
	case *ast.Assign:
		for _, t := range s.Targets {
			if walk(t) {
				return true
			}
		}
		for _, v := range s.Values {
			if walk(v) {
				return true
			}
		}
	case *ast.FuncCall:
		if walk(s.Receiver) || walk(s.Function) {
			return true
		}
		for _, arg := range s.Args {
			if walk(arg) {
				return true
			}
		}
	case *ast.Return:
		for _, item := range s.Items {
			if walk(item) {
				return true
			}
		}
	case *ast.DoBlock:
		for _, sub := range s.Block {
			if stmtTouchesRoot(sub, rootName) {
				return true
			}
		}
	case *ast.If:
		if walk(s.Cond) {
			return true
		}
		for _, sub := range s.Then {
			if stmtTouchesRoot(sub, rootName) {
				return true
			}
		}
		for _, sub := range s.Else {
			if stmtTouchesRoot(sub, rootName) {
				return true
			}
		}
	case *ast.WhileLoop, *ast.RepeatUntilLoop, *ast.ForLoopNumeric, *ast.ForLoopGeneric:
		// 保守：任何循环都可能反复执行——直接停
		return true
	case *ast.Goto, *ast.Label:
		return true
	}
	return false
}

// ============================================================================
// 代码生成
// ============================================================================

// renderField 生成单个字段的 Lua 源码片段。
func renderField(f *ctorField) string {
	valPart := ""
	if f.sub != nil {
		valPart = renderTree(f.sub)
	} else {
		valPart = expr_to_string(f.valExpr)
	}
	if f.keyStr == "" {
		// 数组位
		return valPart
	}
	// 命名字段：统一用 ['name'] 或 [N] 风格
	keyPart := ""
	switch k := f.keyExpr.(type) {
	case *ast.ConstIdent:
		keyPart = "[" + quoteLuaString(k.Value) + "]"
	case *ast.ConstString:
		keyPart = "[" + quoteLuaString(k.Value) + "]"
	case *ast.ConstInt:
		keyPart = "[" + k.Value + "]"
	default:
		// opaque key：直接原样输出
		if f.keyExpr != nil {
			keyPart = "[" + expr_to_string(f.keyExpr) + "]"
		}
	}
	return keyPart + "=" + valPart
}

func renderTree(t *ctorTree) string {
	parts := make([]string, 0, len(t.fields))
	for _, f := range t.fields {
		parts = append(parts, renderField(f))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// stmtLineRangeIncludingRHS 计算赋值语句的完整行范围，尤其是右侧含多行 TableConstructor 时。
// 对 assign a = {...}，会扫到匹配的 `}`。
// 其他情况退化为 find_stmt_line_range。
func stmtLineRangeIncludingRHS(stmt ast.Stmt) (int, int) {
	s, e := find_stmt_line_range(stmt)
	if s < 0 {
		s = stmt.Line()
		e = s
	}
	assign, isA := stmt.(*ast.Assign)
	if !isA || len(assign.Values) != 1 {
		return s, e
	}
	if _, isCons := assign.Values[0].(*ast.TableConstructor); !isCons {
		return s, e
	}
	// Count code braces only; braces inside quoted strings/comments are opaque.
	num := 0
	for i := s; i <= len(gfilecontent); i++ {
		for _, tok := range luaSourceTokens(gfilecontent[i-1]) {
			if tok.literal {
				continue
			}
			if tok.text == "{" {
				num++
			}
			if tok.text == "}" {
				num--
			}
		}
		if num == 0 {
			if i > e {
				e = i
			}
			break
		}
	}

	return s, e
}

// ============================================================================
// 入口
// ============================================================================

func opt_func_table_constructor(func_decl *ast.FuncDecl) {
	cand := findMergeCandidateRecursive(func_decl.Block)
	if cand == nil {
		return
	}

	has_opt = true

	// 收集所有要删除的行（被吸收的语句的行范围）
	deletedLines := make(map[int]bool)
	for _, a := range cand.absorbed {
		s, e := stmtLineRangeIncludingRHS(a)
		for ln := s; ln <= e; ln++ {
			deletedLines[ln] = true
		}
	}

	// 构造新构造器行：保留原始 `local a =` 或 `a =` 前缀
	origLine := gfilecontent[cand.ctorLine-1]
	eqIdx := strings.Index(origLine, "=")
	if eqIdx < 0 {
		// 理论上不会发生：构造器语句一定有 =
		log.Printf("opt_func_table_constructor: no '=' on line %d, skipping", cand.ctorLine)
		has_opt = false
		return
	}
	leftContent := origLine[:eqIdx]
	newCtorLine := strings.TrimRight(leftContent, " ") + " = " + renderTree(cand.tree) + " -- opt by oLua"

	// 组装新 gfilecontent：
	//   1. 原构造器所在的多行 → 只在起始行放新行，其余行删除
	//   2. 被吸收赋值的行范围 → 全部删除
	//   3. 其他行原样保留
	var newContent []string
	for ln := 1; ln <= len(gfilecontent); ln++ {
		if ln == cand.ctorLine {
			newContent = append(newContent, newCtorLine)
			continue
		}
		if ln > cand.ctorLine && ln <= cand.ctorEndLine {
			// 原构造器的续行：删除
			continue
		}
		if deletedLines[ln] {
			continue
		}
		newContent = append(newContent, gfilecontent[ln-1])
	}
	gfilecontent = newContent

	log.Printf("opt table_constructor at: %s:%d (absorbed %d stmts)", gfilename, cand.ctorLine, len(cand.absorbed))
	goptcount++
}

// Restrict construction folding to values with no reads, calls, coercions, or
// sequence fields (which can have duplicate implicit/explicit integer keys).
func safeConstructorLiteral(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.ConstInt, *ast.ConstFloat, *ast.ConstBool, *ast.ConstNil, *ast.ConstString:
		return true
	case *ast.TableConstructor:
		seen := map[string]bool{}
		for i, k := range v.Keys {
			key, ok := canonicalKeyString(k)
			if _, named := k.(*ast.ConstString); !named {
				return false
			}
			if !ok || seen[key] || !safeConstructorLiteral(v.Vals[i]) {
				return false
			}
			seen[key] = true
		}
		return true
	}
	return false
}
func standaloneConstructorStmt(stmt ast.Stmt) bool {
	line := stmt.Line()
	if line < 1 || line > len(gfilecontent) {
		return false
	}
	parsed, err := ast.Parse(gfilecontent[line-1], 1)
	if err != nil || len(parsed) != 1 {
		return false
	}
	_, ok := parsed[0].(*ast.Assign)
	return ok
}
