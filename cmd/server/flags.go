package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

var flagRunAddr string

func parseFlags() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "ошибка флагов: %v\n", err)
		os.Exit(1)
	}
}
