package docs

import "context"

// Collector defines the interface for infrastructure configuration collectors.
type Collector interface {
	Name() string
	Available() bool
	Discover(ctx context.Context) ([]Application, error)
	Collect(ctx context.Context, appID string) (*CollectedConfig, error)
}

// CollectedConfig holds the raw configuration content retrieved by a collector.
type CollectedConfig struct {
	Content string `json:"content"`
	Format  string `json:"format"` // "json", "yaml", "toml", "text"
}
