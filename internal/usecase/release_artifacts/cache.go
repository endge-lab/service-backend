package release_artifacts

import (
	"container/list"
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// artifactKey адресует конкретную неизменяемую версию artifact.
type artifactKey struct {
	workspaceID string
	releaseID   string
	checksum    string
}

type cacheEntry struct {
	key      artifactKey
	artifact entities.ReleaseArtifact
	size     int
}

// cacheItems объединяет быстрый поиск по ключу и порядок LRU.
// Все его методы вызываются Reader только под Reader.mu.
type cacheItems struct {
	byKey map[artifactKey]*list.Element
	lru   *list.List
	bytes int
}

func newCacheItems() cacheItems {
	return cacheItems{byKey: make(map[artifactKey]*list.Element), lru: list.New()}
}

func (c *cacheItems) get(key artifactKey) (entities.ReleaseArtifact, bool) {
	element, ok := c.byKey[key]
	if !ok {
		return entities.ReleaseArtifact{}, false
	}
	c.lru.MoveToFront(element)
	return cloneArtifact(element.Value.(cacheEntry).artifact), true
}

func (c *cacheItems) insert(key artifactKey, artifact *entities.ReleaseArtifact, maxBytes int, metrics cacheMetrics, ctx context.Context) {
	value := cacheEntry{key: key, artifact: cloneArtifact(*artifact), size: len(artifact.Data)}
	element := c.lru.PushFront(value)
	c.byKey[key] = element
	c.bytes += value.size
	metrics.addItems(ctx, 1, value.size)

	for c.bytes > maxBytes {
		c.remove(c.lru.Back(), metrics, ctx, true)
	}
}

// removeStaleVersions освобождает bytes старой checksum того же release.
// В нормальной immutable-модели это защитный сценарий, а не обычное вытеснение LRU.
func (c *cacheItems) removeStaleVersions(key artifactKey, metrics cacheMetrics, ctx context.Context) {
	for current, element := range c.byKey {
		if current.workspaceID == key.workspaceID && current.releaseID == key.releaseID && current.checksum != key.checksum {
			c.remove(element, metrics, ctx, false)
		}
	}
}

func (c *cacheItems) remove(element *list.Element, metrics cacheMetrics, ctx context.Context, eviction bool) {
	if element == nil {
		return
	}
	value := element.Value.(cacheEntry)
	delete(c.byKey, value.key)
	c.lru.Remove(element)
	c.bytes -= value.size
	metrics.addItems(ctx, -1, -value.size)
	if eviction {
		metrics.recordEviction(ctx)
	}
}

func cloneArtifact(value entities.ReleaseArtifact) entities.ReleaseArtifact {
	value.Data = append([]byte(nil), value.Data...)
	return value
}
