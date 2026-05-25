package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pp-portfolio-classifier/internal/model"
)

func TestBackupExistingCreatesBakFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "classified.xml")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := backupExisting(path); err != nil {
		t.Fatalf("backupExisting returned error: %v", err)
	}
	data, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("unexpected backup content: %q", string(data))
	}
}

func TestWriteReportIncludesFailuresWarningsAndOther(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	security := &model.Security{
		Name: "Example",
		ISIN: "IE0000000001",
		Report: &model.HoldingReport{
			Warnings: []string{"Country: unknown mapping code \"ZZZ\""},
			Taxonomies: map[string]map[string]float64{
				"Asset Type": {"Other": 25},
			},
		},
	}

	if err := writeReport(path, []*model.Security{security}, []string{"Broken (XX): failed"}); err != nil {
		t.Fatalf("writeReport returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Broken (XX): failed", "Country: unknown mapping code", "Example (IE0000000001): Other 25.00%"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q: %s", want, text)
		}
	}
}
