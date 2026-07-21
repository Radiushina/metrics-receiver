package main

import (
	"go/ast"
	"go/types"

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
				if !isOSExitCall(pass, call) {
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

// isOSExitCall сообщает, является ли вызов функцией os.Exit.
// Учитывает любой псевдоним импорта (myos "os", o "os", . "os" и т.д.):
// идентификация идёт по пути пакета объекта из pass.TypesInfo.Uses, а не по имени идентификатора.
func isOSExitCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	if pass.TypesInfo == nil {
		return false
	}

	var obj types.Object
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		// myos.Exit / os.Exit — смотрим на идентификатор метода Exit
		obj = pass.TypesInfo.Uses[fun.Sel]
	case *ast.Ident:
		// import . "os" → Exit(1)
		obj = pass.TypesInfo.Uses[fun]
	default:
		return false
	}

	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}
	return fn.Pkg().Path() == "os" && fn.Name() == "Exit"
}
