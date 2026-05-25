package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"pp-portfolio-classifier/internal/config"
	"pp-portfolio-classifier/internal/model"
	"pp-portfolio-classifier/internal/morningstar"
	"pp-portfolio-classifier/internal/ppxml"
	"pp-portfolio-classifier/internal/taxonomy"
)

func Run(ctx context.Context, opts config.Options) error {
	fmt.Printf("input=%s output=%s csv=%s report=%s dry_run=%t domain=%s stocks=%t crypto=%t bonds_in_funds=%t top_holdings=%d country_by_region=%t\n",
		opts.InputFile, opts.OutputFile, opts.CSVFile, opts.ReportFile, opts.DryRun, opts.Domain, opts.RetrieveStocks, opts.EnableCrypto, opts.BondsInFunds, opts.TopHoldings, opts.CountryByRegion)

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
	var failed []string
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
			failed = append(failed, fmt.Sprintf("%s (%s): %v", security.Name, security.ISIN, err))
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

	if opts.ReportFile != "" {
		fmt.Printf("writing report: %s\n", opts.ReportFile)
		if err := writeReport(opts.ReportFile, active, failed); err != nil {
			return err
		}
	}
	if opts.DryRun {
		fmt.Printf("dry-run: skipped xml and csv writes\n")
		fmt.Printf("done: %d securities classified, %d failed\n", len(active), failures)
		return nil
	}

	fmt.Printf("writing xml: %s\n", opts.OutputFile)
	if opts.Backup {
		if err := backupExisting(opts.OutputFile); err != nil {
			return err
		}
	}
	if err := file.Write(opts.OutputFile); err != nil {
		return err
	}
	fmt.Printf("writing csv: %s\n", opts.CSVFile)
	if opts.Backup {
		if err := backupExisting(opts.CSVFile); err != nil {
			return err
		}
	}
	if err := file.DumpCSV(opts.CSVFile, active); err != nil {
		return err
	}
	fmt.Printf("done: %d securities updated, %d failed\n", len(active), failures)
	return nil
}

func backupExisting(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(path + ".bak")
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	fmt.Printf("backup written: %s\n", path+".bak")
	return nil
}

func writeReport(path string, securities []*model.Security, failures []string) error {
	var b strings.Builder
	b.WriteString("Portfolio Classification Report\n")
	b.WriteString("===============================\n\n")
	b.WriteString(fmt.Sprintf("Classified securities: %d\n", len(securities)))
	b.WriteString(fmt.Sprintf("Failed securities: %d\n\n", len(failures)))

	if len(failures) > 0 {
		b.WriteString("Failures\n")
		b.WriteString("--------\n")
		for _, failure := range failures {
			b.WriteString("- " + failure + "\n")
		}
		b.WriteString("\n")
	}

	warnings := collectWarnings(securities)
	if len(warnings) > 0 {
		b.WriteString("Warnings\n")
		b.WriteString("--------\n")
		for _, warning := range warnings {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}

	others := collectOtherClassifications(securities)
	if len(others) > 0 {
		b.WriteString("Broad Other Classifications\n")
		b.WriteString("---------------------------\n")
		for _, item := range others {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}

	if len(failures) == 0 && len(warnings) == 0 && len(others) == 0 {
		b.WriteString("No failures, warnings, or broad Other classifications found.\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func collectWarnings(securities []*model.Security) []string {
	seen := map[string]bool{}
	var out []string
	for _, security := range securities {
		if security == nil || security.Report == nil {
			continue
		}
		for _, warning := range security.Report.Warnings {
			line := fmt.Sprintf("%s (%s): %s", security.Name, security.ISIN, warning)
			if !seen[line] {
				seen[line] = true
				out = append(out, line)
			}
		}
	}
	sort.Strings(out)
	return out
}

func collectOtherClassifications(securities []*model.Security) []string {
	var out []string
	for _, security := range securities {
		if security == nil || security.Report == nil {
			continue
		}
		if weight := security.Report.Taxonomies["Asset Type"]["Other"]; weight > 0 {
			out = append(out, fmt.Sprintf("%s (%s): Other %.2f%%", security.Name, security.ISIN, weight))
		}
	}
	sort.Strings(out)
	return out
}
