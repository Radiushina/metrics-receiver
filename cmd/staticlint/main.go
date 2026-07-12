package main

import (
	"strings"

	"github.com/alexkohler/nakedret/v2"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	multichecker.Main(analyzers()...)
}

func analyzers() []*analysis.Analyzer {
	result := make([]*analysis.Analyzer, 0, 128)
	result = append(result, standardPasses()...)
	result = append(result, staticcheckSA()...)
	if a := staticcheckST1000(); a != nil {
		result = append(result, a)
	}
	result = append(result,
		bodyclose.Analyzer,
		nakedret.NakedReturnAnalyzer(&nakedret.NakedReturnRunner{MaxLength: 5}),
		OSExitAnalyzer,
	)
	return result
}

func staticcheckSA() []*analysis.Analyzer {
	out := make([]*analysis.Analyzer, 0, 40)
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			out = append(out, a.Analyzer)
		}
	}
	return out
}

func staticcheckST1000() *analysis.Analyzer {
	for _, a := range staticcheck.Analyzers {
		if a.Analyzer.Name == "ST1000" {
			return a.Analyzer
		}
	}
	return nil
}
