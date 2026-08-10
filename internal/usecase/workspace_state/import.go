package workspace_state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

const importPlanLifetime = 30 * time.Minute

// PlanImport валидирует полный snapshot и сохраняет краткоживущий план подтверждения.
func (s *Coordinator) PlanImport(ctx context.Context, bundle entities.PortableBundle) (*entities.ImportPlan, error) {
	current, scope, err := s.writeContext(ctx)
	if err != nil {
		return nil, err
	}
	if !canAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	normalization := normalizePortableBundle(&bundle)
	plan := &entities.ImportPlan{
		Valid: true, TargetWorkspace: scope.Workspace.Identity,
		TargetETag:           workspaceETag(scope.Workspace.Generation, scope.Workspace.HeadSequence),
		ExpectedHeadSequence: scope.Workspace.HeadSequence,
		MissingIntegrations:  []string{}, Unsupported: []string{}, ValidationErrors: []string{},
		Warnings: []string{"Existing workspace documents and history will be removed"},
	}
	if normalization.IgnoredIntegrations > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("Installed integrations from snapshot were ignored: %d; target workspace integrations will be preserved", normalization.IgnoredIntegrations))
	}
	if normalization.IgnoredDeletedDocuments > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d soft-deleted documents were ignored", normalization.IgnoredDeletedDocuments))
	}
	if normalization.IgnoredLegacyFolders > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d legacy system folders were ignored", normalization.IgnoredLegacyFolders))
	}
	if normalization.NormalizedFolderReferences > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d legacy folder references were normalized", normalization.NormalizedFolderReferences))
	}
	if bundle.Kind != "workspace-snapshot" {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "kind must be workspace-snapshot")
	}
	if bundle.SchemaVersion != SchemaVersion {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, fmt.Sprintf("unsupported schemaVersion %d", bundle.SchemaVersion))
	}
	if stringField(bundle.Workspace, "displayName") == "" {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "workspace.displayName is required")
	}
	incomingWorkspaceIdentity := stringField(bundle.Workspace, "identity")
	if identityErr := validateIdentity(incomingWorkspaceIdentity); identityErr != nil {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "workspace.identity: "+identityErr.Error())
	} else if incomingWorkspaceIdentity != scope.Workspace.Identity {
		plan.Warnings = append(plan.Warnings, "Incoming workspace identity will not replace target workspace identity")
	}
	if dataMode := stringField(bundle.Workspace, "dataMode"); dataMode != "development" && dataMode != "production" {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "workspace.dataMode must be development or production")
	}
	if secretErr := validateSecrets(bundle.Workspace["configuration"]); secretErr != nil {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "workspace.configuration: "+secretErr.Error())
	}
	seen := map[string]bool{}
	for _, kind := range Collections {
		if _, exists := bundle.Documents[kind]; !exists {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, "documents."+kind+" is required")
		}
	}
	for kind, items := range bundle.Documents {
		if slices.Contains(UnsupportedCollections, kind) || !slices.Contains(Collections, kind) {
			if len(items) == 0 {
				delete(bundle.Documents, kind)
				continue
			}
			plan.Valid = false
			plan.Unsupported = append(plan.Unsupported, kind)
			continue
		}
		for _, item := range items {
			if validationErr := validateDocument(kind, item); validationErr != nil {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, kind+":"+stringField(item, "identity")+": "+validationErr.Error())
				continue
			}
			key := kind + ":" + stringField(item, "identity")
			if seen[key] {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, "duplicate document "+key)
			}
			seen[key] = true
			plan.Incoming.Documents++
		}
	}
	integrationIdentities := map[string]bool{}
	for _, item := range bundle.InstalledIntegrations {
		identity := stringField(item, "identity")
		if identityErr := validateIdentity(identity); identityErr != nil {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, "installed integration identity: "+identityErr.Error())
			continue
		}
		if integrationIdentities[identity] {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, "duplicate installed integration "+identity)
			continue
		}
		integrationIdentities[identity] = true
		version := stringField(item, "version")
		if version == "" || len(version) > 160 {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, "installed integration "+identity+" version is required and must not exceed 160 characters")
			continue
		}
		if configuration, exists := item["configuration"]; exists && configuration != nil {
			if _, ok := configuration.(map[string]any); !ok {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, "installed integration "+identity+" configuration must be an object")
				continue
			}
			if secretErr := validateSecrets(configuration); secretErr != nil {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, "installed integration "+identity+" configuration: "+secretErr.Error())
				continue
			}
		}
		if _, integrationErr := s.repository.GetIntegration(ctx, identity, false); integrationErr != nil {
			if errors.Is(integrationErr, ports.ErrNotFound) {
				plan.Valid = false
				plan.MissingIntegrations = append(plan.MissingIntegrations, identity)
				continue
			}
			return nil, integrationErr
		}
		plan.Incoming.Integrations++
	}
	if relationErrors := validateSnapshotRelations(bundle); len(relationErrors) > 0 {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, relationErrors...)
	}
	slices.Sort(plan.Unsupported)
	slices.Sort(plan.MissingIntegrations)
	plan.WillRemove, err = s.repository.CountWorkspaceSnapshotState(ctx, scope.Workspace.ID)
	if err != nil {
		return nil, err
	}
	if !plan.Valid {
		return plan, nil
	}
	raw := mustJSON(bundle)
	plan.SnapshotChecksum = checksum(raw)
	expiresAt := time.Now().UTC().Add(importPlanLifetime)
	stored, err := s.repository.CreateSnapshotImportPlan(ctx, entities.SnapshotImportPlan{
		ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, SnapshotChecksum: plan.SnapshotChecksum, Snapshot: raw,
		ExpectedGeneration: scope.Workspace.Generation, ExpectedHeadSequence: scope.Workspace.HeadSequence,
		CreatedBy: entities.Actor{ID: current.User.ID}, ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}
	plan.ID = stored.ID
	plan.ExpiresAt = &stored.ExpiresAt
	return plan, nil
}

// Import полностью заменяет domain state данными ранее проверенного snapshot.
func (s *Coordinator) Import(ctx context.Context, planID, confirmation, ifMatch string) (result *entities.SnapshotImportResult, err error) {
	current, scope, err := s.writeContext(ctx)
	if err != nil {
		return nil, err
	}
	if !canAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	if strings.TrimSpace(confirmation) != scope.Workspace.Identity {
		return nil, domainerrors.InvalidInput("workspace_confirmation_invalid", "confirmation must match target workspace identity")
	}
	plan, err := s.repository.GetSnapshotImportPlan(ctx, scope.Workspace.ID, strings.TrimSpace(planID), current.User.ID)
	if err != nil {
		return nil, mapImportPlanNotFound(err)
	}
	if plan.AppliedAt != nil {
		return nil, domainerrors.Conflict("import_plan_applied", "Import plan has already been applied")
	}
	if time.Now().UTC().After(plan.ExpiresAt) {
		return nil, domainerrors.Conflict("import_plan_expired", "Import plan has expired")
	}
	expectedETag := workspaceETag(plan.ExpectedGeneration, plan.ExpectedHeadSequence)
	if normalizeETag(ifMatch) != normalizeETag(expectedETag) {
		return nil, domainerrors.New("precondition_failed", "Workspace changed after import plan", 412)
	}
	var bundle entities.PortableBundle
	if err = json.Unmarshal(plan.Snapshot, &bundle); err != nil {
		return nil, domainerrors.Internal("import_plan_corrupted", "Stored import plan is corrupted")
	}
	result = &entities.SnapshotImportResult{WorkspaceIdentity: scope.Workspace.Identity}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if txErr := s.repository.LockWorkspaceSnapshot(txctx, scope.Workspace.ID); txErr != nil {
			return txErr
		}
		live, txErr := s.repository.GetWorkspace(txctx, scope.Workspace.Identity)
		if txErr != nil {
			return txErr
		}
		if live.Generation != plan.ExpectedGeneration || live.HeadSequence != plan.ExpectedHeadSequence {
			return domainerrors.New("precondition_failed", "Workspace changed after import plan", 412)
		}
		backupRaw, txErr := s.repository.ExportWorkspace(txctx, live.ID, nil)
		if txErr != nil {
			return txErr
		}
		expiresAt := time.Now().UTC().AddDate(0, 0, s.backupRetentionDays)
		backup, txErr := s.repository.CreateSnapshotBackup(txctx, entities.SnapshotBackup{
			ID: uuid.NewString(), WorkspaceID: live.ID, Kind: "pre_import", Description: pointer("Automatic backup before snapshot import"),
			SchemaVersion: SchemaVersion, Checksum: checksum(backupRaw), Data: backupRaw,
			CreatedBy: entities.Actor{ID: current.User.ID}, ExpiresAt: &expiresAt,
		})
		if txErr != nil {
			return txErr
		}
		baselines, txErr := s.repository.DocumentRevisionBaselines(txctx, live.ID)
		if txErr != nil {
			return txErr
		}
		existingIntegrations, txErr := s.repository.ListWorkspaceIntegrations(txctx, live.ID)
		if txErr != nil {
			return txErr
		}
		reset, txErr := s.repository.ResetWorkspaceSnapshotState(txctx, live.ID, bundle.Workspace, current.User.ID)
		if txErr != nil {
			return txErr
		}
		if txErr = s.repository.ReplaceWorkspaceIntegrations(txctx, reset.ID, existingIntegrations, current.User.ID); txErr != nil {
			return txErr
		}
		batch, txErr := s.repository.CreateMutationBatch(txctx, &reset.ID, "bootstrap_import", current.User.ID)
		if txErr != nil {
			return txErr
		}
		txctx = context.WithValue(txctx, mutationBatchContextKey{}, batch)
		if txErr = s.recordWorkspaceRevision(txctx, *reset, "create"); txErr != nil {
			return txErr
		}
		createdRootTypes := map[string]bool{}
		for _, kind := range Collections {
			if kind == "folders" {
				continue
			}
			entityType := entities.FolderEntityType(kind)
			if createdRootTypes[entityType] {
				continue
			}
			createdRootTypes[entityType] = true
			rootIdentity := entities.RootFolderIdentity(kind)
			root := entities.Document{ID: uuid.NewString(), WorkspaceID: reset.ID, Type: "folders", Identity: rootIdentity, DisplayName: "Root " + entityType, ManagedBy: "system", Meta: json.RawMessage(`{}`), Data: mustJSON(map[string]any{"entityType": entityType, "isRoot": true}), Active: true, Revision: nextImportedRevision(baselines, "folders", rootIdentity), CreatedBy: entities.Actor{ID: current.User.ID}, UpdatedBy: entities.Actor{ID: current.User.ID}}
			created, insertErr := s.repository.InsertDocument(txctx, root, nil)
			if insertErr != nil {
				return insertErr
			}
			if _, insertErr = s.recordRevision(txctx, *created, "create", nil); insertErr != nil {
				return insertErr
			}
		}
		importScope := entities.WorkspaceAccess{Workspace: *reset, Role: scope.Role}
		for _, kind := range restoreOrder() {
			items, orderErr := orderPortableItems(kind, bundle.Documents[kind])
			if orderErr != nil {
				return fmt.Errorf("order imported %s: %w", kind, orderErr)
			}
			for _, item := range items {
				document := documentFromInput(kind, reset.ID, item, current.User.ID)
				document.Revision = nextImportedRevision(baselines, kind, document.Identity)
				folderID, resolveErr := s.resolveFolder(txctx, importScope, kind, item)
				if resolveErr != nil {
					return fmt.Errorf("resolve imported %s:%s folder: %w", kind, document.Identity, resolveErr)
				}
				created, insertErr := s.repository.InsertDocument(txctx, document, folderID)
				if insertErr != nil {
					return fmt.Errorf("insert imported %s:%s: %w", kind, document.Identity, insertErr)
				}
				if insertErr = s.replaceStructuredRelations(txctx, *created); insertErr != nil {
					return fmt.Errorf("relate imported %s:%s: %w", kind, document.Identity, insertErr)
				}
				if _, insertErr = s.recordRevision(txctx, *created, "create", nil); insertErr != nil {
					return fmt.Errorf("record imported %s:%s revision: %w", kind, document.Identity, insertErr)
				}
				result.Imported.Documents++
			}
		}
		pending, txErr := s.repository.PendingRevisions(txctx, reset.ID, 0)
		if txErr != nil {
			return txErr
		}
		head := int64(0)
		for _, revision := range pending {
			if revision.WorkspaceSequence != nil && *revision.WorkspaceSequence > head {
				head = *revision.WorkspaceSequence
			}
		}
		commit, txErr := s.repository.CreateCommit(txctx, entities.Commit{ID: uuid.NewString(), WorkspaceID: reset.ID, BaseSequence: 0, HeadSequence: head, Message: "Snapshot import baseline", RevisionPolicy: "preserve", Operation: "bootstrap_import", CreatedBy: entities.Actor{ID: current.User.ID}}, commitChanges(pending))
		if txErr != nil {
			return txErr
		}
		ids := make([]string, 0, len(pending))
		for _, revision := range pending {
			ids = append(ids, revision.ID)
		}
		if txErr = s.repository.AttachRevisionsToCommit(txctx, commit.ID, ids); txErr != nil {
			return txErr
		}
		if txErr = s.repository.MarkSnapshotImportPlanApplied(txctx, plan.ID); txErr != nil {
			return txErr
		}
		result.Backup = *backup
		result.InitialCommitID = commit.ID
		return nil
	})
	return result, mapConflict(err)
}

func validateSnapshotRelations(bundle entities.PortableBundle) []string {
	available := map[string]map[string]bool{}
	for kind, items := range bundle.Documents {
		available[kind] = map[string]bool{}
		for _, item := range items {
			available[kind][stringField(item, "identity")] = true
		}
	}
	result := []string{}
	for kind, items := range bundle.Documents {
		for _, item := range items {
			identity := stringField(item, "identity")
			if kind == "folders" {
				parent := stringField(item, "parentIdentity")
				entityType := stringField(item, "entityType")
				if parent != "" && parent != entities.RootFolderIdentity(entityType) && !available["folders"][parent] {
					result = append(result, kind+":"+identity+": parentIdentity target is missing")
				}
			} else if folder := stringField(item, "folderIdentity"); folder != "" && folder != entities.RootFolderIdentity(kind) && !available["folders"][folder] {
				result = append(result, kind+":"+identity+": folderIdentity target is missing")
			}
			if kind == "updates" && !available["stores"][stringField(item, "storeIdentity")] {
				result = append(result, kind+":"+identity+": storeIdentity target is missing")
			}
			if kind == "projects" {
				for _, environment := range relationIdentityList(item["allowedEnvironments"]) {
					if !available["environments"][environment] {
						result = append(result, kind+":"+identity+": allowed environment "+environment+" is missing")
					}
				}
			}
			if kind == "vocabs" {
				target := stringField(item, "authProfileIdentity")
				if target != "" && !available["auth-profiles"][target] {
					result = append(result, kind+":"+identity+": authProfileIdentity target is missing")
				}
			}
		}
	}
	return result
}

func workspaceETag(generation string, head int64) string {
	return fmt.Sprintf(`"%s:%d"`, generation, head)
}

func normalizeETag(value string) string {
	return strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "W/")), `"`)
}

func nextImportedRevision(baselines map[string]int, kind, identity string) int {
	return baselines[kind+":"+identity] + 1
}

func pointer(value string) *string { return &value }

func mapImportPlanNotFound(err error) error {
	if errors.Is(err, ports.ErrNotFound) {
		return domainerrors.NotFound("import_plan_not_found", "Import plan was not found")
	}
	return err
}
