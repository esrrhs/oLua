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
	case *ast.FuncCall:
		left_call := left.(*ast.FuncCall)
		switch right.(type) {
		case *ast.FuncCall:
			right_call := right.(*ast.FuncCall)
			if check_expr_same(left_call.Function, right_call.Function) {
				if len(left_call.Args) == len(right_call.Args) {
					for i := 0; i < len(left_call.Args); i++ {
						if !check_expr_same(left_call.Args[i], right_call.Args[i]) {
							return false
						}
					}
					return true
				}
			}
		}
	case *ast.TableConstructor:
		left_constructor := left.(*ast.TableConstructor)
		switch right.(type) {
		case *ast.TableConstructor:
			right_constructor := right.(*ast.TableConstructor)
			if len(left_constructor.Keys) == len(right_constructor.Keys) {
				for i := 0; i < len(left_constructor.Keys); i++ {
					if !check_expr_same(left_constructor.Keys[i], right_constructor.Keys[i]) {
						return false
					}
					if !check_expr_same(left_constructor.Vals[i], right_constructor.Vals[i]) {
						return false
					}
				}
				return true
			}
		}
	case *ast.ConstString:
		left_string := left.(*ast.ConstString)
		switch right.(type) {
		case *ast.ConstString:
			right_string := right.(*ast.ConstString)
			if left_string.Value == right_string.Value {
				return true
			}
		}
	case *ast.ConstInt:
		left_int := left.(*ast.ConstInt)
		switch right.(type) {
		case *ast.ConstInt:
			right_int := right.(*ast.ConstInt)
			if left_int.Value == right_int.Value {
				return true
			}
		}
	case *ast.ConstFloat:
		left_float := left.(*ast.ConstFloat)
		switch right.(type) {
		case *ast.ConstFloat:
			right_float := right.(*ast.ConstFloat)
			if left_float.Value == right_float.Value {
				return true
			}
		}
	case *ast.ConstBool:
		left_bool := left.(*ast.ConstBool)
		switch right.(type) {
		case *ast.ConstBool:
			right_bool := right.(*ast.ConstBool)
			if left_bool.Value == right_bool.Value {
				return true
			}
		}
	case *ast.ConstNil:
		switch right.(type) {
		case *ast.ConstNil:
			return true
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

func expr_to_string(expr ast.Expr) string {
	expr_str := ""
	switch expr.(type) {
	case *ast.ConstIdent:
		expr_str = expr.(*ast.ConstIdent).Value
	case *ast.TableAccessor:
		expr_str = expr_to_string(expr.(*ast.TableAccessor).Obj) + "." + expr_to_string(expr.(*ast.TableAccessor).Key)
	case *ast.FuncCall:
		expr_str = expr_to_string(expr.(*ast.FuncCall).Function) + "("
		for i, arg := range expr.(*ast.FuncCall).Args {
			if i > 0 {
				expr_str += ","
			}
			expr_str += expr_to_string(arg)
		}
		expr_str += ")"
	case *ast.TableConstructor:
		expr_str = "{"
		for i, field := range expr.(*ast.TableConstructor).Keys {
			if i > 0 {
				expr_str += ","
			}
			switch field.(type) {
			case *ast.ConstIdent:
				expr_str += "[" + field.(*ast.ConstIdent).Value + "]"
			case *ast.ConstString:
				expr_str += "['" + field.(*ast.ConstString).Value + "']"
			case *ast.ConstInt:
				expr_str += "[" + field.(*ast.ConstInt).Value + "]"
			}
			expr_str += "="
			expr_str += expr_to_string(expr.(*ast.TableConstructor).Vals[i])
		}
		expr_str += "}"
	case *ast.ConstString:
		expr_str = expr.(*ast.ConstString).Value
	case *ast.ConstInt:
		expr_str = expr.(*ast.ConstInt).Value
	case *ast.ConstFloat:
		expr_str = expr.(*ast.ConstFloat).Value
	case *ast.ConstNil:
		expr_str = "nil"
	case *ast.ConstBool:
		if expr.(*ast.ConstBool).Value {
			expr_str = "true"
		} else {
			expr_str = "false"
		}
	}
	return expr_str
}

func can_expr_to_string(expr ast.Expr) bool {
	ret := false
	switch expr.(type) {
	case *ast.ConstIdent:
		ret = true
	case *ast.TableAccessor:
		ret = can_expr_to_string(expr.(*ast.TableAccessor).Obj) && can_expr_to_string(expr.(*ast.TableAccessor).Key)
	case *ast.FuncCall:
		ret = can_expr_to_string(expr.(*ast.FuncCall).Function)
		for _, arg := range expr.(*ast.FuncCall).Args {
			ret = ret && can_expr_to_string(arg)
		}
	case *ast.TableConstructor:
		for i, field := range expr.(*ast.TableConstructor).Keys {
			switch field.(type) {
			case *ast.ConstIdent:
			case *ast.ConstString:
			case *ast.ConstInt:
			default:
				return false
			}
			ret = ret && can_expr_to_string(expr.(*ast.TableConstructor).Vals[i])
		}
	case *ast.ConstString:
		ret = true
	case *ast.ConstInt:
		ret = true
	case *ast.ConstFloat:
		ret = true
	case *ast.ConstNil:
		ret = true
	case *ast.ConstBool:
		ret = true
	}
	return ret
}
