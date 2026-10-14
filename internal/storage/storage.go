package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
)

type Document struct {
	ID        string                                       `json:"-"`
	Timestamp time.Time                                    `json:"@timestamp"`
	Resource  struct{ Type, ID, Name string }              `json:"resource"`
	Cloud     struct{ Provider, Project, Location string } `json:"cloud"`
	GCP       any                                          `json:"gcp"`
}
type Store interface {
	Write(context.Context, []Document) error
	Ping(context.Context) error
}
type Elasticsearch struct {
	base, prefix, user, password string
	http                         *http.Client
}

func New(cfg config.Elasticsearch) *Elasticsearch {
	return &Elasticsearch{base: strings.TrimRight(cfg.Addresses[0], "/"), prefix: cfg.IndexPrefix, user: cfg.Username, password: cfg.Password, http: http.DefaultClient}
}
func (s *Elasticsearch) index(now time.Time) string {
	return fmt.Sprintf("%s-%s", s.prefix, now.UTC().Format("2006.01.02"))
}
func (s *Elasticsearch) Write(ctx context.Context, docs []Document) error {
	var body bytes.Buffer
	for _, d := range docs {
		meta, _ := json.Marshal(map[string]any{"index": map[string]string{"_index": s.index(d.Timestamp), "_id": d.ID}})
		data, err := json.Marshal(d)
		if err != nil {
			return err
		}
		body.Write(meta)
		body.WriteByte('\n')
		body.Write(data)
		body.WriteByte('\n')
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/_bulk", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if s.user != "" {
		req.SetBasicAuth(s.user, s.password)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch bulk write: %s", resp.Status)
	}
	return nil
}
func (s *Elasticsearch) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.base, nil)
	if err != nil {
		return err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("elasticsearch ping: %s", resp.Status)
	}
	return nil
}
