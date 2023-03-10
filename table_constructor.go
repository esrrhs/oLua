package main

import (
	"github.com/milochristiansen/lua/ast"
	"log"
	"strings"
)

func find_last_table_constructor(block []ast.Stmt) (bool, []ast.Stmt, ast.Stmt, int, int) {
	var r_ok bool
	var r_block []ast.Stmt
	var r_stmt ast.Stmt
	var r_used_count int
	var r_end_line int
	for _, stmt := range block {
		switch nn := stmt.(type) {
		case *ast.Assign:
			assign := stmt.(*ast.Assign)
			is_new := false
			if len(assign.Values) == 1 {
				switch assign.Values[0].(type) {
				case *ast.TableConstructor:
					if can_expr_to_string(assign.Targets[0]) {
						is_new = true
					}
				}
			}
			if is_new {
				used_count, end_line := get_used_table_constructor_assign(block, stmt)
				if used_count > 0 {
					r_ok, r_block, r_stmt, r_used_count, r_end_line = true, block, stmt, used_count, end_line
				}
			}
		case *ast.DoBlock:
			ok, ret_block, ret_stmt, ret_used_count, ret_end_line := find_last_table_constructor(nn.Block)
			if ok {
				r_ok, r_block, r_stmt, r_used_count, r_end_line = true, ret_block, ret_stmt, ret_used_count, ret_end_line
			}
		case *ast.If:
			ok, ret_block, ret_stmt, ret_used_count, ret_end_line := find_last_table_constructor(nn.Then)
			if ok {
				r_ok, r_block, r_stmt, r_used_count, r_end_line = true, ret_block, ret_stmt, ret_used_count, ret_end_line
			}
			ok, ret_block, ret_stmt, ret_used_count, ret_end_line = find_last_table_constructor(nn.Else)
			if ok {
				r_ok, r_block, r_stmt, r_used_count, r_end_line = true, ret_block, ret_stmt, ret_used_count, ret_end_line
			}
		case *ast.WhileLoop:
			ok, ret_block, ret_stmt, ret_used_count, ret_end_line := find_last_table_constructor(nn.Block)
			if ok {
				r_ok, r_block, r_stmt, r_used_count, r_end_line = true, ret_block, ret_stmt, ret_used_count, ret_end_line
			}
		case *ast.RepeatUntilLoop:
			ok, ret_block, ret_stmt, ret_used_count, ret_end_line := find_last_table_constructor(nn.Block)
			if ok {
				r_ok, r_block, r_stmt, r_used_count, r_end_line = true, ret_block, ret_stmt, ret_used_count, ret_end_line
			}
		case *ast.ForLoopNumeric:
			ok, ret_block, ret_stmt, ret_used_count, ret_end_line := find_last_table_constructor(nn.Block)
			if ok {
				r_ok, r_block, r_stmt, r_used_count, r_end_line = true, ret_block, ret_stmt, ret_used_count, ret_end_line
			}
		case *ast.ForLoopGeneric:
			ok, ret_block, ret_stmt, ret_used_count, ret_end_line := find_last_table_constructor(nn.Block)
			if ok {
				r_ok, r_block, r_stmt, r_used_count, r_end_line = true, ret_block, ret_stmt, ret_used_count, ret_end_line
			}
		}
	}
	return r_ok, r_block, r_stmt, r_used_count, r_end_line
}

func get_used_table_constructor_assign(block []ast.Stmt, assign_stmt ast.Stmt) (int, int) {
	target := assign_stmt.(*ast.Assign).Targets[0]
	use_count := 0
	next := false
	var last_value ast.Node
	for _, stmt := range block {
		if stmt == assign_stmt {
			next = true
			continue
		}
		if next {
			has_use := false
			switch stmt.(type) {
			case *ast.Assign:
				assign := stmt.(*ast.Assign)
				if len(assign.Targets) == 1 {
					switch assign.Targets[0].(type) {
					case *ast.TableAccessor:
						accessor := assign.Targets[0].(*ast.TableAccessor)
						obj := accessor.Obj
						if check_expr_same(obj, target) {
							if can_expr_to_string(assign.Values[0]) {
								switch accessor.Key.(type) {
								case *ast.ConstIdent:
									has_use = true
								case *ast.ConstString:
									has_use = true
								case *ast.ConstInt:
									has_use = true
								}
								last_value = assign.Values[0]
							}
						}
					}
				}
			}
			if has_use {
				use_count++
			} else {
				break
			}
		}
	}
	_, end_line := find_stmt_line_range(last_value)
	return use_count, end_line
}

func opt_func_table_constructor(func_decl *ast.FuncDecl) {
	ok, ret_block, ret_stmt, ret_used_count, ret_end_line := find_last_table_constructor(func_decl.Block)
	if !ok {
		return
	}

	has_opt = true

	new_cons := replace_table_constructor_used(ret_block, ret_stmt, ret_used_count)
	log.Println("opt_func_table_constructor", new_cons)

	start_line := ret_stmt.Line()

	content := gfilecontent[start_line-1]
	left_content := content[:strings.Index(content, "=")]
	insert_line := strings.TrimRight(left_content, " ") + " = {" + strings.Join(new_cons, ", ") + "}" + " -- opt by oLua"

	var filecontent []string
	filecontent = append(filecontent, gfilecontent[:start_line-1]...)
	filecontent = append(filecontent, insert_line)
	filecontent = append(filecontent, gfilecontent[start_line+ret_end_line-start_line:]...)
	gfilecontent = filecontent

	log.Printf("opt at: %s:%d", gfilename, start_line)
	goptcount++
}

func replace_table_constructor_used(block []ast.Stmt, assign_stmt ast.Stmt, used_count int) []string {
	var new_keys []ast.Expr
	var new_vals []ast.Expr

	old_value := assign_stmt.(*ast.Assign).Values[0]
	switch old_value.(type) {
	case *ast.TableConstructor:
		old_cons := old_value.(*ast.TableConstructor)
		for _, k := range old_cons.Keys {
			new_keys = append(new_keys, k)
		}
		for _, v := range old_cons.Vals {
			new_vals = append(new_vals, v)
		}
	}

	next := false
	c := 0
	var ret []string
	for _, stmt := range block {
		if stmt == assign_stmt {
			next = true
			continue
		}
		if next {
			if c < used_count {
				c++
				switch stmt.(type) {
				case *ast.Assign:
					assign := stmt.(*ast.Assign)
					if len(assign.Targets) == 1 {
						switch assign.Targets[0].(type) {
						case *ast.TableAccessor:
							accessor := assign.Targets[0].(*ast.TableAccessor)
							new_keys = append(new_keys, accessor.Key)
							new_vals = append(new_vals, assign.Values[0])
						}
					}
				}
			}
		}
	}

	for i, k := range new_keys {
		if k == nil {
			ret = append(ret, expr_to_string(new_vals[i]))
			continue
		}
		switch k.(type) {
		case *ast.ConstIdent:
			ret = append(ret, "["+k.(*ast.ConstIdent).Value+"]="+expr_to_string(new_vals[i]))
		case *ast.ConstString:
			ret = append(ret, "['"+k.(*ast.ConstString).Value+"']="+expr_to_string(new_vals[i]))
		case *ast.ConstInt:
			ret = append(ret, "["+k.(*ast.ConstInt).Value+"]="+expr_to_string(new_vals[i]))
		}
	}

	return ret
}
