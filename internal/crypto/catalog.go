package crypto

import (
	"strings"

	"pp-portfolio-classifier/internal/model"
)

type Entry struct {
	Label   string
	Aliases []string
}

var catalog = []Entry{
	{Label: "Bitcoin", Aliases: []string{"bitcoin", "btc"}},
	{Label: "Ethereum", Aliases: []string{"ethereum", "eth", "ether"}},
	{Label: "Solana", Aliases: []string{"solana", "sol"}},
	{Label: "XRP", Aliases: []string{"xrp", "ripple"}},
	{Label: "Cardano", Aliases: []string{"cardano", "ada"}},
	{Label: "Avalanche", Aliases: []string{"avalanche", "avax"}},
	{Label: "Polkadot", Aliases: []string{"polkadot", "dot"}},
	{Label: "Chainlink", Aliases: []string{"chainlink", "link"}},
	{Label: "Litecoin", Aliases: []string{"litecoin", "ltc"}},
	{Label: "Dogecoin", Aliases: []string{"dogecoin", "doge"}},
}

func Resolve(security *model.Security) (string, bool) {
	if security == nil {
		return "", false
	}
	text := normalizedText(security)
	if text == "" {
		return "", false
	}
	for _, entry := range catalog {
		for _, alias := range entry.Aliases {
			if strings.Contains(text, alias) {
				return entry.Label, true
			}
		}
	}
	if strings.Contains(text, "crypto") || strings.Contains(text, "digital asset") || strings.Contains(text, "blockchain") {
		return "Crypto", true
	}
	if strings.EqualFold(strings.TrimSpace(security.Feed), "COINGECKO") {
		return "Crypto", true
	}
	return "", false
}

func IsCrypto(security *model.Security) bool {
	_, ok := Resolve(security)
	return ok
}

func normalizedText(security *model.Security) string {
	parts := []string{
		strings.ToLower(security.Name),
		strings.ToLower(security.Ticker),
		strings.ToLower(security.Feed),
		strings.ToLower(security.CoinID),
	}
	return strings.Join(parts, " ")
}
