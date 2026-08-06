// Package release_artifacts владеет чтением immutable JSON артефактов релизов.
package release_artifacts

import (
	"context"
	"sync"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"go.opentelemetry.io/otel/metric"
)

// Reader возвращает artifact из bounded LRU или постоянного хранилища.
// Он не проверяет права: это делают use case до вызова Read.
type Reader struct {
	repository ports.ReleaseArtifactRepository
	config     config.ReleaseArtifactCacheConfig
	metrics    cacheMetrics

	mu          sync.Mutex
	cache       cacheItems
	inflight    map[artifactKey]*loadFlight
	loadTimeout time.Duration
}

// NewReader создаёт единый на процесс reader. Кэш не разделяется между репликами.
func NewReader(repository ports.ReleaseArtifactRepository, cacheConfig config.ReleaseArtifactCacheConfig, meter metric.Meter) (*Reader, error) {
	metrics, err := newCacheMetrics(meter)
	if err != nil {
		return nil, err
	}
	return &Reader{
		repository:  repository,
		config:      cacheConfig,
		metrics:     metrics,
		cache:       newCacheItems(),
		inflight:    make(map[artifactKey]*loadFlight),
		loadTimeout: artifactLoadTimeout,
	}, nil
}

// Read получает artifact конкретного immutable release. Checksum входит в key:
// если целостность release когда-либо будет нарушена, старые bytes не выдадутся.
func (r *Reader) Read(ctx context.Context, operation, workspaceID string, release entities.Release) (*entities.ReleaseArtifact, error) {
	key := artifactKey{workspaceID: workspaceID, releaseID: release.ID, checksum: release.Checksum}
	if !r.config.Enabled {
		return r.loadWithoutCache(ctx, operation, workspaceID, release)
	}

	r.mu.Lock()
	if artifact, ok := r.cache.get(key); ok {
		r.mu.Unlock()
		r.metrics.recordRequest(ctx, operation, "hit")
		return &artifact, nil
	}
	flight, exists := r.inflight[key]
	if !exists {
		flight = &loadFlight{done: make(chan struct{})}
		r.inflight[key] = flight
	}
	r.mu.Unlock()

	if !exists {
		r.startFlight(ctx, operation, key, workspaceID, release, flight)
	}
	return r.waitForFlight(ctx, operation, flight)
}

func (r *Reader) startFlight(requestCtx context.Context, operation string, key artifactKey, workspaceID string, release entities.Release, flight *loadFlight) {
	loadCtx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), r.loadTimeout)
	go func() {
		defer cancel()
		r.runFlight(loadCtx, operation, key, workspaceID, release, flight)
	}()
}

func (r *Reader) runFlight(ctx context.Context, operation string, key artifactKey, workspaceID string, release entities.Release, flight *loadFlight) {
	startedAt := time.Now()
	artifact, err := r.load(ctx, workspaceID, release)
	r.metrics.recordLoad(ctx, operation, time.Since(startedAt))

	r.mu.Lock()
	if err == nil && r.canStore(len(artifact.Data)) {
		r.cache.removeStaleVersions(key, r.metrics, ctx)
		r.cache.insert(key, artifact, r.config.MaxBytes, r.metrics, ctx)
	}
	if err == nil {
		value := cloneArtifact(*artifact)
		flight.artifact = &value
	}
	flight.err = err
	delete(r.inflight, key)
	close(flight.done)
	r.mu.Unlock()
}

func (r *Reader) waitForFlight(ctx context.Context, operation string, flight *loadFlight) (*entities.ReleaseArtifact, error) {
	select {
	case <-flight.done:
		if flight.err != nil {
			r.metrics.recordRequest(ctx, operation, "error")
			return nil, flight.err
		}
		if r.canStore(len(flight.artifact.Data)) {
			r.metrics.recordRequest(ctx, operation, "miss")
		} else {
			r.metrics.recordRequest(ctx, operation, "bypass")
		}
		artifact := cloneArtifact(*flight.artifact)
		return &artifact, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *Reader) loadWithoutCache(ctx context.Context, operation, workspaceID string, release entities.Release) (*entities.ReleaseArtifact, error) {
	startedAt := time.Now()
	artifact, err := r.load(ctx, workspaceID, release)
	r.metrics.recordLoad(ctx, operation, time.Since(startedAt))
	if err != nil {
		r.metrics.recordRequest(ctx, operation, "error")
		return nil, err
	}
	r.metrics.recordRequest(ctx, operation, "bypass")
	value := cloneArtifact(*artifact)
	return &value, nil
}

func (r *Reader) load(ctx context.Context, workspaceID string, release entities.Release) (*entities.ReleaseArtifact, error) {
	artifact, err := r.repository.GetReleaseArtifact(ctx, workspaceID, release.ID)
	if err != nil {
		return nil, err
	}
	if artifact.ReleaseID != release.ID || artifact.WorkspaceID != workspaceID || artifact.Checksum != release.Checksum {
		return nil, domainerrors.Internal("release_artifact_inconsistent", "Release artifact metadata is inconsistent")
	}
	return artifact, nil
}

func (r *Reader) canStore(size int) bool {
	return size <= r.config.MaxItemBytes && size <= r.config.MaxBytes
}
