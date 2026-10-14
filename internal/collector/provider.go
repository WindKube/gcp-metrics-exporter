package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// Provider lazily creates and caches one registry for each configured project.
// It keeps resolved credentials in the worker process rather than putting them
// in Temporal workflow inputs or schedules.
type Provider struct {
	projects   map[string]config.Project
	registries map[string]*Registry
	mu         sync.Mutex
}

func NewProvider(projects []config.Project) *Provider {
	byName := make(map[string]config.Project, len(projects))
	for _, project := range projects {
		byName[project.Name] = project
	}
	return &Provider{projects: byName, registries: make(map[string]*Registry, len(projects))}
}

func (p *Provider) Get(ctx context.Context, projectName string) (*Registry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if registry, ok := p.registries[projectName]; ok {
		return registry, nil
	}
	project, ok := p.projects[projectName]
	if !ok {
		return nil, fmt.Errorf("project %q is not configured", projectName)
	}
	registry, err := NewGoogleRegistry(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("create Google clients for project %q: %w", projectName, err)
	}
	p.registries[projectName] = registry
	return registry, nil
}

func clientOptions(ctx context.Context, access config.Access) ([]option.ClientOption, error) {
	var base []option.ClientOption
	if access.Credentials != "" {
		var metadata struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(access.Credentials), &metadata); err != nil {
			return nil, fmt.Errorf("configured credentials are not valid JSON")
		}
		switch metadata.Type {
		case "service_account", "authorized_user", "external_account":
		default:
			return nil, fmt.Errorf("configured credentials have unsupported type %q", metadata.Type)
		}
		credentials, err := google.CredentialsFromJSON(ctx, []byte(access.Credentials), cloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("configured credentials could not be loaded")
		}
		base = []option.ClientOption{option.WithTokenSource(credentials.TokenSource)}
	}
	if access.Impersonation == nil {
		return base, nil
	}
	impersonation := access.Impersonation
	impersonationConfig := impersonate.CredentialsConfig{
		TargetPrincipal: impersonation.TargetServiceAccount,
		Scopes:          []string{cloudPlatformScope},
		Delegates:       impersonation.Delegates,
	}
	if impersonation.Lifetime != "" {
		lifetime, err := time.ParseDuration(impersonation.Lifetime)
		if err != nil {
			return nil, fmt.Errorf("parse impersonation lifetime: %w", err)
		}
		impersonationConfig.Lifetime = lifetime
	}
	tokenSource, err := impersonate.CredentialsTokenSource(ctx, impersonationConfig, base...)
	if err != nil {
		return nil, fmt.Errorf("create impersonated credentials: %w", err)
	}
	return []option.ClientOption{option.WithTokenSource(tokenSource)}, nil
}
