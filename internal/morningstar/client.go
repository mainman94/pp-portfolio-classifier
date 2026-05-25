package morningstar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pp-portfolio-classifier/internal/config"
	"pp-portfolio-classifier/internal/crypto"
	"pp-portfolio-classifier/internal/model"
	"pp-portfolio-classifier/internal/taxonomy"
)

type Client struct {
	httpClient *http.Client
	domain     string
	token      string
}

func New(domain string) *Client {
	return &Client{
		domain: domain,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) LoadReport(ctx context.Context, security *model.Security, opts config.Options) (*model.HoldingReport, error) {
	if security == nil {
		return nil, fmt.Errorf("security is nil")
	}
	if security.ISIN == "" {
		if report := c.tryCryptoFallback(security, opts); report != nil {
			return report, nil
		}
		return nil, fmt.Errorf("missing isin")
	}
	isin := security.ISIN
	token, err := c.bearerToken(ctx)
	if err != nil {
		if report := c.tryCryptoFallback(security, opts); report != nil {
			return report, nil
		}
		return nil, err
	}

	snapshot, err := c.snapshot(ctx, token, isin, "snapshot")
	if err != nil {
		if report := c.tryCryptoFallback(security, opts); report != nil {
			return report, nil
		}
		return nil, err
	}
	report := model.NewHoldingReport()
	report.Security = asString(firstItem(snapshot, "Name"))
	if report.Security == "" {
		report.Security = security.Name
	}
	report.Type = asString(firstItem(snapshot, "Type"))

	for _, def := range taxonomy.Ordered {
		report.Taxonomies[def.Name] = map[string]float64{}
	}
	if report.Type == "" {
		if opts.EnableCrypto && crypto.IsCrypto(security) {
			forceCryptoClassification(report, security)
			return report, nil
		}
		return nil, fmt.Errorf("no snapshot type for %s", isin)
	}

	switch report.Type {
	case "Fund":
		if err := c.loadFund(ctx, token, isin, opts, report); err != nil {
			return nil, err
		}
	case "Stock":
		if !opts.RetrieveStocks {
			if opts.EnableCrypto && crypto.IsCrypto(security) {
				forceCryptoClassification(report, security)
			}
			return report, nil
		}
		c.loadStock(snapshot, report)
	default:
		if opts.EnableCrypto && crypto.IsCrypto(security) {
			forceCryptoClassification(report, security)
			return report, nil
		}
		return nil, fmt.Errorf("unsupported security type %q for %s", report.Type, isin)
	}
	if opts.EnableCrypto && crypto.IsCrypto(security) {
		forceCryptoClassification(report, security)
	}
	return report, nil
}

func (c *Client) loadFund(ctx context.Context, token, isin string, opts config.Options, report *model.HoldingReport) error {
	data, err := c.snapshot(ctx, token, isin, "ITsnapshot")
	if err != nil {
		return err
	}
	items := rootItems(data)
	if len(items) > 0 {
		report.FundType = asString(itemMap(items[0], "CategoryBroadAssetClass", "Name"))
	}
	if report.FundType == "" {
		report.FundType = "Equity"
	}

	netEquity := 1.0
	netBonds := 0.0
	if allocs := findAssetAllocations(data); len(allocs) > 0 {
		for _, alloc := range allocs {
			code := asString(alloc["Type"])
			value := asFloat(alloc["Value"])
			switch code {
			case "1":
				netEquity = min(1.0, value/100.0)
			case "3":
				netBonds = min(1.0, value/100.0)
			}
			name := mapAssetType(report, code)
			if name != "" && value > 0 {
				report.Taxonomies["Asset Type"][name] += value
			}
		}
	}
	if len(report.Taxonomies["Asset Type"]) == 0 {
		report.Taxonomies["Asset Type"] = inferFundAssetType(report.FundType)
	}

	addMapped(report, "Stock Style", findBreakdown(data, "StyleBoxBreakdown"), taxonomy.StockStyleMap, netEquity*100.0)
	addMapped(report, "Stock Sector", findBreakdown(data, "GlobalStockSectorBreakdown"), taxonomy.StockSectorMap, netEquity*100.0)

	if !opts.BondsInFunds {
		netBonds = 0
	}
	if netBonds > 0 {
		etfData, err := c.snapshot(ctx, token, isin, "ETFsnapshot")
		if err == nil {
			addMapped(report, "Bond Style", findBreakdown(etfData, "BondStyleBoxBreakdown"), taxonomy.BondStyleMap, netBonds*100.0)
		}
		addMapped(report, "Bond Sector", findBreakdown(data, "GlobalBondSectorBreakdownLevel1"), taxonomy.BondSectorMap, netBonds*100.0)
	}

	addGeo(report, "Region", findRegionalExposure(data), taxonomy.MapRegion, netEquity*100.0)
	addGeo(report, "Country", findCountryExposure(data, "Equity"), taxonomy.MapCountry, netEquity*100.0)

	if opts.BondsInFunds && netBonds > 0 {
		addGeo(report, "Region", findCountryExposure(data, "Bond"), func(code string) string {
			return taxonomy.BondSuffix(mapRegion(report, code), opts.SegregateBonds)
		}, netBonds*100.0)
		addGeo(report, "Country", findCountryExposure(data, "Bond"), func(code string) string {
			return taxonomy.BondSuffix(mapCountry(report, code), opts.SegregateBonds)
		}, netBonds*100.0)
	}

	if opts.CountryByRegion {
		for name, value := range report.Taxonomies["Country"] {
			if strings.HasSuffix(name, " (Bonds)") {
				continue
			}
			report.Taxonomies["Country@Region"][name] = value
		}
	}

	if opts.TopHoldings != 0 {
		holdings := findHoldings(data)
		limit := opts.TopHoldings
		if limit < 0 || limit > len(holdings) {
			limit = len(holdings)
		}
		for i := 0; i < limit; i++ {
			name := strings.TrimSpace(asString(holdings[i]["SecurityName"]))
			if name == "" {
				continue
			}
			weight := asFloat(holdings[i]["Weighting"])
			if opts.BondsInFunds || strings.HasPrefix(asString(holdings[i]["DetailHoldingTypeId"]), "E") {
				report.Taxonomies["Holding"][name] += weight
			}
		}
	}
	return nil
}

func (c *Client) loadStock(snapshot any, report *model.HoldingReport) {
	items := rootItems(snapshot)
	if len(items) == 0 {
		return
	}
	item := items[0]
	if v := mapAssetTypeStock(report, asString(item["Type"])); v != "" {
		report.Taxonomies["Asset Type"][v] = 100
	}
	if v := asString(itemMap(item, "Sector", "SectorCode")); v != "" {
		report.Taxonomies["Stock Sector"][mapOrWarn(report, "Stock Sector", v, taxonomy.StockSectorMap)] = 100
	}
	if v := asString(item["InvestmentStyle"]); v != "" {
		report.Taxonomies["Stock Style"][mapOrWarn(report, "Stock Style", v, taxonomy.StockStyleMap)] = 100
	}
	if v := asString(item["Country"]); v != "" {
		country := mapCountry(report, v)
		region := mapRegion(report, v)
		report.Taxonomies["Country"][country] = 100
		report.Taxonomies["Region"][region] = 100
		report.Taxonomies["Country@Region"][country] = 100
	}
	if v := asString(item["Name"]); v != "" {
		report.Taxonomies["Holding"][v] = 100
	}
}

func (c *Client) bearerToken(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://www.morningstar.%s/Common/funds/snapshot/PortfolioSAL.aspx", c.domain), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	match := regexp.MustCompile(`const maasToken = "([^"]+)"`).FindSubmatch(body)
	if len(match) != 2 {
		return "", fmt.Errorf("morningstar bearer token not found")
	}
	c.token = string(match[1])
	return c.token, nil
}

func (c *Client) snapshot(ctx context.Context, token, isin, viewID string) (any, error) {
	values := url.Values{}
	values.Set("idtype", "ISIN")
	values.Set("viewid", viewID)
	values.Set("currencyId", "EUR")
	values.Set("responseViewFormat", "json")
	values.Set("languageId", "en-UK")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.emea-api.morningstar.com/ecint/v1/securities/"+isin+"?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snapshot request failed for %s with %d", isin, resp.StatusCode)
	}
	var out any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func addMapped(report *model.HoldingReport, taxonomyName string, rows []map[string]any, mapping map[string]string, scale float64) {
	for _, row := range rows {
		code := asString(row["Type"])
		value := asFloat(row["Value"])
		name := mapOrWarn(report, taxonomyName, code, mapping)
		if name == "" || value <= 0 {
			continue
		}
		report.Taxonomies[taxonomyName][name] += value * scale / 100.0
	}
}

func addGeo(report *model.HoldingReport, taxonomyName string, rows []map[string]any, mapper func(string) string, scale float64) {
	for _, row := range rows {
		code := asString(row["Type"])
		value := asFloat(row["Value"])
		name := mapper(code)
		if name == code {
			addWarning(report, "%s: unknown mapping code %q", taxonomyName, code)
		}
		if name == "" || value <= 0 {
			continue
		}
		report.Taxonomies[taxonomyName][name] += value * scale / 100.0
	}
}

func findAssetAllocations(data any) []map[string]any {
	var out []map[string]any
	for _, row := range findPortfolioRows(data, "AssetAllocations") {
		if asString(row["Type"]) == "MorningStarDefault" && asString(row["SalePosition"]) == "N" {
			out = append(out, mapsFromArray(row["BreakdownValues"])...)
		}
	}
	return out
}

func findBreakdown(data any, field string) []map[string]any {
	for _, row := range findPortfolioRows(data, field) {
		if asString(row["SalePosition"]) == "N" {
			return mapsFromArray(row["BreakdownValues"])
		}
	}
	return nil
}

func findRegionalExposure(data any) []map[string]any {
	return findBreakdown(data, "RegionalExposure")
}

func findCountryExposure(data any, secType string) []map[string]any {
	for _, row := range findPortfolioRows(data, "CountryExposure") {
		if asString(row["Type"]) == secType && asString(row["SalePosition"]) == "N" {
			return mapsFromArray(row["BreakdownValues"])
		}
	}
	return nil
}

func findHoldings(data any) []map[string]any {
	items := rootItems(data)
	if len(items) == 0 {
		return nil
	}
	portfolios := itemArray(items[0], "Portfolios")
	if len(portfolios) == 0 {
		return nil
	}
	return mapsFromArray(portfolios[0]["PortfolioHoldings"])
}

func findPortfolioRows(data any, field string) []map[string]any {
	items := rootItems(data)
	if len(items) == 0 {
		return nil
	}
	portfolios := itemArray(items[0], "Portfolios")
	if len(portfolios) == 0 {
		return nil
	}
	return mapsFromArray(portfolios[0][field])
}

func firstItem(data any, key string) any {
	items := rootItems(data)
	if len(items) == 0 {
		return nil
	}
	return items[0][key]
}

func rootItems(data any) []map[string]any {
	switch raw := data.(type) {
	case []any:
		return mapsFromArray(raw)
	case []map[string]any:
		return raw
	case map[string]any:
		return []map[string]any{raw}
	}
	return nil
}

func itemArray(item map[string]any, key string) []map[string]any {
	return mapsFromArray(item[key])
}

func itemMap(item any, keys ...string) any {
	current := item
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

func mapsFromArray(value any) []map[string]any {
	var out []map[string]any
	switch raw := value.(type) {
	case []any:
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
	case []map[string]any:
		return raw
	}
	return out
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func asFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

func mapOr(key string, mapping map[string]string) string {
	if value, ok := mapping[key]; ok {
		return value
	}
	return key
}

func mapOrWarn(report *model.HoldingReport, taxonomyName, key string, mapping map[string]string) string {
	if key == "" {
		return ""
	}
	if value, ok := mapping[key]; ok {
		return value
	}
	addWarning(report, "%s: unknown mapping code %q", taxonomyName, key)
	return key
}

func mapAssetType(report *model.HoldingReport, code string) string {
	if value, ok := taxonomy.AssetTypeMap[code]; ok {
		return value
	}
	if code != "" {
		addWarning(report, "Asset Type: unknown mapping code %q", code)
	}
	return code
}

func mapAssetTypeStock(report *model.HoldingReport, kind string) string {
	if value, ok := taxonomy.AssetTypeStockMap[kind]; ok {
		return value
	}
	if kind != "" {
		addWarning(report, "Asset Type: unknown stock type %q", kind)
	}
	return kind
}

func mapRegion(report *model.HoldingReport, code string) string {
	value := taxonomy.MapRegion(code)
	if value == code && code != "" {
		addWarning(report, "Region: unknown mapping code %q", code)
	}
	return value
}

func mapCountry(report *model.HoldingReport, code string) string {
	value := taxonomy.MapCountry(code)
	if value == code && code != "" {
		addWarning(report, "Country: unknown mapping code %q", code)
	}
	return value
}

func addWarning(report *model.HoldingReport, format string, args ...any) {
	if report == nil {
		return
	}
	message := fmt.Sprintf(format, args...)
	for _, existing := range report.Warnings {
		if existing == message {
			return
		}
	}
	report.Warnings = append(report.Warnings, message)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func forceCryptoClassification(report *model.HoldingReport, security *model.Security) {
	for name := range report.Taxonomies {
		report.Taxonomies[name] = map[string]float64{}
	}
	report.Type = "Crypto"
	report.Taxonomies["Asset Type"]["Crypto"] = 100.0
	if label, ok := crypto.Resolve(security); ok {
		report.Security = label
		if label != "Crypto" {
			report.Taxonomies["Holding"][label] = 100.0
		}
	} else if security != nil {
		report.Security = security.Name
	}
}

func (c *Client) tryCryptoFallback(security *model.Security, opts config.Options) *model.HoldingReport {
	if !opts.EnableCrypto {
		return nil
	}
	if !crypto.IsCrypto(security) {
		return nil
	}
	report := model.NewHoldingReport()
	report.Security = security.Name
	for _, def := range taxonomy.Ordered {
		report.Taxonomies[def.Name] = map[string]float64{}
	}
	forceCryptoClassification(report, security)
	return report
}

func inferFundAssetType(fundType string) map[string]float64 {
	clean := strings.TrimSpace(fundType)
	lower := strings.ToLower(clean)
	switch clean {
	case "Equity":
		return map[string]float64{"Stocks": 100}
	case "Fixed Income":
		return map[string]float64{"Bonds": 100}
	case "Commodities":
		return map[string]float64{"Commodities": 100}
	case "Miscellaneous":
		return map[string]float64{"Alternative": 100}
	case "Allocation":
		return map[string]float64{"Allocation": 100}
	default:
		if strings.Contains(lower, "money market") || strings.Contains(lower, "cash") {
			return map[string]float64{"Cash": 100}
		}
		if strings.Contains(lower, "real estate") || strings.Contains(lower, "property") || strings.Contains(lower, "reit") {
			return map[string]float64{"Real Estate": 100}
		}
		if strings.Contains(lower, "commodity") || strings.Contains(lower, "commodities") {
			return map[string]float64{"Commodities": 100}
		}
		if strings.Contains(lower, "convertible") {
			return map[string]float64{"Convertible": 100}
		}
		if strings.Contains(lower, "allocation") || strings.Contains(lower, "mixed") || strings.Contains(lower, "multi asset") {
			return map[string]float64{"Allocation": 100}
		}
		if strings.Contains(lower, "alternative") || strings.Contains(lower, "derivative") || strings.Contains(lower, "hedge") {
			return map[string]float64{"Alternative": 100}
		}
		return map[string]float64{"Other": 100}
	}
}
