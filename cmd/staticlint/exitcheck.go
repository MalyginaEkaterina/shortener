package main

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
)

// ExitCheckAnalyzer analyzer for check for using os.Exit in main() of main package
var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check for using os.Exit in main() of main package",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.File:
				if x.Name.Name == "main" {
					return true
				} else {
					return false
				}
			case *ast.FuncDecl:
				if x.Name.Name == "main" {
					return true
				} else {
					return false
				}
			case *ast.CallExpr:
				if f, ok := x.Fun.(*ast.SelectorExpr); ok {
					if p, ok := f.X.(*ast.Ident); ok {
						if p.Name == "os" && f.Sel.Name == "Exit" {
							pass.Reportf(x.Pos(), "os.Exit call is used in main() of main package")
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
