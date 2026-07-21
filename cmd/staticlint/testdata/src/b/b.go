package main

import "os"

func main() {
	run()
}

func run() {
	// os.Exit вне func main — анализатор не должен срабатывать.
	os.Exit(1)
}
