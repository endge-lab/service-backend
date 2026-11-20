package user_state

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

type UseCase struct {
	repo        ports.UserStateRepository
	facets      ports.FacetRepository
	tx          ports.TxManager
	environment string
}

func NewUseCase(repo ports.UserStateRepository, facets ports.FacetRepository, tx ports.TxManager, environment string) *UseCase {
	return &UseCase{repo: repo, facets: facets, tx: tx, environment: environment}
}

type Snapshot struct {
	Partition     string                    `json:"partition"`
	ActorID       string                    `json:"actorId"`
	ResetRevision int64                     `json:"resetRevision"`
	Entries       []entities.UserStateEntry `json:"entries"`
	Application   string                    `json:"application,omitempty"`
}

type Operation struct {
	Key          string          `json:"key"`
	Value        json.RawMessage `json:"value,omitempty" swaggertype:"object" extensions:"x-endge-json-value"`
	Delete       bool            `json:"delete,omitempty"`
	BaseRevision int64           `json:"baseRevision"`
	ID           string          `json:"id"`
}

type Batch struct {
	Partition     string      `json:"partition"`
	ActorID       string      `json:"actorId,omitempty"`
	ResetRevision int64       `json:"resetRevision"`
	Operations    []Operation `json:"operations"`
}

type Outcome struct {
	Key    string                   `json:"key"`
	Status string                   `json:"status"`
	Entry  *entities.UserStateEntry `json:"entry"`
}

type BatchResult struct {
	ResetRevision int64     `json:"resetRevision"`
	Results       []Outcome `json:"results"`
}

type UserPage struct {
	Items []entities.UserStateUser `json:"items"`
	Next  string                   `json:"next,omitempty"`
}

func (s *UseCase) Context(ctx context.Context, facets map[string]string) (entities.CurrentActor, entities.UserStateScope, Scope, error) {
	actor, access, err := userContext(ctx)
	if err != nil {
		return actor, entities.UserStateScope{}, Scope{}, err
	}
	key, scope, err := s.resolveScope(ctx, access.Workspace, facets)
	return actor, key, scope, err
}

func userContext(ctx context.Context) (entities.CurrentActor, entities.WorkspaceAccess, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return actor, entities.WorkspaceAccess{}, err
	}
	if _, isToken := entities.AccessTokenFromContext(ctx); isToken {
		return actor, entities.WorkspaceAccess{}, domainerrors.Forbidden("user_state_session_required", "User settings require an interactive session")
	}
	access, err := shared.Access(ctx)
	return actor, access, err
}

func (s *UseCase) Scopes(ctx context.Context) ([]Scope, error) {
	_, access, err := userContext(ctx)
	if err != nil {
		return nil, err
	}
	scopes, err := s.repo.UserStateScopes(ctx, entities.UserStateScope{WorkspaceID: access.Workspace.ID, Environment: s.environment})
	if err != nil {
		return nil, err
	}
	items := []Scope{}
	for _, raw := range scopes {
		var facets map[string]string
		if err := json.Unmarshal([]byte(raw), &facets); err != nil {
			return nil, err
		}
		_, scope, err := s.resolveScope(ctx, access.Workspace, facets)
		if domainerrors.CodeOf(err) == "user_state_context_invalid" {
			continue
		}
		if err != nil {
			return nil, err
		}
		items = append(items, scope)
	}
	return items, nil
}

func (s *UseCase) Users(ctx context.Context, facets map[string]string, search, after string) (UserPage, error) {
	_, key, _, err := s.Context(ctx, facets)
	if err != nil {
		return UserPage{}, err
	}
	if len(search) > 160 || (after != "" && !validID(after)) {
		return UserPage{}, domainerrors.InvalidInput("user_state_filter_invalid", "Invalid user search or cursor")
	}
	const pageSize = 40
	users, err := s.repo.UserStateUsers(ctx, key, strings.TrimSpace(search), after, pageSize+1)
	if err != nil {
		return UserPage{}, err
	}
	result := UserPage{Items: users}
	if len(users) > pageSize {
		result.Items = users[:pageSize]
		result.Next = users[pageSize-1].ID
	}
	return result, nil
}

func (s *UseCase) Read(ctx context.Context, facets map[string]string, user string, managed bool, application string) (Snapshot, error) {
	actor, key, scope, err := s.Context(ctx, facets)
	if err != nil {
		return Snapshot{}, err
	}
	if !managed {
		if scope.Driver != "backend" {
			return Snapshot{}, driverDisabled()
		}
		user = actor.User.ID
	}
	key, err = settingsScope(key, managed, application)
	if err != nil {
		return Snapshot{}, err
	}
	if !validID(user) {
		return Snapshot{}, invalidTarget()
	}
	result := Snapshot{ActorID: actor.User.ID, Partition: partition(key), Application: key.Application}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repo.LockUserState(txctx, key, user); err != nil {
			return err
		}
		result.ResetRevision, err = s.repo.UserStateRevision(txctx, key, user)
		if err != nil {
			return err
		}
		result.Entries, err = s.repo.ReadUserState(txctx, key, user)
		return err
	})
	return result, err
}

func (s *UseCase) Save(ctx context.Context, facets map[string]string, user string, managed bool, application string, batch Batch) (BatchResult, error) {
	actor, access, err := userContext(ctx)
	if err != nil {
		return BatchResult{}, err
	}
	if managed && !shared.CanWrite(access.Role) {
		return BatchResult{}, domainerrors.Forbidden("workspace_editor_required", "Workspace Editor role is required")
	}
	key, scope, err := s.resolveScope(ctx, access.Workspace, facets)
	if err != nil {
		return BatchResult{}, err
	}
	if scope.Driver != "backend" {
		return BatchResult{}, driverDisabled()
	}
	if !managed {
		if batch.ActorID != actor.User.ID {
			return BatchResult{}, domainerrors.Forbidden("user_state_actor_changed", "The active session does not match this settings queue")
		}
		user = actor.User.ID
	}
	key, err = settingsScope(key, managed, application)
	if err != nil {
		return BatchResult{}, err
	}
	if !validID(user) {
		return BatchResult{}, invalidTarget()
	}
	if batch.Partition != partition(key) {
		return BatchResult{}, domainerrors.Forbidden("user_state_partition_changed", "The current server context does not match this settings queue")
	}
	if err := validateBatch(batch); err != nil {
		return BatchResult{}, err
	}

	result := BatchResult{Results: []Outcome{}}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repo.LockUserState(txctx, key, user); err != nil {
			return err
		}
		rev, err := s.repo.UserStateRevision(txctx, key, user)
		if err != nil {
			return err
		}
		result.ResetRevision = rev
		entries, err := s.repo.ReadUserState(txctx, key, user)
		if err != nil {
			return err
		}
		// The management endpoint edits stored settings and does not provision users.
		if managed && len(entries) == 0 && rev == batch.ResetRevision {
			return domainerrors.NotFound("user_state_user_not_found", "No saved settings for this user in this context")
		}
		byKey := make(map[string]*entities.UserStateEntry, len(entries))
		for i := range entries {
			byKey[entries[i].Key] = &entries[i]
		}
		var changed []string
		for _, op := range batch.Operations {
			entry := byKey[op.Key]
			outcome := Outcome{Key: op.Key, Status: "conflict", Entry: entry}
			revision := int64(0)
			if entry != nil {
				revision = entry.Revision
			}
			switch {
			case rev != batch.ResetRevision:
				outcome.Status = "reset"
			case entry != nil && entry.MutationID == op.ID:
				outcome.Status = "saved"
			case revision == op.BaseRevision:
				saved, err := s.repo.WriteUserState(txctx, key, user, actor.User.ID, entities.UserStateEntry{
					Key: op.Key, Value: op.Value, Deleted: op.Delete, MutationID: op.ID,
				})
				if err != nil {
					return err
				}
				outcome.Status, outcome.Entry = "saved", &saved
				changed = append(changed, op.Key)
			}
			result.Results = append(result.Results, outcome)
		}
		if managed && len(changed) != 0 {
			return s.repo.RecordUserStateAdminAction(txctx, key, user, actor.User.ID, "edit", changed, int64(len(changed)))
		}
		return nil
	})
	return result, err
}

func (s *UseCase) Reset(ctx context.Context, facets map[string]string, user, confirmation, expectedPartition string) (int64, error) {
	actor, access, err := shared.AdminContext(ctx)
	if err != nil {
		return 0, err
	}
	key, scope, err := s.resolveScope(ctx, access.Workspace, facets)
	if err != nil {
		return 0, err
	}
	if scope.Driver != "backend" {
		return 0, driverDisabled()
	}
	if expectedPartition != scope.Partition {
		return 0, domainerrors.Forbidden("user_state_partition_changed", "The current server context does not match this settings reset")
	}
	expected := "reset-all"
	if user != "" {
		if !validID(user) {
			return 0, invalidTarget()
		}
		expected = "reset-user"
	}
	if confirmation != expected {
		return 0, domainerrors.InvalidInput("user_state_confirmation_required", "Explicit confirmation is required")
	}
	var count int64
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repo.LockUserState(txctx, key, user); err != nil {
			return err
		}
		count, err = s.repo.ResetUserState(txctx, key, user)
		if err != nil {
			return err
		}
		return s.repo.RecordUserStateAdminAction(txctx, key, user, actor.User.ID, expected, []string{}, count)
	})
	return count, err
}

func partition(key entities.UserStateScope) string {
	if key.Application != "" {
		return "app:" + key.Application + ":" + key.Environment
	}
	return key.WorkspaceID + ":" + key.Environment
}

func validID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id.String() == value && id != uuid.Nil
}

func validateBatch(batch Batch) error {
	if batch.ResetRevision < 0 || len(batch.Operations) == 0 || len(batch.Operations) > 32 {
		return domainerrors.InvalidInput("user_state_batch_invalid", "Invalid settings batch")
	}
	seen := map[string]bool{}
	for _, op := range batch.Operations {
		if !validID(op.ID) || op.BaseRevision < 0 || seen[op.Key] || !validOperation(op) {
			return domainerrors.InvalidInput("user_state_operation_invalid", "Invalid key, revision or value in settings batch")
		}
		seen[op.Key] = true
	}
	return nil
}

func invalidTarget() error {
	return domainerrors.InvalidInput("user_state_target_invalid", "Invalid settings target user")
}

func driverDisabled() error {
	return domainerrors.Forbidden("user_state_backend_disabled", "Backend persistence is inactive in this context")
}
