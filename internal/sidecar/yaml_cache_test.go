package sidecar

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestYAMLCache(t *testing.T) {
	t.Run("should validate valid YAML files", func(t *testing.T) {
		// ARRANGE
		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.yaml")
		os.WriteFile(validFile, []byte("key: value\n"), 0644)
		cache := NewYAMLCache()

		// ACT
		err := cache.Validate([]string{validFile})

		// ASSERT
		if err != nil {
			t.Errorf("Expected no error for valid YAML, got %v", err)
		}
	})

	t.Run("should reject invalid YAML files", func(t *testing.T) {
		// ARRANGE
		tmpDir := t.TempDir()
		invalidFile := filepath.Join(tmpDir, "invalid.yaml")
		os.WriteFile(invalidFile, []byte("key: [unclosed\n"), 0644)
		cache := NewYAMLCache()

		// ACT
		err := cache.Validate([]string{invalidFile})

		// ASSERT
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("should use cache for unchanged files", func(t *testing.T) {
		// ARRANGE
		tmpDir := t.TempDir()
		yamlFile := filepath.Join(tmpDir, "data.yaml")
		os.WriteFile(yamlFile, []byte("key: value\n"), 0644)
		cache := NewYAMLCache()

		// First validation (cache miss)
		cache.Validate([]string{yamlFile})
		firstHits := cache.GetCacheHits()

		// Second validation (cache hit)
		cache.Validate([]string{yamlFile})
		secondHits := cache.GetCacheHits()

		// ASSERT
		if secondHits <= firstHits {
			t.Errorf("Expected cache hits to increase, got first=%d second=%d", firstHits, secondHits)
		}
	})

	t.Run("should revalidate on mtime change", func(t *testing.T) {
		// ARRANGE
		tmpDir := t.TempDir()
		yamlFile := filepath.Join(tmpDir, "data.yaml")
		os.WriteFile(yamlFile, []byte("key: old\n"), 0644)
		cache := NewYAMLCache()

		// First validation
		cache.Validate([]string{yamlFile})

		// Modify file
		time.Sleep(10 * time.Millisecond) // Ensure mtime changes
		os.WriteFile(yamlFile, []byte("key: new\n"), 0644)

		// ACT
		err := cache.Validate([]string{yamlFile})

		// ASSERT
		if err != nil {
			t.Errorf("Expected revalidation to succeed, got %v", err)
		}
	})

	t.Run("should revalidate on size change", func(t *testing.T) {
		// ARRANGE
		tmpDir := t.TempDir()
		yamlFile := filepath.Join(tmpDir, "data.yaml")
		os.WriteFile(yamlFile, []byte("key: value\n"), 0644)
		cache := NewYAMLCache()
		cache.Validate([]string{yamlFile})

		// Change size
		os.WriteFile(yamlFile, []byte("key: longer_value\n"), 0644)

		// ACT
		err := cache.Validate([]string{yamlFile})

		// ASSERT
		if err != nil {
			t.Errorf("Expected revalidation to succeed, got %v", err)
		}
	})

	t.Run("should handle missing files gracefully", func(t *testing.T) {
		// ARRANGE
		cache := NewYAMLCache()

		// ACT
		err := cache.Validate([]string{"/nonexistent/file.yaml"})

		// ASSERT
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})

	t.Run("should be concurrent-safe", func(t *testing.T) {
		// ARRANGE
		tmpDir := t.TempDir()
		yamlFile := filepath.Join(tmpDir, "data.yaml")
		os.WriteFile(yamlFile, []byte("key: value\n"), 0644)
		cache := NewYAMLCache()

		// ACT — 10 concurrent validations
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.Validate([]string{yamlFile})
			}()
		}
		wg.Wait()

		// ASSERT — no panic occurred
	})
}
