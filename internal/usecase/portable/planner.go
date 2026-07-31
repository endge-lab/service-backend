package portable

import (
	"context"
	"encoding/json"
	"strings"

	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

type Planner struct {
	registry *Registry
	tx       ports.TxManager
}

func NewPlanner(registry *Registry, tx ports.TxManager) *Planner {
	return &Planner{registry: registry, tx: tx}
}

// Import создаёт import plan и выполняет его через entity adapters.
//
// Параметры:
//
//	ctx       - контекст выполнения; при atomic import получает transaction context;
//	documents - переносимые documents без foreign UUID;
//	options   - conflict policy, explicit rename map и atomic mode.
//
// Что делает функция:
//
//	Нормализует documents, проверяет conflicts и unresolved relations, создаёт
//	или находит base entities, строит entityType+identity → target UUID map и
//	только затем передаёт resolved relations entity-specific adapters. Canonical
//	source/JSON не переписывается.
//
// Возвращаемые значения:
//
//	ImportResult - created, updated, skipped и document-level errors;
//	error        - validation/storage error; при atomic failure также вызывает rollback.
func (p *Planner) Import(ctx context.Context, documents []PortableDocument, options ImportOptions) (result ImportResult, err error) {
	prepared, preparationErrors, err := p.prepare(ctx, documents, options)
	if err != nil {
		return result, err
	}
	if len(preparationErrors) > 0 {
		result.Errors = preparationErrors
		if options.Atomic {
			return result, apperrors.InvalidInput("invalid_relation", "atomic import contains invalid documents")
		}
	}

	execute := func(executionCtx context.Context) error {
		executed, executeErr := p.execute(executionCtx, prepared)
		result.Created = executed.Created
		result.Updated = executed.Updated
		result.Skipped = executed.Skipped
		result.Errors = append(result.Errors, executed.Errors...)
		err = executeErr
		return err
	}
	if options.Atomic {
		if p.tx == nil {
			return result, apperrors.Internal("internal_error", "portable atomic import transaction manager is unavailable")
		}
		if err = p.tx.WithinTransaction(ctx, execute); err != nil {
			return result, err
		}
		return result, nil
	}
	if err = execute(ctx); err != nil {
		return result, err
	}
	return result, nil
}

type preparedDocument struct {
	document       PortableDocument
	sourceKey      EntityKey
	targetKey      EntityKey
	adapter        EntityPortableAdapter
	existingID     uuid.UUID
	exists         bool
	resolvedDirect []ResolvedRelation
}

func (p *Planner) prepare(ctx context.Context, documents []PortableDocument, options ImportOptions) ([]preparedDocument, []ImportError, error) {
	if p == nil || p.registry == nil {
		return nil, nil, apperrors.Internal("internal_error", "portable adapter registry is unavailable")
	}
	policy, err := normalizeConflictPolicy(options.ConflictPolicy)
	if err != nil {
		return nil, nil, err
	}
	prepared := make([]preparedDocument, 0, len(documents))
	keys := make(map[EntityKey]struct{}, len(documents))
	for _, document := range documents {
		document, err = normalizeDocument(document)
		if err != nil {
			return nil, nil, err
		}
		sourceKey := EntityKey{EntityType: document.EntityType, Identity: document.Identity}
		if _, duplicate := keys[sourceKey]; duplicate {
			return nil, nil, apperrors.InvalidInput("validation_error", "portable documents must have unique entity type and identity")
		}
		keys[sourceKey] = struct{}{}
		adapter, ok := p.registry.Adapter(document.EntityType)
		if !ok {
			return nil, nil, apperrors.NotFound("portable_adapter_not_found", "portable adapter is not registered")
		}
		prepared = append(prepared, preparedDocument{document: document, sourceKey: sourceKey, targetKey: sourceKey, adapter: adapter})
	}

	errorsByDocument := make([]ImportError, 0)
	invalid := make(map[EntityKey]struct{})
	for index := range prepared {
		item := &prepared[index]
		existingID, found, findErr := item.adapter.FindByIdentity(ctx, item.document.Identity)
		if findErr != nil {
			return nil, nil, findErr
		}
		item.existingID, item.exists = existingID, found
		if !found {
			continue
		}
		switch policy {
		case ConflictPolicyFail:
			errorsByDocument = append(errorsByDocument, importError(item.sourceKey, "import_identity_conflict", "target identity already exists"))
			invalid[item.sourceKey] = struct{}{}
		case ConflictPolicyOverwrite:
		case ConflictPolicyRename:
			renamedIdentity, exists := options.RenameIdentities[item.sourceKey]
			renamedIdentity = strings.TrimSpace(renamedIdentity)
			if !exists || renamedIdentity == "" || renamedIdentity == item.document.Identity {
				errorsByDocument = append(errorsByDocument, importError(item.sourceKey, "import_identity_conflict", "rename policy requires an explicit new identity"))
				invalid[item.sourceKey] = struct{}{}
				continue
			}
			if _, collision, collisionErr := item.adapter.FindByIdentity(ctx, renamedIdentity); collisionErr != nil {
				return nil, nil, collisionErr
			} else if collision {
				errorsByDocument = append(errorsByDocument, importError(item.sourceKey, "import_identity_conflict", "renamed target identity already exists"))
				invalid[item.sourceKey] = struct{}{}
				continue
			}
			item.document.Identity = renamedIdentity
			item.targetKey.Identity = renamedIdentity
			item.exists = false
			item.existingID = uuid.Nil
		}
	}
	for index := range prepared {
		item := &prepared[index]
		if _, failed := invalid[item.sourceKey]; failed {
			continue
		}
		for _, relation := range item.document.Relations {
			relationKey := EntityKey{EntityType: relation.EntityType, Identity: relation.Identity}
			if _, imported := keys[relationKey]; imported {
				if _, unavailable := invalid[relationKey]; unavailable {
					errorsByDocument = append(errorsByDocument, importError(item.sourceKey, "unresolved_dependency", "referenced import document is invalid"))
					invalid[item.sourceKey] = struct{}{}
				}
				continue
			}
			adapter, ok := p.registry.Adapter(relation.EntityType)
			if !ok {
				errorsByDocument = append(errorsByDocument, importError(item.sourceKey, "unresolved_dependency", "relation adapter is not registered"))
				invalid[item.sourceKey] = struct{}{}
				continue
			}
			targetID, found, findErr := adapter.FindByIdentity(ctx, relation.Identity)
			if findErr != nil {
				return nil, nil, findErr
			}
			if !found {
				errorsByDocument = append(errorsByDocument, importError(item.sourceKey, "unresolved_dependency", "relation identity is not available in target workspace"))
				invalid[item.sourceKey] = struct{}{}
				continue
			}
			item.resolvedDirect = append(item.resolvedDirect, ResolvedRelation{Path: relation.Path, EntityType: relation.EntityType, Identity: relation.Identity, TargetID: targetID})
		}
	}
	valid := make([]preparedDocument, 0, len(prepared))
	for _, item := range prepared {
		if _, failed := invalid[item.sourceKey]; !failed {
			valid = append(valid, item)
		}
	}
	return valid, errorsByDocument, nil
}

func (p *Planner) execute(ctx context.Context, prepared []preparedDocument) (ImportResult, error) {
	result := ImportResult{Created: []EntityKey{}, Updated: []EntityKey{}, Skipped: []EntityKey{}, Errors: []ImportError{}}
	targetIDs := make(map[EntityKey]uuid.UUID, len(prepared))
	for _, item := range prepared {
		if item.exists {
			if err := item.adapter.OverwriteBase(ctx, item.existingID, item.document); err != nil {
				return result, err
			}
			targetIDs[item.sourceKey] = item.existingID
			result.Updated = append(result.Updated, item.targetKey)
			continue
		}
		id, err := item.adapter.CreateBase(ctx, item.document)
		if err != nil {
			return result, err
		}
		targetIDs[item.sourceKey] = id
		result.Created = append(result.Created, item.targetKey)
	}
	for _, item := range prepared {
		resolved := append([]ResolvedRelation(nil), item.resolvedDirect...)
		for _, relation := range item.document.Relations {
			relationKey := EntityKey{EntityType: relation.EntityType, Identity: relation.Identity}
			if targetID, imported := targetIDs[relationKey]; imported {
				resolved = append(resolved, ResolvedRelation{Path: relation.Path, EntityType: relation.EntityType, Identity: relation.Identity, TargetID: targetID})
			}
		}
		if err := item.adapter.ApplyRelations(ctx, targetIDs[item.sourceKey], resolved); err != nil {
			return result, err
		}
	}
	return result, nil
}

func normalizeConflictPolicy(value ConflictPolicy) (ConflictPolicy, error) {
	if value == "" {
		return ConflictPolicyFail, nil
	}
	switch value {
	case ConflictPolicyFail, ConflictPolicyOverwrite, ConflictPolicyRename:
		return value, nil
	default:
		return "", apperrors.InvalidInput("validation_error", "portable conflict policy is invalid")
	}
}

func normalizeDocument(document PortableDocument) (PortableDocument, error) {
	document.EntityType = strings.TrimSpace(document.EntityType)
	document.Identity = strings.TrimSpace(document.Identity)
	if document.EntityType == "" || document.Identity == "" {
		return PortableDocument{}, apperrors.InvalidInput("validation_error", "portable entity type and identity are required")
	}
	if len(document.Canonical) == 0 || !json.Valid(document.Canonical) {
		return PortableDocument{}, apperrors.InvalidInput("validation_error", "portable canonical document must be valid JSON")
	}
	seenPaths := make(map[string]struct{}, len(document.Relations))
	for index := range document.Relations {
		relation := &document.Relations[index]
		relation.Path = strings.TrimSpace(relation.Path)
		relation.EntityType = strings.TrimSpace(relation.EntityType)
		relation.Identity = strings.TrimSpace(relation.Identity)
		if relation.Path == "" || relation.EntityType == "" || relation.Identity == "" {
			return PortableDocument{}, apperrors.InvalidInput("validation_error", "portable relation path, entity type and identity are required")
		}
		if _, duplicate := seenPaths[relation.Path]; duplicate {
			return PortableDocument{}, apperrors.InvalidInput("validation_error", "portable relation paths must be unique")
		}
		seenPaths[relation.Path] = struct{}{}
	}
	return document, nil
}

func importError(document EntityKey, code, message string) ImportError {
	return ImportError{Document: document, Code: code, Message: message}
}
