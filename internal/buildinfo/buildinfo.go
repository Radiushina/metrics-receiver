// Package buildinfo печатает информацию о сборке в stdout.
package buildinfo

import "fmt"

// Print выводит версию, дату и коммит сборки.
func Print(version, date, commit string) {
	_, _ = fmt.Printf("Build version: %s\n", version)
	_, _ = fmt.Printf("Build date: %s\n", date)
	_, _ = fmt.Printf("Build commit: %s\n", commit)
}
