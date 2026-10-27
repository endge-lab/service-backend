package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// ServerName — имя реализации в ответе initialize.
const ServerName = "endge"

// Config — сведения, которые MCP показывает агенту: версия backend и адрес документации.
type Config struct {
	Service      string
	Version      string
	Env          string
	AgentDocsURL string
}

// permissions — набор инструментов изменения. Для персонального токена он сужается
// scope и флагами токена, поэтому агент не видит недоступных ему операций.
type permissions struct {
	write, commit, delete bool
}

// permissionsFrom вычисляет набор по токену запроса; сессия пользователя получает все
// инструменты, а права проверяет роль в workspace внутри use case.
func permissionsFrom(ctx context.Context) permissions {
	token, ok := entities.AccessTokenFromContext(ctx)
	if !ok {
		return permissions{write: true, commit: true, delete: true}
	}
	write := token.CanWrite()
	return permissions{write: write, commit: write && token.CanCommit, delete: write && token.CanDelete}
}

// instructions — порядок работы и правила для агента; передаются клиенту в initialize.
func instructions(docs *docsSource) string {
	var text strings.Builder
	if docs != nil {
		fmt.Fprintf(&text, "Сначала прочитайте ресурс llms-agent (%s): ядро документации для агента — правила, коды ошибок, авторинг Source и все операции. Схемы тел — шаблон ресурса llms-full/api/{group}.txt, Source каждого типа — llms-full/reference/{type}.txt, любая страница — {path}.md.\n", docs.uri("llms-agent.txt"))
	}
	text.WriteString(`Инструменты работают с Endge Service Backend от имени владельца токена (или пользователя JWT) через те же use case, что REST API.
Порядок: endge_status → endge_list_workspaces → чтение (endge_get_workspace, endge_list_documents, endge_get_document) → изменение → endge_commit_plan.
Персональный токен выпущен на один workspace: параметр workspace можно не передавать. С JWT передавайте workspace в каждый инструмент.
Документы описываются полем source на DSL Endge (defineQuery, defineComponent и т. д.). Перед созданием прочитайте похожий документ той же коллекции; поле версии формата коллекции (sourceVersion или modelVersion) указано в описании endge_create_document.
Фасеты и их документы (тенанты, регионы) — коллекции facets и facet-documents тех же инструментов; для facet-documents передавайте facetIdentity.
Изменение, удаление и восстановление требуют текущей revision; при revision_conflict перечитайте документ. Откат к прошлой версии: endge_document_revisions → endge_get_revision → endge_restore_revision.
Изменения агента помечаются токеном в истории ревизий. Удаление и коммит выполняйте только по явной просьбе пользователя; перед коммитом покажите план (endge_commit_plan перечисляет документы). История коммитов — endge_list_commits, изменения коммита — endge_get_commit.
Ошибка инструмента содержит code backend, HTTP-статус и подсказку hint.`)
	if links := docs.links(); links != nil {
		text.WriteString("\nДокументация для агентов (ресурсы MCP):")
		for _, link := range links {
			fmt.Fprintf(&text, "\n- %s: %s", link["title"], link["url"])
		}
	}
	return text.String()
}

// newServer собирает MCP-сервер с инструментами, разрешёнными набором permissions.
// Сервер не хранит состояния запроса и переиспользуется всеми запросами с тем же набором.
func newServer(cfg Config, useCases UseCases, contracts Contracts, logger *zap.Logger, allowed permissions, docs *docsSource) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: ServerName, Title: "Endge Service Backend", Version: cfg.Version}, &mcp.ServerOptions{Instructions: instructions(docs)})
	t := &tools{cfg: cfg, useCases: useCases, contracts: contracts, logger: logger, docs: docs}
	t.registerReadTools(server)
	if allowed.write {
		t.registerWriteTools(server)
	}
	if allowed.delete {
		t.registerDeleteTool(server)
	}
	if allowed.commit {
		t.registerCommitTool(server)
	}
	docs.register(server)
	return server
}

// inputSchema выводит JSON Schema из структуры входа и уточняет её (enum, пределы, значения по умолчанию).
func inputSchema[In any](customize func(properties map[string]*jsonschema.Schema)) *jsonschema.Schema {
	schema, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("mcp input schema: %v", err))
	}
	if customize != nil {
		customize(schema.Properties)
	}
	return schema
}

// rawDefault кодирует значение по умолчанию для схемы.
func rawDefault(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func intRange(minimum, maximum float64) (*float64, *float64) { return &minimum, &maximum }
