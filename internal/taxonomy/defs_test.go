package taxonomy

import "testing"

func TestRegionForMappedCountryName(t *testing.T) {
	if got := RegionForCountry("Switzerland"); got != "Europe Developed" {
		t.Fatalf("unexpected region: %q", got)
	}
}

func TestRegionForMappedCountryNameWithBondSuffix(t *testing.T) {
	if got := RegionForCountry("Switzerland (Bonds)"); got != "Europe Developed" {
		t.Fatalf("unexpected region with bond suffix: %q", got)
	}
}
