package promclient

import (
	"net/http"
	"sort"
	"strings"

	lru "github.com/hashicorp/golang-lru"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type ProviderCache struct {
	provider        promClientProvider
	promClientCache *lru.Cache
	httpClientCache *lru.Cache
}

type promClientProvider interface {
	GetPromClient(map[string]string) (apiv1.API, error)
	GetHTTPClient(map[string]string) (*http.Client, error)
}

func NewProviderCache(p promClientProvider) (*ProviderCache, error) {
	promClientCache, err := lru.New(500)
	if err != nil {
		return nil, err
	}

	httpClientCache, err := lru.New(500)
	if err != nil {
		return nil, err
	}

	return &ProviderCache{
		provider:        p,
		promClientCache: promClientCache,
		httpClientCache: httpClientCache,
	}, nil
}

func (c *ProviderCache) GetPromClient(headers map[string]string) (apiv1.API, error) {
	key := c.key(headers)
	if client, ok := c.promClientCache.Get(key); ok {
		return client.(apiv1.API), nil
	}

	client, err := c.provider.GetPromClient(headers)
	if err != nil {
		return nil, err
	}

	c.promClientCache.Add(key, client)
	return client, nil
}

func (c *ProviderCache) GetHTTPClient(headers map[string]string) (*http.Client, error) {
	key := c.key(headers)
	if client, ok := c.httpClientCache.Get(key); ok {
		return client.(*http.Client), nil
	}

	client, err := c.provider.GetHTTPClient(headers)
	if err != nil {
		return nil, err
	}

	c.httpClientCache.Add(key, client)
	return client, nil
}

func (c *ProviderCache) key(headers map[string]string) string {
	vals := make([]string, len(headers))
	var i int
	for _, v := range headers {
		vals[i] = v
		i++
	}
	sort.Strings(vals)
	return strings.Join(vals, "")
}
