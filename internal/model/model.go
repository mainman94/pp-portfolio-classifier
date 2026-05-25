package model

type Security struct {
	Name      string
	ISIN      string
	Ticker    string
	Feed      string
	CoinID    string
	UUID      string
	IsRetired bool
	Note      string
	SourceRef string
	AltISIN   string
	Report    *HoldingReport
	AltReport *HoldingReport
}

type HoldingReport struct {
	SecID      string
	Security   string
	Type       string
	FundType   string
	Taxonomies map[string]map[string]float64
	Warnings   []string
}

func NewHoldingReport() *HoldingReport {
	return &HoldingReport{
		Taxonomies: map[string]map[string]float64{},
	}
}
