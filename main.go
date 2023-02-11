package main

import (
	"bufio"
	"flag"
	"github.com/milochristiansen/lua/ast"
	"log"
	"os"
	"strings"
)

var input = flag.String("input", "input.lua", "Input file")
var output = flag.String("output", "output.lua", "Output file")

var has_opt bool

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	read_file(*input)
	write_file(*output)
	has_opt = true
	n := 0
	for has_opt {
		has_opt = false
		read_file(*output)
		parse_lua()
		opt_lua()
		write_file(*output)
		n++
	}
	log.Println("n:", n)
}

var gfilecontent []string

func read_file(filename string) {
	var filecontent []string
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		filecontent = append(filecontent, scanner.Text())
	}
	gfilecontent = filecontent
}

var gblock []ast.Stmt

func parse_lua() {
	source := ""
	for _, line := range gfilecontent {
		source += line + "\n"
	}

	block, err := ast.Parse(source, 1)
	if err != nil {
		log.Fatal(err)
	}
	gblock = block
}

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

func opt_lua() {
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if has_opt {
			*ok = false
			return
		}
		if n != nil {
			switch n.(type) {
			case *ast.FuncDecl:
				func_decl := n.(*ast.FuncDecl)
				opt_func(func_decl)
			}
		}
	}}
	for _, stmt := range gblock {
		ast.Walk(&f, stmt)
	}
}

func opt_func(func_decl *ast.FuncDecl) {
	block := func_decl.Block

	first_table_access_assign_new_str := ""
	first_line := 0

	// find first assign: xxx.yyy.zzz = New()
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if first_table_access_assign_new_str != "" {
			*ok = false
			return
		}
		if n != nil {
			switch n.(type) {
			case *ast.Assign:
				assign := n.(*ast.Assign)
				is_new := false
				if len(assign.Values) == 1 {
					switch assign.Values[0].(type) {
					case *ast.FuncCall:
						func_call := assign.Values[0].(*ast.FuncCall)
						if len(func_call.Args) == 0 {
							function := func_call.Function
							switch function.(type) {
							case *ast.ConstIdent:
								ident := function.(*ast.ConstIdent)
								if ident.Value == "New" {
									is_new = true
								}
							}
						}
					}
				}
				if is_new {
					switch assign.Targets[0].(type) {
					case *ast.TableAccessor:
						line := assign.Targets[0].(*ast.TableAccessor).Line()
						content := gfilecontent[line-1]
						content = strings.TrimSpace(content)
						if strings.HasPrefix(content, "local ") {
							content = strings.Replace(content, "local ", "", 1)
						}
						params := strings.Split(content, "=")
						if len(params) == 2 {
							params[0] = strings.TrimSpace(params[0])
							params[1] = strings.TrimSpace(params[1])
							if params[1] == "New()" {
								if g_first_table_access_assign_new_str_his[params[0]] == 0 {
									g_first_table_access_assign_new_str_his[params[0]] = 1
									first_table_access_assign_new_str = params[0]
									first_line = line
									log.Println("first_table_access_assign_new_str:", first_table_access_assign_new_str, " ", line)
								}
							}
						}
					}
				}
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}

	if first_table_access_assign_new_str == "" {
		return
	}

	has_opt = true

	new_table_access_assign_new_str := strings.ReplaceAll(first_table_access_assign_new_str, ".", "_")
	new_table_access_assign_new_str = strings.ReplaceAll(new_table_access_assign_new_str, ":", "_")
	new_table_access_assign_new_str = strings.ReplaceAll(new_table_access_assign_new_str, "[", "_")
	new_table_access_assign_new_str = strings.ReplaceAll(new_table_access_assign_new_str, "]", "_")
	new_table_access_assign_new_str = strings.ReplaceAll(new_table_access_assign_new_str, "\"", "_")
	new_table_access_assign_new_str = strings.ReplaceAll(new_table_access_assign_new_str, "'", "_")

	f = lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			line := n.Line()
			if line > first_line {
				gfilecontent[line-1] = replace_table_access(gfilecontent[line-1], first_table_access_assign_new_str, new_table_access_assign_new_str)
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}

	// insert local define
	var filecontent []string
	filecontent = append(filecontent, gfilecontent[:first_line]...)
	filecontent = append(filecontent, "local "+new_table_access_assign_new_str+" = "+first_table_access_assign_new_str)
	filecontent = append(filecontent, gfilecontent[first_line:]...)
	gfilecontent = filecontent
}

func write_file(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	for _, line := range gfilecontent {
		file.WriteString(line + "\n")
	}
}

func replace_table_access(content string, src string, dst string) string {
	tmp := content
	begin := 0
	for {
		idx := strings.Index(tmp[begin:], src)
		if idx == -1 {
			break
		}
		idx += begin
		if idx > 0 {
			// front must not be . and a-zA-Z0-9_
			if tmp[idx-1] == '.' || (tmp[idx-1] >= 'a' && tmp[idx-1] <= 'z') || (tmp[idx-1] >= 'A' && tmp[idx-1] <= 'Z') || (tmp[idx-1] >= '0' && tmp[idx-1] <= '9') || tmp[idx-1] == '_' {
				// use next
				begin = idx + len(src)
				continue
			}
		}
		if idx+len(src) < len(tmp) {
			// back must not be . and a-zA-Z0-9_
			if tmp[idx+len(src)] == '.' || (tmp[idx+len(src)] >= 'a' && tmp[idx+len(src)] <= 'z') || (tmp[idx+len(src)] >= 'A' && tmp[idx+len(src)] <= 'Z') || (tmp[idx+len(src)] >= '0' && tmp[idx+len(src)] <= '9') || tmp[idx+len(src)] == '_' {
				// use next
				begin = idx + len(src)
				continue
			}
		}
		tmp = tmp[:idx] + dst + tmp[idx+len(src):]
	}
	return tmp
}
