package main

import (
	"github.com/milochristiansen/lua/ast"
	"math"
	"strings"
)

type lua_visitor struct {
	f func(n ast.Node, ok *bool)
}

func (lv *lua_visitor) Visit(n ast.Node) ast.Visitor {
	if n != nil {
		ok := true
		lv.f(n, &ok)
		if ok {
			return lv
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func find_end_line(block []ast.Stmt, stmt ast.Stmt) int {

	var next_stmt ast.Stmt
	var next_index int
	for i, s := range block {
		if s == stmt {
			next_index = i
		}
	}

	for i := next_index + 1; i < len(block); i++ {
		empty := false
		switch block[i].(type) {
		case *ast.DoBlock:
			if len(block[i].(*ast.DoBlock).Block) == 0 {
				empty = true
			}
			break
		}
		if !empty {
			next_stmt = block[i]
			break
		}
	}

	if next_stmt == nil {
		start_line := stmt.Line()
		cur := 0
		for i := start_line; i < len(gfilecontent); i++ {
			line := gfilecontent[i-1]
			left := strings.Count(line, "{")
			cur += left
			right := strings.Count(line, "}")
			cur -= right
			if cur == 0 {
				return i + 1
			}
		}
		return len(gfilecontent) + 1
	} else {
		minline := math.MaxInt32
		f := lua_visitor{f: func(n ast.Node, ok *bool) {
			if n != nil {
				line := n.Line()
				if line < minline {
					minline = line
				}
			}
		}}
		ast.Walk(&f, next_stmt)
		return minline
	}
}

func check_expr_same(left ast.Expr, right ast.Expr) bool {
	switch left.(type) {
	case *ast.ConstIdent:
		left_ident := left.(*ast.ConstIdent)
		switch right.(type) {
		case *ast.ConstIdent:
			right_ident := right.(*ast.ConstIdent)
			if left_ident.Value == right_ident.Value {
				return true
			}
		}
	case *ast.TableAccessor:
		left_accessor := left.(*ast.TableAccessor)
		switch right.(type) {
		case *ast.TableAccessor:
			right_accessor := right.(*ast.TableAccessor)
			if check_expr_same(left_accessor.Obj, right_accessor.Obj) && check_expr_same(left_accessor.Key, right_accessor.Key) {
				return true
			}
		}
	}
	return false
}

func get_content_space(content string) string {
	for index, c := range content {
		if c != ' ' && c != '\t' {
			return content[:index]
		}
	}
	return ""
}
