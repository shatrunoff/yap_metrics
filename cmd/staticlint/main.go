// Multichecker - Расширенный статический анализатор Go
//
// Этот инструмент объединяет:
// 1. Стандартные анализаторы golang.org/x/tools/go/analysis/passes
// 2. Все анализаторы класса SA staticcheck.io
// 3. По одному анализатору из других классов staticcheck.io
// 4. Два дополнительных публичных анализатора
// 5. Пользовательский анализатор noosexit
//
// Использование:
//
//	staticlint [флаги] [пакеты...]
//
// Флаги:
//
//	-json       Вывод в формате JSON
//	-enable     Включить конкретный анализатор
//	-disable    Отключить конкретный анализатор
//
// Примеры:
//
//	staticlint ./...                     # Проверить все пакеты
//	staticlint -json .                   # Проверить текущий пакет, вывод в JSON
//	staticlint -enable.SA1000 ./...      # Включить только SA1000
//	staticlint -disable.printf ./...     # Отключить printf анализатор
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/shatrunoff/yap_metrics/cmd/staticlint/custom"
)

func main() {
	var analyzers []*analysis.Analyzer

	// Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
	analyzers = append(analyzers,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		findcall.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		pkgfact.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		usesgenerics.Analyzer,
	)

	// Все анализаторы класса SA из staticcheck
	for _, v := range staticcheck.Analyzers {
		// Берем только анализаторы класса SA (Style & Accuracy)
		if len(v.Analyzer.Name) > 2 && v.Analyzer.Name[0:2] == "SA" {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// По одному анализатору из других классов staticcheck.io
	// Вспомогательные переменные для хранения найденных анализаторов
	var simpleAnalyzer, styleAnalyzer, quickfixAnalyzer *analysis.Analyzer
	var simpleAnalyzerFound, styleAnalyzerFound, quickfixAnalyzerFound bool

	// Ищем нужные анализаторы в staticcheck
	for _, v := range staticcheck.Analyzers {
		name := v.Analyzer.Name

		// S1000 - Simple: Упрощение булевых выражений
		if name == "S1000" {
			simpleAnalyzer = v.Analyzer
			simpleAnalyzerFound = true
		}

		// QF1001 - Quickfix: Упрощение range
		if name == "QF1001" {
			quickfixAnalyzer = v.Analyzer
			quickfixAnalyzerFound = true
		}

		// S1012 - Simple: Упрощение кода
		if name == "S1012" {
			quickfixAnalyzer = v.Analyzer
			quickfixAnalyzerFound = true
		}

		// U1000 - Unused code: Поиск неиспользуемого кода
		if name == "U1000" {
			quickfixAnalyzer = v.Analyzer
			quickfixAnalyzerFound = true
		}
	}

	// Ищем ST1000 в stylecheck
	for _, v := range stylecheck.Analyzers {
		if v.Analyzer.Name == "ST1000" {
			styleAnalyzer = v.Analyzer
			styleAnalyzerFound = true
			break
		}
	}

	// Добавляем найденные анализаторы
	if simpleAnalyzerFound {
		analyzers = append(analyzers, simpleAnalyzer)
	}
	if styleAnalyzerFound {
		analyzers = append(analyzers, styleAnalyzer)
	}
	if quickfixAnalyzerFound {
		analyzers = append(analyzers, quickfixAnalyzer)
	}

	// Два дополнительных публичных анализатора
	// Ищем S1012 и U1000
	var s1012Analyzer, u1000Analyzer *analysis.Analyzer
	var s1012Found, u1000Found bool

	for _, v := range staticcheck.Analyzers {
		name := v.Analyzer.Name
		if name == "S1012" {
			s1012Analyzer = v.Analyzer
			s1012Found = true
		}
		if name == "U1000" {
			u1000Analyzer = v.Analyzer
			u1000Found = true
		}

		// Можно выйти раньше, если оба найдены
		if s1012Found && u1000Found {
			break
		}
	}

	// Добавляем дополнительные анализаторы
	if s1012Found {
		analyzers = append(analyzers, s1012Analyzer)
	}
	if u1000Found {
		analyzers = append(analyzers, u1000Analyzer)
	}

	// Пользовательский анализатор noosexit
	analyzers = append(analyzers, custom.Analyzer)

	multichecker.Main(analyzers...)
}
