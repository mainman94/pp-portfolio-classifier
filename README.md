# pp-portfolio-classifier

Go rewrite of the Portfolio Performance classifier based on Morningstar data.

The tool reads a Portfolio Performance XML file, updates or creates taxonomies, and writes a classified output XML plus a CSV dump of the fetched classification data.

## Status

Implemented:
- fund and ETF classification
- optional stock classification via `-stocks`
- optional crypto fallback classification via `-crypto`
- taxonomies: `Asset Type`, `Stock Style`, `Stock Sector`, `Bond Style`, `Bond Sector`, `Region`, `Country`, `Country@Region`, `Holding`
- Python-style CLI argument order such as `input.xml output.xml -stocks -country_by_region`

## Build

```bash
go build -o pp-classifier .
```

If your environment cannot use the default Go cache:

```bash
GOCACHE=/tmp/go-build GOPATH=/tmp/go go build -o pp-classifier .
```

## Usage

```bash
./pp-classifier <input_file> [output_file] [flags]
```

Examples:

```bash
./pp-classifier portfolio.xml
./pp-classifier portfolio.xml pp_classified.xml -stocks -country_by_region
./pp-classifier Hauptmann.xml Hauptmann_fin.xml -bonds_in_funds -top_holdings 1000 -country_by_region -stocks
./pp-classifier Hauptmann.xml Hauptmann_fin.xml -bonds_in_funds -top_holdings 1000 -country_by_region -stocks -crypto
./pp-classifier portfolio.xml pp_classified.xml -csv fetched.csv
./pp-classifier portfolio.xml pp_classified.xml -dry-run -report report.txt
```

## Main Flags

- `-d <domain>`
  Morningstar domain for the auth token. Default: `de`

- `-csv <file>`
  CSV output path for fetched classification data. Default: `pp_data_fetched.csv`

- `-report <file>`
  Write a human-readable report with failures, unknown mappings, and broad `Other` classifications.

- `-dry-run`
  Fetch and classify securities, update taxonomies in memory, and optionally write `-report`, but skip XML and CSV writes.

- `-no_backup`
  Disable automatic `.bak` creation before overwriting an existing XML or CSV output file.

- `-stocks`
  Enable classification for individual stocks.

- `-crypto`
  Enable internal crypto classification. Without this flag, crypto-specific fallback handling stays off.

- `-top_holdings <0|10|25|50|100|1000|3200>`
  Controls how holdings are retrieved.

- `-bonds_in_funds`
  Include bond-related classification for mixed and bond funds.

- `-seg_bonds`
  Separate bond-related region and country entries, for example `France (Bonds)`.

- `-country_by_region`
  Create or update the `Country@Region` taxonomy.

## Output

The tool writes:
- the classified XML output file
- the fetched classification CSV, defaulting to `pp_data_fetched.csv` in the current working directory
- an optional report when `-report` is set

It also prints progress while running, including fetch status per security and taxonomy update steps.
When XML or CSV output paths already exist, the previous file is copied to `<path>.bak` before overwrite unless `-no_backup` is set.

## Notes

- Use a copy of your Portfolio Performance XML, not the original.
- The input must be the classic PP XML format without `id` attributes.
- Morningstar coverage varies by instrument, so some products may still need manual cleanup.
- Unknown Morningstar mapping codes are preserved as raw categories and reported as warnings.
