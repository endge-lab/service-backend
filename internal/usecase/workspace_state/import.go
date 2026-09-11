package workspace_state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	configurationdomain "github.com/endge-lab/service-backend/internal/domain/configuration"
	"github.com/endge-lab/service-backend/internal/domain/domainversion"
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
	if schemaErr := validateWorkspaceSchemaVersion(bundle.SchemaVersion, s.schemaVersion); schemaErr != nil {
		return nil, schemaErr
	}
	providedDomainVersion := bundle.DomainVersion
	computedSourceDomainVersion, normalization, versionErr := normalizePortableBundleForImport(&bundle)
	if versionErr != nil {
		return nil, domainerrors.InvalidInput("domain_version_invalid", "Domain version could not be computed")
	}
	plan := &entities.ImportPlan{
		Valid: true, TargetWorkspace: scope.Workspace.Identity,
		TargetETag:           workspaceETag(scope.Workspace.Generation, scope.Workspace.HeadSequence),
		ExpectedHeadSequence: scope.Workspace.HeadSequence,
		MissingIntegrations:  []string{}, Unsupported: []string{}, ValidationErrors: []string{},
		Warnings: []string{},
	}
	if normalization.IgnoredIntegrations > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("Installed integrations from snapshot were ignored: %d; target workspace integrations will be preserved", normalization.IgnoredIntegrations))
	}
	if normalization.IgnoredLegacyFolders > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d legacy system folders were ignored", normalization.IgnoredLegacyFolders))
	}
	if normalization.NormalizedFolderReferences > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d legacy folder references were normalized", normalization.NormalizedFolderReferences))
	}
	if normalization.MigratedLegacyActions > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d legacy Action documents with empty Source were migrated to default Action Source", normalization.MigratedLegacyActions))
	}
	if normalization.MigratedLegacyVocabs > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d legacy external Payload Vocab documents were migrated to Source", normalization.MigratedLegacyVocabs))
	}
	if bundle.Kind != "workspace-snapshot" {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "kind must be workspace-snapshot")
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
	if documentStructure := stringField(bundle.Workspace, "documentStructure"); !entities.IsWorkspaceDocumentStructure(documentStructure) {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "workspace.documentStructure must be frontend or custom")
	}
	if secretErr := validateSecrets(bundle.Workspace["configuration"]); secretErr != nil {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "workspace.configuration: "+secretErr.Error())
	}
	if configurationErr := configurationdomain.ValidateValuesShape(bundle.Workspace["configuration"]); configurationErr != nil {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "workspace.configuration: "+configurationErr.Error())
	}
	seen := map[string]bool{}
	for _, kind := range Collections {
		if _, exists := bundle.Documents[kind]; !exists {
			if kind == entities.CollectionConfigurations {
				continue
			}
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
			if validationErr := validatePortableDocument(kind, item); validationErr != nil {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, kind+":"+stringField(item, "identity")+": "+validationErr.Error())
				continue
			}
			key := kind + ":" + portableDocumentKey(kind, item)
			if seen[key] {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, "duplicate document "+key)
			}
			seen[key] = true
			plan.Incoming.Documents++
		}
	}
	if positionErr := validateFacetPositions(bundle.Documents[entities.CollectionFacets]); positionErr != nil {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "documents.facets: "+positionErr.Error())
	}
	sfcEditingDefaultsAdded, versionErr := finalizeImportDomainVersion(&bundle, providedDomainVersion)
	if versionErr != nil {
		return nil, domainerrors.InvalidInput("domain_version_invalid", "Domain version could not be computed")
	}
	if providedDomainVersion != "" && providedDomainVersion != computedSourceDomainVersion {
		plan.Valid = false
		plan.ValidationErrors = append(plan.ValidationErrors, "domainVersion does not match portable domain content")
	}
	if sfcEditingDefaultsAdded {
		plan.Warnings = append(plan.Warnings, "Workspace SFC editing defaults were added for compatibility")
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
	if plan.Valid {
		latest, latestErr := s.repository.LatestCommit(ctx, scope.Workspace.ID)
		if latestErr != nil {
			return nil, latestErr
		}
		if latest.HeadSequence != scope.Workspace.HeadSequence {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, "workspace has uncommitted revisions; create a commit before import")
		} else if err = s.planSnapshotChanges(ctx, scope.Workspace.ID, bundle, plan); err != nil {
			return nil, err
		}
		if plan.Deletes > 0 {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("Documents absent from snapshot will be marked as deleted: %d", plan.Deletes))
		}
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

func validateWorkspaceSchemaVersion(exported int, supported int) error {
	if exported == supported {
		return nil
	}
	return domainerrors.WithDetails(
		domainerrors.InvalidInput(
			"workspace_schema_unsupported",
			fmt.Sprintf("Версия схемы экспорта %d не поддерживается backend; обновите исходный backend и сформируйте новый экспорт", exported),
		),
		map[string]any{
			"exportedSchemaVersion":  exported,
			"supportedSchemaVersion": supported,
		},
	)
}

// Import приводит domain state к ранее проверенному snapshot через обратимые revisions.
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
	domainversion.CanonicalizeInPlace(&bundle)
	if _, versionErr := finalizeImportDomainVersion(&bundle, bundle.DomainVersion); versionErr != nil {
		return nil, domainerrors.Internal("import_plan_corrupted", "Stored import plan domain version is invalid")
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
		latest, txErr := s.repository.LatestCommit(txctx, live.ID)
		if txErr != nil {
			return txErr
		}
		if latest.HeadSequence != live.HeadSequence {
			return domainerrors.Conflict("import_requires_clean_commit", "Workspace has uncommitted revisions")
		}
		batch, txErr := s.repository.CreateMutationBatch(txctx, &live.ID, "import", current.User.ID)
		if txErr != nil {
			return txErr
		}
		txctx = context.WithValue(txctx, mutationBatchContextKey{}, batch)
		revisions := []entities.Revision{}
		workspacePatch := map[string]any{}
		for _, key := range []string{"displayName", "description", "dataMode", "documentStructure", "configuration", "meta", "active"} {
			if value, exists := bundle.Workspace[key]; exists {
				workspacePatch[key] = value
			}
		}
		updatedWorkspace, txErr := s.repository.UpdateWorkspace(txctx, live.Identity, workspacePatch, live.Revision, current.User.ID)
		if txErr != nil {
			return txErr
		}
		workspaceRevision, txErr := s.recordWorkspaceRevision(txctx, *updatedWorkspace, "update")
		if txErr != nil {
			return txErr
		}
		revisions = append(revisions, *workspaceRevision)

		existing, txErr := s.loadSnapshotDocuments(txctx, live.ID)
		if txErr != nil {
			return txErr
		}
		incoming := snapshotDocumentIdentities(bundle)
		importScope := entities.WorkspaceAccess{Workspace: *updatedWorkspace, Role: scope.Role}
		for _, kind := range restoreOrder() {
			items, orderErr := orderPortableItems(kind, bundle.Documents[kind])
			if orderErr != nil {
				return fmt.Errorf("order imported %s: %w", kind, orderErr)
			}
			for _, item := range items {
				identity := stringField(item, "identity")
				key := portableDocumentKey(kind, item)
				folderID, resolveErr := s.resolveFolder(txctx, importScope, kind, item)
				if resolveErr != nil {
					return fmt.Errorf("resolve imported %s:%s folder: %w", kind, identity, resolveErr)
				}
				document := documentFromInput(kind, live.ID, item, current.User.ID)
				operation := "create"
				var stored *entities.Document
				if previous, exists := existing[kind][key]; exists {
					document = replaceDocumentFromInput(previous, item, current.User.ID)
					if kind == entities.CollectionFacets && portableDocumentDeleted(item) && previous.DeletedAt == nil {
						// Active nested documents are reconciled later in restore order;
						// keep the parent active until that has completed.
						document.DeletedAt = nil
						document.Active = true
					}
					if kind == entities.CollectionFacets {
						position := facetPosition(previous)
						if previous.DeletedAt != nil {
							position = -1
						}
						document = replaceDocumentDataField(document, "position", position)
					}
					switch {
					case previous.DeletedAt == nil && document.DeletedAt != nil:
						operation = "delete"
					case previous.DeletedAt != nil && document.DeletedAt == nil:
						operation = "restore"
					default:
						operation = "update"
					}
					stored, txErr = s.repository.UpdateDocument(txctx, document, previous.Revision, folderID)
				} else {
					if kind == entities.CollectionFacets && document.DeletedAt == nil {
						document = replaceDocumentDataField(document, "position", -1)
					}
					stored, txErr = s.repository.InsertDocument(txctx, document, folderID)
				}
				if txErr != nil {
					return fmt.Errorf("apply imported %s:%s: %w", kind, identity, txErr)
				}
				if txErr = s.replaceStructuredRelations(txctx, *stored); txErr != nil {
					return fmt.Errorf("relate imported %s:%s: %w", kind, identity, txErr)
				}
				revision, revisionErr := s.recordRevision(txctx, *stored, operation, nil)
				if revisionErr != nil {
					return fmt.Errorf("record imported %s:%s revision: %w", kind, identity, revisionErr)
				}
				revisions = append(revisions, *revision)
				switch operation {
				case "create":
					result.Creates++
				case "delete":
					result.Deletes++
				case "restore":
					result.Restores++
				default:
					result.Updates++
				}
				result.Imported.Documents++
			}
		}
		existing, txErr = s.loadSnapshotDocuments(txctx, live.ID)
		if txErr != nil {
			return txErr
		}
		for _, document := range documentsMissingFromSnapshot(existing, incoming) {
			now := time.Now().UTC()
			next := document
			next.Active = false
			next.DeletedAt = &now
			next.UpdatedBy = entities.Actor{ID: current.User.ID}
			folderID, resolveErr := s.resolveDocumentFolder(txctx, importScope, next)
			if resolveErr != nil {
				return fmt.Errorf("resolve deleted %s:%s folder: %w", next.Type, next.Identity, resolveErr)
			}
			deleted, deleteErr := s.repository.UpdateDocument(txctx, next, document.Revision, folderID)
			if deleteErr != nil {
				return fmt.Errorf("soft-delete imported %s:%s: %w", next.Type, next.Identity, deleteErr)
			}
			revision, revisionErr := s.recordRevision(txctx, *deleted, "delete", nil)
			if revisionErr != nil {
				return revisionErr
			}
			revisions = append(revisions, *revision)
			result.Deletes++
		}
		for _, item := range bundle.Documents[entities.CollectionFacets] {
			if !portableDocumentDeleted(item) {
				continue
			}
			facet, facetErr := s.repository.GetFacet(txctx, live.ID, stringField(item, "identity"), true)
			if facetErr != nil {
				return facetErr
			}
			if facet.DeletedAt != nil {
				continue
			}
			now := time.Now().UTC()
			next := *facet
			next.Active = false
			next.DeletedAt = &now
			next.UpdatedBy = entities.Actor{ID: current.User.ID}
			deleted, deleteErr := s.repository.UpdateFacet(txctx, next, facet.Revision)
			if deleteErr != nil {
				return fmt.Errorf("soft-delete imported facet:%s: %w", facet.Identity, deleteErr)
			}
			revision, revisionErr := s.recordRevision(txctx, *deleted, "delete", nil)
			if revisionErr != nil {
				return revisionErr
			}
			revisions = append(revisions, *revision)
			result.Deletes++
		}
		currentFacets, txErr := s.repository.ListFacets(txctx, live.ID, false)
		if txErr != nil {
			return txErr
		}
		changedFacets, txErr := s.repository.ReorderFacets(txctx, live.ID, facetReorderItems(bundle, currentFacets), current.User.ID)
		if txErr != nil {
			return fmt.Errorf("reorder imported facets: %w", txErr)
		}
		for _, document := range changedFacets {
			revision, revisionErr := s.recordRevision(txctx, document, "update", nil)
			if revisionErr != nil {
				return revisionErr
			}
			revisions = append(revisions, *revision)
		}
		head := latest.HeadSequence
		for _, revision := range revisions {
			if revision.WorkspaceSequence != nil && *revision.WorkspaceSequence > head {
				head = *revision.WorkspaceSequence
			}
		}
		message := "Import workspace snapshot " + shortChecksum(plan.SnapshotChecksum)
		commit, txErr := s.repository.CreateCommit(txctx, entities.Commit{ID: uuid.NewString(), WorkspaceID: live.ID, ParentCommitID: &latest.ID, BaseSequence: latest.HeadSequence, HeadSequence: head, Message: message, RevisionPolicy: "preserve", Operation: "import", CreatedBy: entities.Actor{ID: current.User.ID}}, commitChanges(revisions))
		if txErr != nil {
			return txErr
		}
		if bundle.DomainVersion != "" && commit.DomainVersion != bundle.DomainVersion {
			return domainerrors.Conflict("domain_version_mismatch", "Imported workspace does not match the source domain version")
		}
		ids := make([]string, 0, len(revisions))
		for _, revision := range revisions {
			ids = append(ids, revision.ID)
		}
		if txErr = s.repository.AttachRevisionsToCommit(txctx, commit.ID, ids); txErr != nil {
			return txErr
		}
		if txErr = s.repository.MarkSnapshotImportPlanApplied(txctx, plan.ID); txErr != nil {
			return txErr
		}
		result.CommitID = commit.ID
		result.ParentCommitID = latest.ID
		result.DomainVersion = commit.DomainVersion
		return nil
	})
	return result, mapConflict(err)
}

func (s *Coordinator) planSnapshotChanges(ctx context.Context, workspaceID string, bundle entities.PortableBundle, plan *entities.ImportPlan) error {
	existing, err := s.loadSnapshotDocuments(ctx, workspaceID)
	if err != nil {
		return err
	}
	incoming := snapshotDocumentIdentities(bundle)
	tombstones := snapshotDocumentTombstones(bundle)
	for kind, identities := range incoming {
		for identity := range identities {
			document, exists := existing[kind][identity]
			switch {
			case !exists:
				plan.Creates++
			case tombstones[kind][identity] && document.DeletedAt != nil:
				plan.Updates++
			case tombstones[kind][identity] && document.DeletedAt == nil:
				if kind == entities.CollectionFacets {
					plan.Updates++
				}
				plan.Deletes++
			case document.DeletedAt != nil:
				plan.Restores++
			default:
				plan.Updates++
			}
		}
	}
	plan.Deletes = len(documentsMissingFromSnapshot(existing, incoming))
	return nil
}

func (s *Coordinator) loadSnapshotDocuments(ctx context.Context, workspaceID string) (map[string]map[string]entities.Document, error) {
	result := make(map[string]map[string]entities.Document, len(Collections))
	for _, kind := range Collections {
		documents, err := s.repository.ListAllDocuments(ctx, workspaceID, kind, true)
		if err != nil {
			return nil, err
		}
		result[kind] = make(map[string]entities.Document, len(documents))
		for _, document := range documents {
			result[kind][storedDocumentKey(document)] = document
		}
	}
	return result, nil
}

func snapshotDocumentIdentities(bundle entities.PortableBundle) map[string]map[string]bool {
	result := make(map[string]map[string]bool, len(Collections))
	for _, kind := range Collections {
		result[kind] = map[string]bool{}
		for _, item := range bundle.Documents[kind] {
			result[kind][portableDocumentKey(kind, item)] = true
		}
	}
	return result
}

func snapshotDocumentTombstones(bundle entities.PortableBundle) map[string]map[string]bool {
	result := make(map[string]map[string]bool, len(Collections))
	for _, kind := range Collections {
		result[kind] = map[string]bool{}
		for _, item := range bundle.Documents[kind] {
			result[kind][portableDocumentKey(kind, item)] = portableDocumentDeleted(item)
		}
	}
	return result
}

func documentsMissingFromSnapshot(existing map[string]map[string]entities.Document, incoming map[string]map[string]bool) []entities.Document {
	result := []entities.Document{}
	order := restoreOrder()
	for index := len(order) - 1; index >= 0; index-- {
		kind := order[index]
		if kind == entities.CollectionFolders {
			continue
		}
		documents := []entities.Document{}
		for identity, document := range existing[kind] {
			if !incoming[kind][identity] && document.DeletedAt == nil && document.ManagedBy != entities.ManagedBySystem {
				documents = append(documents, document)
			}
		}
		sort.Slice(documents, func(i, j int) bool { return documents[i].Identity < documents[j].Identity })
		result = append(result, documents...)
	}
	folders := []entities.Document{}
	for identity, document := range existing[entities.CollectionFolders] {
		if !incoming[entities.CollectionFolders][identity] && document.DeletedAt == nil && document.ManagedBy != entities.ManagedBySystem {
			folders = append(folders, document)
		}
	}
	sort.Slice(folders, func(i, j int) bool {
		left := folderDepth(folders[i], existing[entities.CollectionFolders])
		right := folderDepth(folders[j], existing[entities.CollectionFolders])
		if left == right {
			return folders[i].Identity < folders[j].Identity
		}
		return left > right
	})
	return append(result, folders...)
}

func folderDepth(document entities.Document, folders map[string]entities.Document) int {
	depth := 0
	seen := map[string]bool{document.Identity: true}
	current := document
	for current.FolderIdentity != nil && *current.FolderIdentity != "" {
		parentIdentity := *current.FolderIdentity
		if seen[parentIdentity] {
			break
		}
		parent, exists := folders[parentIdentity]
		if !exists {
			break
		}
		seen[parentIdentity] = true
		depth++
		current = parent
	}
	return depth
}

func shortChecksum(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func validateSnapshotRelations(bundle entities.PortableBundle) []string {
	available := map[string]map[string]bool{}
	folderScopes := map[string]string{}
	deletedFacets := map[string]bool{}
	for kind, items := range bundle.Documents {
		available[kind] = map[string]bool{}
		for _, item := range items {
			available[kind][stringField(item, "identity")] = true
			if kind == entities.CollectionFolders {
				folderScopes[stringField(item, "identity")] = defaultString(stringField(item, "scope"), entities.FolderScopeCollection)
			}
			if kind == entities.CollectionFacets {
				deletedFacets[stringField(item, "identity")] = portableDocumentDeleted(item)
			}
		}
	}
	result := []string{}
	for kind, items := range bundle.Documents {
		for _, item := range items {
			identity := stringField(item, "identity")
			if kind == entities.CollectionFolders {
				parent := stringField(item, "parentIdentity")
				entityType := stringField(item, "entityType")
				if parent != "" && parent != entities.RootFolderIdentity(entityType) && !available[entities.CollectionFolders][parent] {
					result = append(result, kind+":"+identity+": parentIdentity target is missing")
				} else if parent != "" && folderScopes[parent] != defaultString(stringField(item, "scope"), entities.FolderScopeCollection) {
					result = append(result, kind+":"+identity+": parentIdentity scope does not match")
				}
			} else if !slices.Contains(entities.FacetCollections, kind) {
				if folder := stringField(item, "folderIdentity"); folder != "" && folder != entities.RootFolderIdentity(kind) && !available[entities.CollectionFolders][folder] {
					result = append(result, kind+":"+identity+": folderIdentity target is missing")
				}
				workspaceFolder := defaultString(stringField(item, "workspaceFolderIdentity"), entities.WorkspaceRootFolderIdentity)
				if !available[entities.CollectionFolders][workspaceFolder] {
					result = append(result, kind+":"+identity+": workspaceFolderIdentity target is missing")
				} else if folderScopes[workspaceFolder] != entities.FolderScopeWorkspace {
					result = append(result, kind+":"+identity+": workspaceFolderIdentity target is not a Workspace folder")
				}
			}
			if kind == entities.CollectionFacetDocuments {
				facetIdentity := stringField(item, "facetIdentity")
				if !available[entities.CollectionFacets][facetIdentity] {
					result = append(result, kind+":"+facetIdentity+"/"+identity+": parent facet is missing")
				} else if deletedFacets[facetIdentity] && !portableDocumentDeleted(item) {
					result = append(result, kind+":"+facetIdentity+"/"+identity+": active document cannot belong to a deleted facet")
				}
			}
			if kind == entities.CollectionUpdates && !available[entities.CollectionStores][stringField(item, "storeIdentity")] {
				result = append(result, kind+":"+identity+": storeIdentity target is missing")
			}
			if kind == entities.CollectionProjects {
				for _, environment := range relationIdentityList(item["allowedEnvironments"]) {
					if !available["environments"][environment] {
						result = append(result, kind+":"+identity+": allowed environment "+environment+" is missing")
					}
				}
			}
			if kind == entities.CollectionVocabs {
				target := stringField(item, "authProfileIdentity")
				if target != "" && !available[entities.CollectionAuthProfiles][target] {
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

func mapImportPlanNotFound(err error) error {
	if errors.Is(err, ports.ErrNotFound) {
		return domainerrors.NotFound("import_plan_not_found", "Import plan was not found")
	}
	return err
}
