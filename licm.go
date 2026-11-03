package main

import (
	"log"
	"sort"
	"strings"

	"github.com/milochristiansen/lua/ast"
)

// opt_func_licm performs loop-invariant code motion (v1) on a function.
// One hoist per round: insert a local before the loop header and rewrite body reads.
func opt_func_licm(func_decl *ast.FuncDecl) {
	if has_opt {
		return
	}
	if func_decl == nil || len(func_decl.Block) == 0 {
		return
	}
	threshold := *opt_licm_threshold
	if threshold < 2 {
		threshold = 2
	}
	tryLicmInBlock(func_decl.Block, func_decl, threshold)
}

// tryLicmInBlock walks statements in source order. For each loop, try a hoist
// before recursing into nested blocks. Nested FuncDecl bodies are skipped
// (handled as their own FuncDecl by opt_lua).
func tryLicmInBlock(block []ast.Stmt, func_decl *ast.FuncDecl, threshold int) {
	if has_opt {
		return
	}
	for _, stmt := range block {
		if has_opt {
			return
		}
		switch s := stmt.(type) {
		case *ast.WhileLoop:
			if tryHoistFromLoop(s, s.Block, []ast.Expr{s.Cond}, nil, func_decl, threshold) {
				return
			}
			tryLicmInBlock(s.Block, func_decl, threshold)
		case *ast.RepeatUntilLoop:
			if tryHoistFromLoop(s, s.Block, []ast.Expr{s.Cond}, nil, func_decl, threshold) {
				return
			}
			tryLicmInBlock(s.Block, func_decl, threshold)
		case *ast.ForLoopNumeric:
			binders := map[string]bool{s.Counter: true}
			if tryHoistFromLoop(s, s.Block, nil, binders, func_decl, threshold) {
				return
			}
			tryLicmInBlock(s.Block, func_decl, threshold)
		case *ast.ForLoopGeneric:
			binders := make(map[string]bool)
			for _, name := range s.Locals {
				binders[name] = true
			}
			if tryHoistFromLoop(s, s.Block, nil, binders, func_decl, threshold) {
				return
			}
			tryLicmInBlock(s.Block, func_decl, threshold)
		case *ast.DoBlock:
			tryLicmInBlock(s.Block, func_decl, threshold)
		case *ast.If:
			tryLicmInBlock(s.Then, func_decl, threshold)
			if has_opt {
				return
			}
			tryLicmInBlock(s.Else, func_decl, threshold)
		}
	}
}

// tryHoistFromLoop attempts one LICM hoist for a single loop.
// reevalExprs are expressions re-evaluated each iteration (while/repeat cond);
// their reads count toward the threshold. For-header init/limit/step do not.
func tryHoistFromLoop(loopStmt ast.Stmt, body []ast.Stmt, reevalExprs []ast.Expr, binders map[string]bool, func_decl *ast.FuncDecl, threshold int) bool {
	if has_opt {
		return true
	}
	if loopHasGotoOrLabel(body) {
		return false
	}
	if binders == nil {
		binders = map[string]bool{}
	}

	counts := countLicmBodyReads(body, reevalExprs)
	type cand struct {
		path  string
		count int
	}
	var cands []cand
	for path, n := range counts {
		if n < threshold {
			continue
		}
		if !strings.Contains(path, ".") {
			continue
		}
		if pathHasBinder(path, binders) {
			continue
		}
		// No write / invalidation anywhere in the loop statement.
		if stmtContainsWrite(loopStmt, path) {
			continue
		}
		// Conservative v1: any non-pure call in body or reeval exprs blocks hoist.
		if licmHasImpureCall(body) || exprsHaveImpureCall(reevalExprs) {
			continue
		}
		cands = append(cands, cand{path: path, count: n})
	}
	if len(cands) == 0 {
		return false
	}

	sort.Slice(cands, func(i, j int) bool {
		di := strings.Count(cands[i].path, ".")
		dj := strings.Count(cands[j].path, ".")
		if di != dj {
			return di > dj
		}
		if cands[i].count != cands[j].count {
			return cands[i].count > cands[j].count
		}
		return cands[i].path < cands[j].path
	})

	best := cands[0]
	return applyLicmHoist(loopStmt, body, best.path, func_decl)
}

func pathHasBinder(path string, binders map[string]bool) bool {
	for _, part := range strings.Split(path, ".") {
		if binders[part] {
			return true
		}
	}
	return false
}

func loopHasGotoOrLabel(block []ast.Stmt) bool {
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

func licmHasImpureCall(block []ast.Stmt) bool {
	found := false
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n == nil || found {
			return
		}
		switch e := n.(type) {
		case *ast.FuncDecl:
			*ok = false
		case *ast.FuncCall:
			name, nameOk := getFuncCallName(e)
			if !nameOk || !isPureFunction(name) {
				found = true
				*ok = false
			}
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

func exprsHaveImpureCall(exprs []ast.Expr) bool {
	for _, expr := range exprs {
		if exprHasImpureCall(expr) {
			return true
		}
	}
	return false
}

func exprHasImpureCall(expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	found := false
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n == nil || found {
			return
		}
		switch e := n.(type) {
		case *ast.FuncDecl:
			*ok = false
		case *ast.FuncCall:
			name, nameOk := getFuncCallName(e)
			if !nameOk || !isPureFunction(name) {
				found = true
				*ok = false
			}
		}
	}}
	ast.Walk(&f, expr)
	return found
}

// countLicmBodyReads counts static dotted path reads in the loop body and
// re-evaluated expressions (while/repeat conditions). Nested FuncDecl excluded.
// Parent paths get +1 per distinct leaf path per statement (same as table_access).
func countLicmBodyReads(body []ast.Stmt, reevalExprs []ast.Expr) map[string]int {
	counts := make(map[string]int)

	var walkExpr func(expr ast.Expr, leafPaths map[string]bool)
	walkExpr = func(expr ast.Expr, leafPaths map[string]bool) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.TableAccessor:
			path, ok := getExprPath(e)
			if ok && strings.Contains(path, ".") {
				leafPaths[path] = true
				return
			}
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
		case *ast.FuncDecl:
			return
		}
	}

	addLeaves := func(leafPaths map[string]bool) {
		parentsSeen := make(map[string]bool)
		for path := range leafPaths {
			counts[path]++
			parts := strings.Split(path, ".")
			for depth := 2; depth < len(parts); depth++ {
				parent := strings.Join(parts[:depth], ".")
				key := parent + "|" + path
				if !parentsSeen[key] {
					parentsSeen[key] = true
					counts[parent]++
				}
			}
		}
	}

	var walkStmt func(stmt ast.Stmt)
	walkStmt = func(stmt ast.Stmt) {
		if stmt == nil {
			return
		}
		stmtLine := stmt.Line()
		if stmtLine > 0 && stmtLine <= len(gfilecontent) {
			if strings.Contains(gfilecontent[stmtLine-1], "-- opt by oLua") {
				return
			}
		}
		leafPaths := make(map[string]bool)
		switch s := stmt.(type) {
		case *ast.Assign:
			if s.LocalFunc {
				return
			}
			for _, t := range s.Targets {
				// Only count embedded reads in dynamic targets; static assign
				// targets are writes, not body reads for LICM.
				if _, okPath := getExprPath(t); okPath {
					continue
				}
				walkExpr(t, leafPaths)
			}
			for _, v := range s.Values {
				if _, isFunc := v.(*ast.FuncDecl); isFunc {
					continue
				}
				walkExpr(v, leafPaths)
			}
		case *ast.FuncCall:
			walkExpr(s, leafPaths)
		case *ast.DoBlock:
			for _, b := range s.Block {
				walkStmt(b)
			}
			return
		case *ast.If:
			walkExpr(s.Cond, leafPaths)
			addLeaves(leafPaths)
			for _, b := range s.Then {
				walkStmt(b)
			}
			for _, b := range s.Else {
				walkStmt(b)
			}
			return
		case *ast.WhileLoop:
			walkExpr(s.Cond, leafPaths)
			addLeaves(leafPaths)
			for _, b := range s.Block {
				walkStmt(b)
			}
			return
		case *ast.RepeatUntilLoop:
			for _, b := range s.Block {
				walkStmt(b)
			}
			leafPaths = make(map[string]bool)
			walkExpr(s.Cond, leafPaths)
			addLeaves(leafPaths)
			return
		case *ast.ForLoopNumeric:
			for _, b := range s.Block {
				walkStmt(b)
			}
			return
		case *ast.ForLoopGeneric:
			for _, b := range s.Block {
				walkStmt(b)
			}
			return
		case *ast.Return:
			for _, it := range s.Items {
				walkExpr(it, leafPaths)
			}
		case *ast.Goto, *ast.Label:
			return
		}
		addLeaves(leafPaths)
	}

	for _, stmt := range body {
		walkStmt(stmt)
	}
	for _, expr := range reevalExprs {
		leafPaths := make(map[string]bool)
		walkExpr(expr, leafPaths)
		addLeaves(leafPaths)
	}
	return counts
}

func applyLicmHoist(loopStmt ast.Stmt, body []ast.Stmt, path string, func_decl *ast.FuncDecl) bool {
	headerLine := loopStmt.Line()
	if headerLine < 1 || headerLine > len(gfilecontent) {
		return false
	}

	minL, maxL := find_stmt_line_range(loopStmt)
	if minL < 1 {
		minL = headerLine
	}
	if maxL < minL {
		maxL = headerLine
	}

	baseName := table_access_to_local_name(path)
	localName := getUniqueLocalName(func_decl.Block, baseName)

	nestedRanges := collectNestedFuncLineRanges(body)

	for lineNum := minL; lineNum <= maxL && lineNum <= len(gfilecontent); lineNum++ {
		if lineInRanges(lineNum, nestedRanges) {
			continue
		}
		line := gfilecontent[lineNum-1]
		if strings.Contains(line, "-- opt by oLua") {
			continue
		}
		gfilecontent[lineNum-1] = replace_table_access(line, path, localName)
	}

	indent := get_content_space(gfilecontent[headerLine-1])
	insertLine := indent + "local " + localName + " = " + path + " -- opt by oLua"

	var filecontent []string
	filecontent = append(filecontent, gfilecontent[:headerLine-1]...)
	filecontent = append(filecontent, insertLine)
	filecontent = append(filecontent, gfilecontent[headerLine-1:]...)
	gfilecontent = filecontent

	log.Printf("opt licm at: %s:%d path=%s -> %s", gfilename, headerLine, path, localName)
	goptcount++
	has_opt = true
	return true
}
