package sidecar

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// YAMLCache caches YAML validation results based on file mtime+size
type YAMLCache struct {
	entries   map[string]cacheEntry
	mu        sync.RWMutex
	cacheHits int
}

type cacheEntry struct {
	mtime    int64
	size     int64
	checksum string
}

// NewYAMLCache creates a new YAML validation cache
func NewYAMLCache() *YAMLCache {
	return &YAMLCache{
		entries: make(map[string]cacheEntry),
	}
}

// Validate validates YAML files with mtime+size caching
func (c *YAMLCache) Validate(paths []string) error {
	for _, path := range paths {
		// Get file info
		stat, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		mtime := stat.ModTime().Unix()
		size := stat.Size()

		// Check cache
		c.mu.RLock()
		cached, exists := c.entries[path]
		c.mu.RUnlock()

		if exists && cached.mtime == mtime && cached.size == size {
			// Cache hit
			c.mu.Lock()
			c.cacheHits++
			c.mu.Unlock()
			continue
		}

		// Cache miss — validate file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var parsed map[string]interface{}
		if err := yaml.Unmarshal(content, &parsed); err != nil {
			return fmt.Errorf("invalid YAML %s: %w", path, err)
		}

		// Update cache
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))
		c.mu.Lock()
		c.entries[path] = cacheEntry{
			mtime:    mtime,
			size:     size,
			checksum: checksum[:16], // First 16 chars for size
		}
		c.mu.Unlock()
	}

	return nil
}

// GetCacheHits returns the number of cache hits (for testing)
func (c *YAMLCache) GetCacheHits() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cacheHits
}
