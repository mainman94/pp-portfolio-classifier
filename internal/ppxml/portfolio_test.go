package ppxml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pp-portfolio-classifier/internal/model"
)

func TestNestedSecurityRef(t *testing.T) {
	got := nestedSecurityRef("../../../../../../../../securities/security[2]")
	want := "../../../../../../../../../../securities/security[2]"
	if got != want {
		t.Fatalf("unexpected nested ref: got %q want %q", got, want)
	}
}

func TestSecuritiesKeepsCryptoWithoutISIN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "portfolio.xml")
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<client>
  <securities>
    <security>
      <uuid>1</uuid>
      <name>Bitcoin</name>
      <tickerSymbol>BTC</tickerSymbol>
      <feed>COINGECKO</feed>
      <property type="FEED" name="COINGECKOCOINID">bitcoin</property>
      <isRetired>false</isRetired>
    </security>
    <security>
      <uuid>2</uuid>
      <name>No ISIN Plain Security</name>
      <feed>MANUAL</feed>
      <isRetired>false</isRetired>
    </security>
  </securities>
</client>`
	if err := os.WriteFile(path, []byte(xml), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	file, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	securities, err := file.Securities()
	if err != nil {
		t.Fatalf("Securities returned error: %v", err)
	}
	if len(securities) != 1 {
		t.Fatalf("expected exactly one retained security, got %d", len(securities))
	}
	if securities[0].Name != "Bitcoin" {
		t.Fatalf("unexpected retained security: %q", securities[0].Name)
	}
}

func TestUpdateTaxonomyRemovesStaleAssignmentForSecurity(t *testing.T) {
	root, err := Parse([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<client>
  <securities>
    <security><name>Example</name><isin>US0000000001</isin></security>
  </securities>
  <taxonomies>
    <taxonomy>
      <name>Asset Type</name>
      <root>
        <children>
          <classification>
            <name>Bonds</name>
            <assignments>
              <assignment>
                <investmentVehicle class="security" reference="../../../../../../../../securities/security"></investmentVehicle>
                <weight>10000</weight>
              </assignment>
            </assignments>
          </classification>
        </children>
      </root>
    </taxonomy>
  </taxonomies>
</client>`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	file := &File{root: root}
	security := &model.Security{
		Name:      "Example",
		ISIN:      "US0000000001",
		SourceRef: securityRef(0),
		Report: &model.HoldingReport{Taxonomies: map[string]map[string]float64{
			"Asset Type": {"Stocks": 100},
		}},
	}

	if err := file.UpdateTaxonomy("Asset Type", []*model.Security{security}, -1, false); err != nil {
		t.Fatalf("UpdateTaxonomy returned error: %v", err)
	}

	children := root.Child("taxonomies").Child("taxonomy").Child("root").Child("children")
	for _, class := range children.ChildrenNamed("classification") {
		assignments := class.Child("assignments")
		if assignments == nil {
			continue
		}
		if childText(class, "name") == "Bonds" && len(assignments.ChildrenNamed("assignment")) != 0 {
			t.Fatalf("stale Bonds assignment was not removed")
		}
		if childText(class, "name") == "Stocks" && len(assignments.ChildrenNamed("assignment")) != 1 {
			t.Fatalf("expected one new Stocks assignment, got %d", len(assignments.ChildrenNamed("assignment")))
		}
	}
}

func TestMinimalPortfolioXMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "portfolio.xml")
	output := filepath.Join(dir, "classified.xml")
	csvPath := filepath.Join(dir, "fetched.csv")
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<client>
  <securities>
    <security>
      <uuid>1</uuid>
      <name>Example ETF</name>
      <isin>IE0000000001</isin>
      <isRetired>false</isRetired>
    </security>
  </securities>
</client>`
	if err := os.WriteFile(input, []byte(xml), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	file, err := Load(input)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	securities, err := file.Securities()
	if err != nil {
		t.Fatalf("Securities returned error: %v", err)
	}
	if len(securities) != 1 {
		t.Fatalf("expected one security, got %d", len(securities))
	}
	securities[0].Report = &model.HoldingReport{Taxonomies: map[string]map[string]float64{
		"Asset Type": {"Stocks": 100},
		"Region":     {"North America": 60, "Europe Developed": 40},
	}}

	if err := file.UpdateTaxonomy("Asset Type", securities, -1, false); err != nil {
		t.Fatalf("UpdateTaxonomy Asset Type returned error: %v", err)
	}
	if err := file.UpdateTaxonomy("Region", securities, -1, false); err != nil {
		t.Fatalf("UpdateTaxonomy Region returned error: %v", err)
	}
	if err := file.Write(output); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := file.DumpCSV(csvPath, securities); err != nil {
		t.Fatalf("DumpCSV returned error: %v", err)
	}

	written, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	text := string(written)
	for _, want := range []string{"<name>Asset Type</name>", "<name>Stocks</name>", "<name>Region</name>", "<name>North America</name>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("written XML missing %q", want)
		}
	}
	if _, err := Parse(written); err != nil {
		t.Fatalf("written XML is not parseable: %v", err)
	}

	csvData, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if !strings.Contains(string(csvData), "IE0000000001,Asset Type,Stocks,1,Example ETF") {
		t.Fatalf("CSV missing expected classification row: %s", string(csvData))
	}
}
