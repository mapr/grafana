package manager

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/grafana/grafana/pkg/registry/apis/secret/encryption"
	"github.com/grafana/grafana/pkg/registry/apis/secret/encryption/cipher"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/prometheus/client_golang/prometheus"
)

type ossDataKeyCache struct {
	mtx       sync.RWMutex
	byId      map[string]map[string]*encryption.DataKeyCacheEntry
	byLabel   map[string]map[string]*encryption.DataKeyCacheEntry
	cacheTTL  time.Duration
	optCipher cipher.Cipher
	cfg       *setting.Cfg
}

func ProvideOSSDataKeyCache(cfg *setting.Cfg, cipher cipher.Cipher) encryption.DataKeyCache {
	return &ossDataKeyCache{
		byId:      make(map[string]map[string]*encryption.DataKeyCacheEntry),
		byLabel:   make(map[string]map[string]*encryption.DataKeyCacheEntry),
		cacheTTL:  cfg.SecretsManagement.DataKeysCacheTTL,
		optCipher: cipher,
		cfg:       cfg,
	}
}

func (c *ossDataKeyCache) GetById(namespace, id string) (_ *encryption.DataKeyCacheEntry, exists bool) {
	defer func() {
		cacheReadsCounter.With(prometheus.Labels{
			"hit":    strconv.FormatBool(exists),
			"method": "byId",
		}).Inc()
	}()

	c.mtx.RLock()
	defer c.mtx.RUnlock()

	entries, exists := c.byId[namespace]
	if !exists {
		return nil, false
	}
	entry, exists := entries[id]
	if !exists || entry.IsExpired() || entry.Namespace != namespace {
		return nil, false
	}

	if c.cfg.SecretsManagement.UseCipherForDataKeyCache {
		dek, err := c.optCipher.Decrypt(context.TODO(), entry.EncryptedDataKey, string(entry.Id))
		if err != nil {
			return nil, false
		}
		// Create a copy to avoid mutating the cached entry
		entryCopy := &encryption.DataKeyCacheEntry{
			Namespace:  entry.Namespace,
			Id:         entry.Id,
			Label:      entry.Label,
			DataKey:    dek,
			Active:     entry.Active,
			Expiration: entry.Expiration,
		}
		return entryCopy, true
	}

	return entry, true
}

func (c *ossDataKeyCache) GetByLabel(namespace, label string) (_ *encryption.DataKeyCacheEntry, exists bool) {
	defer func() {
		cacheReadsCounter.With(prometheus.Labels{
			"hit":    strconv.FormatBool(exists),
			"method": "byLabel",
		}).Inc()
	}()

	c.mtx.RLock()
	defer c.mtx.RUnlock()

	entries, exists := c.byLabel[namespace]
	if !exists {
		return nil, false
	}
	entry, exists := entries[label]
	if !exists || entry.IsExpired() || entry.Namespace != namespace {
		return nil, false
	}

	if c.cfg.SecretsManagement.UseCipherForDataKeyCache {
		dek, err := c.optCipher.Decrypt(context.TODO(), entry.EncryptedDataKey, string(entry.Id))
		if err != nil {
			return nil, false
		}
		// Create a copy to avoid mutating the cached entry
		entryCopy := &encryption.DataKeyCacheEntry{
			Namespace:  entry.Namespace,
			Id:         entry.Id,
			Label:      entry.Label,
			DataKey:    dek,
			Active:     entry.Active,
			Expiration: entry.Expiration,
		}
		return entryCopy, true
	}

	return entry, true
}

func (c *ossDataKeyCache) AddById(namespace string, entry *encryption.DataKeyCacheEntry) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	entry.Expiration = time.Now().Add(c.cacheTTL)
	entry.Namespace = namespace

	entries, exists := c.byId[namespace]
	if !exists {
		entries = make(map[string]*encryption.DataKeyCacheEntry)
		c.byId[namespace] = entries
	}
	entries[entry.Id] = entry
}

func (c *ossDataKeyCache) AddByLabel(namespace string, entry *encryption.DataKeyCacheEntry) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	entry.Expiration = time.Now().Add(c.cacheTTL)
	entry.Namespace = namespace

	entries, exists := c.byLabel[namespace]
	if !exists {
		entries = make(map[string]*encryption.DataKeyCacheEntry)
		c.byLabel[namespace] = entries
	}
	entries[entry.Label] = entry
}

func (c *ossDataKeyCache) RemoveExpired() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	for _, entries := range c.byId {
		for id, entry := range entries {
			if entry.IsExpired() {
				delete(entries, id)
			}
		}
	}

	for _, entries := range c.byLabel {
		for label, entry := range entries {
			if entry.IsExpired() {
				delete(entries, label)
			}
		}
	}
}

func (c *ossDataKeyCache) Flush(namespace string) {
	c.mtx.Lock()
	c.byId[namespace] = make(map[string]*encryption.DataKeyCacheEntry)
	c.byLabel[namespace] = make(map[string]*encryption.DataKeyCacheEntry)
	c.mtx.Unlock()
}
