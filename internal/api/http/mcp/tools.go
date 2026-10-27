package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/commits"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// tools — обработчики инструментов MCP. Каждый вызывает application use case напрямую
// с context запроса, поэтому identity, роль и отметка токена те же, что в REST.
type tools struct {
	cfg       Config
	useCases  UseCases
	contracts Contracts
	logger    *zap.Logger
	docs      *docsSource
}

func boolPtr(value bool) *bool { return &value }

// annotations задаёт подсказки клиенту: чтение, обратимое изменение или разрушительное действие.
func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, OpenWorldHint: boolPtr(false)}
}

func mutating(title string, destructive bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, DestructiveHint: boolPtr(destructive), OpenWorldHint: boolPtr(false)}
}

// collectionProperty ограничивает collection коллекциями документного API и фасетов.
func collectionProperty(properties map[string]*jsonschema.Schema) {
	enum := make([]any, 0, len(toolCollections))
	for _, name := range toolCollections {
		enum = append(enum, name)
	}
	properties["collection"].Enum = enum
}

// revisionProperty требует неотрицательную revision.
func revisionProperty(properties map[string]*jsonschema.Schema) {
	properties["revision"].Minimum = new(float64)
}

// pageProperties задаёт пределы и значение limit по умолчанию.
func pageProperties(properties map[string]*jsonschema.Schema, defaultLimit int) {
	properties["limit"].Minimum, properties["limit"].Maximum = intRange(1, 500)
	properties["limit"].Default = rawDefault(defaultLimit)
	properties["offset"].Minimum = new(float64)
}

const (
	collectionDescription = "Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"
	facetDescription      = "Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"
)

type workspaceInput struct {
	Workspace string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
}

type listDocumentsInput struct {
	Collection     string `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Workspace      string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	FacetIdentity  string `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	FolderIdentity string `json:"folderIdentity,omitempty" jsonschema:"Только документы этой папки"`
	IncludeDeleted bool   `json:"includeDeleted,omitempty" jsonschema:"Включить мягко удалённые документы"`
	WithSource     bool   `json:"withSource,omitempty" jsonschema:"Вернуть документы целиком, вместе с source и meta"`
	Limit          int    `json:"limit,omitempty" jsonschema:"Размер страницы"`
	Offset         int    `json:"offset,omitempty" jsonschema:"Смещение"`
}

type documentInput struct {
	FacetIdentity  string `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	Collection     string `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Identity       string `json:"identity" jsonschema:"Identity документа"`
	Workspace      string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	IncludeDeleted bool   `json:"includeDeleted,omitempty" jsonschema:"Разрешить чтение мягко удалённого документа"`
}

type workspaceSnapshotInput struct {
	Workspace string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	Full      bool   `json:"full,omitempty" jsonschema:"Вернуть полный snapshot со всеми source; может быть большим"`
}

type revisionsInput struct {
	FacetIdentity string `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	Collection    string `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Identity      string `json:"identity" jsonschema:"Identity документа"`
	Workspace     string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	WithSnapshot  bool   `json:"withSnapshot,omitempty" jsonschema:"Включить снимки документа в каждой ревизии"`
	Limit         int    `json:"limit,omitempty" jsonschema:"Размер страницы, от новых ревизий к старым"`
	Offset        int    `json:"offset,omitempty" jsonschema:"Смещение"`
}

type createInput struct {
	FacetIdentity string         `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	Collection    string         `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Workspace     string         `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	Document      map[string]any `json:"document" jsonschema:"Тело создания, как в REST POST /api/v1/<коллекция> (схема <коллекция>.CreateRequest): identity, displayName, source, поле версии формата (sourceVersion или modelVersion — по коллекции, см. описание инструмента), folderIdentity, description, meta. Фасет: identity, displayName, icon, color, meta. Документ фасета: identity, displayName, description, configuration, meta, active"`
}

type updateInput struct {
	FacetIdentity string         `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	Collection    string         `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Identity      string         `json:"identity" jsonschema:"Identity документа"`
	Revision      int            `json:"revision" jsonschema:"Текущая revision документа (поле revision из endge_get_document)"`
	Workspace     string         `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	Changes       map[string]any `json:"changes" jsonschema:"Изменяемые поля, например {\"source\": \"...\"}"`
}

type mutateInput struct {
	FacetIdentity string `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	Collection    string `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Identity      string `json:"identity" jsonschema:"Identity документа"`
	Revision      int    `json:"revision" jsonschema:"Текущая revision документа (поле revision из endge_get_document)"`
	Workspace     string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
}

type commitInput struct {
	Workspace            string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	Message              string `json:"message" jsonschema:"Сообщение коммита"`
	ExpectedHeadSequence int64  `json:"expectedHeadSequence" jsonschema:"headSequence из endge_commit_plan"`
	RevisionPolicy       string `json:"revisionPolicy,omitempty" jsonschema:"preserve (по умолчанию) — сохранить ревизии, squash — свернуть по документу"`
}

func (t *tools) registerReadTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_status",
		Description: "Версия backend, способ входа, пользователь, права токена и ссылки на документацию. Вызывайте первым.",
		Annotations: readOnly("Состояние подключения к Endge"),
	}, t.status)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_list_workspaces",
		Description: "Пользователь, признак Platform Admin и доступные workspace с ролями. С персональным токеном — только workspace токена и роль, суженная правами токена.",
		Annotations: readOnly("Workspace и роли"),
	}, t.listWorkspaces)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_list_documents",
		Description: "Документы коллекции: кратко (без source) или целиком с withSource=true. collection=facets — фасеты (тенанты, регионы); collection=facet-documents с facetIdentity — документы фасета.",
		Annotations: readOnly("Список документов коллекции"),
		InputSchema: inputSchema[listDocumentsInput](func(p map[string]*jsonschema.Schema) {
			collectionProperty(p)
			pageProperties(p, 100)
		}),
	}, t.listDocuments)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_get_document",
		Description: "Документ целиком: source, meta и текущая revision. Фасет и документ фасета (configuration) — коллекции facets и facet-documents.",
		Annotations: readOnly("Документ"),
		InputSchema: inputSchema[documentInput](collectionProperty),
	}, t.getDocument)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_get_workspace",
		Description: "Содержимое workspace. По умолчанию — документы по коллекциям кратко (identity, displayName, revision); full=true возвращает полный snapshot со всеми source.",
		Annotations: readOnly("Содержимое workspace"),
	}, t.getWorkspace)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_document_revisions",
		Description: "История ревизий документа от новых к старым: id ревизии, номер, автор, операция, время, коммит, accessTokenId/accessTokenName для изменений агента. Снимки — только с withSnapshot=true или в endge_get_revision.",
		Annotations: readOnly("История ревизий документа"),
		InputSchema: inputSchema[revisionsInput](func(p map[string]*jsonschema.Schema) {
			collectionProperty(p)
			pageProperties(p, 20)
		}),
	}, t.revisions)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_get_revision",
		Description: "Одна ревизия документа со снимком (snapshot) — состоянием документа после этой ревизии. Используйте перед откатом endge_restore_revision, чтобы сравнить с текущим документом.",
		Annotations: readOnly("Ревизия документа"),
		InputSchema: inputSchema[revisionInput](collectionProperty),
	}, t.getRevision)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_commit_plan",
		Description: "План коммита: документы с ревизиями, ожидающими коммита (коллекция, identity, displayName, число ревизий, операции, авторы, токены агентов), счётчики, участники и headSequence для endge_commit. Ничего не меняет. Покажите пользователю чужие правки в плане до коммита.",
		Annotations: readOnly("План коммита"),
	}, t.commitPlan)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_list_commits",
		Description: "История коммитов workspace от новых к старым: id, сообщение, автор, время, headSequence, политика ревизий. Изменения коммита — endge_get_commit.",
		Annotations: readOnly("История коммитов"),
		InputSchema: inputSchema[listCommitsInput](func(p map[string]*jsonschema.Schema) { pageProperties(p, 20) }),
	}, t.listCommits)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_get_commit",
		Description: "Коммит и его изменения (diff, как REST GET /api/v1/commits/{id}/diff): для каждого документа — коллекция, identity, операция, ревизии до и после. Содержимое версий — endge_get_revision по beforeRevisionId/afterRevisionId.",
		Annotations: readOnly("Изменения коммита"),
	}, t.getCommit)
}

func (t *tools) registerWriteTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "endge_create_document",
		Description: "Создаёт документ в коллекции. Поля как в REST (схема <коллекция>.CreateRequest справочника API, ресурс llms-full/api/{group}.txt): identity, displayName, source, поле версии формата и другие. Возвращает краткий вид с revision.\n" +
			"Поле версии формата по коллекциям (неизвестное поле — validation_error):\n" + formatVersionSummary(t.contracts.FormatVersions, entities.DocumentCollections) + "\n" +
			"- facets: нет поля версии; тело — identity, displayName, icon (имя иконки, например MapPin), color (#rrggbb), meta\n" +
			"- facet-documents (с facetIdentity): нет поля версии; тело — identity, displayName, description, configuration (обязательно: {\"mode\":\"inherit\",\"patch\":{…}} или {\"mode\":\"replace\",\"value\":{…}}), meta, active",
		Annotations: mutating("Создать документ", false),
		InputSchema: inputSchema[createInput](collectionProperty),
	}, t.createDocument)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_update_document",
		Description: "Частично изменяет документ (как PATCH): передайте только изменяемые поля и текущую revision. Меняя source, передавайте поле версии формата коллекции (sourceVersion или modelVersion у components, см. endge_create_document). Возвращает краткий вид с новой revision.",
		Annotations: mutating("Изменить документ", false),
		InputSchema: inputSchema[updateInput](func(p map[string]*jsonschema.Schema) {
			collectionProperty(p)
			revisionProperty(p)
		}),
	}, t.updateDocument)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_restore_document",
		Description: "Восстанавливает мягко удалённый документ. Вернуть прошлое содержимое существующего документа — endge_restore_revision.",
		Annotations: mutating("Восстановить документ", false),
		InputSchema: inputSchema[mutateInput](func(p map[string]*jsonschema.Schema) {
			collectionProperty(p)
			revisionProperty(p)
		}),
	}, t.restoreDocument)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_restore_revision",
		Description: "Откатывает документ к прошлой ревизии (как REST POST .../revisions/{revisionId}/restore с If-Match): содержимое берётся из снимка ревизии, создаётся новая ревизия restore, история не переписывается. Нужны revisionId из endge_document_revisions и текущая revision документа.",
		Annotations: mutating("Откатить документ к ревизии", false),
		InputSchema: inputSchema[restoreRevisionInput](func(p map[string]*jsonschema.Schema) {
			collectionProperty(p)
			revisionProperty(p)
		}),
	}, t.restoreRevision)
}

func (t *tools) registerDeleteTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_delete_document",
		Description: "Мягко удаляет документ, фасет или документ фасета. Только по явной просьбе пользователя; восстановление — endge_restore_document.",
		Annotations: mutating("Удалить документ", true),
		InputSchema: inputSchema[mutateInput](func(p map[string]*jsonschema.Schema) {
			collectionProperty(p)
			revisionProperty(p)
		}),
	}, t.deleteDocument)
}

func (t *tools) registerCommitTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_commit",
		Description: "Фиксирует ожидающие ревизии в коммит. Сначала покажите пользователю endge_commit_plan; выполняйте только по его просьбе.",
		Annotations: mutating("Создать коммит", false),
		// Длину message, enum revisionPolicy и границы headSequence проверяет DTO REST
		// (Contracts.Commit), а не схема SDK: правило одно на оба транспорта.
	}, t.commit)
}

// workspaceContext повторяет RequireWorkspace: по умолчанию берёт workspace токена,
// проверяет доступ use case и кладёт WorkspaceAccess в context.
func (t *tools) workspaceContext(ctx context.Context, requested string) (context.Context, error) {
	identity := strings.TrimSpace(requested)
	if token, ok := entities.AccessTokenFromContext(ctx); ok && identity == "" {
		identity = token.WorkspaceIdentity
	}
	if identity == "" {
		return nil, domainerrors.InvalidInput("workspace_required", "workspace is required for user sessions")
	}
	access, err := t.useCases.Workspaces.Authorize(ctx, identity)
	if err != nil {
		return nil, err
	}
	return entities.WithWorkspaceAccess(ctx, access), nil
}

// documentContract возвращает DTO REST коллекции. Коллекция уже проверена
// store, а BindContracts покрывает все коллекции (тест contracts_test.go);
// отсутствие контракта — ошибка сборки, поэтому проверка отклоняет любое тело.
func (t *tools) documentContract(collection string) shared.DocumentContract {
	if contract, ok := t.contracts.Documents[collection]; ok {
		return contract
	}
	missing := shared.RequestContract{}
	return shared.DocumentContract{Create: missing, Patch: missing}
}

// commitRequestJSON переводит вход инструмента в тело REST POST /api/v1/commits:
// workspace — параметр маршрута MCP, а не поле DTO.
func commitRequestJSON(in commitInput) []byte {
	body := map[string]any{"message": in.Message, "expectedHeadSequence": in.ExpectedHeadSequence}
	if in.RevisionPolicy != "" {
		body["revisionPolicy"] = in.RevisionPolicy
	}
	payload, _ := json.Marshal(body)
	return payload
}

// documentView — документ в форме REST-ответа (payload + серверные поля), без typed nil.
func documentView(document entities.Document) (map[string]any, error) {
	value, err := shared.DocumentMap(document)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	return result, json.Unmarshal(raw, &result)
}

func documentSummary(document entities.Document) (map[string]any, error) {
	view, err := documentView(document)
	if err != nil {
		return nil, err
	}
	return summarize(view), nil
}

func (t *tools) status(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	result := map[string]any{
		"service": t.cfg.Service, "version": t.cfg.Version, "env": t.cfg.Env,
		"mcp": map[string]any{"transport": "streamable-http", "stateless": true},
	}
	if actor, ok := entities.CurrentActorFromContext(ctx); ok && actor.User != nil {
		result["user"] = map[string]any{"id": actor.User.ID, "username": actor.User.Username, "displayName": actor.User.DisplayName}
	}
	if token, ok := entities.AccessTokenFromContext(ctx); ok {
		result["authentication"] = "access_token"
		result["accessToken"] = map[string]any{
			"id": token.TokenID, "workspace": token.WorkspaceIdentity, "scope": token.Scope,
			"commit": token.CanWrite() && token.CanCommit, "delete": token.CanWrite() && token.CanDelete,
			"expiresAt": token.ExpiresAt.UTC().Format(time.RFC3339),
		}
		result["defaultWorkspace"] = token.WorkspaceIdentity
	} else {
		result["authentication"] = "user"
		result["defaultWorkspace"] = nil
	}
	if links := t.docs.links(); links != nil {
		result["docs"] = links
	}
	return jsonResult(result)
}

func (t *tools) listWorkspaces(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	current, err := t.useCases.Session.Current(ctx)
	if err != nil {
		return errorResult(t.logger, "endge_list_workspaces", err)
	}
	items := make([]map[string]any, 0, len(current.Workspaces))
	for _, access := range current.Workspaces {
		items = append(items, map[string]any{"identity": access.Workspace.Identity, "displayName": access.Workspace.DisplayName, "role": access.Role})
	}
	result := map[string]any{"platformAdmin": current.PlatformAdmin, "workspaces": items}
	if current.User != nil {
		result["user"] = map[string]any{"id": current.User.ID, "username": current.User.Username, "displayName": current.User.DisplayName}
	}
	return jsonResult(result)
}

// documentTarget проверяет коллекцию и workspace: общий пролог инструментов документов.
func (t *tools) documentTarget(ctx context.Context, collection, facetIdentity, workspace string) (context.Context, collectionStore, string, error) {
	store, facet, err := t.store(collection, facetIdentity)
	if err != nil {
		return nil, nil, "", err
	}
	ctx, err = t.workspaceContext(ctx, workspace)
	if err != nil {
		return nil, nil, "", err
	}
	return ctx, store, facet, nil
}

func (t *tools) listDocuments(ctx context.Context, _ *mcp.CallToolRequest, in listDocumentsInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_list_documents"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	filter := ports.DocumentFilter{IncludeDeleted: in.IncludeDeleted, Limit: in.Limit, Offset: in.Offset}
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	if folder := strings.TrimSpace(in.FolderIdentity); folder != "" {
		filter.FolderIdentity = &folder
	}
	values, err := store.list(ctx, facet, filter)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	items := make([]map[string]any, 0, len(values))
	for _, value := range values {
		item, mapErr := store.view(value)
		if mapErr != nil {
			return errorResult(t.logger, tool, mapErr)
		}
		if !in.WithSource {
			item = summarize(item)
		}
		items = append(items, item)
	}
	result := map[string]any{"collection": in.Collection, "items": items, "total": len(items), "limit": filter.Limit, "offset": filter.Offset}
	if facet != "" {
		result["facetIdentity"] = facet
	}
	return jsonResult(result)
}

func (t *tools) getDocument(ctx context.Context, _ *mcp.CallToolRequest, in documentInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_get_document"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	document, err := store.get(ctx, facet, in.Identity, in.IncludeDeleted)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	view, err := store.view(*document)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return jsonResult(view)
}

func (t *tools) getWorkspace(ctx context.Context, _ *mcp.CallToolRequest, in workspaceSnapshotInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_get_workspace"
	ctx, err := t.workspaceContext(ctx, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	raw, err := t.useCases.Domain.Live(ctx)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	if in.Full {
		return rawResult(raw)
	}
	var snapshot struct {
		DomainVersion string                      `json:"domainVersion"`
		Workspace     map[string]any              `json:"workspace"`
		Documents     map[string][]map[string]any `json:"documents"`
	}
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		return errorResult(t.logger, tool, domainerrors.Wrap(err, "snapshot_mapping_failed", "Не удалось разобрать snapshot", 500))
	}
	collections := map[string][]map[string]any{}
	for name, items := range snapshot.Documents {
		summaries := make([]map[string]any, 0, len(items))
		for _, item := range items {
			summaries = append(summaries, summarize(item))
		}
		collections[name] = summaries
	}
	workspace := map[string]any{}
	for _, key := range []string{"identity", "displayName", "description", "state"} {
		if value, ok := snapshot.Workspace[key]; ok {
			workspace[key] = value
		}
	}
	return jsonResult(map[string]any{"workspace": workspace, "domainVersion": snapshot.DomainVersion, "documents": collections})
}

func (t *tools) commitPlan(ctx context.Context, _ *mcp.CallToolRequest, in workspaceInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_commit_plan"
	ctx, err := t.workspaceContext(ctx, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	plan, err := t.useCases.Commits.PlanWithDocuments(ctx)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	if plan.Documents == nil {
		plan.Documents = []commits.PlanDocument{}
	}
	return jsonResult(plan)
}

func (t *tools) createDocument(ctx context.Context, _ *mcp.CallToolRequest, in createInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_create_document"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	payload, err := json.Marshal(in.Document)
	if err != nil {
		return errorResult(t.logger, tool, domainerrors.InvalidInput("invalid_json", "document must be a JSON object"))
	}
	return t.writeSummary(tool, in.Collection, store, func() (*entities.Document, error) { return store.create(ctx, facet, payload) })
}

func (t *tools) updateDocument(ctx context.Context, _ *mcp.CallToolRequest, in updateInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_update_document"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	payload, err := json.Marshal(in.Changes)
	if err != nil {
		return errorResult(t.logger, tool, domainerrors.InvalidInput("invalid_json", "changes must be a JSON object"))
	}
	return t.writeSummary(tool, in.Collection, store, func() (*entities.Document, error) {
		return store.patch(ctx, facet, in.Identity, payload, in.Revision)
	})
}

func (t *tools) restoreDocument(ctx context.Context, _ *mcp.CallToolRequest, in mutateInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_restore_document"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return t.writeSummary(tool, in.Collection, store, func() (*entities.Document, error) { return store.restore(ctx, facet, in.Identity, in.Revision) })
}

// deleteDocument дублирует проверку флага delete: инструмент скрыт от такого токена,
// но вызов по имени не должен обходить политику REST.
func (t *tools) deleteDocument(ctx context.Context, _ *mcp.CallToolRequest, in mutateInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_delete_document"
	if !permissionsFrom(ctx).delete {
		return errorResult(t.logger, tool, domainerrors.Forbidden("access_token_delete_forbidden", "Access token does not allow deletion"))
	}
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	document, err := store.remove(ctx, facet, in.Identity, in.Revision)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	view, err := store.view(*document)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return jsonResult(map[string]any{"deleted": true, "document": summarize(view)})
}

// commit дублирует проверку флага commit по той же причине, что deleteDocument.
func (t *tools) commit(ctx context.Context, _ *mcp.CallToolRequest, in commitInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_commit"
	if !permissionsFrom(ctx).commit {
		return errorResult(t.logger, tool, domainerrors.Forbidden("access_token_commit_forbidden", "Access token does not allow commits and releases"))
	}
	ctx, err := t.workspaceContext(ctx, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	if err = t.contracts.Commit.Validate(commitRequestJSON(in)); err != nil {
		return errorResult(t.logger, tool, err)
	}
	policy := strings.TrimSpace(in.RevisionPolicy)
	if policy == "" {
		policy = "preserve"
	}
	created, err := t.useCases.Commits.Create(ctx, in.Message, policy, in.ExpectedHeadSequence)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return jsonResult(created)
}

// writeSummary выполняет изменение и возвращает краткий вид документа с новой revision.
// Ошибка проверки тела дополняется подсказкой о поле версии формата коллекции.
func (t *tools) writeSummary(tool, collection string, store collectionStore, operation func() (*entities.Document, error)) (*mcp.CallToolResult, any, error) {
	document, err := operation()
	if err != nil {
		return errorResultWithHint(t.logger, tool, err, t.formatVersionHint(collection, err))
	}
	view, err := store.view(*document)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return jsonResult(summarize(view))
}

// formatVersionHint напоминает поле версии формата коллекции при ошибке проверки тела.
func (t *tools) formatVersionHint(collection string, err error) string {
	switch domainerrors.CodeOf(err) {
	case "validation_error", "source_version_invalid", "query_source_version_invalid", "simulation_source_version_invalid":
	default:
		return ""
	}
	version, ok := t.contracts.FormatVersions[collection]
	if !ok {
		return ""
	}
	if version.Field == "" {
		return "У коллекции " + collection + " нет поля версии формата: не передавайте sourceVersion и modelVersion."
	}
	return "Поле версии формата коллекции " + collection + ": " + version.describe() + "."
}
