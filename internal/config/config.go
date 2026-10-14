package config

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/robfig/cron/v3"
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
}

type Temporal struct {
	Address   string `yaml:"address" json:"address"`
	Namespace string `yaml:"namespace" json:"namespace"`
	TaskQueue string `yaml:"task_queue" json:"task_queue"`
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
}

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
	}
}

func (c *Config) Validate(knownResources []string) error {
	if len(c.Projects) == 0 {
		return fmt.Errorf("projects must not be empty")
	}
	if c.Global.ScheduleTimezone == "" {
		return fmt.Errorf("schedule_timezone must not be empty")
	}
	for _, p := range c.Projects {
		if p.Name == "" || p.ID == "" {
			return fmt.Errorf("each project needs name and id")
		}
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
	}
	if len(c.Storage.Elasticsearch.Addresses) == 0 {
		return fmt.Errorf("storage.elasticsearch.addresses must not be empty")
	}
	if strings.TrimSpace(c.Storage.Elasticsearch.IndexPrefix) == "" {
		return fmt.Errorf("storage.elasticsearch.index_prefix must not be empty")
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
