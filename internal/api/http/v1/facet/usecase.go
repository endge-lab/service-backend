package facet

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/facets"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type UseCase interface {
	List(context.Context, bool) ([]entities.Document, error)
	Get(context.Context, string, bool) (*entities.Document, error)
	Create(context.Context, resourceusecase.FacetCreateInput) (*entities.Document, error)
	Patch(context.Context, string, resourceusecase.FacetPatchInput, int) (*entities.Document, error)
	Delete(context.Context, string, int) (*entities.Document, error)
	Restore(context.Context, string, int) (*entities.Document, error)
	Reorder(context.Context, []ports.FacetOrderItem) ([]entities.Document, error)
	ListDocuments(context.Context, string, ports.DocumentFilter) ([]entities.Document, error)
	GetDocument(context.Context, string, string, bool) (*entities.Document, error)
	CreateDocument(context.Context, string, resourceusecase.FacetDocumentCreateInput) (*entities.Document, error)
	PatchDocument(context.Context, string, string, resourceusecase.FacetDocumentPatchInput, int) (*entities.Document, error)
	DeleteDocument(context.Context, string, string, int) (*entities.Document, error)
	RestoreDocument(context.Context, string, string, int) (*entities.Document, error)
	ListRevisions(context.Context, string) ([]entities.Revision, error)
	GetRevision(context.Context, string, string) (*entities.Revision, error)
	RestoreRevision(context.Context, string, string, int) (*entities.Document, error)
	ListDocumentRevisions(context.Context, string, string) ([]entities.Revision, error)
	GetDocumentRevision(context.Context, string, string, string) (*entities.Revision, error)
	RestoreDocumentRevision(context.Context, string, string, string, int) (*entities.Document, error)
}

func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
