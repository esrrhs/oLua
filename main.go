package main

import (
	"bufio"
	"flag"
	"github.com/milochristiansen/lua/ast"
	"log"
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
	if *opt_table_constructor {
		opt_func_table_constructor(func_decl)
		if has_opt {
			return
		}
	}
	if *opt_table_access {
		opt_func_table_access(func_decl)
		if has_opt {
			return
		}
	}
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
