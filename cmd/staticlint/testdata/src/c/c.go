package c

import "os"

// Пакет не main — даже при вызове os.Exit в функции main замечаний быть не должно.
func main() {
	os.Exit(1)
}
