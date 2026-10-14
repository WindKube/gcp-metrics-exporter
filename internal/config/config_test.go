package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolvesProjectDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`global:
  locations: [europe-west1]
resources: [cloud_run_service]
storage:
  elasticsearch:
    addresses: [http://localhost:9200]
projects:
  - name: test
    id: test-project
`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, []string{"cloud_run_service"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Projects[0].ScrapeSchedule != DefaultSchedule {
		t.Fatalf("schedule = %q", c.Projects[0].ScrapeSchedule)
	}
	if c.Projects[0].Locations[0] != "europe-west1" {
		t.Fatal("locations were not inherited")
	}
}
