package main

import (
	"log"
	"sort"
	"strings"

	"github.com/milochristiansen/lua/ast"
)

// privateTablePaths proves a deliberately small closed subset, rather than
// trusting dotted spelling or a function-name purity whitelist. Tables must be
// freshly constructed in the function's outer block from literals, immutable,
// never passed to a call, never captured, and never shadowed. Every other read
// is a local scalar; no external code/metamethod can run between cached reads.
// The result maps existing paths to dominating declaration lines. Unknown means
// skip, not "probably pure". This is not a general Lua effect/type analysis.
func privateTablePaths(fd *ast.FuncDecl) map[string]int {
	if fd == nil || len(fd.Block) == 0 || fd.Block[0].Line() <= fd.Line() {
		return nil
	}
	type shape struct {
		fields map[string]*shape
	}
	scal := func() *shape { return &shape{} }
	env := map[string]*shape{}
	used := map[string]bool{}
	for _, p := range fd.Params {
		used[p] = true
	} // parameter values are unknown
	paths := map[string]int{}
	stmtLines := map[int]bool{}
	var literal func(ast.Expr) *shape
	literal = func(e ast.Expr) *shape {
		switch v := e.(type) {
		case *ast.ConstInt, *ast.ConstFloat, *ast.ConstString, *ast.ConstBool, *ast.ConstNil:
			return scal()
		case *ast.Parens:
			return literal(v.Inner)
		case *ast.Operator:
			// Negative numeric literal only; no runtime coercions or metamethods.
			if v.Op == ast.OpUMinus {
				r := v.Right
				if r == nil {
					r = v.Left
				}
				if _, ok := r.(*ast.ConstInt); ok {
					return scal()
				}
				if _, ok := r.(*ast.ConstFloat); ok {
					return scal()
				}
			}
		case *ast.TableConstructor:
			t := &shape{fields: map[string]*shape{}}
			for i, k := range v.Keys {
				key, ok := k.(*ast.ConstString)
				if !ok {
					return nil
				}
				value := literal(v.Vals[i])
				if value == nil {
					return nil
				}
				// Duplicate keys and constructors with sequence fields are not proven.
				if _, exists := t.fields[key.Value]; exists {
					return nil
				}
				t.fields[key.Value] = value
			}
			return t
		}
		return nil
	}
	var expr func(ast.Expr) *shape
	expr = func(e ast.Expr) *shape {
		if e == nil {
			return scal()
		}
		switch v := e.(type) {
		case *ast.ConstIdent:
			return env[v.Value]
		case *ast.TableAccessor:
			parent := expr(v.Obj)
			key, ok := v.Key.(*ast.ConstString)
			if parent == nil || parent.fields == nil || !ok {
				return nil
			}
			return parent.fields[key.Value] // missing values deliberately not proven
		case *ast.Parens:
			return expr(v.Inner)
		case *ast.Operator:
			a, b := expr(v.Left), expr(v.Right)
			if a != nil && b != nil && a.fields == nil && b.fields == nil {
				return scal()
			}
			return nil
		default:
			return literal(e)
		}
	}
	var addPaths func(string, *shape, int)
	addPaths = func(path string, t *shape, line int) {
		paths[path] = line
		for key, v := range t.fields {
			if key != "" && isLuaIdentStart(key[0]) && !isLuaKeyword(key) {
				valid := true
				for i := 1; i < len(key); i++ {
					if !isLuaIdentPart(key[i]) {
						valid = false
					}
				}
				if valid {
					addPaths(path+"."+key, v, line)
				}
			}
		}
	}
	var block func([]ast.Stmt, bool) bool
	block = func(stmts []ast.Stmt, outer bool) bool {
		if !outer {
			saved := make(map[string]*shape, len(env))
			for name, value := range env {
				saved[name] = value
			}
			defer func() { env = saved }()
		}
		for _, stmt := range stmts {
			// The legacy access rewriter inserts whole lines. Reject inline statements
			// and tokens spanning lines, which cannot be edited with that granularity.
			line := stmt.Line()
			if line <= fd.Line() || stmtLines[line] {
				return false
			}
			stmtLines[line] = true
			switch s := stmt.(type) {
			case *ast.Assign:
				if s.LocalFunc || len(s.Targets) != 1 || len(s.Values) != 1 {
					return false
				}
				id, ok := s.Targets[0].(*ast.ConstIdent)
				if !ok {
					return false
				} // no table/global writes
				val := expr(s.Values[0])
				if val == nil {
					return false
				}
				if s.LocalDecl {
					if used[id.Value] {
						return false
					}
					used[id.Value] = true
					if val.fields != nil {
						if !outer {
							return false
						}
						// A table-valued alias is safe only because all table writes/calls are
						// forbidden for this entire function.
						addPaths(id.Value, val, line)
					}
					env[id.Value] = val
				} else {
					old := env[id.Value]
					if old == nil || old.fields != nil || val.fields != nil {
						return false
					}
				}
			case *ast.Return:
				for _, v := range s.Items {
					if expr(v) == nil {
						return false
					}
				}
			case *ast.ForLoopNumeric:
				if used[s.Counter] || expr(s.Init) == nil || expr(s.Limit) == nil || expr(s.Step) == nil {
					return false
				}
				used[s.Counter] = true
				env[s.Counter] = scal()
				if !block(s.Block, false) {
					return false
				}
				delete(env, s.Counter)
			case *ast.WhileLoop:
				if expr(s.Cond) == nil || !block(s.Block, false) {
					return false
				}
			case *ast.RepeatUntilLoop:
				if !block(s.Block, false) || expr(s.Cond) == nil {
					return false
				}
			case *ast.If:
				if expr(s.Cond) == nil || !block(s.Then, false) || !block(s.Else, false) {
					return false
				}
			case *ast.DoBlock:
				if !block(s.Block, false) {
					return false
				}
			default:
				return false // calls, closures, generic iterators, goto/break, etc.
			}
		}
		return true
	}
	// Lua limits simultaneous locals to 200; keep headroom for cache aliases.
	if !block(fd.Block, true) || len(used) >= 190 {
		return nil
	}
	// Block-local scalars cannot affect table safety, but raw multiline source
	// editing can. Until the rewriter is fully token-span based, reject it.
	lo, hi := funcBodyLineRange(fd)
	if lo < 1 || hi > len(gfilecontent) {
		return nil
	}
	source := strings.Join(gfilecontent[lo-1:hi], "\n")
	for _, tok := range luaSourceTokens(source) {
		if tok.literal && strings.Contains(tok.text, "\n") {
			return nil
		}
	}
	return paths
}

// In the proven immutable subset there are no invalidation groups. Insert once
// in the outer function scope, after the root declaration and before the first
// containing statement. Never introduce a branch-local alias used elsewhere.
func optimizePrivateTableAccess(fd *ast.FuncDecl) {
	if has_opt {
		return
	}
	proven := privateTablePaths(fd)
	if proven == nil {
		return
	}
	threshold := *opt_table_access_threshold
	if threshold < 2 {
		threshold = 2
	}
	counts := countLicmBodyReads(fd.Block, nil)
	var candidates []string
	for path, count := range counts {
		if count >= threshold && strings.Contains(path, ".") && proven[path] > 0 && !isOluaGeneratedName(strings.Split(path, ".")[0]) {
			candidates = append(candidates, path)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if strings.Count(a, ".") != strings.Count(b, ".") {
			return strings.Count(a, ".") > strings.Count(b, ".")
		}
		if counts[a] != counts[b] {
			return counts[a] > counts[b]
		}
		return a < b
	})
	for _, path := range candidates {
		first := 0
		for _, stmt := range fd.Block {
			if stmtContainsRead(stmt, path) {
				first = stmt.Line()
				break
			}
		}
		if first <= proven[path] {
			continue
		}
		_, last := funcBodyLineRange(fd)
		name := getUniqueLocalName(fd.Block, table_access_to_local_name(path))
		changed := false
		for line := first; line <= last; line++ {
			if !strings.Contains(gfilecontent[line-1], "-- opt by oLua") {
				before := gfilecontent[line-1]
				gfilecontent[line-1] = replace_table_access(before, path, name)
				changed = changed || before != gfilecontent[line-1]
			}
		}
		if !changed {
			continue
		}
		insertion := get_content_space(gfilecontent[first-1]) + "local " + name + " = " + path + " -- opt by oLua"
		lines := append([]string{}, gfilecontent[:first-1]...)
		lines = append(lines, insertion)
		lines = append(lines, gfilecontent[first-1:]...)
		gfilecontent = lines
		goptcount++
		has_opt = true
		log.Printf("opt private table_access at: %s:%d path=%s", gfilename, first, path)
		return
	}
}
