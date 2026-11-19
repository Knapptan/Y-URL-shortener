// Package main содержит пользовательский анализатор noosexit,
// который запрещает прямой вызов os.Exit() в функции main пакета main.
//
// Анализатор noosexit:
//
// Проверяет, что в функции main пакета main нет прямых вызовов os.Exit().
// Это необходимо для корректной очистки ресурсов при завершении работы приложения.
// Вместо os.Exit() рекомендуется использовать log.Fatal() или return с обработкой ошибки,
// что позволяет deferred функциям корректно выполнить очистку ресурсов.
//
// Пример нарушения:
//
//	package main
//
//	import "os"
//
//	func main() {
//	    if err := doSomething(); err != nil {
//	        os.Exit(1) // нарушение правила
//	    }
//	}
//
// Правильный подход:
//
//	package main
//
//	import (
//	    "log"
//	)
//
//	func main() {
//	    if err := doSomething(); err != nil {
//	        log.Fatal(err) // корректная обработка - deferred функции выполнятся
//	    }
//	}
package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noosexit проверяет, что в функции main пакета main нет вызовов os.Exit.
var noosexit = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "forbids direct os.Exit calls in the main function of the main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	// Пропускаем пакеты из чужих модулей (зависимости, тулчейн, кэш).
	pkgPath := pass.Pkg.Path()
	if !strings.HasPrefix(pkgPath, "github.com/Knapptan/Y-URL-shortener") &&
		!strings.HasPrefix(pkgPath, "mynoosexit") {
		return nil, nil
	}

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" {
			return
		}

		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Пропускаем не-.go файлы (кэш, скомпилированные артефакты).
			if f := pass.Fset.File(call.Pos()); f == nil || !strings.HasSuffix(f.Name(), ".go") {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "os" || sel.Sel.Name != "Exit" {
				return true
			}

			obj := pass.TypesInfo.Uses[ident]
			if obj == nil {
				return true
			}
			if _, ok := obj.(*types.PkgName); !ok {
				return true
			}

			pass.Reportf(call.Pos(), "direct call to os.Exit in main function is prohibited")
			return true
		})
	})

	return nil, nil
}
