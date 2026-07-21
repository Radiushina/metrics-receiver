// Package buildinfo печатает информацию о сборке в stdout.
package buildinfo

import "fmt"

// Print выводит версию, дату и коммит сборки.
func Print(version, date, commit string) {
	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}
