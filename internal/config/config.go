package config

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/windkube/gcp-metrics-exporter/internal/queries"
	"gopkg.in/yaml.v3"
)

const DefaultSchedule = "0 * * * *"

type Config struct {
	Global    Global    `yaml:"global" json:"global"`
	Storage   Storage   `yaml:"storage" json:"storage"`
	Resources []string  `yaml:"resources" json:"resources"`
	Projects  []Project `yaml:"projects" json:"projects"`
}

type Global struct {
	ScrapeSchedule   string   `yaml:"scrape_schedule" json:"scrape_schedule"`
	ScheduleTimezone string   `yaml:"schedule_timezone" json:"schedule_timezone"`
	Locations        []string `yaml:"locations" json:"locations"`
	Temporal         Temporal `yaml:"temporal" json:"temporal"`
	// Billing is retained as a migration fallback. New configurations should
	// configure billing on the individual project.
	Billing Billing `yaml:"billing" json:"billing"`
}

type Temporal struct {
	Address   string `yaml:"address" json:"address"`
	Namespace string `yaml:"namespace" json:"namespace"`
	TaskQueue string `yaml:"task_queue" json:"task_queue"`
}

// Billing identifies the BigQuery billing-export table and the project used to
// run the query. The table must be a fully-qualified BigQuery table name.
type Billing struct {
	QueryProject    string `yaml:"query_project" json:"query_project"`
	Location        string `yaml:"location" json:"location"`
	Table           string `yaml:"table" json:"table"`
	GroupByService  bool   `yaml:"group_by_service" json:"group_by_service"`
	GroupByCostType bool   `yaml:"group_by_cost_type" json:"group_by_cost_type"`
}

type Storage struct {
	Elasticsearch Elasticsearch `yaml:"elasticsearch" json:"elasticsearch"`
}

type Elasticsearch struct {
	Addresses   []string `yaml:"addresses" json:"addresses"`
	IndexPrefix string   `yaml:"index_prefix" json:"index_prefix"`
	Username    string   `yaml:"-" json:"-"`
	Password    string   `yaml:"-" json:"-"`
}

type Project struct {
	Name           string   `yaml:"name" json:"name"`
	ID             string   `yaml:"id" json:"id"`
	Locations      []string `yaml:"locations" json:"locations"`
	Resources      []string `yaml:"resources" json:"resources"`
	ScrapeSchedule string   `yaml:"scrape_schedule" json:"scrape_schedule"`
	Billing        *Billing `yaml:"billing" json:"billing,omitempty"`
	Access         Access   `yaml:"access" json:"access"`
}

// Access configures the identity used for all Google APIs for one project.
// Credentials contains resolved credential JSON and is deliberately excluded
// from config show output.
type Access struct {
	Credentials   string         `yaml:"credentials" json:"-"`
	Impersonation *Impersonation `yaml:"impersonation" json:"impersonation,omitempty"`
}

type Impersonation struct {
	TargetServiceAccount string   `yaml:"target_service_account" json:"target_service_account"`
	Delegates            []string `yaml:"delegates" json:"delegates,omitempty"`
	Lifetime             string   `yaml:"lifetime" json:"lifetime,omitempty"`
}

var envReference = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)
var serviceAccountEmail = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.iam\.gserviceaccount\.com$`)

func Load(path string, knownResources []string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.applyDefaults()
	if err := c.resolveCredentials(); err != nil {
		return nil, err
	}
	c.Storage.Elasticsearch.Username = os.Getenv("GCME_STORAGE_ELASTICSEARCH_USERNAME")
	c.Storage.Elasticsearch.Password = os.Getenv("GCME_STORAGE_ELASTICSEARCH_PASSWORD")
	if err := c.Validate(knownResources); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Global.ScrapeSchedule == "" {
		c.Global.ScrapeSchedule = DefaultSchedule
	}
	if c.Global.ScheduleTimezone == "" {
		c.Global.ScheduleTimezone = "UTC"
	}
	if c.Global.Temporal.Address == "" {
		c.Global.Temporal.Address = "localhost:7233"
	}
	if c.Global.Temporal.Namespace == "" {
		c.Global.Temporal.Namespace = "default"
	}
	if c.Global.Temporal.TaskQueue == "" {
		c.Global.Temporal.TaskQueue = "gcp-metrics-exporter"
	}
	if c.Storage.Elasticsearch.IndexPrefix == "" {
		c.Storage.Elasticsearch.IndexPrefix = "gcp-inventory"
	}
	for i := range c.Projects {
		p := &c.Projects[i]
		if len(p.Locations) == 0 {
			p.Locations = slices.Clone(c.Global.Locations)
		}
		if len(p.Resources) == 0 {
			p.Resources = slices.Clone(c.Resources)
		}
		if p.ScrapeSchedule == "" {
			p.ScrapeSchedule = c.Global.ScrapeSchedule
		}
		if p.Billing == nil && slices.Contains(p.Resources, "billing") && hasBillingConfig(c.Global.Billing) {
			billing := c.Global.Billing
			p.Billing = &billing
		}
	}
}

func (c *Config) Validate(knownResources []string) error {
	if len(c.Projects) == 0 {
		return fmt.Errorf("projects must not be empty")
	}
	if c.Global.ScheduleTimezone == "" {
		return fmt.Errorf("schedule_timezone must not be empty")
	}
	names := make(map[string]struct{}, len(c.Projects))
	for _, p := range c.Projects {
		if p.Name == "" || p.ID == "" {
			return fmt.Errorf("each project needs name and id")
		}
		if _, ok := names[p.Name]; ok {
			return fmt.Errorf("project name %q is duplicated", p.Name)
		}
		names[p.Name] = struct{}{}
		if len(p.Locations) == 0 {
			return fmt.Errorf("project %q has no locations", p.Name)
		}
		if _, err := cron.ParseStandard(p.ScrapeSchedule); err != nil {
			return fmt.Errorf("project %q scrape_schedule: %w", p.Name, err)
		}
		for _, resource := range p.Resources {
			if !slices.Contains(knownResources, resource) {
				return fmt.Errorf("project %q has unknown resource %q", p.Name, resource)
			}
		}
		billingEnabled := slices.Contains(p.Resources, "billing")
		if p.Billing != nil && !billingEnabled {
			return fmt.Errorf("project %q billing is configured but billing is not enabled in resources", p.Name)
		}
		if billingEnabled {
			if err := validateBilling(p.Name, p.Billing); err != nil {
				return err
			}
		}
		if err := validateAccess(p.Name, p.Access); err != nil {
			return err
		}
	}
	if len(c.Storage.Elasticsearch.Addresses) == 0 {
		return fmt.Errorf("storage.elasticsearch.addresses must not be empty")
	}
	if strings.TrimSpace(c.Storage.Elasticsearch.IndexPrefix) == "" {
		return fmt.Errorf("storage.elasticsearch.index_prefix must not be empty")
	}
	return nil
}

func (c *Config) resolveCredentials() error {
	for i := range c.Projects {
		p := &c.Projects[i]
		if p.Access.Credentials == "" {
			continue
		}
		matches := envReference.FindStringSubmatch(p.Access.Credentials)
		if matches == nil {
			return fmt.Errorf("project %q access.credentials must be an environment reference such as ${GCME_CREDENTIALS}", p.Name)
		}
		value, ok := os.LookupEnv(matches[1])
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("project %q access.credentials references an unset or empty environment variable", p.Name)
		}
		p.Access.Credentials = value
	}
	return nil
}

func hasBillingConfig(b Billing) bool {
	return strings.TrimSpace(b.QueryProject) != "" || strings.TrimSpace(b.Location) != "" || strings.TrimSpace(b.Table) != ""
}

func validateBilling(project string, billing *Billing) error {
	if billing == nil {
		return fmt.Errorf("project %q billing must be configured when billing is enabled", project)
	}
	if strings.TrimSpace(billing.QueryProject) == "" {
		return fmt.Errorf("project %q billing.query_project must not be empty", project)
	}
	if strings.TrimSpace(billing.Table) == "" {
		return fmt.Errorf("project %q billing.table must not be empty", project)
	}
	if strings.TrimSpace(billing.Location) == "" {
		return fmt.Errorf("project %q billing.location must not be empty", project)
	}
	if err := queries.ValidateTable(billing.Table); err != nil {
		return fmt.Errorf("project %q billing.table: %w", project, err)
	}
	return nil
}

func validateAccess(project string, access Access) error {
	if access.Impersonation == nil {
		return nil
	}
	impersonation := access.Impersonation
	if !serviceAccountEmail.MatchString(impersonation.TargetServiceAccount) {
		return fmt.Errorf("project %q access.impersonation.target_service_account must be a service account email", project)
	}
	for _, delegate := range impersonation.Delegates {
		if !serviceAccountEmail.MatchString(delegate) {
			return fmt.Errorf("project %q access.impersonation.delegates must contain service account emails", project)
		}
	}
	if impersonation.Lifetime == "" {
		return nil
	}
	lifetime, err := time.ParseDuration(impersonation.Lifetime)
	if err != nil || lifetime <= 0 || lifetime > time.Hour {
		return fmt.Errorf("project %q access.impersonation.lifetime must be greater than zero and at most 1h", project)
	}
	return nil
}

func (c *Config) Project(name string) (Project, bool) {
	for _, p := range c.Projects {
		if p.Name == name {
			return p, true
		}
	}
	return Project{}, false
}
