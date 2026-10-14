package config

import (
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

func TestLoadValidatesBillingLocationAndTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`global:
  locations: [europe-west1]
  billing:
    query_project: billing-project
    location: EU
    table: billing-project.billing_dataset.gcp_billing_export_v1_ABCDEF
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
	if _, err := Load(path, []string{"billing"}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadUsesProjectBillingAndGlobalMigrationFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`global:
  locations: [europe-west1]
  billing:
    query_project: legacy-billing-project
    location: EU
    table: legacy-billing-project.billing_dataset.gcp_billing_export_v1_ABCDEF
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
      query_project: separate-billing-project
      location: US
      table: separate-billing-project.billing_dataset.gcp_billing_export_v1_123456
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
	if got, want := c.Projects[1].Billing.QueryProject, "separate-billing-project"; got != want {
		t.Fatalf("separate query project = %q, want %q", got, want)
	}
}

func TestLoadResolvesCredentialReferenceWithoutExposingItInJSON(t *testing.T) {
	t.Setenv("GCME_TEST_CREDENTIALS", `{"type":"authorized_user","refresh_token":"sensitive"}`)
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
	if err == nil || !strings.Contains(err.Error(), "service account email") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvironmentReferenceName(t *testing.T) {
	for _, test := range []struct {
		value string
		name  string
		valid bool
	}{
		{value: "${GCME_CREDENTIALS}", name: "GCME_CREDENTIALS", valid: true},
		{value: "${_GCME_2}", name: "_GCME_2", valid: true},
		{value: "GCME_CREDENTIALS"},
		{value: "${2GCME}"},
		{value: "${GCME-NAME}"},
		{value: "${}"},
	} {
		name, valid := environmentReferenceName(test.value)
		if name != test.name || valid != test.valid {
			t.Fatalf("environmentReferenceName(%q) = (%q, %t), want (%q, %t)", test.value, name, valid, test.name, test.valid)
		}
	}
}

func TestIsServiceAccountEmail(t *testing.T) {
	if !isServiceAccountEmail("exporter@target-project.iam.gserviceaccount.com") {
		t.Fatal("expected valid service account email")
	}
	for _, value := range []string{"", "exporter@example.com", "exporter@@target.iam.gserviceaccount.com", "exporter @target.iam.gserviceaccount.com"} {
		if isServiceAccountEmail(value) {
			t.Fatalf("accepted invalid service account email %q", value)
		}
	}
}
