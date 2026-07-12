// Package main — multichecker для статического анализа проекта metrics-receiver.
//
// # Запуск
//
// Из корня репозитория:
//
//	go run ./cmd/staticlint ./...
//
// Или после сборки:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// Multichecker принимает пути пакетов так же, как go vet.
//
// # Состав анализаторов
//
// Стандартные анализаторы golang.org/x/tools/go/analysis/passes:
// asmdecl, assign, atomic, bools, buildtag, cgocall, composites, copylock,
// defers, directive, errorsas, framepointer, httpresponse, ifaceassert, loopclosure,
// lostcancel, nilfunc, nilness, printf, shift, sigchanyzer, slog, stdmethods,
// stdversion, stringintconv, structtag, testinggoroutine, tests, timeformat,
// unmarshal, unsafeptr, unusedresult, waitgroup и др.
//
// Staticcheck SA* — все анализаторы класса SA (simple checks):
// проверки ошибок, подозрительных конструкций, устаревших API и т.п.
//
// Staticcheck ST1000 — анализатор класса ST: рекомендует godoc-комментарий
// у экспортируемых имён пакета.
//
// Публичные анализаторы:
//   - bodyclose — проверяет закрытие HTTP response.Body;
//   - nakedret — предупреждает о «голых» return с именованными результатами.
//
// Собственный анализатор:
//   - osexitanalyzer — запрещает os.Exit внутри func main() пакета main.
package main
