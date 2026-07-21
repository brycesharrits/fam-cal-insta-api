package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
)

// jwksCache fetches and caches a remote JWKS with a TTL.
// One instance per JWKS URL.
type jwksCache struct {
	url        string
	httpClient *http.Client
	ttl        time.Duration

	mu        sync.Mutex
	set       jwk.Set
	fetchedAt time.Time
}

func newJWKSCache(url string, ttl time.Duration) *jwksCache {
	return &jwksCache{
		url:        url,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		ttl:        ttl,
	}
}

func (c *jwksCache) Get(ctx context.Context) (jwk.Set, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.set != nil && time.Since(c.fetchedAt) < c.ttl {
		return c.set, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching JWKS from %s: %w", c.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS fetch returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	set, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parsing JWKS: %w", err)
	}
	c.set = set
	c.fetchedAt = time.Now()
	return set, nil
}
