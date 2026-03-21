package wsaa

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"time"
)

// DefaultRenewalMargin is how early to renew a token before it expires.
const DefaultRenewalMargin = 30 * time.Minute

// CachedClient wraps a WSAA Client with a TokenStore for automatic caching.
// It avoids redundant LoginCms calls by reusing valid tokens.
type CachedClient struct {
	client        *Client
	store         TokenStore
	renewalMargin time.Duration
}

// CachedOption configures the CachedClient.
type CachedOption func(*CachedClient)

// WithRenewalMargin sets how early to renew tokens before expiration.
func WithRenewalMargin(d time.Duration) CachedOption {
	return func(cc *CachedClient) {
		cc.renewalMargin = d
	}
}

// NewCachedClient creates a CachedClient that caches tokens in the given store.
func NewCachedClient(client *Client, store TokenStore, opts ...CachedOption) *CachedClient {
	cc := &CachedClient{
		client:        client,
		store:         store,
		renewalMargin: DefaultRenewalMargin,
	}
	for _, o := range opts {
		o(cc)
	}
	return cc
}

// cacheKey builds a deterministic key for the token store.
func cacheKey(cuit int64, service string) string {
	return fmt.Sprintf("wsaa:%d:%s", cuit, service)
}

// Authenticate returns a cached token if still valid, otherwise obtains a new
// one from WSAA and caches it.
func (cc *CachedClient) Authenticate(ctx context.Context, cert *x509.Certificate, key *rsa.PrivateKey, cuit int64, service string, production bool) (*TokenAcceso, error) {
	k := cacheKey(cuit, service)

	// Try cache first
	token, err := cc.store.Get(ctx, k)
	if err != nil {
		return nil, fmt.Errorf("token store get: %w", err)
	}
	if token != nil && !token.IsExpired(cc.renewalMargin) {
		return token, nil
	}

	// Cache miss or expired — authenticate
	token, err = cc.client.Authenticate(ctx, cert, key, service, production)
	if err != nil {
		return nil, err
	}

	// Store in cache (best-effort, don't fail the operation)
	if storeErr := cc.store.Set(ctx, k, token); storeErr != nil {
		// Log but don't return error — the token is still valid
		_ = storeErr
	}

	return token, nil
}

// Invalidate removes a cached token, forcing re-authentication on next call.
func (cc *CachedClient) Invalidate(ctx context.Context, cuit int64, service string) error {
	return cc.store.Delete(ctx, cacheKey(cuit, service))
}
