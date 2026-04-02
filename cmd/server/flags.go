package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

var flagRunAddr string

func parseFlags() (exitCode int, err error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, flag.ErrHelp
		}
		return 1, fmt.Errorf("ошибка флагов: %v\n", err)
	}
	return 0, nil
}
