// Package facets owns the explicit lifecycle of dynamic facet definitions and
// their nested documents. It intentionally does not participate in the generic
// Domain document lifecycle or runtime configuration resolution.
package facets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	configurationdomain "github.com/endge-lab/service-backend/internal/domain/configuration"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/history"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

var (
	iconTokenPattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	colorPattern     = regexp.MustCompile(`^#[0-9a-f]{6}$`)
)

type FacetCreateInput struct {
	Identity    string
	DisplayName string
	Icon        string
	Color       string
	Meta        map[string]any
}

type FacetPatchInput struct {
	Identity    *string
	DisplayName *string
	Icon        *string
	Color       *string
	Meta        *map[string]any
}

type FacetDocumentCreateInput struct {
	Identity      string
	DisplayName   string
	Description   *string
	Configuration map[string]any
	Meta          map[string]any
	Active        *bool
}

type FacetDocumentPatchInput struct {
	Identity      *string
	DisplayName   *string
	Description   *string
	Configuration *map[string]any
	Meta          *map[string]any
	Active        *bool
}

type UseCase struct {
	repository ports.FacetRepository
	tx         ports.TxManager
	history    *history.Recorder
}

func NewUseCase(repository ports.FacetRepository, tx ports.TxManager, recorder *history.Recorder) *UseCase {
	return &UseCase{repository: repository, tx: tx, history: recorder}
}

func (s *UseCase) List(ctx context.Context, includeDeleted bool) ([]entities.Document, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	return s.repository.ListFacets(ctx, scope.Workspace.ID, includeDeleted)
}

func (s *UseCase) Get(ctx context.Context, identity string, includeDeleted bool) (*entities.Document, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.GetFacet(ctx, scope.Workspace.ID, strings.TrimSpace(identity), includeDeleted)
	return value, shared.MapNotFound(err)
}

func (s *UseCase) Create(ctx context.Context, input FacetCreateInput) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	input = normalizeFacetCreate(input)
	if err = validateFacet(input.Identity, input.DisplayName, input.Icon, input.Color, input.Meta); err != nil {
		return nil, err
	}
	value := entities.Document{
		ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, Type: entities.CollectionFacets,
		Identity: input.Identity, DisplayName: input.DisplayName, ManagedBy: "user",
		Meta: mustJSON(input.Meta), Data: mustJSON(map[string]any{"icon": input.Icon, "color": input.Color, "position": -1}),
		Active: true, Revision: 1, CreatedBy: entities.Actor{ID: current.User.ID}, UpdatedBy: entities.Actor{ID: current.User.ID},
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		created, txErr := s.repository.InsertFacet(txctx, value)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *created, "create", nil); txErr != nil {
			return txErr
		}
		result = created
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) Patch(ctx context.Context, identity string, input FacetPatchInput, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	existing, err := s.repository.GetFacet(ctx, scope.Workspace.ID, strings.TrimSpace(identity), true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	next := *existing
	data := dataMap(existing.Data)
	if input.Identity != nil {
		next.Identity = strings.TrimSpace(*input.Identity)
	}
	if input.DisplayName != nil {
		next.DisplayName = strings.TrimSpace(*input.DisplayName)
	}
	if input.Icon != nil {
		data["icon"] = strings.TrimSpace(*input.Icon)
	}
	if input.Color != nil {
		data["color"] = strings.TrimSpace(*input.Color)
	}
	if input.Meta != nil {
		next.Meta = mustJSON(*input.Meta)
	}
	if next.Identity != existing.Identity {
		count, countErr := s.repository.CountActiveFacetDocuments(ctx, scope.Workspace.ID, existing.Identity)
		if countErr != nil {
			return nil, countErr
		}
		if count > 0 {
			return nil, domainerrors.WithDetails(domainerrors.Conflict("facet_identity_locked", "Facet identity cannot be changed while active documents exist"), map[string]any{"documentCount": count})
		}
	}
	if err = validateFacet(next.Identity, next.DisplayName, stringValue(data["icon"]), stringValue(data["color"]), rawObject(next.Meta)); err != nil {
		return nil, err
	}
	delete(data, "documentCount")
	next.Data = mustJSON(data)
	next.UpdatedBy = entities.Actor{ID: current.User.ID}
	if checksumDocument(*existing) == checksumDocument(next) {
		return existing, nil
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacet(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "update", nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) Delete(ctx context.Context, identity string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	existing, err := s.repository.GetFacet(ctx, scope.Workspace.ID, strings.TrimSpace(identity), true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if existing.DeletedAt != nil {
		return existing, nil
	}
	count, err := s.repository.CountActiveFacetDocuments(ctx, scope.Workspace.ID, existing.Identity)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, domainerrors.WithDetails(domainerrors.Conflict("facet_not_empty", "Facet cannot be deleted while active documents exist"), map[string]any{"documentCount": count})
	}
	now := time.Now().UTC()
	next := *existing
	next.DeletedAt, next.Active, next.UpdatedBy = &now, false, entities.Actor{ID: current.User.ID}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacet(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "delete", nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) Restore(ctx context.Context, identity string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	existing, err := s.repository.GetFacet(ctx, scope.Workspace.ID, strings.TrimSpace(identity), true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if existing.DeletedAt == nil {
		return existing, nil
	}
	next := *existing
	data := dataMap(existing.Data)
	data["position"] = -1
	delete(data, "documentCount")
	next.Data = mustJSON(data)
	next.DeletedAt, next.Active, next.UpdatedBy = nil, true, entities.Actor{ID: current.User.ID}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacet(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "restore", nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) Reorder(ctx context.Context, order []ports.FacetOrderItem) (result []entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if len(order) > 1000 {
		return nil, domainerrors.InvalidInput("facet_reorder_too_large", "Facet reorder cannot contain more than 1000 items")
	}
	seen := map[string]bool{}
	for index := range order {
		order[index].Identity = strings.TrimSpace(order[index].Identity)
		if validationErr := validateIdentity(order[index].Identity); validationErr != nil {
			return nil, validationErr
		}
		if order[index].ExpectedRevision <= 0 {
			return nil, shared.PreconditionRequired()
		}
		if seen[order[index].Identity] {
			return nil, domainerrors.InvalidInput("facet_reorder_duplicate", "Facet reorder contains duplicate identities")
		}
		seen[order[index].Identity] = true
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		changed, txErr := s.repository.ReorderFacets(txctx, scope.Workspace.ID, order, current.User.ID)
		if txErr != nil {
			return txErr
		}
		if len(changed) > 0 {
			txctx, txErr = s.history.BeginBatch(txctx, &scope.Workspace.ID, "update", current.User.ID)
			if txErr != nil {
				return txErr
			}
		}
		for _, value := range changed {
			if _, txErr = s.history.RecordDocument(txctx, value, "update", nil); txErr != nil {
				return txErr
			}
		}
		result, txErr = s.repository.ListFacets(txctx, scope.Workspace.ID, false)
		return txErr
	})
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "set mismatch") {
		return nil, domainerrors.Conflict("facet_reorder_set_mismatch", "Facet set changed; reload before reordering")
	}
	return result, shared.MapConflict(err)
}

func (s *UseCase) ListDocuments(ctx context.Context, facetIdentity string, filter ports.DocumentFilter) ([]entities.Document, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	if _, err = s.repository.GetFacet(ctx, scope.Workspace.ID, strings.TrimSpace(facetIdentity), true); err != nil {
		return nil, shared.MapNotFound(err)
	}
	return s.repository.ListFacetDocuments(ctx, scope.Workspace.ID, strings.TrimSpace(facetIdentity), filter)
}

func (s *UseCase) GetDocument(ctx context.Context, facetIdentity, identity string, includeDeleted bool) (*entities.Document, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.GetFacetDocument(ctx, scope.Workspace.ID, strings.TrimSpace(facetIdentity), strings.TrimSpace(identity), includeDeleted)
	return value, shared.MapNotFound(err)
}

func (s *UseCase) CreateDocument(ctx context.Context, facetIdentity string, input FacetDocumentCreateInput) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	facetIdentity = strings.TrimSpace(facetIdentity)
	if _, err = s.repository.GetFacet(ctx, scope.Workspace.ID, facetIdentity, false); err != nil {
		return nil, shared.MapNotFound(err)
	}
	input = normalizeFacetDocumentCreate(input)
	if err = validateFacetDocument(input.Identity, input.DisplayName, input.Configuration, input.Meta); err != nil {
		return nil, err
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	value := entities.Document{
		ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, Type: entities.CollectionFacetDocuments,
		Identity: input.Identity, DisplayName: input.DisplayName, Description: normalizeOptional(input.Description), ManagedBy: "user",
		Meta: mustJSON(input.Meta), Data: mustJSON(map[string]any{"facetIdentity": facetIdentity, "configuration": input.Configuration}),
		Active: active, Revision: 1, CreatedBy: entities.Actor{ID: current.User.ID}, UpdatedBy: entities.Actor{ID: current.User.ID},
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		created, txErr := s.repository.InsertFacetDocument(txctx, value)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *created, "create", nil); txErr != nil {
			return txErr
		}
		result = created
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) PatchDocument(ctx context.Context, facetIdentity, identity string, input FacetDocumentPatchInput, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	facetIdentity, identity = strings.TrimSpace(facetIdentity), strings.TrimSpace(identity)
	if _, err = s.repository.GetFacet(ctx, scope.Workspace.ID, facetIdentity, false); err != nil {
		return nil, shared.MapNotFound(err)
	}
	existing, err := s.repository.GetFacetDocument(ctx, scope.Workspace.ID, facetIdentity, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	next := *existing
	data := dataMap(existing.Data)
	if input.Identity != nil {
		next.Identity = strings.TrimSpace(*input.Identity)
	}
	if input.DisplayName != nil {
		next.DisplayName = strings.TrimSpace(*input.DisplayName)
	}
	if input.Description != nil {
		next.Description = normalizeOptional(input.Description)
	}
	if input.Configuration != nil {
		data["configuration"] = *input.Configuration
	}
	if input.Meta != nil {
		next.Meta = mustJSON(*input.Meta)
	}
	if input.Active != nil {
		next.Active = *input.Active
	}
	configuration, _ := data["configuration"].(map[string]any)
	if err = validateFacetDocument(next.Identity, next.DisplayName, configuration, rawObject(next.Meta)); err != nil {
		return nil, err
	}
	next.Data = mustJSON(data)
	next.UpdatedBy = entities.Actor{ID: current.User.ID}
	if checksumDocument(*existing) == checksumDocument(next) {
		return existing, nil
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacetDocument(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "update", nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) DeleteDocument(ctx context.Context, facetIdentity, identity string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	existing, err := s.repository.GetFacetDocument(ctx, scope.Workspace.ID, strings.TrimSpace(facetIdentity), strings.TrimSpace(identity), true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if existing.DeletedAt != nil {
		return existing, nil
	}
	now := time.Now().UTC()
	next := *existing
	next.DeletedAt, next.Active, next.UpdatedBy = &now, false, entities.Actor{ID: current.User.ID}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacetDocument(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "delete", nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) RestoreDocument(ctx context.Context, facetIdentity, identity string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	facetIdentity, identity = strings.TrimSpace(facetIdentity), strings.TrimSpace(identity)
	if _, err = s.repository.GetFacet(ctx, scope.Workspace.ID, facetIdentity, false); err != nil {
		return nil, domainerrors.Conflict("facet_deleted", "Facet document can be restored only inside an active facet")
	}
	existing, err := s.repository.GetFacetDocument(ctx, scope.Workspace.ID, facetIdentity, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if existing.DeletedAt == nil {
		return existing, nil
	}
	next := *existing
	next.DeletedAt, next.Active, next.UpdatedBy = nil, true, entities.Actor{ID: current.User.ID}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacetDocument(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "restore", nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) ListRevisions(ctx context.Context, identity string) ([]entities.Revision, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	values, err := s.repository.ListFacetRevisions(ctx, scope.Workspace.ID, strings.TrimSpace(identity))
	return values, shared.MapNotFound(err)
}

func (s *UseCase) GetRevision(ctx context.Context, identity, revisionID string) (*entities.Revision, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.GetFacetRevision(ctx, scope.Workspace.ID, strings.TrimSpace(identity), revisionID)
	return value, shared.MapNotFound(err)
}

func (s *UseCase) ListDocumentRevisions(ctx context.Context, facetIdentity, identity string) ([]entities.Revision, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	values, err := s.repository.ListFacetDocumentRevisions(ctx, scope.Workspace.ID, strings.TrimSpace(facetIdentity), strings.TrimSpace(identity))
	return values, shared.MapNotFound(err)
}

func (s *UseCase) GetDocumentRevision(ctx context.Context, facetIdentity, identity, revisionID string) (*entities.Revision, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.GetFacetDocumentRevision(ctx, scope.Workspace.ID, strings.TrimSpace(facetIdentity), strings.TrimSpace(identity), revisionID)
	return value, shared.MapNotFound(err)
}

func (s *UseCase) RestoreRevision(ctx context.Context, identity, revisionID string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	identity = strings.TrimSpace(identity)
	existing, err := s.repository.GetFacet(ctx, scope.Workspace.ID, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	revision, err := s.repository.GetFacetRevision(ctx, scope.Workspace.ID, identity, revisionID)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	var target entities.Document
	if err = json.Unmarshal(revision.Snapshot, &target); err != nil {
		return nil, domainerrors.Internal("revision_snapshot_invalid", "Facet revision snapshot is invalid")
	}
	target.ID, target.WorkspaceID, target.Type, target.Revision = existing.ID, existing.WorkspaceID, entities.CollectionFacets, existing.Revision
	target.UpdatedBy = entities.Actor{ID: current.User.ID}
	data := dataMap(target.Data)
	delete(data, "documentCount")
	target.Data = mustJSON(data)
	if err = validateFacet(target.Identity, target.DisplayName, stringValue(data["icon"]), stringValue(data["color"]), rawObject(target.Meta)); err != nil {
		return nil, err
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacet(txctx, target, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "restore", &revision.ID); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func (s *UseCase) RestoreDocumentRevision(ctx context.Context, facetIdentity, identity, revisionID string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	facetIdentity, identity = strings.TrimSpace(facetIdentity), strings.TrimSpace(identity)
	if _, err = s.repository.GetFacet(ctx, scope.Workspace.ID, facetIdentity, false); err != nil {
		return nil, domainerrors.Conflict("facet_deleted", "Facet document revision can be restored only inside an active facet")
	}
	existing, err := s.repository.GetFacetDocument(ctx, scope.Workspace.ID, facetIdentity, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	revision, err := s.repository.GetFacetDocumentRevision(ctx, scope.Workspace.ID, facetIdentity, identity, revisionID)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	var target entities.Document
	if err = json.Unmarshal(revision.Snapshot, &target); err != nil {
		return nil, domainerrors.Internal("revision_snapshot_invalid", "Facet document revision snapshot is invalid")
	}
	target.ID, target.WorkspaceID, target.Type, target.Revision = existing.ID, existing.WorkspaceID, entities.CollectionFacetDocuments, existing.Revision
	target.UpdatedBy = entities.Actor{ID: current.User.ID}
	data := dataMap(target.Data)
	data["facetIdentity"] = facetIdentity
	target.Data = mustJSON(data)
	configuration, _ := data["configuration"].(map[string]any)
	if err = validateFacetDocument(target.Identity, target.DisplayName, configuration, rawObject(target.Meta)); err != nil {
		return nil, err
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateFacetDocument(txctx, target, expected)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *updated, "restore", &revision.ID); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

func normalizeFacetCreate(input FacetCreateInput) FacetCreateInput {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Icon = strings.TrimSpace(input.Icon)
	input.Color = strings.TrimSpace(input.Color)
	if input.Meta == nil {
		input.Meta = map[string]any{}
	}
	return input
}

func normalizeFacetDocumentCreate(input FacetDocumentCreateInput) FacetDocumentCreateInput {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Configuration == nil {
		input.Configuration = map[string]any{"mode": "inherit", "patch": map[string]any{}}
	}
	if input.Meta == nil {
		input.Meta = map[string]any{}
	}
	return input
}

func validateFacet(identity, displayName, icon, color string, meta map[string]any) error {
	if err := validateIdentity(identity); err != nil {
		return err
	}
	if displayName == "" || len(displayName) > 255 {
		return domainerrors.InvalidInput("display_name_invalid", "displayName is required and must not exceed 255 characters")
	}
	if len(icon) > 80 || !iconTokenPattern.MatchString(icon) {
		return domainerrors.InvalidInput("facet_icon_invalid", "icon must be a PascalCase token")
	}
	if !colorPattern.MatchString(color) {
		return domainerrors.InvalidInput("facet_color_invalid", "color must be canonical lowercase #rrggbb")
	}
	if err := shared.ValidateSecrets(meta); err != nil {
		return err
	}
	return nil
}

func validateFacetDocument(identity, displayName string, configuration, meta map[string]any) error {
	if err := validateIdentity(identity); err != nil {
		return err
	}
	if displayName == "" || len(displayName) > 255 {
		return domainerrors.InvalidInput("display_name_invalid", "displayName is required and must not exceed 255 characters")
	}
	mode, _ := configuration["mode"].(string)
	switch mode {
	case "inherit":
		if _, ok := configuration["patch"].(map[string]any); !ok {
			return domainerrors.InvalidInput("facet_configuration_invalid", "inherit contribution requires an object patch")
		}
	case "replace":
		value, ok := configuration["value"].(map[string]any)
		if !ok {
			return domainerrors.InvalidInput("facet_configuration_invalid", "replace contribution requires an object value")
		}
		if err := configurationdomain.ValidateValuesShape(value); err != nil {
			return domainerrors.InvalidInput("facet_configuration_invalid", err.Error())
		}
	default:
		return domainerrors.InvalidInput("facet_configuration_invalid", "configuration mode must be inherit or replace")
	}
	if err := shared.ValidateSecrets(configuration); err != nil {
		return err
	}
	return shared.ValidateSecrets(meta)
}

func validateIdentity(value string) error {
	length := len(strings.TrimSpace(value))
	if length == 0 {
		return domainerrors.InvalidInput("identity_required", "identity is required")
	}
	if length > 160 {
		return domainerrors.InvalidInput("identity_too_long", "identity must not exceed 160 characters")
	}
	return nil
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}
	return &text
}

func dataMap(raw json.RawMessage) map[string]any {
	result := map[string]any{}
	_ = json.Unmarshal(raw, &result)
	return result
}

func rawObject(raw json.RawMessage) map[string]any { return dataMap(raw) }
func mustJSON(value any) json.RawMessage           { raw, _ := json.Marshal(value); return raw }
func stringValue(value any) string                 { text, _ := value.(string); return strings.TrimSpace(text) }

func checksumDocument(value entities.Document) string {
	raw := mustJSON(map[string]any{"identity": value.Identity, "displayName": value.DisplayName, "description": value.Description, "meta": json.RawMessage(value.Meta), "data": json.RawMessage(value.Data), "active": value.Active, "deletedAt": value.DeletedAt})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
