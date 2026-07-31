package dependencies

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestReplaceForOwnerNormalizesDeduplicatesAndUsesTransaction(t *testing.T) {
	workspaceID := uuid.New()
	ownerID := uuid.New()
	repository := &repositoryStub{}
	tx := &txManagerStub{}
	service := newDependenciesService(repository, tx)

	verificationError := " parser is not available "
	err := service.ReplaceForOwner(workspaceContext(workspaceID), entities.DomainDependencyOwner{Type: " type ", ID: ownerID, Identity: " OrderList "}, DependencyExtractionResult{
		References: []entities.DomainDependencyReference{
			{Type: " type ", Identity: " Money ", SourcePath: " schema.fields[0].type "},
			{Type: "type", Identity: "Money", SourcePath: "schema.fields[0].type"},
		},
		VerificationState: entities.DomainDependencyVerificationStateUnverified,
		VerificationError: &verificationError,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.calls != 1 || repository.replaceCalls != 1 {
		t.Fatalf("transaction calls = %d, replace calls = %d", tx.calls, repository.replaceCalls)
	}
	if repository.owner.Type != "type" || repository.owner.Identity != "OrderList" || len(repository.references) != 1 || repository.references[0].Identity != "Money" {
		t.Fatalf("owner = %#v, references = %#v", repository.owner, repository.references)
	}
	if repository.state != entities.DomainDependencyVerificationStateUnverified || repository.verificationError == nil || *repository.verificationError != "parser is not available" {
		t.Fatalf("state = %q, error = %#v", repository.state, repository.verificationError)
	}
}

func TestReplaceForOwnerRejectsVerifiedExtractionError(t *testing.T) {
	message := "unexpected"
	tx := &txManagerStub{}
	service := newDependenciesService(&repositoryStub{}, tx)
	err := service.ReplaceForOwner(workspaceContext(uuid.New()), entities.DomainDependencyOwner{Type: "type", ID: uuid.New(), Identity: "Orders"}, DependencyExtractionResult{
		VerificationState: entities.DomainDependencyVerificationStateVerified,
		VerificationError: &message,
	})
	if code := apperrors.CodeOf(err); code != "validation_error" {
		t.Fatalf("error code = %q", code)
	}
	if tx.calls != 0 {
		t.Fatalf("transaction calls = %d, want 0", tx.calls)
	}
}

func TestEnsureNotReferencedReturnsEntitySpecificConflict(t *testing.T) {
	repository := &repositoryStub{usages: entities.DomainDependencyUsages{
		Items: []entities.DomainDependencyUsage{{OwnerType: "filter", OwnerID: uuid.New(), OwnerIdentity: "orders", SourcePath: "fields[0].type", VerificationState: entities.DomainDependencyVerificationStateVerified}},
		Total: 23,
	}}
	service := newDependenciesService(repository, &txManagerStub{})
	err := service.EnsureNotReferenced(workspaceContext(uuid.New()), "type", "type", "Orders")
	if code := apperrors.CodeOf(err); code != "type_in_use" {
		t.Fatalf("error code = %q", code)
	}
	details := apperrors.DetailsOf(err)
	if details["total"] != int64(23) || repository.ensureLimit != deleteGuardUsageLimit {
		t.Fatalf("details = %#v, limit = %d", details, repository.ensureLimit)
	}
}

func TestListUsagesAppliesPageDefaultsAndBounds(t *testing.T) {
	repository := &repositoryStub{usages: entities.DomainDependencyUsages{}}
	service := newDependenciesService(repository, &txManagerStub{})
	result, err := service.ListUsages(workspaceContext(uuid.New()), ListUsagesInput{DependencyType: " type ", DependencyIdentity: " Orders "})
	if err != nil {
		t.Fatal(err)
	}
	if result.Items == nil || result.Limit != defaultUsagesLimit || result.Offset != 0 || repository.listOptions.Limit != defaultUsagesLimit || repository.listType != "type" || repository.listIdentity != "Orders" {
		t.Fatalf("result = %#v, options = %#v", result, repository.listOptions)
	}

	limit := maxUsagesLimit + 1
	if _, err := service.ListUsages(workspaceContext(uuid.New()), ListUsagesInput{DependencyType: "type", DependencyIdentity: "Orders", Limit: &limit}); apperrors.CodeOf(err) != "validation_error" {
		t.Fatalf("limit error code = %q", apperrors.CodeOf(err))
	}
}

func TestDependencyExtractorIsTyped(t *testing.T) {
	extractor := fakeDependencyExtractor{}
	result, err := extractor.Extract(testDocument{Identity: "Orders"})
	if err != nil || result.VerificationState != entities.DomainDependencyVerificationStateVerified || len(result.References) != 1 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

func newDependenciesService(repository ports.DomainDependenciesRepository, tx ports.TxManager) *Dependencies {
	return NewDependenciesService(DependenciesParams{Repository: repository, TxManager: tx, Observability: observability.NewCore(otel.Tracer("dependencies-test"), zap.NewNop())})
}

func workspaceContext(workspaceID uuid.UUID) context.Context {
	return entities.WithWorkspaceID(context.Background(), workspaceID)
}

type repositoryStub struct {
	ports.DomainDependenciesRepository
	replaceCalls      int
	owner             entities.DomainDependencyOwner
	references        []entities.DomainDependencyReference
	state             entities.DomainDependencyVerificationState
	verificationError *string
	usages            entities.DomainDependencyUsages
	ensureLimit       int
	listType          string
	listIdentity      string
	listOptions       ports.DomainDependenciesListOptions
}

func (s *repositoryStub) ReplaceForOwner(_ context.Context, owner entities.DomainDependencyOwner, references []entities.DomainDependencyReference, state entities.DomainDependencyVerificationState, verificationError *string) error {
	s.replaceCalls++
	s.owner = owner
	s.references = references
	s.state = state
	s.verificationError = verificationError
	return nil
}

func (s *repositoryStub) EnsureNotReferenced(_ context.Context, _, _ string, limit int) (entities.DomainDependencyUsages, error) {
	s.ensureLimit = limit
	return s.usages, nil
}

func (s *repositoryStub) ListUsages(_ context.Context, dependencyType, dependencyIdentity string, options ports.DomainDependenciesListOptions) (entities.DomainDependencyUsages, error) {
	s.listType = dependencyType
	s.listIdentity = dependencyIdentity
	s.listOptions = options
	return s.usages, nil
}

type txManagerStub struct{ calls int }

func (s *txManagerStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	s.calls++
	return fn(ctx)
}

type testDocument struct{ Identity string }

type fakeDependencyExtractor struct{}

func (fakeDependencyExtractor) Extract(document testDocument) (DependencyExtractionResult, error) {
	return DependencyExtractionResult{References: []entities.DomainDependencyReference{{Type: "type", Identity: document.Identity, SourcePath: "schema.identity"}}, VerificationState: entities.DomainDependencyVerificationStateVerified}, nil
}
