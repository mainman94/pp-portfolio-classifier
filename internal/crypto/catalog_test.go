package crypto

import (
	"testing"

	"pp-portfolio-classifier/internal/model"
)

func TestResolveByTickerAndFeed(t *testing.T) {
	security := &model.Security{
		Name:   "Bitcoin",
		Ticker: "BTC",
		Feed:   "COINGECKO",
	}
	label, ok := Resolve(security)
	if !ok {
		t.Fatalf("expected crypto match")
	}
	if label != "Bitcoin" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func TestResolveGenericCrypto(t *testing.T) {
	security := &model.Security{Name: "Global Digital Asset Basket ETP"}
	label, ok := Resolve(security)
	if !ok || label != "Crypto" {
		t.Fatalf("expected generic crypto label, got %q ok=%v", label, ok)
	}
}
