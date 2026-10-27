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

// agentDoc — раздел llms-документации для агентов, публикуемый как ресурс MCP.
type agentDoc struct {
	path, name, title, description string
}

var agentDocs = []agentDoc{
	{"/llms.txt", "llms", "Обзор и оглавление", "Краткий обзор Endge и оглавление разделов для агентов."},
	{"/llms-full/agents.txt", "agents", "Работа агента с Endge", "Порядок работы агента, правила изменения документов и коммитов."},
	{"/llms-full/reference.txt", "reference", "Справочник Source", "DSL документов: defineQuery, defineComponent, defineType и другие."},
	{"/llms-full/api.txt", "api", "HTTP API backend", "Справочник HTTP API Service Backend."},
}

// docLinks возвращает абсолютные ссылки на разделы документации или nil, если адрес не настроен.
func docLinks(base string) []map[string]string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil
	}
	links := make([]map[string]string, 0, len(agentDocs))
	for _, doc := range agentDocs {
		links = append(links, map[string]string{"title": doc.title, "url": base + doc.path})
	}
	return links
}

// instructions — порядок работы и правила для агента; передаются клиенту в initialize.
func instructions(cfg Config) string {
	var text strings.Builder
	text.WriteString(`Инструменты работают с Endge Service Backend от имени владельца токена (или пользователя JWT) через те же use case, что REST API.
Порядок: endge_status → endge_list_workspaces → чтение (endge_get_workspace, endge_list_documents, endge_get_document) → изменение → endge_commit_plan.
Персональный токен выпущен на один workspace: параметр workspace можно не передавать. С JWT передавайте workspace в каждый инструмент.
Документы описываются полем source на DSL Endge (defineQuery, defineComponent и т. д.). Перед созданием прочитайте похожий документ той же коллекции.
Изменение, удаление и восстановление требуют текущей revision; при revision_conflict перечитайте документ.
Изменения агента помечаются токеном в истории ревизий. Удаление и коммит выполняйте только по явной просьбе пользователя; перед коммитом покажите план.
Ошибка инструмента содержит code backend, HTTP-статус и подсказку hint.`)
	if links := docLinks(cfg.AgentDocsURL); links != nil {
		text.WriteString("\nДокументация для агентов (ресурсы MCP):")
		for _, link := range links {
			fmt.Fprintf(&text, "\n- %s: %s", link["title"], link["url"])
		}
	}
	return text.String()
}

// newServer собирает MCP-сервер с инструментами, разрешёнными набором permissions.
// Сервер не хранит состояния запроса и переиспользуется всеми запросами с тем же набором.
func newServer(cfg Config, useCases UseCases, contracts Contracts, logger *zap.Logger, allowed permissions) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: ServerName, Title: "Endge Service Backend", Version: cfg.Version}, &mcp.ServerOptions{Instructions: instructions(cfg)})
	t := &tools{cfg: cfg, useCases: useCases, contracts: contracts, logger: logger}
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
	registerDocResources(server, cfg.AgentDocsURL)
	return server
}

// registerDocResources публикует ссылки на llms-документацию; содержимое агент читает по URL.
func registerDocResources(server *mcp.Server, base string) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return
	}
	for _, doc := range agentDocs {
		uri := base + doc.path
		description := doc.description
		server.AddResource(&mcp.Resource{URI: uri, Name: doc.name, Title: doc.title, Description: description, MIMEType: "text/plain"},
			func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				text := fmt.Sprintf("%s\n%s\nПолный текст: %s", doc.title, description, uri)
				return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "text/plain", Text: text}}}, nil
			})
	}
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
