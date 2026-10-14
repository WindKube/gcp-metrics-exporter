package config

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestLoadRequiresBillingSettingsWhenBillingIsEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`global:
  locations: [europe-west1]
resources: [billing]
storage:
  elasticsearch:
    addresses: [http://localhost:9200]
projects:
  - name: test
    id: test-project
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path, []string{"billing"})
	if err == nil || err.Error() != "project \"test\" billing must be configured when billing is enabled" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadValidatesBillingLocationDatasetAndTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`global:
  locations: [europe-west1]
  billing:
    location: EU
    dataset: billing-project.billing_dataset
    table: gcp_billing_export_v1_ABCDEF
resources: [billing]
storage:
  elasticsearch:
    addresses: [http://localhost:9200]
projects:
  - name: test
    id: test-project
`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, []string{"billing"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.Projects[0].Billing.QueryProject, "test-project"; got != want {
		t.Fatalf("query project = %q, want %q", got, want)
	}
}

func TestLoadUsesProjectBillingAndGlobalMigrationFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`global:
  locations: [europe-west1]
  billing:
    query_project: legacy-billing-project
    location: EU
    dataset: legacy-billing-project.billing_dataset
    table: gcp_billing_export_v1_ABCDEF
resources: [cloud_run_service]
storage:
  elasticsearch:
    addresses: [http://localhost:9200]
projects:
  - name: legacy
    id: legacy-project
    resources: [billing]
  - name: separate
    id: separate-project
    resources: [billing]
    billing:
      location: US
      dataset: separate-billing-project.billing_dataset
      table: gcp_billing_export_v1_123456
`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, []string{"billing", "cloud_run_service"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.Projects[0].Billing.QueryProject, "legacy-billing-project"; got != want {
		t.Fatalf("legacy query project = %q, want %q", got, want)
	}
	if got, want := c.Projects[1].Billing.QueryProject, "separate-project"; got != want {
		t.Fatalf("separate query project = %q, want %q", got, want)
	}
}

func TestLoadResolvesCredentialReferenceWithoutExposingItInJSON(t *testing.T) {
	t.Setenv("GCME_TEST_CREDENTIALS", base64.StdEncoding.EncodeToString([]byte(`{"type":"authorized_user","refresh_token":"sensitive"}`)))
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
    access:
      credentials: "${GCME_TEST_CREDENTIALS}"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, []string{"cloud_run_service"})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Projects[0].Access.Credentials; !strings.Contains(got, "sensitive") {
		t.Fatalf("credential reference was not resolved")
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "sensitive") {
		t.Fatalf("config JSON exposed credentials: %s", b)
	}
}

func TestLoadRejectsNonBase64Credentials(t *testing.T) {
	t.Setenv("GCME_TEST_CREDENTIALS", `{"type":"authorized_user"}`)
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
    access:
      credentials: "${GCME_TEST_CREDENTIALS}"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path, []string{"cloud_run_service"})
	if err == nil || !strings.Contains(err.Error(), "base64-encoded") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidAccessConfiguration(t *testing.T) {
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
    access:
      credentials: not-an-environment-reference
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path, []string{"cloud_run_service"})
	if err == nil || !strings.Contains(err.Error(), "environment reference") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidImpersonationTarget(t *testing.T) {
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
    access:
      impersonation:
        target_service_account: not-a-service-account
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path, []string{"cloud_run_service"})
	if err == nil || !strings.Contains(err.Error(), "valid email address") {
		t.Fatalf("unexpected error: %v", err)
	}
}
