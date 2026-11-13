// Package main реализует multichecker — агрегатор статических анализаторов
// Go для проекта Y-URL-shortener.
//
// Состав анализаторов:
//
//   - стандартные passes из golang.org/x/tools/go/analysis/passes;
//   - все анализаторы класса SA из honnef.co/go/tools/staticcheck;
//   - ST1000, ST1020 из honnef.co/go/tools/stylecheck;
//   - S1000 из honnef.co/go/tools/simple;
//   - публичные анализаторы errcheck и whitespace;
//   - собственный noosexit — запрещает os.Exit в main пакета main.
//
// # Использование
//
// Соберите бинарник:
//
//	go build -o staticlint ./cmd/staticlint
//
// Запустите на проекте:
//
//	./staticlint ./...
//
// Или через go vet:
//
//	go vet -vettool=$(which staticlint) ./...
//
// # Собственный анализатор noosexit
//
// Запрещает прямой вызов os.Exit в функции main пакета main.
// Причина: os.Exit завершает программу немедленно и не даёт выполниться
// отложенным функциям (defer), что приводит к утечкам ресурсов
// (незакрытые подключения к БД, файлы, HTTP-клиенты).
package main

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/simple/s1000"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck/st1000"
	"honnef.co/go/tools/stylecheck/st1020"

	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/ultraware/whitespace"
)

func main() {
	analyzersPasses := []*analysis.Analyzer{
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		deepequalerrors.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilness.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	}

	var (
		checksNameSA = "SA"
		checksAll    []*analysis.Analyzer
	)

	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, checksNameSA) {
			checksAll = append(checksAll, v.Analyzer)
		}
	}

	checksAll = append(checksAll, st1000.SCAnalyzer.Analyzer, st1020.SCAnalyzer.Analyzer)
	checksAll = append(checksAll, analyzersPasses...)
	checksAll = append(checksAll, s1000.Analyzer)

	// Публичные анализаторы от третьей стороны.
	checksAll = append(checksAll,
		errcheck.Analyzer,           // необработанные ошибки
		whitespace.NewAnalyzer(nil), // лишние пробелы
	)

	// Собственный анализатор: запрет os.Exit в main().
	checksAll = append(checksAll, noosexit)

	multichecker.Main(checksAll...)
}
