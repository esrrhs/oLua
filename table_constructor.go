package main

import (
	"fmt"
	"github.com/milochristiansen/lua/ast"
	"log"
	"strings"
)

func find_first_table_constructor(block []ast.Stmt) (bool, []ast.Stmt, ast.Stmt, int) {
	for _, stmt := range block {
		switch nn := stmt.(type) {
		case *ast.Assign:
			assign := stmt.(*ast.Assign)
			is_new := false
			if len(assign.Values) == 1 {
				switch assign.Values[0].(type) {
				case *ast.TableConstructor:
					tc := assign.Values[0].(*ast.TableConstructor)
					if len(tc.Keys) == 0 && len(tc.Vals) == 0 {
						is_new = true
					}
				}
			}
			if is_new {
				used_count := get_used_table_constructor_assign(block, stmt)
				if used_count > 0 {
					return true, block, stmt, used_count
				}
			}
		case *ast.DoBlock:
			ok, ret_block, ret_stmt, ret_used_count := find_first_table_constructor(nn.Block)
			if ok {
				return true, ret_block, ret_stmt, ret_used_count
			}
		case *ast.If:
			ok, ret_block, ret_stmt, ret_used_count := find_first_table_constructor(nn.Then)
			if ok {
				return true, ret_block, ret_stmt, ret_used_count
			}
			ok, ret_block, ret_stmt, ret_used_count = find_first_table_constructor(nn.Else)
			if ok {
				return true, ret_block, ret_stmt, ret_used_count
			}
		case *ast.WhileLoop:
			ok, ret_block, ret_stmt, ret_used_count := find_first_table_constructor(nn.Block)
			if ok {
				return true, ret_block, ret_stmt, ret_used_count
			}
		case *ast.RepeatUntilLoop:
			ok, ret_block, ret_stmt, ret_used_count := find_first_table_constructor(nn.Block)
			if ok {
				return true, ret_block, ret_stmt, ret_used_count
			}
		case *ast.ForLoopNumeric:
			ok, ret_block, ret_stmt, ret_used_count := find_first_table_constructor(nn.Block)
			if ok {
				return true, ret_block, ret_stmt, ret_used_count
			}
		case *ast.ForLoopGeneric:
			ok, ret_block, ret_stmt, ret_used_count := find_first_table_constructor(nn.Block)
			if ok {
				return true, ret_block, ret_stmt, ret_used_count
			}
		}
	}
	return false, nil, nil, 0
}

func get_used_table_constructor_assign(block []ast.Stmt, assign_stmt ast.Stmt) int {
	target := assign_stmt.(*ast.Assign).Targets[0]
	use_count := 0
	next := false
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
							switch accessor.Key.(type) {
							case *ast.ConstIdent:
								has_use = true
							case *ast.ConstString:
								has_use = true
							case *ast.ConstInt:
								has_use = true
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
	return use_count
}

func opt_func_table_constructor(func_decl *ast.FuncDecl) {
	ok, ret_block, ret_stmt, ret_used_count := find_first_table_constructor(func_decl.Block)
	if !ok {
		return
	}

	has_opt = true

	new_cons := replace_table_constructor_used(ret_block, ret_stmt, ret_used_count)
	log.Println("opt_func_table_constructor", new_cons)

	start_line := ret_stmt.Line()

	content := gfilecontent[start_line-1]
	left_content := content[:strings.Index(content, "=")]
	insert_line := left_content + " = {" + strings.Join(new_cons, ", ") + "}" + " -- opt by oLua"

	var filecontent []string
	filecontent = append(filecontent, gfilecontent[:start_line-1]...)
	filecontent = append(filecontent, insert_line)
	filecontent = append(filecontent, gfilecontent[start_line+ret_used_count:]...)
	gfilecontent = filecontent

	log.Printf("opt at: %s:%d", gfilename, start_line)
	goptcount++
}

func replace_table_constructor_used(block []ast.Stmt, assign_stmt ast.Stmt, used_count int) []string {
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
							content := gfilecontent[stmt.Line()-1]
							pot := strings.Index(content, "=")
							if pot > 0 {
								right := content[pot+1:]
								switch accessor.Key.(type) {
								case *ast.ConstIdent:
									key := accessor.Key.(*ast.ConstIdent).Value
									ret = append(ret, fmt.Sprintf("[%s] = %s", key, right))
								case *ast.ConstString:
									key := accessor.Key.(*ast.ConstString).Value
									ret = append(ret, fmt.Sprintf("['%s'] = %s", key, right))
								case *ast.ConstInt:
									key := accessor.Key.(*ast.ConstInt).Value
									ret = append(ret, fmt.Sprintf("[%s] = %s", key, right))
								}
							}
						}
					}
				}
			}
		}
	}
	return ret
}
