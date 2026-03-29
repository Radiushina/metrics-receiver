package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

var flagRunAddr string
var flagPollInterval int64
var flagReportInterval int64

func parseFlags() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&flagRunAddr, "a", "localhost:8080", "HTTP server host:port (scheme http:// added automatically)")
	fs.Int64Var(&flagPollInterval, "p", 2, "poll interval: how often to read runtime.MemStats (seconds)")
	fs.Int64Var(&flagReportInterval, "r", 10, "report interval: how often to send metrics to the server (seconds)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "ошибка флагов: %v\n", err)
		os.Exit(1)
	}
}

func serverBaseURL() string {
	s := strings.TrimSpace(flagRunAddr)
	if s == "" {
		s = "localhost:8080"
	}
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "/")
	return "http://" + s
}

func getPollInterval() time.Duration {
	return time.Duration(flagPollInterval) * time.Second
}

func getReportInterval() time.Duration {
	return time.Duration(flagReportInterval) * time.Second
}
