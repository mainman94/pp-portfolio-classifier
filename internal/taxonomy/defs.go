package taxonomy

import (
	"fmt"
	"strings"
)

type Definition struct {
	Name string
}

var Ordered = []Definition{
	{Name: "Asset Type"},
	{Name: "Stock Style"},
	{Name: "Stock Sector"},
	{Name: "Bond Style"},
	{Name: "Bond Sector"},
	{Name: "Region"},
	{Name: "Country"},
	{Name: "Country@Region"},
	{Name: "Holding"},
}

var Colors = []string{
	"#1a5fb4", "#1c71d8", "#3584e4", "#62a0ea", "#99c1f1", "#26a269", "#33d17a", "#57e389",
	"#f5c211", "#f6d32d", "#e5a50a", "#ff7800", "#e66100", "#c64600", "#ed333b", "#c01c28",
	"#9141ac", "#813d9c", "#613583", "#77767b", "#9a9996", "#c0bfbc", "#5e5c64", "#241f31",
	"#2ec27e", "#f66151", "#dc8add", "#8ff0a4", "#f9f06b", "#ffbe6f",
}

var AssetTypeMap = map[string]string{
	"1":  "Stocks",
	"2":  "Preferred Stock",
	"3":  "Bonds",
	"4":  "Convertible",
	"5":  "Other",
	"6":  "Commodities",
	"7":  "Cash",
	"8":  "Real Estate",
	"99": "Not classified",
}

var AssetTypeStockMap = map[string]string{
	"Stock": "Stocks",
}

var StockStyleMap = map[string]string{
	"1": "Large Value", "2": "Large Blend", "3": "Large Growth",
	"4": "Mid-Cap Value", "5": "Mid-Cap Blend", "6": "Mid-Cap Growth",
	"7": "Small Value", "8": "Small Blend", "9": "Small Growth",
}

var StockSectorMap = map[string]string{
	"101": "Basic Materials",
	"102": "Consumer Cyclical",
	"103": "Financial Services",
	"104": "Real Estate",
	"205": "Consumer Defensive",
	"206": "Healthcare",
	"207": "Utilities",
	"308": "Communication Services",
	"309": "Energy",
	"310": "Industrials",
	"311": "Technology",
}

var BondStyleMap = map[string]string{
	"1": "High Quality - Short Term", "2": "High Quality - Intermediate Term", "3": "High Quality - Long Term",
	"4": "Medium Quality - Short Term", "5": "Medium Quality - Intermediate Term", "6": "Medium Quality - Long Term",
	"7": "Low Quality - Short Term", "8": "Low Quality - Intermediate Term", "9": "Low Quality - Long Term",
}

var BondSectorMap = map[string]string{
	"10": "Government",
	"20": "Municipal",
	"30": "Corporate",
	"40": "Securitized",
	"50": "Cash",
	"60": "Derivative",
}

var RegionCodeMap = map[string]string{
	"1": "North America", "2": "North America", "3": "Central & Latin America",
	"4": "United Kingdom", "5": "Europe Developed", "6": "Europe Developed",
	"7": "Europe Emerging", "8": "Middle East / Africa", "9": "Middle East / Africa",
	"10": "Japan", "11": "Australasia", "12": "Asia Developed", "13": "Asia Emerging",
}

var RegionCountryMap = map[string]string{
	"USA": "North America", "CAN": "North America",
	"MEX": "Central & Latin America", "BRA": "Central & Latin America", "ARG": "Central & Latin America", "CHL": "Central & Latin America", "COL": "Central & Latin America", "PER": "Central & Latin America", "URY": "Central & Latin America",
	"GBR": "United Kingdom", "IMN": "United Kingdom", "GGY": "United Kingdom", "JEY": "United Kingdom",
	"DEU": "Europe Developed", "FRA": "Europe Developed", "ESP": "Europe Developed", "ITA": "Europe Developed", "NLD": "Europe Developed", "BEL": "Europe Developed", "AUT": "Europe Developed", "CHE": "Europe Developed", "IRL": "Europe Developed", "PRT": "Europe Developed", "DNK": "Europe Developed", "SWE": "Europe Developed", "NOR": "Europe Developed", "FIN": "Europe Developed", "ISL": "Europe Developed", "LUX": "Europe Developed", "MLT": "Europe Developed", "GRC": "Europe Developed", "CYP": "Europe Developed", "SVN": "Europe Developed",
	"POL": "Europe Emerging", "CZE": "Europe Emerging", "HUN": "Europe Emerging", "ROU": "Europe Emerging", "BGR": "Europe Emerging", "HRV": "Europe Emerging", "SVK": "Europe Emerging", "EST": "Europe Emerging", "LVA": "Europe Emerging", "LTU": "Europe Emerging", "SRB": "Europe Emerging", "UKR": "Europe Emerging", "TUR": "Europe Emerging", "RUS": "Europe Emerging",
	"ZAF": "Middle East / Africa", "SAU": "Middle East / Africa", "ARE": "Middle East / Africa", "EGY": "Middle East / Africa", "QAT": "Middle East / Africa", "KWT": "Middle East / Africa", "MAR": "Middle East / Africa", "NGA": "Middle East / Africa", "ISR": "Middle East / Africa",
	"JPN": "Japan",
	"AUS": "Australasia", "NZL": "Australasia",
	"HKG": "Asia Developed", "SGP": "Asia Developed", "KOR": "Asia Developed", "TWN": "Asia Developed",
	"CHN": "Asia Emerging", "IND": "Asia Emerging", "IDN": "Asia Emerging", "THA": "Asia Emerging", "MYS": "Asia Emerging", "PHL": "Asia Emerging", "VNM": "Asia Emerging", "PAK": "Asia Emerging",
	"XSN": "Supranational",
}

var CountryCodeMap = map[string]string{
	"USA": "UnitedStates", "CAN": "Canada", "MEX": "Mexico", "BRA": "Brazil", "ARG": "Argentina", "CHL": "Chile", "COL": "Colombia", "PER": "Peru", "URY": "Uruguay",
	"GBR": "UnitedKingdom", "IMN": "IsleofMan", "GGY": "Guernsey", "JEY": "Jersey",
	"DEU": "Germany", "FRA": "France", "ESP": "Spain", "ITA": "Italy", "NLD": "Netherlands", "BEL": "Belgium", "AUT": "Austria", "CHE": "Switzerland", "IRL": "Ireland", "PRT": "Portugal", "DNK": "Denmark", "SWE": "Sweden", "NOR": "Norway", "FIN": "Finland", "ISL": "Iceland", "LUX": "Luxembourg", "MLT": "Malta", "GRC": "Greece", "CYP": "Cyprus", "SVN": "Slovenia",
	"POL": "Poland", "CZE": "CzechRepublic", "HUN": "Hungary", "ROU": "Romania", "BGR": "Bulgaria", "HRV": "Croatia", "SVK": "Slovakia", "EST": "Estonia", "LVA": "Latvia", "LTU": "Lithuania", "SRB": "Serbia", "UKR": "Ukraine", "TUR": "Turkey", "RUS": "Russia",
	"ZAF": "SouthAfrica", "SAU": "SaudiArabia", "ARE": "UnitedArabEmirates", "EGY": "Egypt", "QAT": "Qatar", "KWT": "Kuwait", "MAR": "Morocco", "NGA": "Nigeria", "ISR": "Israel",
	"JPN": "Japan",
	"AUS": "Australia", "NZL": "NewZealand",
	"HKG": "HongKong", "SGP": "Singapore", "KOR": "SouthKorea", "TWN": "Taiwan",
	"CHN": "China", "IND": "India", "IDN": "Indonesia", "THA": "Thailand", "MYS": "Malaysia", "PHL": "Philippines", "VNM": "Vietnam", "PAK": "Pakistan",
	"XSN": "Supranational",
}

func MapAssetType(code string) string {
	if v, ok := AssetTypeMap[code]; ok {
		return v
	}
	return code
}

func MapAssetTypeStock(kind string) string {
	if v, ok := AssetTypeStockMap[kind]; ok {
		return v
	}
	return kind
}

func MapRegion(code string) string {
	if v, ok := RegionCodeMap[code]; ok {
		return v
	}
	if v, ok := RegionCountryMap[code]; ok {
		return v
	}
	return code
}

func MapCountry(code string) string {
	if v, ok := CountryCodeMap[code]; ok {
		return v
	}
	return code
}

func RegionForCountry(category string) string {
	category = strings.TrimSpace(strings.TrimSuffix(category, " (Bonds)"))
	if region, ok := RegionCountryMap[category]; ok {
		return region
	}
	if region, ok := regionForMappedCountry(category); ok {
		return region
	}
	return "Not Found"
}

func regionForMappedCountry(category string) (string, bool) {
	for code, mappedCountry := range CountryCodeMap {
		if mappedCountry == category {
			region, ok := RegionCountryMap[code]
			return region, ok
		}
	}
	return "", false
}

func CleanText(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")
	return strings.TrimSpace(value)
}

func DisplayCountry(code string) string {
	value := MapCountry(code)
	if strings.Contains(value, " ") {
		return value
	}
	if strings.ToUpper(value) == value {
		return value
	}
	var parts []string
	start := 0
	for i := 1; i < len(value); i++ {
		if value[i] >= 'A' && value[i] <= 'Z' {
			parts = append(parts, value[start:i])
			start = i
		}
	}
	parts = append(parts, value[start:])
	return strings.Join(parts, " ")
}

func RootDataKey(name string) (string, bool) {
	switch name {
	case "Asset Type":
		return "assetclasses", true
	default:
		return "", false
	}
}

func RootDimensions(name string) []string {
	switch name {
	case "Asset Type":
		return []string{"Asset Class"}
	default:
		return nil
	}
}

func ValidateWeight(weight float64) float64 {
	if weight < 0 {
		return 0
	}
	if weight > 100 {
		return 100
	}
	return weight
}

func BondSuffix(name string, segregate bool) string {
	if segregate {
		return fmt.Sprintf("%s (Bonds)", name)
	}
	return name
}
