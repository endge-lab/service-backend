package portable

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestPlannerRemapsImportedRelationByIdentity(t *testing.T) {
	types := newFakeAdapter("type")
	filters := newFakeAdapter("filter")
	planner := newPlannerForTest(t, nil, types, filters)

	result, err := planner.Import(context.Background(), []PortableDocument{
		portableDocument("type", "Money"),
		{EntityType: "filter", Identity: "Orders", Canonical: json.RawMessage(`{"field":"amount"}`), Relations: []PortableRelation{{Path: "fields[0].type", EntityType: "type", Identity: "Money"}}},
	}, ImportOptions{ConflictPolicy: ConflictPolicyFail})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 2 || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	moneyID := types.ids["Money"]
	ordersID := filters.ids["Orders"]
	relations := filters.relations[ordersID]
	if len(relations) != 1 || relations[0].TargetID != moneyID || relations[0].EntityType != "type" {
		t.Fatalf("relations = %#v, moneyID = %s", relations, moneyID)
	}
	if string(filters.documents[ordersID].Canonical) != `{"field":"amount"}` {
		t.Fatalf("canonical document was rewritten: %s", filters.documents[ordersID].Canonical)
	}
}

func TestPlannerConflictPolicies(t *testing.T) {
	t.Run("fail", func(t *testing.T) {
		types := newFakeAdapter("type")
		types.seed("Orders")
		result, err := newPlannerForTest(t, nil, types).Import(context.Background(), []PortableDocument{portableDocument("type", "Orders")}, ImportOptions{ConflictPolicy: ConflictPolicyFail})
		if err != nil || len(result.Errors) != 1 || result.Errors[0].Code != "import_identity_conflict" || len(result.Created) != 0 {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
	})

	t.Run("overwrite", func(t *testing.T) {
		types := newFakeAdapter("type")
		originalID := types.seed("Orders")
		result, err := newPlannerForTest(t, nil, types).Import(context.Background(), []PortableDocument{portableDocument("type", "Orders")}, ImportOptions{ConflictPolicy: ConflictPolicyOverwrite})
		if err != nil || len(result.Updated) != 1 || types.overwrites[originalID] != 1 {
			t.Fatalf("result = %#v, overwrites = %#v, err = %v", result, types.overwrites, err)
		}
	})

	t.Run("rename requires explicit target", func(t *testing.T) {
		types := newFakeAdapter("type")
		types.seed("Orders")
		result, err := newPlannerForTest(t, nil, types).Import(context.Background(), []PortableDocument{portableDocument("type", "Orders")}, ImportOptions{ConflictPolicy: ConflictPolicyRename})
		if err != nil || len(result.Errors) != 1 || len(types.ids) != 1 {
			t.Fatalf("result = %#v, ids = %#v, err = %v", result, types.ids, err)
		}
	})

	t.Run("rename creates explicit target", func(t *testing.T) {
		types := newFakeAdapter("type")
		types.seed("Orders")
		key := EntityKey{EntityType: "type", Identity: "Orders"}
		result, err := newPlannerForTest(t, nil, types).Import(context.Background(), []PortableDocument{portableDocument("type", "Orders")}, ImportOptions{ConflictPolicy: ConflictPolicyRename, RenameIdentities: map[EntityKey]string{key: "OrdersImported"}})
		if err != nil || len(result.Created) != 1 || result.Created[0].Identity != "OrdersImported" || types.ids["OrdersImported"] == uuid.Nil {
			t.Fatalf("result = %#v, ids = %#v, err = %v", result, types.ids, err)
		}
	})
}

func TestPlannerNonAtomicKeepsValidDocumentsAndReportsUnresolvedRelation(t *testing.T) {
	types := newFakeAdapter("type")
	filters := newFakeAdapter("filter")
	planner := newPlannerForTest(t, nil, types, filters)

	result, err := planner.Import(context.Background(), []PortableDocument{
		portableDocument("type", "Money"),
		{EntityType: "filter", Identity: "Orders", Canonical: json.RawMessage(`{}`), Relations: []PortableRelation{{Path: "fields[0].type", EntityType: "type", Identity: "Unknown"}}},
	}, ImportOptions{ConflictPolicy: ConflictPolicyFail})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 || result.Created[0] != (EntityKey{EntityType: "type", Identity: "Money"}) || len(result.Errors) != 1 || result.Errors[0].Code != "unresolved_dependency" {
		t.Fatalf("result = %#v", result)
	}
	if _, exists := filters.ids["Orders"]; exists {
		t.Fatal("invalid filter must not be persisted in non-atomic import")
	}
}

func TestPlannerAtomicRollsBackOnAdapterFailure(t *testing.T) {
	types := newFakeAdapter("type")
	filters := newFakeAdapter("filter")
	filters.applyErr = errors.New("apply relations failed")
	tx := &fakeTxManager{adapters: []*fakeAdapter{types, filters}}
	planner := newPlannerForTest(t, tx, types, filters)

	_, err := planner.Import(context.Background(), []PortableDocument{
		portableDocument("type", "Money"),
		{EntityType: "filter", Identity: "Orders", Canonical: json.RawMessage(`{}`), Relations: []PortableRelation{{Path: "fields[0].type", EntityType: "type", Identity: "Money"}}},
	}, ImportOptions{ConflictPolicy: ConflictPolicyFail, Atomic: true})
	if err == nil || tx.calls != 1 {
		t.Fatalf("error = %v, transaction calls = %d", err, tx.calls)
	}
	if len(types.ids) != 0 || len(filters.ids) != 0 {
		t.Fatalf("atomic import left persisted data: types=%#v filters=%#v", types.ids, filters.ids)
	}
}

func newPlannerForTest(t *testing.T, tx *fakeTxManager, adapters ...EntityPortableAdapter) *Planner {
	t.Helper()
	registry, err := NewRegistry(adapters...)
	if err != nil {
		t.Fatal(err)
	}
	return NewPlanner(registry, tx)
}

func portableDocument(entityType, identity string) PortableDocument {
	return PortableDocument{EntityType: entityType, Identity: identity, Canonical: json.RawMessage(`{}`)}
}

type fakeAdapter struct {
	entityType string
	ids        map[string]uuid.UUID
	documents  map[uuid.UUID]PortableDocument
	relations  map[uuid.UUID][]ResolvedRelation
	overwrites map[uuid.UUID]int
	applyErr   error
}

func newFakeAdapter(entityType string) *fakeAdapter {
	return &fakeAdapter{entityType: entityType, ids: map[string]uuid.UUID{}, documents: map[uuid.UUID]PortableDocument{}, relations: map[uuid.UUID][]ResolvedRelation{}, overwrites: map[uuid.UUID]int{}}
}

func (a *fakeAdapter) EntityType() string { return a.entityType }

func (a *fakeAdapter) Export(_ context.Context, id uuid.UUID) (PortableDocument, error) {
	return a.documents[id], nil
}

func (a *fakeAdapter) FindByIdentity(_ context.Context, identity string) (uuid.UUID, bool, error) {
	id, found := a.ids[identity]
	return id, found, nil
}

func (a *fakeAdapter) CreateBase(_ context.Context, document PortableDocument) (uuid.UUID, error) {
	id := uuid.New()
	a.ids[document.Identity] = id
	a.documents[id] = document
	return id, nil
}

func (a *fakeAdapter) OverwriteBase(_ context.Context, id uuid.UUID, document PortableDocument) error {
	a.documents[id] = document
	a.overwrites[id]++
	return nil
}

func (a *fakeAdapter) ApplyRelations(_ context.Context, id uuid.UUID, relations []ResolvedRelation) error {
	if a.applyErr != nil {
		return a.applyErr
	}
	a.relations[id] = append([]ResolvedRelation(nil), relations...)
	return nil
}

func (a *fakeAdapter) seed(identity string) uuid.UUID {
	id := uuid.New()
	a.ids[identity] = id
	a.documents[id] = portableDocument(a.entityType, identity)
	return id
}

type fakeAdapterSnapshot struct {
	ids        map[string]uuid.UUID
	documents  map[uuid.UUID]PortableDocument
	relations  map[uuid.UUID][]ResolvedRelation
	overwrites map[uuid.UUID]int
}

type fakeTxManager struct {
	adapters []*fakeAdapter
	calls    int
}

func (m *fakeTxManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	m.calls++
	snapshots := make([]fakeAdapterSnapshot, len(m.adapters))
	for index, adapter := range m.adapters {
		snapshots[index] = adapter.snapshot()
	}
	if err := fn(ctx); err != nil {
		for index, adapter := range m.adapters {
			adapter.restore(snapshots[index])
		}
		return err
	}
	return nil
}

func (a *fakeAdapter) snapshot() fakeAdapterSnapshot {
	snapshot := fakeAdapterSnapshot{ids: map[string]uuid.UUID{}, documents: map[uuid.UUID]PortableDocument{}, relations: map[uuid.UUID][]ResolvedRelation{}, overwrites: map[uuid.UUID]int{}}
	for key, value := range a.ids {
		snapshot.ids[key] = value
	}
	for key, value := range a.documents {
		snapshot.documents[key] = value
	}
	for key, value := range a.relations {
		snapshot.relations[key] = append([]ResolvedRelation(nil), value...)
	}
	for key, value := range a.overwrites {
		snapshot.overwrites[key] = value
	}
	return snapshot
}

func (a *fakeAdapter) restore(snapshot fakeAdapterSnapshot) {
	a.ids = snapshot.ids
	a.documents = snapshot.documents
	a.relations = snapshot.relations
	a.overwrites = snapshot.overwrites
}
