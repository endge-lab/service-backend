package workspace_state

import (
	"context"
	"encoding/json"
	"time"

	configurationdomain "github.com/endge-lab/service-backend/internal/domain/configuration"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

// planExactRestore строит точный план восстановления состояния.
func (s *Coordinator) planExactRestore(ctx context.Context, scope entities.WorkspaceAccess, bundle entities.PortableBundle) (*entities.ImportPlan, error) {
	plan := &entities.ImportPlan{Valid: true, ExpectedHeadSequence: scope.Workspace.HeadSequence}
	for _, kind := range restoreOrder() {
		targets := map[string]bool{}
		for _, item := range bundle.Documents[kind] {
			targets[stringField(item, "identity")] = true
		}
		current, err := s.repository.ListDocuments(ctx, scope.Workspace.ID, kind, ports.DocumentFilter{IncludeDeleted: true, Limit: 100000})
		if err != nil {
			return nil, err
		}
		for _, doc := range current {
			if targets[doc.Identity] {
				plan.Updates++
			} else if doc.DeletedAt == nil {
				plan.Updates++
			}
		}
		for identity := range targets {
			found := false
			for _, doc := range current {
				if doc.Identity == identity {
					found = true
					break
				}
			}
			if !found {
				plan.Creates++
			}
		}
	}
	return plan, nil
}

// restoreBundle собирает переносимый пакет из снимка для восстановления.
func (s *Coordinator) restoreBundle(ctx context.Context, bundle entities.PortableBundle, expected int64, operation, message string) (result *entities.Commit, err error) {
	removeLegacySSEFromPortableBundle(&bundle)
	current, scope, err := s.writeContext(ctx)
	if err != nil {
		return nil, err
	}
	if !canAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if e := s.repository.LockWorkspaceSnapshot(txctx, scope.Workspace.ID); e != nil {
			return e
		}
		live, e := s.repository.GetWorkspace(txctx, scope.Workspace.Identity)
		if e != nil {
			return e
		}
		if expected != live.HeadSequence {
			return domainerrors.Conflict("head_sequence_conflict", "Workspace changed after preview")
		}
		latest, e := s.repository.LatestCommit(txctx, live.ID)
		if e != nil {
			return e
		}
		scope.Workspace = *live

		batch, e := s.repository.CreateMutationBatch(txctx, &scope.Workspace.ID, operation, current.User.ID)
		if e != nil {
			return e
		}
		txctx = context.WithValue(txctx, mutationBatchContextKey{}, batch)
		previousIntegrations, e := s.repository.ListWorkspaceIntegrations(txctx, scope.Workspace.ID)
		if e != nil {
			return e
		}
		if e := s.repository.ReplaceWorkspaceIntegrations(txctx, scope.Workspace.ID, bundle.InstalledIntegrations, current.User.ID); e != nil {
			return e
		}
		integrationsChanged := checksum(mustJSON(previousIntegrations)) != checksum(mustJSON(bundle.InstalledIntegrations))
		workspaceRevisionRecorded := false
		workspacePatch := map[string]any{}
		for _, key := range []string{"identity", "displayName", "description", "dataMode", "configuration", "meta"} {
			if value, ok := bundle.Workspace[key]; ok {
				workspacePatch[key] = value
			}
		}
		if configuration, exists := workspacePatch["configuration"]; exists {
			workspacePatch["configuration"] = configurationdomain.EnsureSFCEditingDefaults(configuration)
		}
		if len(workspacePatch) > 0 {
			live, e := s.repository.GetWorkspace(txctx, scope.Workspace.Identity)
			if e != nil {
				return e
			}
			currentState := map[string]any{"identity": live.Identity, "displayName": live.DisplayName, "description": live.Description, "dataMode": live.DataMode, "configuration": json.RawMessage(live.Configuration), "meta": json.RawMessage(live.Meta)}
			if checksum(mustJSON(currentState)) != checksum(mustJSON(workspacePatch)) {
				updated, e := s.repository.UpdateWorkspace(txctx, live.Identity, workspacePatch, live.Revision, current.User.ID)
				if e != nil {
					return e
				}
				if _, e = s.recordWorkspaceRevision(txctx, *updated, "restore"); e != nil {
					return e
				}
				workspaceRevisionRecorded = true
			}
		}
		if integrationsChanged && !workspaceRevisionRecorded {
			live, e := s.repository.GetWorkspace(txctx, scope.Workspace.Identity)
			if e != nil {
				return e
			}
			updated, e := s.repository.UpdateWorkspace(txctx, live.Identity, map[string]any{}, live.Revision, current.User.ID)
			if e != nil {
				return e
			}
			if _, e = s.recordWorkspaceRevision(txctx, *updated, "restore"); e != nil {
				return e
			}
		}
		for _, kind := range restoreOrder() {
			orderedTargets, orderErr := orderPortableItems(kind, bundle.Documents[kind])
			if orderErr != nil {
				return orderErr
			}
			targets := map[string]map[string]any{}
			for _, item := range orderedTargets {
				if e := validateProjectContract(kind, item); e != nil {
					return e
				}
				targets[stringField(item, "identity")] = item
			}
			existing, er := s.repository.ListDocuments(txctx, scope.Workspace.ID, kind, ports.DocumentFilter{IncludeDeleted: true, Limit: 100000})
			if er != nil {
				return er
			}
			seen := map[string]bool{}
			for _, doc := range existing {
				item, ok := targets[doc.Identity]
				if !ok {
					if kind == "folders" && doc.ManagedBy == "system" {
						continue
					}
					if doc.DeletedAt == nil {
						now := time.Now().UTC()
						next := doc
						next.DeletedAt = &now
						next.Active = false
						next.UpdatedBy = entities.Actor{ID: current.User.ID}
						folderID, e := s.resolveDocumentFolder(txctx, scope, next)
						if e != nil {
							return e
						}
						updated, e := s.repository.UpdateDocument(txctx, next, doc.Revision, folderID)
						if e != nil {
							return e
						}
						if _, e = s.recordRevision(txctx, *updated, "delete", nil); e != nil {
							return e
						}
					}
					continue
				}
				seen[doc.Identity] = true
				next := replaceDocumentFromInput(doc, item, current.User.ID)
				folderID, e := s.resolveDocumentFolder(txctx, scope, next)
				if e != nil {
					return e
				}
				if checksumContent(doc) != checksumContent(next) {
					updated, e := s.repository.UpdateDocument(txctx, next, doc.Revision, folderID)
					if e != nil {
						return e
					}
					if e = s.replaceStructuredRelations(txctx, *updated); e != nil {
						return e
					}
					if _, e = s.recordRevision(txctx, *updated, "restore", nil); e != nil {
						return e
					}
				}
			}
			for _, item := range orderedTargets {
				identity := stringField(item, "identity")
				if seen[identity] {
					continue
				}
				doc := documentFromInput(kind, scope.Workspace.ID, item, current.User.ID)
				folderID, e := s.resolveFolder(txctx, scope, kind, item)
				if e != nil {
					return e
				}
				created, e := s.repository.InsertDocument(txctx, doc, folderID)
				if e != nil {
					return e
				}
				if e = s.replaceStructuredRelations(txctx, *created); e != nil {
					return e
				}
				if _, e = s.recordRevision(txctx, *created, "restore", nil); e != nil {
					return e
				}
			}
		}
		latest, e = s.repository.LatestCommit(txctx, scope.Workspace.ID)
		if e != nil {
			return e
		}
		pending, e := s.repository.PendingRevisions(txctx, scope.Workspace.ID, latest.HeadSequence)
		if e != nil {
			return e
		}
		if len(pending) == 0 {
			return domainerrors.Conflict("nothing_to_restore", "Workspace already matches target state")
		}
		head := scope.Workspace.HeadSequence
		for _, revision := range pending {
			if revision.WorkspaceSequence != nil && *revision.WorkspaceSequence > head {
				head = *revision.WorkspaceSequence
			}
		}
		value := entities.Commit{ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, ParentCommitID: &latest.ID, BaseSequence: latest.HeadSequence, HeadSequence: head, Message: message, RevisionPolicy: "preserve", Operation: operation, CreatedBy: entities.Actor{ID: current.User.ID}}
		result, e = s.repository.CreateCommit(txctx, value, commitChanges(pending))
		if e != nil {
			return e
		}
		ids := []string{}
		for _, revision := range pending {
			ids = append(ids, revision.ID)
		}
		return s.repository.AttachRevisionsToCommit(txctx, result.ID, ids)
	})
	return result, mapConflict(err)
}

// restoreOrder задаёт порядок восстановления коллекций.
func restoreOrder() []string {
	return []string{"folders", "environments", "navigations", "auth-profiles", "stores", "projects", "vocabs", "updates", "tenants", "types", "configurations", "queries", "data-views", "compositions", "streams", "mocks", "components", "actions", "filters", "converters", "computations", "i18n-bundles", "styles"}
}
