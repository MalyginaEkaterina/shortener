package main

import (
	"4d63.com/gochecknoglobals/checknoglobals"
	sqlclosecheck "github.com/ryanrolds/sqlclosecheck/pkg/analyzer"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// Checks adds analyzers into multichecker
func Checks() []*analysis.Analyzer {
	var checks []*analysis.Analyzer
	checks = append(checks, copylock.Analyzer)
	checks = append(checks, httpresponse.Analyzer)
	checks = append(checks, loopclosure.Analyzer)
	checks = append(checks, printf.Analyzer)
	checks = append(checks, shadow.Analyzer)
	checks = append(checks, structtag.Analyzer)
	staticChecks := map[string]bool{
		"S1010":  true,
		"ST1005": true,
		"QF1002": true,
	}
	for _, v := range staticcheck.Analyzers {
		checks = append(checks, v.Analyzer)
	}
	for _, v := range stylecheck.Analyzers {
		if staticChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}
	for _, v := range simple.Analyzers {
		if staticChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}
	for _, v := range quickfix.Analyzers {
		if staticChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}
	checks = append(checks, ExitCheckAnalyzer)
	checks = append(checks, checknoglobals.Analyzer())
	checks = append(checks, sqlclosecheck.NewAnalyzer())
	return checks
}
