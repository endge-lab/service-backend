package config

import "fmt"

// ReleaseArtifactCacheConfig ограничивает локальный кеш immutable JSON релизов.
// Каждая реплика приложения использует собственный кеш.
type ReleaseArtifactCacheConfig struct {
	Enabled      bool
	MaxBytes     int
	MaxItemBytes int
}

func loadReleaseArtifactCacheConfig() (ReleaseArtifactCacheConfig, error) {
	config := ReleaseArtifactCacheConfig{
		Enabled:      envBool("RELEASE_ARTIFACT_CACHE_ENABLED", true),
		MaxBytes:     envIntAllowZero("RELEASE_ARTIFACT_CACHE_MAX_BYTES", 64*1024*1024),
		MaxItemBytes: envIntAllowZero("RELEASE_ARTIFACT_CACHE_MAX_ITEM_BYTES", 16*1024*1024),
	}
	if err := config.Validate(); err != nil {
		return ReleaseArtifactCacheConfig{}, err
	}
	return config, nil
}

func (c ReleaseArtifactCacheConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.MaxBytes <= 0 {
		return fmt.Errorf("RELEASE_ARTIFACT_CACHE_MAX_BYTES must be positive when cache is enabled")
	}
	if c.MaxItemBytes <= 0 {
		return fmt.Errorf("RELEASE_ARTIFACT_CACHE_MAX_ITEM_BYTES must be positive when cache is enabled")
	}
	return nil
}
