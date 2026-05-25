package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Options struct {
	Domain          string
	InputFile       string
	OutputFile      string
	CSVFile         string
	ReportFile      string
	DryRun          bool
	Backup          bool
	RetrieveStocks  bool
	EnableCrypto    bool
	TopHoldings     int
	HoldingViewID   string
	BondsInFunds    bool
	SegregateBonds  bool
	CountryByRegion bool
}

func Parse(args []string) (Options, error) {
	fs := flag.NewFlagSet("pp-portfolio-classifier", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Println("Usage:")
		fmt.Println("  pp-classifier <input_file> [output_file] [flags]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  pp-classifier portfolio.xml")
		fmt.Println("  pp-classifier portfolio.xml pp_classified.xml -stocks -country_by_region")
		fmt.Println("  pp-classifier portfolio.xml out.xml -bonds_in_funds -top_holdings 1000 -stocks -crypto")
		fmt.Println("")
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println("")
		fmt.Println("Notes:")
		fmt.Println("  - Positional arguments and flags can be mixed in Python-style order.")
		fmt.Println("  - Supported -top_holdings values: 0, 10, 25, 50, 100, 1000, 3200.")
		fmt.Println("  - -crypto enables the internal crypto catalog for crypto classification.")
	}

	var opts Options
	var topHoldings string

	fs.StringVar(&opts.Domain, "d", "de", "Morningstar domain used to retrieve the auth token")
	fs.StringVar(&opts.CSVFile, "csv", "pp_data_fetched.csv", "CSV output file for fetched classification data")
	fs.StringVar(&opts.ReportFile, "report", "", "write a human-readable classification report to this file")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "fetch and classify without writing XML or CSV output")
	opts.Backup = true
	fs.BoolFunc("no_backup", "do not create a .bak file before overwriting an existing output file", func(string) error {
		opts.Backup = false
		return nil
	})
	fs.BoolVar(&opts.RetrieveStocks, "stocks", false, "retrieve classifications for individual stocks")
	fs.BoolVar(&opts.EnableCrypto, "crypto", false, "enable crypto classification using the internal crypto catalog")
	fs.StringVar(&topHoldings, "top_holdings", "10", "0, 10, 25, 50, 100, 1000 or 3200")
	fs.BoolVar(&opts.BondsInFunds, "bonds_in_funds", false, "include bond information in funds")
	fs.BoolVar(&opts.SegregateBonds, "seg_bonds", false, "segregate bond-related country and region categories")
	fs.BoolVar(&opts.CountryByRegion, "country_by_region", false, "create taxonomy Country@Region")

	if err := fs.Parse(normalizeArgs(args)); err != nil {
		return Options{}, err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return Options{}, errors.New("input file required")
	}
	opts.InputFile = rest[0]
	opts.OutputFile = "pp_classified.xml"
	if len(rest) > 1 {
		opts.OutputFile = rest[1]
	}

	switch topHoldings {
	case "0":
		opts.HoldingViewID = "Top10"
		opts.TopHoldings = 0
	case "10":
		opts.HoldingViewID = "Top10"
		opts.TopHoldings = -1
	case "25":
		opts.HoldingViewID = "Top25"
		opts.TopHoldings = -1
	case "50", "100", "1000":
		opts.HoldingViewID = "Allholdings"
		n, _ := strconv.Atoi(topHoldings)
		opts.TopHoldings = n
	case "3200":
		opts.HoldingViewID = "Allholdings"
		opts.TopHoldings = -1
	default:
		return Options{}, fmt.Errorf("unsupported -top_holdings value %q", topHoldings)
	}

	return opts, nil
}

func normalizeArgs(args []string) []string {
	var flags []string
	var positionals []string

	boolFlags := map[string]bool{
		"-stocks":            true,
		"-crypto":            true,
		"-dry-run":           true,
		"-no_backup":         true,
		"-bonds_in_funds":    true,
		"-seg_bonds":         true,
		"-country_by_region": true,
	}

	valueFlags := map[string]bool{
		"-d":            true,
		"-csv":          true,
		"-report":       true,
		"-top_holdings": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if boolFlags[arg] {
				continue
			}
			if valueFlags[arg] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positionals = append(positionals, arg)
	}

	return append(flags, positionals...)
}
