package morningstar

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"pp-portfolio-classifier/internal/config"
	"pp-portfolio-classifier/internal/model"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestLoadReportReturnsClearErrorForMissingSnapshotType(t *testing.T) {
	client := testClient(func(req *http.Request) string {
		return `[{
			"Name":"No Type Fund"
		}]`
	})

	_, err := client.LoadReport(context.Background(), &model.Security{
		Name: "No Type Fund",
		ISIN: "LU0000000001",
	}, config.Options{})
	if err == nil || !strings.Contains(err.Error(), "no snapshot type") {
		t.Fatalf("expected missing type error, got %v", err)
	}
}

func TestLoadReportFundFallsBackWhenBreakdownsAreMissing(t *testing.T) {
	client := testClient(func(req *http.Request) string {
		switch req.URL.Query().Get("viewid") {
		case "snapshot":
			return `[{"Name":"Sparse Fund","Type":"Fund"}]`
		case "ITsnapshot":
			return `[{"Name":"Sparse Fund","CategoryBroadAssetClass":{"Name":"Fixed Income"}}]`
		default:
			return `[]`
		}
	})

	report, err := client.LoadReport(context.Background(), &model.Security{
		Name: "Sparse Fund",
		ISIN: "LU0000000002",
	}, config.Options{})
	if err != nil {
		t.Fatalf("LoadReport returned error: %v", err)
	}
	if got := report.Taxonomies["Asset Type"]["Bonds"]; got != 100 {
		t.Fatalf("expected Fixed Income fallback to Bonds=100, got %v", got)
	}
}

func TestLoadReportReturnsClearErrorForUnsupportedType(t *testing.T) {
	client := testClient(func(req *http.Request) string {
		return `[{"Name":"Option","Type":"Derivative"}]`
	})

	_, err := client.LoadReport(context.Background(), &model.Security{
		Name: "Option",
		ISIN: "LU0000000003",
	}, config.Options{})
	if err == nil || !strings.Contains(err.Error(), "unsupported security type") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestLoadReportWarnsAboutUnknownStockMappings(t *testing.T) {
	client := testClient(func(req *http.Request) string {
		return `[{
			"Name":"Unknown Stock",
			"Type":"Stock",
			"Sector":{"SectorCode":"999"},
			"InvestmentStyle":"42",
			"Country":"ZZZ"
		}]`
	})

	report, err := client.LoadReport(context.Background(), &model.Security{
		Name: "Unknown Stock",
		ISIN: "US0000000004",
	}, config.Options{RetrieveStocks: true})
	if err != nil {
		t.Fatalf("LoadReport returned error: %v", err)
	}
	warnings := strings.Join(report.Warnings, "\n")
	for _, want := range []string{
		`Stock Sector: unknown mapping code "999"`,
		`Stock Style: unknown mapping code "42"`,
		`Country: unknown mapping code "ZZZ"`,
		`Region: unknown mapping code "ZZZ"`,
	} {
		if !strings.Contains(warnings, want) {
			t.Fatalf("missing warning %q in %q", want, warnings)
		}
	}
}

func TestInferFundAssetTypeAvoidsBroadOtherForKnownSpecialTypes(t *testing.T) {
	cases := map[string]string{
		"Commodities":        "Commodities",
		"Allocation":         "Allocation",
		"Money Market":       "Cash",
		"Real Estate Sector": "Real Estate",
		"Convertible Bond":   "Convertible",
		"Alternative":        "Alternative",
	}
	for input, want := range cases {
		got := inferFundAssetType(input)
		if got[want] != 100 {
			t.Fatalf("inferFundAssetType(%q) = %#v, want %s=100", input, got, want)
		}
	}
}

func testClient(response func(*http.Request) string) *Client {
	return &Client{
		domain: "test",
		token:  "token",
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(response(req))),
			}, nil
		})},
	}
}
