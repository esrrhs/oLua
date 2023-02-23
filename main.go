package main

import (
	"bufio"
	"flag"
	"github.com/milochristiansen/lua/ast"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var input = flag.String("input", "input.lua", "Input file")
var inputpath = flag.String("inputpath", "", "Input path")
var output = flag.String("output", "output.lua", "Output file")

var has_opt bool
var gfilename string
var goptcount int

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)

	if *inputpath != "" {
		opt_path(*inputpath)
	} else {
		opt(*input, *output)
	}
}

func opt_path(inputpath string) {
	filepath.Walk(inputpath, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".lua") {
			return nil
		}
		log.Println("start opt_path:", path)
		opt(path, "./tmp.lua")
		if goptcount > 0 {
			os.Remove(path)
			os.Rename("./tmp.lua", path)
		} else {
			os.Remove("./tmp.lua")
		}
		return nil
	})
}

func opt(input string, output string) {
	gfilename = input
	goptcount = 0
	read_file(input)
	write_file(output)
	has_opt = true
	for has_opt {
		has_opt = false
		read_file(output)
		parse_lua()
		opt_lua()
		write_file(output)
	}
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
		log.Fatalf("%v %v", gfilename, err)
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

func find_first_table_access(block []ast.Stmt) (bool, []ast.Stmt, string, int, int) {
	for _, stmt := range block {
		switch nn := stmt.(type) {
		case *ast.Assign:
			assign := stmt.(*ast.Assign)
			is_new := false
			if len(assign.Values) == 1 {
				switch assign.Values[0].(type) {
				case *ast.TableConstructor:
					is_new = true
				}
			}
			if is_new {
				switch assign.Targets[0].(type) {
				case *ast.TableAccessor:
					obj := assign.Targets[0].(*ast.TableAccessor).Obj
					// check obj is _G
					is_G := false
					switch obj.(type) {
					case *ast.ConstIdent:
						if obj.(*ast.ConstIdent).Value == "_G" {
							is_G = true
						}
					}
					if !is_G {
						line := assign.Targets[0].(*ast.TableAccessor).Line()
						content := gfilecontent[line-1]
						content = strings.TrimSpace(content)
						if strings.HasPrefix(content, "local ") {
							content = strings.Replace(content, "local ", "", 1)
						}
						params := strings.Split(content, "=")
						if len(params) >= 2 {
							params[0] = strings.TrimSpace(params[0])
							if has_used_table_access(block, line, params[0]) {
								endline := find_end_line(block, stmt)
								log.Println("first_table_access_assign:", params[0], " ", line, " ", endline)
								return true, block, params[0], line, endline
							}
						}
					}
				}
			}
		case *ast.DoBlock:
			ok, ret_block, ret_string, ret_line, ret_end_line := find_first_table_access(nn.Block)
			if ok {
				return true, ret_block, ret_string, ret_line, ret_end_line
			}
		case *ast.If:
			ok, ret_block, ret_string, ret_line, ret_end_line := find_first_table_access(nn.Then)
			if ok {
				return true, ret_block, ret_string, ret_line, ret_end_line
			}
			ok, ret_block, ret_string, ret_line, ret_end_line = find_first_table_access(nn.Else)
			if ok {
				return true, ret_block, ret_string, ret_line, ret_end_line
			}
		case *ast.WhileLoop:
			ok, ret_block, ret_string, ret_line, ret_end_line := find_first_table_access(nn.Block)
			if ok {
				return true, ret_block, ret_string, ret_line, ret_end_line
			}
		case *ast.RepeatUntilLoop:
			ok, ret_block, ret_string, ret_line, ret_end_line := find_first_table_access(nn.Block)
			if ok {
				return true, ret_block, ret_string, ret_line, ret_end_line
			}
		case *ast.ForLoopNumeric:
			ok, ret_block, ret_string, ret_line, ret_end_line := find_first_table_access(nn.Block)
			if ok {
				return true, ret_block, ret_string, ret_line, ret_end_line
			}
		case *ast.ForLoopGeneric:
			ok, ret_block, ret_string, ret_line, ret_end_line := find_first_table_access(nn.Block)
			if ok {
				return true, ret_block, ret_string, ret_line, ret_end_line
			}
		}
	}
	return false, nil, "", 0, 0
}

func opt_func(func_decl *ast.FuncDecl) {
	first_table_access_assign_new_str := ""
	first_line := 0

	// find first assign: xxx.yyy.zzz = {xxx = yyy}
	ok, first_block, first_table_access_assign_new_str, first_line, end_line := find_first_table_access(func_decl.Block)
	if !ok {
		return
	}

	has_opt = true

	new_table_access_assign_new_str := table_access_to_local_name(first_table_access_assign_new_str)

	replace_used_table_access(first_block, first_line, first_table_access_assign_new_str, new_table_access_assign_new_str)

	// insert local define
	insert_line := get_content_space(gfilecontent[first_line-1]) + "local " + new_table_access_assign_new_str + " = " + first_table_access_assign_new_str + " -- opt by lua2lua"

	var filecontent []string
	filecontent = append(filecontent, gfilecontent[:end_line-1]...)
	filecontent = append(filecontent, insert_line)
	filecontent = append(filecontent, gfilecontent[end_line-1:]...)
	gfilecontent = filecontent

	log.Printf("opt at: %s:%d", gfilename, first_line)
	goptcount++
}

func get_content_space(content string) string {
	for index, c := range content {
		if c != ' ' && c != '\t' {
			return content[:index]
		}
	}
	return ""
}

func replace_used_table_access(block []ast.Stmt, first_line int, src string, dst string) {

	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			line := n.Line()
			if line > first_line {
				gfilecontent[line-1] = replace_table_access(gfilecontent[line-1], src, dst)
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}

}

func table_access_to_local_name(name string) string {
	ret := strings.ReplaceAll(name, ".", "_")
	ret = strings.ReplaceAll(ret, ":", "_")
	ret = strings.ReplaceAll(ret, "[", "_")
	ret = strings.ReplaceAll(ret, "]", "_")
	ret = strings.ReplaceAll(ret, "\"", "_")
	ret = strings.ReplaceAll(ret, "'", "_")
	return ret
}

func has_used_table_access(block []ast.Stmt, line int, dst string) bool {
	ret := 0
	find_line := make(map[int]int)
	f := lua_visitor{f: func(n ast.Node, ok *bool) {
		if n != nil {
			if n.Line() > line && find_line[n.Line()] == 0 {
				find_line[n.Line()] = 1
				content := gfilecontent[n.Line()-1]
				contain := contain_table_access(content, dst)
				if contain > 0 {
					if !strings.Contains(content, "local "+table_access_to_local_name(dst)+" = "+dst) {
						ret += contain
					}
				}
			}
		}
	}}
	for _, stmt := range block {
		ast.Walk(&f, stmt)
	}
	return ret > 1
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

func contain_table_access(content string, src string) int {
	ret := 0
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
			// back must not be a-zA-Z0-9_
			if (tmp[idx+len(src)] >= 'a' && tmp[idx+len(src)] <= 'z') || (tmp[idx+len(src)] >= 'A' && tmp[idx+len(src)] <= 'Z') || (tmp[idx+len(src)] >= '0' && tmp[idx+len(src)] <= '9') || tmp[idx+len(src)] == '_' {
				// use next
				begin = idx + len(src)
				continue
			}
		}
		begin = idx + len(src)
		ret++
	}
	return ret
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
			if (tmp[idx+len(src)] >= 'a' && tmp[idx+len(src)] <= 'z') || (tmp[idx+len(src)] >= 'A' && tmp[idx+len(src)] <= 'Z') || (tmp[idx+len(src)] >= '0' && tmp[idx+len(src)] <= '9') || tmp[idx+len(src)] == '_' {
				// use next
				begin = idx + len(src)
				continue
			}
		}
		tmp = tmp[:idx] + dst + tmp[idx+len(src):]
	}
	return tmp
}
