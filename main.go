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

var opt_table_access = flag.Bool("opt_table_access", false, "Optimize table access")
var opt_table_constructor = flag.Bool("opt_table_constructor", false, "Optimize table constructor")

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

func opt_func(func_decl *ast.FuncDecl) {
	if *opt_table_access {
		opt_func_table_access(func_decl)
		if has_opt {
			return
		}
	}
	if *opt_table_constructor {
		opt_func_table_constructor(func_decl)
		if has_opt {
			return
		}
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
