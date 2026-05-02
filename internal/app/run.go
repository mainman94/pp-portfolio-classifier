package app

import (
	"context"
	"fmt"

	"pp-portfolio-classifier/internal/config"
	"pp-portfolio-classifier/internal/model"
	"pp-portfolio-classifier/internal/morningstar"
	"pp-portfolio-classifier/internal/ppxml"
	"pp-portfolio-classifier/internal/taxonomy"
)

func Run(ctx context.Context, opts config.Options) error {
	fmt.Printf("input=%s output=%s csv=%s domain=%s stocks=%t crypto=%t bonds_in_funds=%t top_holdings=%d country_by_region=%t\n",
		opts.InputFile, opts.OutputFile, opts.CSVFile, opts.Domain, opts.RetrieveStocks, opts.EnableCrypto, opts.BondsInFunds, opts.TopHoldings, opts.CountryByRegion)

	file, err := ppxml.Load(opts.InputFile)
	if err != nil {
		return err
	}

	securities, err := file.Securities()
	if err != nil {
		return err
	}

	client := morningstar.New(opts.Domain)
	var active []*model.Security
	failures := 0
	for i, security := range securities {
		if security.IsRetired {
			fmt.Printf("[%d/%d] %s (%s): skipped retired\n", i+1, len(securities), security.Name, security.ISIN)
			continue
		}
		fmt.Printf("[%d/%d] %s (%s): fetching\n", i+1, len(securities), security.Name, security.ISIN)
		report, err := client.LoadReport(ctx, security, opts)
		if err != nil {
			fmt.Printf("[%d/%d] %s (%s): error: %v\n", i+1, len(securities), security.Name, security.ISIN, err)
			failures++
			continue
		}
		security.Report = report
		fmt.Printf("[%d/%d] %s (%s): ok type=%s fund_type=%s resolved=%s\n", i+1, len(securities), security.Name, security.ISIN, report.Type, report.FundType, report.Security)
		if security.AltISIN != "" {
			fmt.Printf("[%d/%d] %s: trying fallback ISIN %s\n", i+1, len(securities), security.Name, security.AltISIN)
			altSecurity := *security
			altSecurity.ISIN = security.AltISIN
			if alt, err := client.LoadReport(ctx, &altSecurity, opts); err == nil {
				security.AltReport = alt
				fmt.Printf("[%d/%d] %s: fallback ISIN loaded\n", i+1, len(securities), security.Name)
			} else {
				fmt.Printf("[%d/%d] %s: fallback ISIN failed: %v\n", i+1, len(securities), security.Name, err)
			}
		}
		active = append(active, security)
	}
	if len(active) == 0 && failures > 0 {
		return fmt.Errorf("unable to retrieve data for any security")
	}

	for _, def := range taxonomy.Ordered {
		if def.Name == "Country@Region" && !opts.CountryByRegion {
			continue
		}
		fmt.Printf("updating taxonomy: %s\n", def.Name)
		if err := file.UpdateTaxonomy(def.Name, active, opts.TopHoldings, opts.CountryByRegion); err != nil {
			return err
		}
	}

	fmt.Printf("writing xml: %s\n", opts.OutputFile)
	if err := file.Write(opts.OutputFile); err != nil {
		return err
	}
	fmt.Printf("writing csv: %s\n", opts.CSVFile)
	if err := file.DumpCSV(opts.CSVFile, active); err != nil {
		return err
	}
	fmt.Printf("done: %d securities updated, %d failed\n", len(active), failures)
	return nil
}
