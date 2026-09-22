package release_artifacts

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

const artifactLoadTimeout = 10 * time.Second

// loadFlight хранит результат одной текущей загрузки artifact. Запросы с тем же
// key ждут done вместо повторного SQL-запроса.
type loadFlight struct {
	done     chan struct{}
	artifact *entities.ReleaseArtifact
	err      error
}
