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
	for i, stmt := range block {
		if has_opt {
			return
		}
		// DCE before folding inside the same if (if header is earlier in source order).
		if ifs, ok := stmt.(*ast.If); ok {
			if tryDCEIf(ifs, block, i) {
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
func tryDCEIf(ifs *ast.If, parent []ast.Stmt, idx int) bool {
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

	ifLine := ifs.Line()
	if ifLine < 1 || ifLine > len(gfilecontent) {
		return false
	}
	endLine := findIfEndLine(ifs, parent, idx)
	if endLine < ifLine || endLine > len(gfilecontent) {
		return false
	}

	var keep []ast.Stmt
	if cb.Value {
		keep = ifs.Then
	} else {
		keep = ifs.Else
	}

	keepLines := collectStmtLinesForDCE(keep, ifLine, endLine)
	indent := get_content_space(gfilecontent[ifLine-1])

	var newContent []string
	newContent = append(newContent, gfilecontent[:ifLine-1]...)
	if len(keepLines) == 0 {
		newContent = append(newContent, indent+"-- opt by oLua")
	} else {
		for i, ln := range keepLines {
			if i == 0 && !strings.Contains(ln, "-- opt by oLua") {
				newContent = append(newContent, ln+" -- opt by oLua")
			} else {
				newContent = append(newContent, ln)
			}
		}
	}
	newContent = append(newContent, gfilecontent[endLine:]...)
	gfilecontent = newContent

	log.Printf("opt const_fold DCE if %v at: %s:%d", cb.Value, gfilename, ifLine)
	goptcount++
	has_opt = true
	return true
}

func collectStmtLinesForDCE(keep []ast.Stmt, ifLine, endLine int) []string {
	if len(keep) == 0 {
		return nil
	}
	minL := math.MaxInt32
	maxL := -1
	for _, s := range keep {
		a, b := find_stmt_line_range(s)
		if a < 0 {
			a = s.Line()
			b = a
		}
		if a < minL {
			minL = a
		}
		if b > maxL {
			maxL = b
		}
	}
	if minL < ifLine+1 {
		minL = ifLine + 1
	}
	if maxL < minL {
		return nil
	}
	if maxL >= endLine {
		maxL = endLine - 1
	}
	ifIndent := len(get_content_space(gfilecontent[ifLine-1]))
	var out []string
	for ln := minL; ln <= maxL && ln <= len(gfilecontent); ln++ {
		trimmed := strings.TrimSpace(gfilecontent[ln-1])
		lineIndent := len(get_content_space(gfilecontent[ln-1]))
		if ln > minL && lineIndent == ifIndent {
			if trimmed == "else" || strings.HasPrefix(trimmed, "else ") ||
				strings.HasPrefix(trimmed, "elseif ") || trimmed == "end" ||
				strings.HasPrefix(trimmed, "end ") || strings.HasPrefix(trimmed, "end;") ||
				strings.HasPrefix(trimmed, "end--") {
				break
			}
		}
		out = append(out, gfilecontent[ln-1])
	}
	return out
}

func findIfEndLine(ifs *ast.If, parent []ast.Stmt, idx int) int {
	ifLine := ifs.Line()
	ifIndent := len(get_content_space(gfilecontent[ifLine-1]))
	upper := len(gfilecontent) + 1
	if idx+1 < len(parent) {
		next := parent[idx+1].Line()
		if next > 0 {
			upper = next
		}
	}
	// Prefer forward keyword-depth scan so a later nested/sibling end at the same
	// indent (e.g. another function's if) is not chosen when the if is last in its block.
	if end := scanBlockEndLine(ifLine, ifIndent, upper); end >= ifLine {
		return end
	}
	for ln := upper - 1; ln > ifLine; ln-- {
		if ln < 1 || ln > len(gfilecontent) {
			continue
		}
		trimmed := strings.TrimSpace(gfilecontent[ln-1])
		if trimmed == "end" || strings.HasPrefix(trimmed, "end ") ||
			strings.HasPrefix(trimmed, "end;") || strings.HasPrefix(trimmed, "end--") {
			if len(get_content_space(gfilecontent[ln-1])) == ifIndent {
				return ln
			}
		}
	}
	return ifLine
}

func scanBlockEndLine(ifLine, ifIndent, upper int) int {
	depth := 0
	for ln := ifLine; ln < upper && ln <= len(gfilecontent); ln++ {
		fields := strings.Fields(stripLuaLineComment(gfilecontent[ln-1]))
		for _, tok := range fields {
			switch tok {
			case "if", "while", "for", "function", "repeat":
				depth++
			case "end", "until":
				depth--
				if depth == 0 && len(get_content_space(gfilecontent[ln-1])) == ifIndent {
					return ln
				}
			}
		}
	}
	if upper-1 >= ifLine {
		return upper - 1
	}
	return ifLine
}

func stripLuaLineComment(line string) string {
	inS, inD := false, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '\'' && !inD {
			inS = !inS
			continue
		}
		if c == '"' && !inS {
			inD = !inD
			continue
		}
		if !inS && !inD && c == '-' && i+1 < len(line) && line[i+1] == '-' {
			return line[:i]
		}
	}
	return line
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
	loc := re.FindStringIndex(line)
	if loc == nil {
		return false
	}
	repl := result.lit()
	if repl == "" {
		return false
	}
	newLine := line[:loc[0]] + repl + line[loc[1]:]
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
		i, err := strconv.ParseInt(e.Value, 0, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(e.Value, 64)
			if ferr != nil {
				return foldedValue{}, false
			}
			return foldedValue{kind: foldFloat, f: f}, true
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
		return foldedValue{kind: foldString, s: e.Value}, true
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

	switch name {
	case "OpDiv":
		if rf == 0 {
			return foldedValue{}, false
		}
		return foldedValue{kind: foldFloat, f: lf / rf}, true
	case "OpPow":
		res := math.Pow(lf, rf)
		if math.IsNaN(res) || math.IsInf(res, 0) {
			return foldedValue{}, false
		}
		return foldedValue{kind: foldFloat, f: res}, true
	case "OpIDiv":
		if rf == 0 {
			return foldedValue{}, false
		}
		// Lua // is floor division
		res := math.Floor(lf / rf)
		if bothInt && res >= math.MinInt64 && res <= math.MaxInt64 {
			return foldedValue{kind: foldInt, i: int64(res)}, true
		}
		return foldedValue{kind: foldFloat, f: res}, true
	case "OpMod":
		if rf == 0 {
			return foldedValue{}, false
		}
		// Lua a % b == a - math.floor(a/b)*b
		res := lf - math.Floor(lf/rf)*rf
		if bothInt && res >= math.MinInt64 && res <= math.MaxInt64 {
			return foldedValue{kind: foldInt, i: int64(res)}, true
		}
		return foldedValue{kind: foldFloat, f: res}, true
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
