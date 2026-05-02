package config

import "testing"

func TestParseSupportsPythonStyleArgumentOrder(t *testing.T) {
	opts, err := Parse([]string{
		"input.xml",
		"output.xml",
		"-bonds_in_funds",
		"-top_holdings", "1000",
		"-country_by_region",
		"-stocks",
		"-crypto",
		"-csv", "custom.csv",
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if opts.InputFile != "input.xml" {
		t.Fatalf("unexpected input file: %q", opts.InputFile)
	}
	if opts.OutputFile != "output.xml" {
		t.Fatalf("unexpected output file: %q", opts.OutputFile)
	}
	if !opts.BondsInFunds || !opts.CountryByRegion || !opts.RetrieveStocks || !opts.EnableCrypto {
		t.Fatalf("expected bool flags to be set: %+v", opts)
	}
	if opts.TopHoldings != 1000 || opts.HoldingViewID != "Allholdings" {
		t.Fatalf("unexpected top holdings config: %+v", opts)
	}
	if opts.CSVFile != "custom.csv" {
		t.Fatalf("unexpected csv file: %q", opts.CSVFile)
	}
}

func TestParseRejectsRemovedVoapaFlag(t *testing.T) {
	if _, err := Parse([]string{"-voapa", "2026", "input.xml"}); err == nil {
		t.Fatalf("expected removed -voapa flag to be rejected")
	}
}
