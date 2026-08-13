// Package search implements the production search subsystem: an
// Elasticsearch-backed fast path with a Postgres fallback, wrapped by a
// circuit breaker so a degraded cluster never takes the API down.
package search

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"

	"ispilolite/internal/search/index"
)

// Config holds the connection settings for the Elasticsearch cluster.
type Config struct {
	Addresses []string
	Username  string
	Password  string
	// APIKey, when set, is preferred over username/password.
	APIKey string
	// EnableSniff toggles node discovery. Off by default (many managed
	// clusters sit behind a load balancer that breaks sniffing).
	EnableSniff bool
}

// Client wraps the typed ES client together with index-management helpers.
type Client struct {
	es *elasticsearch.Client
}

// NewClient builds an Elasticsearch client from Config. It does NOT block on
// connectivity — call Ping / EnsureIndices explicitly during startup so the
// caller decides whether a missing cluster is fatal (it isn't: the service
// degrades to the Postgres fallback).
func NewClient(cfg Config) (*Client, error) {
	esCfg := elasticsearch.Config{
		Addresses:            cfg.Addresses,
		Username:             cfg.Username,
		Password:             cfg.Password,
		APIKey:               cfg.APIKey,
		DiscoverNodesOnStart: cfg.EnableSniff,
		// Retry transient failures a couple of times with backoff before the
		// circuit breaker in the service layer trips.
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    2,
		RetryBackoff:  func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}
	return &Client{es: es}, nil
}

// ES exposes the underlying typed client for the repository layer.
func (c *Client) ES() *elasticsearch.Client { return c.es }

// Ping reports whether the cluster is reachable within ctx's deadline.
func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("elasticsearch ping: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch ping status: %s", res.Status())
	}
	return nil
}

// EnsureIndices creates any missing index with its mapping. Existing indices
// are left untouched (mapping migrations are handled by a separate reindex
// flow, not on every boot). Safe to call repeatedly.
func (c *Client) EnsureIndices(ctx context.Context) error {
	for name, body := range index.Mappings {
		exists, err := c.indexExists(ctx, name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := c.createIndex(ctx, name, body); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) indexExists(ctx context.Context, name string) (bool, error) {
	res, err := c.es.Indices.Exists([]string{name}, c.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("index exists %q: %w", name, err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("index exists %q: unexpected status %s", name, res.Status())
	}
}

func (c *Client) createIndex(ctx context.Context, name, body string) error {
	res, err := c.es.Indices.Create(
		name,
		c.es.Indices.Create.WithBody(strings.NewReader(body)),
		c.es.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("create index %q: %w", name, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("create index %q: %s", name, res.String())
	}
	return nil
}
