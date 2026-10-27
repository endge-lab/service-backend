package commits

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// Plan описывает предварительный план операции и её последствия.
type Plan struct {
	BaseSequence  int64            `json:"baseSequence"`
	HeadSequence  int64            `json:"headSequence"`
	RevisionCount int              `json:"revisionCount"`
	DocumentCount int              `json:"documentCount"`
	Contributors  []entities.Actor `json:"contributors"`
	Shared        bool             `json:"shared"`
	// Documents заполняет только PlanWithDocuments; REST-план остаётся прежним.
	Documents []PlanDocument `json:"documents,omitempty"`
}

// PlanDocument — документ с ревизиями, ожидающими коммита.
type PlanDocument struct {
	Collection    string            `json:"collection"`
	Identity      string            `json:"identity"`
	DisplayName   string            `json:"displayName,omitempty"`
	RevisionCount int               `json:"revisionCount"`
	Operations    []string          `json:"operations"`
	Authors       []entities.Actor  `json:"authors"`
	AccessTokens  []PlanAccessToken `json:"accessTokens,omitempty"`
}

// PlanAccessToken — персональный токен, которым сделана хотя бы одна ревизия документа.
type PlanAccessToken struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// Plan строит предварительный план операции без изменения состояния.
func (s *UseCase) Plan(ctx context.Context) (*Plan, error) {
	return s.plan(ctx, false)
}

// PlanWithDocuments строит тот же план и перечисляет документы: коллекцию, identity,
// displayName последней ревизии, число ревизий, операции, авторов и токены.
func (s *UseCase) PlanWithDocuments(ctx context.Context) (*Plan, error) {
	return s.plan(ctx, true)
}

func (s *UseCase) plan(ctx context.Context, withDocuments bool) (*Plan, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	latest, err := s.repository.LatestCommit(ctx, scope.Workspace.ID)
	if err != nil {
		return nil, err
	}
	pending, err := s.repository.PendingRevisions(ctx, scope.Workspace.ID, latest.HeadSequence)
	if err != nil {
		return nil, err
	}
	documents, contributors := map[string]bool{}, map[string]entities.Actor{}
	for _, revision := range pending {
		documents[revision.DocumentType+":"+revision.DocumentID] = true
		contributors[revision.CreatedBy.ID] = revision.CreatedBy
	}
	actors := make([]entities.Actor, 0, len(contributors))
	for _, actor := range contributors {
		actors = append(actors, actor)
	}
	result := &Plan{BaseSequence: latest.HeadSequence, HeadSequence: scope.Workspace.HeadSequence, RevisionCount: len(pending), DocumentCount: len(documents), Contributors: actors, Shared: len(contributors) > 1}
	if withDocuments {
		result.Documents = planDocuments(pending)
	}
	return result, nil
}

// planDocuments группирует ожидающие ревизии по документу в порядке первой ревизии.
// Identity и displayName берутся из последней ревизии: документ мог быть переименован.
func planDocuments(pending []entities.Revision) []PlanDocument {
	result := []PlanDocument{}
	index := map[string]int{}
	for _, revision := range pending {
		key := revision.DocumentType + ":" + revision.DocumentID
		position, ok := index[key]
		if !ok {
			position = len(result)
			index[key] = position
			result = append(result, PlanDocument{Collection: revision.DocumentType, Operations: []string{}, Authors: []entities.Actor{}})
		}
		document := &result[position]
		document.Identity = revision.DocumentIdentity
		if name := snapshotDisplayName(revision.Snapshot); name != "" {
			document.DisplayName = name
		}
		document.RevisionCount++
		if !slices.Contains(document.Operations, revision.Operation) {
			document.Operations = append(document.Operations, revision.Operation)
		}
		if !slices.ContainsFunc(document.Authors, func(actor entities.Actor) bool { return actor.ID == revision.CreatedBy.ID }) {
			document.Authors = append(document.Authors, revision.CreatedBy)
		}
		if revision.AccessTokenID != nil && !slices.ContainsFunc(document.AccessTokens, func(token PlanAccessToken) bool { return token.ID == *revision.AccessTokenID }) {
			token := PlanAccessToken{ID: *revision.AccessTokenID}
			if revision.AccessTokenName != nil {
				token.Name = *revision.AccessTokenName
			}
			document.AccessTokens = append(document.AccessTokens, token)
		}
	}
	return result
}

func snapshotDisplayName(snapshot json.RawMessage) string {
	var value struct {
		DisplayName string `json:"displayName"`
	}
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &value) != nil {
		return ""
	}
	return value.DisplayName
}
