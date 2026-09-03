package config

import (
	"fmt"

	"github.com/endge-lab/service-backend/internal/buildinfo"
	kitconfig "github.com/endge-lab/service-kit-go/config"
)

// Config объединяет общую конфигурацию service-kit с настройками Endge Backend.
type Config struct {
	*kitconfig.ServiceConfig
	WorkspaceSchemaVersion int
	HTTPBasePath           string
	Identity               IdentityConfig
	ConfiguratorAuth       ConfiguratorAuthConfig
	Encryption             EncryptionConfig
	Snapshots              SnapshotConfig
	ReleaseArtifactCache   ReleaseArtifactCacheConfig
	AIWorkbench            AIWorkbenchConfig
}

// Load загружает базовую конфигурацию service-kit и дополняет её настройками
// конкретного backend-сервиса.
func Load() (*Config, error) {
	base, err := kitconfig.LoadServiceConfig()
	if err != nil {
		return nil, err
	}
	buildMetadata, err := buildinfo.Resolve(base.App.Version)
	if err != nil {
		return nil, fmt.Errorf("load build metadata: %w", err)
	}
	base.App.Version = buildMetadata.AppVersion

	httpBasePath, err := loadHTTPBasePath(base)
	if err != nil {
		return nil, err
	}
	identity, err := loadIdentityConfig(base)
	if err != nil {
		return nil, err
	}
	encryption, err := loadEncryptionConfig()
	if err != nil {
		return nil, err
	}
	configuratorAuth, err := loadConfiguratorAuthConfig(base, identity)
	if err != nil {
		return nil, err
	}
	releaseArtifactCache, err := loadReleaseArtifactCacheConfig()
	if err != nil {
		return nil, err
	}
	aiWorkbench, err := loadAIWorkbenchConfig(base)
	if err != nil {
		return nil, err
	}

	return &Config{
		ServiceConfig:          base,
		WorkspaceSchemaVersion: buildMetadata.WorkspaceSchemaVersion,
		HTTPBasePath:           httpBasePath,
		Identity:               identity,
		ConfiguratorAuth:       configuratorAuth,
		Encryption:             encryption,
		Snapshots:              loadSnapshotConfig(),
		ReleaseArtifactCache:   releaseArtifactCache,
		AIWorkbench:            aiWorkbench,
	}, nil
}
