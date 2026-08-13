package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// DocumentFilter задаёт фильтры и пагинацию списка документов.
type DocumentFilter struct {
	IncludeDeleted bool
	FolderIdentity *string
	Active         *bool
	Limit, Offset  int
}

// DocumentRepository задаёт порт хранения документов для use case-слоя.
type DocumentRepository interface {
	ListDocuments(context.Context, string, string, DocumentFilter) ([]entities.Document, error)
	ListAllDocuments(context.Context, string, string, bool) ([]entities.Document, error)
	GetDocument(context.Context, string, string, string, bool) (*entities.Document, error)
	InsertDocument(context.Context, entities.Document, *string) (*entities.Document, error)
	UpdateDocument(context.Context, entities.Document, int, *string) (*entities.Document, error)
	MoveFolderContents(context.Context, string, string, *string, string) ([]entities.Document, error)
	ResolveFolder(context.Context, string, string, string) (*string, error)
	FolderWouldCycle(context.Context, string, string, string) (bool, error)
	ReplaceProjectEnvironments(context.Context, entities.Document, []string) error
}

// DocumentResourceRepository задаёт операции над документами одного типа.
// PostgreSQL-адаптер связывает реализацию ровно с одной таблицей, поэтому use case не передаёт имя таблицы или коллекции.
type DocumentResourceRepository interface {
	List(context.Context, string, DocumentFilter) ([]entities.Document, error)
	Get(context.Context, string, string, bool) (*entities.Document, error)
	Insert(context.Context, entities.Document, *string) (*entities.Document, error)
	Update(context.Context, entities.Document, int, *string) (*entities.Document, error)
}

// ProjectRepository задаёт порт хранения проектов для use case-слоя.
type ProjectRepository interface{ DocumentResourceRepository }

// TenantRepository задаёт порт хранения тенантов для use case-слоя.
type TenantRepository interface{ DocumentResourceRepository }

// EnvironmentRepository задаёт порт хранения окружений для use case-слоя.
type EnvironmentRepository interface{ DocumentResourceRepository }

// FolderRepository задаёт порт хранения папок для use case-слоя.
type FolderRepository interface{ DocumentResourceRepository }

// TypeRepository задаёт порт хранения типов для use case-слоя.
type TypeRepository interface{ DocumentResourceRepository }

// QueryRepository задаёт порт хранения запросов для use case-слоя.
type QueryRepository interface{ DocumentResourceRepository }

// DataViewRepository задаёт порт хранения представлений данных для use case-слоя.
type DataViewRepository interface{ DocumentResourceRepository }

// CompositionRepository задаёт порт хранения композиций для use case-слоя.
type CompositionRepository interface{ DocumentResourceRepository }

// StoreRepository задаёт порт хранения хранилищ для use case-слоя.
type StoreRepository interface{ DocumentResourceRepository }

// StreamRepository задаёт порт хранения потоков для use case-слоя.
type StreamRepository interface{ DocumentResourceRepository }

// UpdateRepository задаёт порт хранения обновлений состояния для use case-слоя.
type UpdateRepository interface{ DocumentResourceRepository }

// MockRepository задаёт порт хранения моков для use case-слоя.
type MockRepository interface{ DocumentResourceRepository }

// ComponentRepository задаёт порт хранения компонентов для use case-слоя.
type ComponentRepository interface{ DocumentResourceRepository }

// ActionRepository задаёт порт хранения действий для use case-слоя.
type ActionRepository interface{ DocumentResourceRepository }

// FilterRepository задаёт порт хранения фильтров для use case-слоя.
type FilterRepository interface{ DocumentResourceRepository }

// ConverterRepository задаёт порт хранения конвертеров для use case-слоя.
type ConverterRepository interface{ DocumentResourceRepository }

// ComputationRepository задаёт порт хранения вычислений для use case-слоя.
type ComputationRepository interface{ DocumentResourceRepository }

// VocabRepository задаёт порт хранения справочников для use case-слоя.
type VocabRepository interface{ DocumentResourceRepository }

// I18nBundleRepository задаёт порт хранения пакетов локализации для use case-слоя.
type I18nBundleRepository interface{ DocumentResourceRepository }

// AuthProfileRepository задаёт порт хранения профилей аутентификации для use case-слоя.
type AuthProfileRepository interface{ DocumentResourceRepository }

// NavigationRepository задаёт порт хранения навигаций для use case-слоя.
type NavigationRepository interface{ DocumentResourceRepository }

// StyleRepository задаёт порт хранения стилей для use case-слоя.
type StyleRepository interface{ DocumentResourceRepository }
