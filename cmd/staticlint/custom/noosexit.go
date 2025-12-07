package custom

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer запрещает прямой вызов os.Exit в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "запрещает прямой вызов os.Exit в функции main пакета main",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Фильтруем вызовы функций и объявления функций
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.FuncDecl)(nil),
	}

	// Флаг, указывающий, находимся ли мы в функции main
	inMain := false
	// Стек для отслеживания вложенности функций (на случай вложенных функций)
	funcStack := []string{}

	inspector.Nodes(nodeFilter, func(node ast.Node, push bool) (proceed bool) {
		// Обрабатываем вход и выход из узлов
		if !push {
			// Выход из узла
			if _, ok := node.(*ast.FuncDecl); ok {
				// Выходим из функции, удаляем из стека
				if len(funcStack) > 0 {
					funcStack = funcStack[:len(funcStack)-1]
				}
				// Обновляем флаг inMain
				inMain = len(funcStack) > 0 && funcStack[len(funcStack)-1] == "main"
			}
			return true
		}

		// Вход в узел
		switch n := node.(type) {
		case *ast.FuncDecl:
			// Входим в функцию
			funcName := n.Name.Name
			funcStack = append(funcStack, funcName)
			inMain = funcName == "main"

		case *ast.CallExpr:
			// Проверяем вызов функции
			if !inMain {
				return true
			}

			// Проверяем, является ли это вызовом os.Exit
			fun, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := fun.X.(*ast.Ident)
			if !ok {
				return true
			}

			// Проверяем, что это вызов os.Exit
			if ident.Name == "os" && fun.Sel.Name == "Exit" {
				// Проверяем, находимся ли мы в пакете main
				if strings.HasSuffix(pass.Pkg.Path(), "main") {
					pass.Reportf(n.Pos(),
						"прямой вызов os.Exit в функции main пакета main запрещен. "+
							"Используйте возврат из main или панику с восстановлением для корректного завершения программы")
				}
			}
		}

		return true
	})

	return nil, nil
}
