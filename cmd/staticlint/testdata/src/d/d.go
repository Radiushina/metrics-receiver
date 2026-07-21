package main

import (
	myos "os"
	o "os"
)

func main() {
	myos.Exit(1) // want "не используйте os.Exit в функции main"
	o.Exit(2)    // want "не используйте os.Exit в функции main"
}
