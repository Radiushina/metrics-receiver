package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// OSExitAnalyzer запрещает прямой вызов os.Exit в функции main пакета main.
var OSExitAnalyzer = &analysis.Analyzer{
	Name: "osexitanalyzer",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  runOSExit,
}

func runOSExit(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Body == nil {
				return true
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Exit" {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "os" {
					return true
				}

				pass.Reportf(call.Pos(), "не используйте os.Exit в функции main")
				return true
			})
			return true
		})
	}

	return nil, nil
}
