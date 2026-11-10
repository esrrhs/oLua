package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/milochristiansen/lua/ast"
)

var input = flag.String("input", "input.lua", "Input file")
var inputpath = flag.String("inputpath", "", "Input path")
var output = flag.String("output", "output.lua", "Output file")

var opt_table_access = flag.Bool("opt_table_access", false, "Optimize table access")
var opt_table_access_threshold = flag.Int("opt_table_access_threshold", 2, "Minimum read count to trigger table access optimization")
var opt_table_access_pure_funcs = flag.String("opt_table_access_pure_funcs", "log_.*", "Legacy name-pattern hint; cannot bypass the private-table safety proof")
var opt_table_access_global = flag.Bool("opt_table_access_global", false, "Legacy compatibility flag; global tables still require a safety proof")
var opt_table_constructor = flag.Bool("opt_table_constructor", false, "Optimize table constructor")
var opt_const_fold = flag.Bool("opt_const_fold", false, "Constant folding and if true/false dead code elimination")
var unsafe_assume_static_globals = flag.Bool("unsafe_assume_static_globals", false, "UNSAFE: assume a stable plain _ENV, libraries and global functions with no rebinding or lookup effects")
var opt_localize_globals = flag.Bool("opt_localize_globals", false, "Localize frequently used globals/library functions")
var opt_localize_globals_threshold = flag.Int("opt_localize_globals_threshold", 2, "Minimum read count to trigger localize globals optimization")
var opt_licm = flag.Bool("opt_licm", false, "Loop-invariant code motion for proven private immutable table paths")
var opt_licm_threshold = flag.Int("opt_licm_threshold", 2, "Minimum body read count to trigger LICM (clamped to >= 2)")

var has_opt bool
var gfilename string
var goptcount int

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)

	if *inputpath != "" {
		opt_path(*inputpath)
	} else {
		if err := opt(*input, *output); err != nil {
			log.Fatal(err)
		}
	}
}

// opt_path 批量处理目录下所有 .lua 文件。
// 单个文件失败时记录错误并继续处理其他文件，不终止整个流程。
// 临时文件写在同目录（os.CreateTemp + *.olua.tmp），成功时直接 Rename 覆盖，不先 Remove 目标。
func opt_path(inputpath string) {
	failed := 0
	processed := 0
	filepath.Walk(inputpath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			log.Printf("walk error on %s: %v", path, err)
			return nil
		}
		if f.IsDir() {
			return nil
		}
		// Skip optimizer temp files (and anything that is not a real .lua source).
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".olua.tmp") || strings.Contains(base, ".olua.tmp") {
			return nil
		}
		if !strings.HasSuffix(path, ".lua") {
			return nil
		}
		log.Println("start opt_path:", path)
		tmp, err := os.CreateTemp(filepath.Dir(path), "oLua-*.olua.tmp")
		if err != nil {
			log.Printf("skip %s due to temp create error: %v", path, err)
			failed++
			return nil
		}
		tmpName := tmp.Name()
		if err := tmp.Close(); err != nil {
			os.Remove(tmpName)
			log.Printf("skip %s due to temp close error: %v", path, err)
			failed++
			return nil
		}
		if err := opt(path, tmpName); err != nil {
			log.Printf("skip %s due to error: %v", path, err)
			os.Remove(tmpName)
			failed++
			return nil
		}
		if goptcount > 0 {
			// Same-dir rename replaces the target atomically on Unix; do not Remove first.
			if err := os.Rename(tmpName, path); err != nil {
				log.Printf("skip %s due to rename error: %v", path, err)
				os.Remove(tmpName)
				failed++
				return nil
			}
		} else {
			os.Remove(tmpName)
		}
		processed++
		return nil
	})
	log.Printf("opt_path done: %d processed, %d failed", processed, failed)
}

// opt 对单个文件执行迭代优化。返回 error 而不是直接 fatal，让批量模式可以跳过失败文件。
func opt(input string, output string) error {
	gfilename = input
	goptcount = 0
	if err := read_file(input); err != nil {
		return fmt.Errorf("read_file failed: %w", err)
	}
	if err := write_file(output); err != nil {
		return fmt.Errorf("write_file failed: %w", err)
	}
	has_opt = true
	for iteration := 0; has_opt; iteration++ {
		if iteration >= 1000 {
			return fmt.Errorf("optimizer did not converge after 1000 iterations")
		}
		has_opt = false
		if err := read_file(output); err != nil {
			return fmt.Errorf("read_file failed: %w", err)
		}
		if err := parse_lua(); err != nil {
			return fmt.Errorf("parse_lua failed: %w", err)
		}
		opt_lua()
		if err := write_file(output); err != nil {
			return fmt.Errorf("write_file failed: %w", err)
		}
	}
	return nil
}

var gfilecontent []string

func read_file(filename string) error {
	var filecontent []string
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		filecontent = append(filecontent, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	gfilecontent = filecontent
	return nil
}

var gblock []ast.Stmt

func parse_lua() error {
	source := ""
	for _, line := range gfilecontent {
		source += line + "\n"
	}

	block, err := ast.Parse(source, 1)
	if err != nil {
		return fmt.Errorf("%v: %v", gfilename, err)
	}
	gblock = block
	return nil
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
	if *opt_const_fold {
		opt_func_const_fold(func_decl)
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
	if *opt_localize_globals {
		opt_func_localize_globals(func_decl)
		if has_opt {
			return
		}
	}
	if *opt_licm {
		opt_func_licm(func_decl)
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

func write_file(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, line := range gfilecontent {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return nil
}
