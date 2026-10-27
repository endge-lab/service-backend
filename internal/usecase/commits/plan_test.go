package commits

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// TestPlanWithDocumentsListsPendingDocuments проверяет состав документов плана:
// группировку по документу, последнее имя, операции, авторов и токены; REST-план без документов.
func TestPlanWithDocumentsListsPendingDocuments(t *testing.T) {
	tokenID, tokenName := "t-1", "Codex"
	owner, agentOwner := entities.Actor{ID: "u-1", Username: "egor"}, entities.Actor{ID: "u-2", Username: "anna"}
	repository := &fakeCommitRepository{pending: []entities.Revision{
		{ID: "r-1", DocumentType: "queries", DocumentID: "d-1", DocumentIdentity: "orders", Operation: "create", CreatedBy: owner, Snapshot: json.RawMessage(`{"displayName":"Заказы"}`)},
		{ID: "r-2", DocumentType: "components", DocumentID: "d-2", DocumentIdentity: "title", Operation: "update", CreatedBy: agentOwner, AccessTokenID: &tokenID, AccessTokenName: &tokenName, Snapshot: json.RawMessage(`{"displayName":"Заголовок"}`)},
		{ID: "r-3", DocumentType: "queries", DocumentID: "d-1", DocumentIdentity: "orders-v2", Operation: "update", CreatedBy: agentOwner, AccessTokenID: &tokenID, AccessTokenName: &tokenName, Snapshot: json.RawMessage(`{"displayName":"Заказы 2"}`)},
		{ID: "r-4", DocumentType: "queries", DocumentID: "d-1", DocumentIdentity: "orders-v2", Operation: "update", CreatedBy: owner, Snapshot: json.RawMessage(`{}`)},
	}}
	useCase := NewUseCase(repository, fakeRevisionRepository{}, fakeTx{}, nil)
	ctx := entities.WithWorkspaceAccess(context.Background(), entities.WorkspaceAccess{Workspace: entities.Workspace{ID: "w-1", HeadSequence: 9}, Role: "editor"})

	plain, err := useCase.Plan(ctx)
	if err != nil || plain.Documents != nil || plain.DocumentCount != 2 || plain.RevisionCount != 4 {
		t.Fatalf("REST plan must keep counters only: %+v %v", plain, err)
	}
	plan, err := useCase.PlanWithDocuments(ctx)
	if err != nil || len(plan.Documents) != 2 || plan.HeadSequence != 9 || plan.BaseSequence != 5 {
		t.Fatalf("plan: %+v %v", plan, err)
	}
	query := plan.Documents[0]
	if query.Collection != "queries" || query.Identity != "orders-v2" || query.DisplayName != "Заказы 2" || query.RevisionCount != 3 {
		t.Fatalf("query entry: %+v", query)
	}
	if !slices.Equal(query.Operations, []string{"create", "update"}) || len(query.Authors) != 2 || len(query.AccessTokens) != 1 || query.AccessTokens[0].Name != "Codex" {
		t.Fatalf("query operations, authors and tokens: %+v", query)
	}
	if component := plan.Documents[1]; component.Collection != "components" || component.RevisionCount != 1 || component.Authors[0].ID != "u-2" {
		t.Fatalf("component entry: %+v", component)
	}
}
