package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/milochristiansen/lua/ast"
)

// localizeGlobalsWhitelist v1: bare globals and selected library functions safe to hoist.
var localizeGlobalsWhitelist = map[string]bool{
	// bare
	"print": true, "error": true, "assert": true, "type": true,
	"tostring": true, "tonumber": true, "select": true,
	"pairs": true, "ipairs": true, "next": true,
	"rawget": true, "rawlen": true, "rawequal": true,
	// math
	"math.abs": true, "math.ceil": true, "math.floor": true,
	"math.max": true, "math.min": true, "math.sqrt": true,
	"math.sin": true, "math.cos": true, "math.tan": true,
	"math.log": true, "math.exp": true, "math.random": true,
	// string
	"string.format": true, "string.len": true, "string.sub": true,
	"string.find": true, "string.match": true, "string.gmatch": true,
	"string.rep": true, "string.lower": true, "string.upper": true,
	"string.byte": true, "string.char": true,
	// table / os
	"table.concat": true,
	"os.clock":     true, "os.time": true, "os.date": true,
}

type localizeCandidate struct {
	path  string
	count int
	first int // first read line (1-based), for stable ordering
}

// opt_func_localize_globals hoists frequently-used whitelist globals/lib functions
// into locals at function entry.
func opt_func_localize_globals(func_decl *ast.FuncDecl) {
	if has_opt {
		return
	}
	if func_decl == nil || len(func_decl.Block) == 0 {
		return
	}

	threshold := *opt_localize_globals_threshold
	if threshold < 1 {
		threshold = 1
	}

	// Single-line functions: body shares the declaration line; skip (no clean insert point).
	firstStmtLine := func_decl.Block[0].Line()
	if firstStmtLine <= func_decl.Line() {
		return
	}

	locals := collectFuncLocals(func_decl)
	hasEnvRebind, writes := scanFuncGlobalWrites(func_decl)
	if hasEnvRebind {
		return
	}

	counts, firstLine := countLocalizeReads(func_decl)

	var candidates []localizeCandidate
	for path, n := range counts {
		if n < threshold {
			continue
		}
		if shouldSkipLocalizePath(path, locals, writes) {
			continue
		}
		candidates = append(candidates, localizeCandidate{
			path:  path,
			count: n,
			first: firstLine[path],
		})
	}
	if len(candidates) == 0 {
		return
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].first != candidates[j].first {
			return candidates[i].first < candidates[j].first
		}
		return candidates[i].path < candidates[j].path
	})

	// Allocate aliases (last path segment; rename when that name is already a local/param).
	// Do not use collectIdentifiers on the whole body: whitelist reads would falsely look like conflicts.
	usedNames := make(map[string]bool)
	for name := range locals {
		usedNames[name] = true
	}
	aliases := make(map[string]string, len(candidates))
	var lhsParts []string
	var rhsParts []string
	for _, c := range candidates {
		base := localizeAliasBase(c.path)
		alias := base
		if usedNames[alias] {
			// Reuse getUniqueLocalName when the last segment collides with a real local.
			alias = getUniqueLocalName(func_decl.Block, base)
			if usedNames[alias] {
				for i := 1; i <= 100; i++ {
					candidate := fmt.Sprintf("%s_%d", base, i)
					if !usedNames[candidate] {
						alias = candidate
						break
					}
				}
				if usedNames[alias] {
					alias = "_opt_" + base
				}
			}
		}
		usedNames[alias] = true
		aliases[c.path] = alias
		lhsParts = append(lhsParts, alias)
		rhsParts = append(rhsParts, c.path)
	}

	nestedRanges := collectNestedFuncLineRanges(func_decl.Block)
	bodyStart, bodyEnd := funcBodyLineRange(func_decl)
	if bodyStart < 1 || bodyEnd < bodyStart {
		return
	}

	// Rewrite body lines (exclude nested funcs and existing oLua lines).
	// Longer paths first so string.format is not disturbed by a later bare replace.
	pathsByLen := make([]string, 0, len(candidates))
	for _, c := range candidates {
		pathsByLen = append(pathsByLen, c.path)
	}
	sort.Slice(pathsByLen, func(i, j int) bool {
		return len(pathsByLen[i]) > len(pathsByLen[j])
	})

	for lineNum := bodyStart; lineNum <= bodyEnd && lineNum <= len(gfilecontent); lineNum++ {
		if lineInRanges(lineNum, nestedRanges) {
			continue
		}
		line := gfilecontent[lineNum-1]
		if strings.Contains(line, "-- opt by oLua") {
			continue
		}
		newLine := line
		for _, path := range pathsByLen {
			alias := aliases[path]
			if alias == path {
				continue
			}
			newLine = replace_table_access(newLine, path, alias)
		}
		gfilecontent[lineNum-1] = newLine
	}

	indent := get_content_space(gfilecontent[firstStmtLine-1])
	insertLine := indent + "local " + strings.Join(lhsParts, ", ") + " = " + strings.Join(rhsParts, ", ") + " -- opt by oLua"

	var filecontent []string
	filecontent = append(filecontent, gfilecontent[:firstStmtLine-1]...)
	filecontent = append(filecontent, insertLine)
	filecontent = append(filecontent, gfilecontent[firstStmtLine-1:]...)
	gfilecontent = filecontent

	log.Printf("opt localize_globals at: %s:%d paths=%v", gfilename, firstStmtLine, rhsParts)
	goptcount++
	has_opt = true
}

func localizeAliasBase(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}
	return path
}

func shouldSkipLocalizePath(path string, locals, writes map[string]bool) bool {
	parts := strings.Split(path, ".")
	// Skip if root is already a local in this function (uses refer to local, not global/lib).
	if locals[parts[0]] {
		return true
	}
	// Skip if there is a global assign to the name or any parent.
	acc := ""
	for i, p := range parts {
		if i == 0 {
			acc = p
		} else {
			acc = acc + "." + p
		}
		if writes[acc] {
			return true
		}
	}
	return false
}

// collectFuncLocals collects names local to this function (not nested FuncDecl bodies).
func collectFuncLocals(func_decl *ast.FuncDecl) map[string]bool {
	locals := make(map[string]bool)
	for _, p := range func_decl.Params {
		locals[p] = true
	}
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n == nil {
			return
		}
		switch e := n.(type) {
		case *ast.FuncDecl:
			*ok = false
		case *ast.Assign:
			if e.LocalDecl || e.LocalFunc {
				for _, t := range e.Targets {
					if id, isIdent := t.(*ast.ConstIdent); isIdent {
						locals[id.Value] = true
					}
				}
			}
		case *ast.ForLoopNumeric:
			locals[e.Counter] = true
		case *ast.ForLoopGeneric:
			for _, name := range e.Locals {
				locals[name] = true
			}
		}
	}}
	for _, stmt := range func_decl.Block {
		ast.Walk(&f, stmt)
	}
	return locals
}

// scanFuncGlobalWrites reports _ENV/_G rebind and non-local assign target paths.
func scanFuncGlobalWrites(func_decl *ast.FuncDecl) (envRebind bool, writes map[string]bool) {
	writes = make(map[string]bool)
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n == nil {
			return
		}
		switch e := n.(type) {
		case *ast.FuncDecl:
			*ok = false
		case *ast.Assign:
			if e.LocalDecl || e.LocalFunc {
				return
			}
			for _, t := range e.Targets {
				path, okPath := getExprPath(t)
				if !okPath {
					continue
				}
				if path == "_ENV" || path == "_G" {
					envRebind = true
				}
				writes[path] = true
			}
		}
	}}
	for _, stmt := range func_decl.Block {
		ast.Walk(&f, stmt)
	}
	return envRebind, writes
}

// countLocalizeReads counts whitelist global/lib reads in the function body,
// excluding nested FuncDecl bodies. Method calls (o:format) are not string.format.
func countLocalizeReads(func_decl *ast.FuncDecl) (counts map[string]int, firstLine map[string]int) {
	counts = make(map[string]int)
	firstLine = make(map[string]int)

	var record func(path string, line int)
	record = func(path string, line int) {
		if !localizeGlobalsWhitelist[path] {
			return
		}
		counts[path]++
		if _, ok := firstLine[path]; !ok || line < firstLine[path] {
			firstLine[path] = line
		}
	}

	var walkExpr func(expr ast.Expr)
	walkExpr = func(expr ast.Expr) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.ConstIdent:
			record(e.Value, e.Line())
		case *ast.TableAccessor:
			if path, ok := getExprPath(e); ok {
				record(path, e.Line())
			}
			// Walk Obj for nested reads; skip const keys so a.print does not count as bare print.
			walkExpr(e.Obj)
			switch e.Key.(type) {
			case *ast.ConstString, *ast.ConstIdent:
				// static key — not a bare global read
			default:
				walkExpr(e.Key)
			}
		case *ast.FuncCall:
			if e.Receiver != nil {
				// Method call o:format() — do not treat Function as string.format.
				walkExpr(e.Receiver)
				// Function is typically ConstString method name; ignore as path.
				for _, a := range e.Args {
					walkExpr(a)
				}
				return
			}
			walkExpr(e.Function)
			for _, a := range e.Args {
				walkExpr(a)
			}
		case *ast.Operator:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *ast.Parens:
			walkExpr(e.Inner)
		case *ast.TableConstructor:
			for i := range e.Keys {
				if e.Keys[i] != nil {
					walkExpr(e.Keys[i])
				}
				walkExpr(e.Vals[i])
			}
		case *ast.FuncDecl:
			// nested function expression — skip body (handled as its own FuncDecl pass)
			return
		}
	}

	var walkStmt func(stmt ast.Stmt)
	walkStmt = func(stmt ast.Stmt) {
		if stmt == nil {
			return
		}
		switch s := stmt.(type) {
		case *ast.Assign:
			if s.LocalFunc {
				// local function name is local; nested body skipped
				for _, v := range s.Values {
					if _, isFunc := v.(*ast.FuncDecl); isFunc {
						continue
					}
					walkExpr(v)
				}
				return
			}
			for _, t := range s.Targets {
				if _, okPath := getExprPath(t); okPath {
					// pure static assign target — write, not read
					continue
				}
				// dynamic target may embed reads (e.g. a[print])
				walkAssignTargetReads(t, record, walkExpr)
			}
			for _, v := range s.Values {
				if _, isFunc := v.(*ast.FuncDecl); isFunc {
					continue
				}
				walkExpr(v)
			}
		case *ast.FuncCall:
			walkExpr(s)
		case *ast.DoBlock:
			for _, b := range s.Block {
				walkStmt(b)
			}
		case *ast.If:
			walkExpr(s.Cond)
			for _, b := range s.Then {
				walkStmt(b)
			}
			for _, b := range s.Else {
				walkStmt(b)
			}
		case *ast.WhileLoop:
			walkExpr(s.Cond)
			for _, b := range s.Block {
				walkStmt(b)
			}
		case *ast.RepeatUntilLoop:
			for _, b := range s.Block {
				walkStmt(b)
			}
			walkExpr(s.Cond)
		case *ast.ForLoopNumeric:
			walkExpr(s.Init)
			walkExpr(s.Limit)
			walkExpr(s.Step)
			for _, b := range s.Block {
				walkStmt(b)
			}
		case *ast.ForLoopGeneric:
			for _, e := range s.Init {
				walkExpr(e)
			}
			for _, b := range s.Block {
				walkStmt(b)
			}
		case *ast.Return:
			for _, it := range s.Items {
				walkExpr(it)
			}
		}
	}

	for _, stmt := range func_decl.Block {
		walkStmt(stmt)
	}
	return counts, firstLine
}

func walkAssignTargetReads(t ast.Expr, record func(string, int), walkExpr func(ast.Expr)) {
	switch e := t.(type) {
	case *ast.ConstIdent:
		return
	case *ast.TableAccessor:
		// Do not record the written path; still scan sub-exprs for embedded reads.
		if _, ok := getExprPath(e); ok {
			// static path write — sub idents are part of the path, not separate reads
			return
		}
		walkExpr(e.Obj)
		walkExpr(e.Key)
	default:
		walkExpr(t)
	}
}

func collectNestedFuncLineRanges(block []ast.Stmt) [][2]int {
	var ranges [][2]int
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n == nil {
			return
		}
		if fd, is := n.(*ast.FuncDecl); is {
			minL, maxL := find_stmt_line_range(fd)
			if minL > 0 && maxL >= minL {
				ranges = append(ranges, [2]int{minL, maxL})
			}
			*ok = false
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}
	return ranges
}

func funcBodyLineRange(func_decl *ast.FuncDecl) (int, int) {
	minL := int(^uint(0) >> 1)
	maxL := -1
	for _, stmt := range func_decl.Block {
		a, b := find_stmt_line_range(stmt)
		if a < 0 {
			continue
		}
		if a < minL {
			minL = a
		}
		if b > maxL {
			maxL = b
		}
	}
	if maxL < 0 {
		return -1, -1
	}
	return minL, maxL
}

func lineInRanges(line int, ranges [][2]int) bool {
	for _, r := range ranges {
		if line >= r[0] && line <= r[1] {
			return true
		}
	}
	return false
}
