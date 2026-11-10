package main

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/milochristiansen/lua/ast"
)

// foldedValue is a compile-time constant produced by const folding.
type foldedValue struct {
	kind foldedKind
	i    int64
	f    float64
	b    bool
	s    string
}

type foldedKind int

const (
	foldInt foldedKind = iota
	foldFloat
	foldBool
	foldString
	foldNil
)

func (v foldedValue) lit() string {
	switch v.kind {
	case foldInt:
		if v.i == math.MinInt64 {
			return "(-9223372036854775807 - 1)"
		}
		return strconv.FormatInt(v.i, 10)
	case foldFloat:
		return formatFoldFloat(v.f)
	case foldBool:
		if v.b {
			return "true"
		}
		return "false"
	case foldString:
		return "'" + v.s + "'"
	case foldNil:
		return "nil"
	}
	return ""
}

func formatFoldFloat(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return ""
	}
	if f == math.Trunc(f) {
		return fmt.Sprintf("%.1f", f)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func (v foldedValue) isFalsey() bool {
	return v.kind == foldNil || (v.kind == foldBool && !v.b)
}

// opt_func_const_fold folds pure constant arith/rel/bool exprs and DCEs if true/false.
// One change per round. Nested FuncDecl bodies are handled separately by opt_lua.
func opt_func_const_fold(func_decl *ast.FuncDecl) {
	if has_opt {
		return
	}
	if func_decl == nil || len(func_decl.Block) == 0 {
		return
	}
	tryConstFoldInBlock(func_decl.Block)
}

func tryConstFoldInBlock(block []ast.Stmt) {
	if has_opt {
		return
	}
	for _, stmt := range block {
		if has_opt {
			return
		}
		// DCE before folding inside the same if (if header is earlier in source order).
		if ifs, ok := stmt.(*ast.If); ok {
			if tryDCEIf(ifs) {
				return
			}
		}
		if tryFoldInStmt(stmt) {
			return
		}
		switch s := stmt.(type) {
		case *ast.DoBlock:
			tryConstFoldInBlock(s.Block)
		case *ast.If:
			tryConstFoldInBlock(s.Then)
			if has_opt {
				return
			}
			tryConstFoldInBlock(s.Else)
		case *ast.WhileLoop:
			tryConstFoldInBlock(s.Block)
		case *ast.RepeatUntilLoop:
			tryConstFoldInBlock(s.Block)
		case *ast.ForLoopNumeric:
			tryConstFoldInBlock(s.Block)
		case *ast.ForLoopGeneric:
			tryConstFoldInBlock(s.Block)
		}
	}
}

// tryDCEIf rewrites/removes if true / if false only (not if 0 / other truthy consts).
// Skips when then/else contains goto, break, or label.
func tryDCEIf(ifs *ast.If) bool {
	if has_opt {
		return true
	}
	cond := unwrapParens(ifs.Cond)
	cb, ok := cond.(*ast.ConstBool)
	if !ok {
		return false
	}
	if ifHasGotoBreakLabel(ifs) {
		return false
	}

	source := strings.Join(gfilecontent, "\n")
	tokens := luaSourceTokens(source)
	start := -1
	for i, tok := range tokens {
		if tok.line == ifs.Line() && tok.text == "if" && !tok.literal {
			if start >= 0 {
				return false
			} // ambiguous same-line AST location
			start = i
		}
	}
	if start < 0 {
		return false
	}
	// Track actual Lua block keywords, not indentation or words inside strings.
	depth, thenAt, branchAt, endAt := 0, -1, -1, -1
	var stack []string
	for i := start; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.literal {
			continue
		}
		switch tok.text {
		case "if", "function", "repeat":
			stack = append(stack, tok.text)
			depth++
		case "for", "while":
			depth++
			stack = append(stack, "loop-header")
		case "do":
			if len(stack) > 0 && stack[len(stack)-1] == "loop-header" {
				stack[len(stack)-1] = "loop"
			} else {
				depth++
				stack = append(stack, "do")
			}
		case "then":
			if depth == 1 && thenAt < 0 {
				thenAt = i
			}
		case "else", "elseif":
			if depth == 1 && branchAt < 0 {
				branchAt = i
			}
		case "end", "until":
			if len(stack) == 0 {
				return false
			}
			stack = stack[:len(stack)-1]
			depth--
			if depth == 0 {
				endAt = i
			}
		}
		if endAt >= 0 {
			break
		}
	}
	if thenAt < 0 || endAt < 0 {
		return false
	}
	body := ""
	if cb.Value {
		stop := endAt
		if branchAt >= 0 {
			stop = branchAt
		}
		body = source[tokens[thenAt].end:tokens[stop].start]
	} else if branchAt >= 0 {
		if tokens[branchAt].text == "else" {
			body = source[tokens[branchAt].end:tokens[endAt].start]
		} else {
			// elseif is a nested if in the AST. Retain its closing end.
			body = "if" + source[tokens[branchAt].end:tokens[endAt].end]
		}
	}
	// An if branch is a lexical scope, even when its condition is constant.
	// Keeping a do/end also preserves validity of a branch-final return.
	indent := get_content_space(gfilecontent[ifs.Line()-1])
	body = strings.TrimLeft(strings.TrimRight(body, " \t\r\n"), "\r\n")
	replacement := "do -- opt by oLua\n" + body + "\n" + indent + "end"
	source = source[:tokens[start].start] + replacement + source[tokens[endAt].end:]
	gfilecontent = strings.Split(source, "\n")
	ifLine := ifs.Line()

	log.Printf("opt const_fold DCE if %v at: %s:%d", cb.Value, gfilename, ifLine)
	goptcount++
	has_opt = true
	return true
}

func ifHasGotoBreakLabel(ifs *ast.If) bool {
	return blockHasGotoBreakLabel(ifs.Then) || blockHasGotoBreakLabel(ifs.Else)
}

func blockHasGotoBreakLabel(block []ast.Stmt) bool {
	found := false
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n == nil || found {
			return
		}
		switch n.(type) {
		case *ast.FuncDecl:
			*ok = false
		case *ast.Goto, *ast.Label:
			found = true
			*ok = false
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
		if found {
			return true
		}
	}
	return false
}

func tryFoldInStmt(stmt ast.Stmt) bool {
	if stmt == nil || has_opt {
		return has_opt
	}

	var foldExpr ast.Expr
	var result foldedValue

	var consider func(expr ast.Expr)
	consider = func(expr ast.Expr) {
		if expr == nil || foldExpr != nil {
			return
		}
		switch e := expr.(type) {
		case *ast.FuncDecl:
			return
		case *ast.Parens:
			if v, ok := evalConstExpr(e); ok {
				if _, isOp := unwrapParens(e).(*ast.Operator); isOp {
					foldExpr = e
					result = v
					return
				}
			}
			consider(e.Inner)
		case *ast.Operator:
			if v, ok := evalConstExpr(e); ok {
				foldExpr = e
				result = v
				return
			}
			consider(e.Left)
			consider(e.Right)
		case *ast.FuncCall:
			consider(e.Receiver)
			consider(e.Function)
			for _, a := range e.Args {
				consider(a)
			}
		case *ast.TableAccessor:
			consider(e.Obj)
			consider(e.Key)
		case *ast.TableConstructor:
			for i := range e.Keys {
				consider(e.Keys[i])
				consider(e.Vals[i])
			}
		}
	}

	switch s := stmt.(type) {
	case *ast.Assign:
		if s.LocalFunc {
			return false
		}
		for _, t := range s.Targets {
			consider(t)
		}
		for _, v := range s.Values {
			if _, isFunc := v.(*ast.FuncDecl); isFunc {
				continue
			}
			consider(v)
		}
	case *ast.FuncCall:
		consider(s)
	case *ast.Return:
		for _, it := range s.Items {
			consider(it)
		}
	case *ast.If:
		consider(s.Cond)
	case *ast.WhileLoop:
		consider(s.Cond)
	case *ast.RepeatUntilLoop:
		consider(s.Cond)
	case *ast.ForLoopNumeric:
		consider(s.Init)
		consider(s.Limit)
		consider(s.Step)
	case *ast.ForLoopGeneric:
		for _, e := range s.Init {
			consider(e)
		}
	}

	if foldExpr == nil {
		return false
	}
	return applyConstFold(foldExpr, result)
}

func applyConstFold(expr ast.Expr, result foldedValue) bool {
	if !exprAllOnOneLine(expr) {
		return false
	}
	lineNum := expr.Line()
	if lineNum < 1 || lineNum > len(gfilecontent) {
		return false
	}
	line := gfilecontent[lineNum-1]
	if strings.Contains(line, "-- opt by oLua") {
		return false
	}
	pat := exprFlexiblePattern(expr)
	if pat == "" {
		return false
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return false
	}
	var loc []int
	offset := 0
	for i := 0; i < lineNum-1; i++ {
		offset += len(gfilecontent[i]) + 1
	}
	tokens := luaSourceTokens(strings.Join(gfilecontent, "\n"))
	matches := 0
	for _, candidate := range re.FindAllStringIndex(line, -1) {
		startOK, endOK, opaque := false, false, false
		for _, tok := range tokens {
			if tok.start == candidate[0]+offset {
				startOK = true
			}
			if tok.end == candidate[1]+offset {
				endOK = true
			}
			if tok.literal && tok.start < candidate[1]+offset && tok.end > candidate[0]+offset {
				opaque = true
			}
		}
		if startOK && endOK && !opaque {
			loc = candidate
			matches++
		}
	}
	if loc == nil || matches != 1 {
		return false
	}
	repl := result.lit()
	if repl == "" {
		return false
	}
	newLine := line[:loc[0]] + "(" + repl + ")" + line[loc[1]:]
	if !strings.Contains(newLine, "-- opt by oLua") {
		newLine = newLine + " -- opt by oLua"
	}
	gfilecontent[lineNum-1] = newLine
	log.Printf("opt const_fold fold at: %s:%d -> %s", gfilename, lineNum, repl)
	goptcount++
	has_opt = true
	return true
}

func exprAllOnOneLine(expr ast.Expr) bool {
	line := expr.Line()
	ok := true
	f := lua_visitor{f: func(n ast.Node, walkOk *bool) {
		if n == nil || !ok {
			return
		}
		if n.Line() != line {
			ok = false
			*walkOk = false
		}
	}}
	ast.Walk(&f, expr)
	return ok
}

func exprFlexiblePattern(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.ConstInt:
		return regexp.QuoteMeta(e.Value)
	case *ast.ConstFloat:
		return regexp.QuoteMeta(e.Value)
	case *ast.ConstBool:
		if e.Value {
			return `true`
		}
		return `false`
	case *ast.ConstNil:
		return `nil`
	case *ast.ConstString:
		return `['"]` + regexp.QuoteMeta(e.Value) + `['"]`
	case *ast.ConstIdent:
		return `\b` + regexp.QuoteMeta(e.Value) + `\b`
	case *ast.Parens:
		inner := exprFlexiblePattern(e.Inner)
		if inner == "" {
			return ""
		}
		return `\(` + `\s*` + inner + `\s*` + `\)`
	case *ast.Operator:
		return operatorFlexiblePattern(e)
	default:
		return ""
	}
}

func operatorFlexiblePattern(op *ast.Operator) string {
	right := op.Right
	left := op.Left
	switch op.Op {
	case ast.OpUMinus, ast.OpBinNot, ast.OpNot, ast.OpLength:
		if right == nil {
			right = left
		}
		rp := exprFlexiblePattern(right)
		if rp == "" {
			return ""
		}
		switch op.Op {
		case ast.OpUMinus:
			return `-` + `\s*` + rp
		case ast.OpBinNot:
			return `~` + `\s*` + rp
		case ast.OpNot:
			return `not` + `\s+` + rp
		case ast.OpLength:
			return `#` + `\s*` + rp
		}
	}

	if left == nil || right == nil {
		return ""
	}
	lp := exprFlexiblePattern(left)
	rp := exprFlexiblePattern(right)
	if lp == "" || rp == "" {
		return ""
	}

	var mid string
	spaced := false
	switch op.Op {
	case ast.OpAdd:
		mid = `\+`
	case ast.OpSub:
		mid = `-`
	case ast.OpMul:
		mid = `\*`
	case ast.OpMod:
		mid = `%`
	case ast.OpPow:
		mid = `\^`
	case ast.OpDiv:
		mid = `/`
	case ast.OpIDiv:
		mid = `//`
	case ast.OpEqual:
		mid = `==`
	case ast.OpNotEqual:
		mid = `~=`
	case ast.OpLessThan:
		mid = `<`
	case ast.OpGreaterThan:
		mid = `>`
	case ast.OpLessOrEqual:
		mid = `<=`
	case ast.OpGreaterOrEqual:
		mid = `>=`
	case ast.OpAnd:
		mid = `and`
		spaced = true
	case ast.OpOr:
		mid = `or`
		spaced = true
	default:
		return ""
	}
	if spaced {
		return lp + `\s+` + mid + `\s+` + rp
	}
	return lp + `\s*` + mid + `\s*` + rp
}

// evalConstExpr evaluates a pure const arith/rel/bool expression tree.
// Skips div/mod by zero, bitwise, concat, length, and non-const leaves.
func evalConstExpr(expr ast.Expr) (foldedValue, bool) {
	expr = unwrapParens(expr)
	switch e := expr.(type) {
	case *ast.ConstInt:
		base := 10
		if strings.HasPrefix(strings.ToLower(e.Value), "0x") {
			base = 0
		}
		i, err := strconv.ParseInt(e.Value, base, 64)
		if err != nil {
			return foldedValue{}, false
		}
		return foldedValue{kind: foldInt, i: i}, true
	case *ast.ConstFloat:
		f, err := strconv.ParseFloat(e.Value, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return foldedValue{}, false
		}
		return foldedValue{kind: foldFloat, f: f}, true
	case *ast.ConstBool:
		return foldedValue{kind: foldBool, b: e.Value}, true
	case *ast.ConstNil:
		return foldedValue{kind: foldNil}, true
	case *ast.ConstString:
		return foldedValue{}, false // escaped strings / locale comparisons stay at runtime
	case *ast.Operator:
		return evalConstOp(e)
	default:
		return foldedValue{}, false
	}
}

func evalConstOp(op *ast.Operator) (foldedValue, bool) {
	switch op.Op {
	case ast.OpAdd, ast.OpSub, ast.OpMul, ast.OpMod, ast.OpPow, ast.OpDiv, ast.OpIDiv,
		ast.OpEqual, ast.OpNotEqual, ast.OpLessThan, ast.OpGreaterThan, ast.OpLessOrEqual, ast.OpGreaterOrEqual,
		ast.OpAnd, ast.OpOr, ast.OpUMinus, ast.OpNot:
		// supported v1
	default:
		return foldedValue{}, false
	}

	if op.Op == ast.OpUMinus || op.Op == ast.OpNot {
		right := op.Right
		if right == nil {
			right = op.Left
		}
		v, ok := evalConstExpr(right)
		if !ok {
			return foldedValue{}, false
		}
		if op.Op == ast.OpNot {
			return foldedValue{kind: foldBool, b: v.isFalsey()}, true
		}
		switch v.kind {
		case foldInt:
			return foldedValue{kind: foldInt, i: -v.i}, true
		case foldFloat:
			return foldedValue{kind: foldFloat, f: -v.f}, true
		default:
			return foldedValue{}, false
		}
	}

	left, okL := evalConstExpr(op.Left)
	if !okL {
		return foldedValue{}, false
	}
	right, okR := evalConstExpr(op.Right)
	if !okR {
		return foldedValue{}, false
	}

	switch op.Op {
	case ast.OpAnd:
		if left.isFalsey() {
			return left, true
		}
		return right, true
	case ast.OpOr:
		if !left.isFalsey() {
			return left, true
		}
		return right, true
	case ast.OpEqual:
		eq, ok := constEqual(left, right)
		if !ok {
			return foldedValue{}, false
		}
		return foldedValue{kind: foldBool, b: eq}, true
	case ast.OpNotEqual:
		eq, ok := constEqual(left, right)
		if !ok {
			return foldedValue{}, false
		}
		return foldedValue{kind: foldBool, b: !eq}, true
	case ast.OpLessThan, ast.OpGreaterThan, ast.OpLessOrEqual, ast.OpGreaterOrEqual:
		cmp, ok := constCompare(left, right)
		if !ok {
			return foldedValue{}, false
		}
		var b bool
		switch op.Op {
		case ast.OpLessThan:
			b = cmp < 0
		case ast.OpGreaterThan:
			b = cmp > 0
		case ast.OpLessOrEqual:
			b = cmp <= 0
		case ast.OpGreaterOrEqual:
			b = cmp >= 0
		}
		return foldedValue{kind: foldBool, b: b}, true
	case ast.OpAdd, ast.OpSub, ast.OpMul, ast.OpMod, ast.OpPow, ast.OpDiv, ast.OpIDiv:
		return evalArithOp(op.Op, left, right)
	default:
		return foldedValue{}, false
	}
}

func constEqual(a, b foldedValue) (bool, bool) {
	if a.kind == foldInt && b.kind == foldInt {
		return a.i == b.i, true
	}
	if !exactFloatComparison(a, b) {
		return false, false
	}
	if a.kind == foldNil || b.kind == foldNil {
		return a.kind == foldNil && b.kind == foldNil, true
	}
	if a.kind == foldBool && b.kind == foldBool {
		return a.b == b.b, true
	}
	if a.kind == foldString && b.kind == foldString {
		return a.s == b.s, true
	}
	af, aOk := asFloat(a)
	bf, bOk := asFloat(b)
	if aOk && bOk {
		return af == bf, true
	}
	if a.kind != b.kind {
		return false, true
	}
	return false, false
}

func constCompare(a, b foldedValue) (int, bool) {
	if a.kind == foldInt && b.kind == foldInt {
		if a.i < b.i {
			return -1, true
		}
		if a.i > b.i {
			return 1, true
		}
		return 0, true
	}
	if !exactFloatComparison(a, b) {
		return 0, false
	}
	af, aOk := asFloat(a)
	bf, bOk := asFloat(b)
	if aOk && bOk {
		switch {
		case af < bf:
			return -1, true
		case af > bf:
			return 1, true
		default:
			return 0, true
		}
	}
	if a.kind == foldString && b.kind == foldString {
		switch {
		case a.s < b.s:
			return -1, true
		case a.s > b.s:
			return 1, true
		default:
			return 0, true
		}
	}
	return 0, false
}

func asFloat(v foldedValue) (float64, bool) {
	switch v.kind {
	case foldInt:
		return float64(v.i), true
	case foldFloat:
		return v.f, true
	default:
		return 0, false
	}
}

func evalArithOp(op interface{}, left, right foldedValue) (foldedValue, bool) {
	// op is ast.opTyp; compare via String()
	name := ""
	if s, ok := op.(fmt.Stringer); ok {
		name = s.String()
	} else {
		name = fmt.Sprint(op)
	}

	lf, lok := asFloat(left)
	rf, rok := asFloat(right)
	if !lok || !rok {
		return foldedValue{}, false
	}

	bothInt := left.kind == foldInt && right.kind == foldInt
	if bothInt && (name == "OpIDiv" || name == "OpMod") {
		a, b := left.i, right.i
		if b == 0 {
			return foldedValue{}, false
		}
		q, r := a/b, a%b
		if r != 0 && (r < 0) != (b < 0) {
			q--
			r += b
		}
		if name == "OpMod" {
			return foldedValue{kind: foldInt, i: r}, true
		}
		return foldedValue{kind: foldInt, i: q}, true
	}
	// Host libm/rounding and Lua floating modulo are not a portable folding
	// specification. Keep division/power/float modulo at runtime.
	if name == "OpPow" || name == "OpDiv" || name == "OpIDiv" || name == "OpMod" {
		return foldedValue{}, false
	}

	switch name {
	case "OpAdd":
		if bothInt {
			return foldedValue{kind: foldInt, i: left.i + right.i}, true
		}
		return foldedValue{kind: foldFloat, f: lf + rf}, true
	case "OpSub":
		if bothInt {
			return foldedValue{kind: foldInt, i: left.i - right.i}, true
		}
		return foldedValue{kind: foldFloat, f: lf - rf}, true
	case "OpMul":
		if bothInt {
			return foldedValue{kind: foldInt, i: left.i * right.i}, true
		}
		return foldedValue{kind: foldFloat, f: lf * rf}, true
	default:
		return foldedValue{}, false
	}
}

// Mixed integer/float comparison must not round a large int through float64.
func exactFloatComparison(a, b foldedValue) bool {
	for _, v := range []foldedValue{a, b} {
		if v.kind == foldInt && (v.i > 1<<53 || v.i < -(1<<53)) {
			return false
		}
	}
	return true
}
