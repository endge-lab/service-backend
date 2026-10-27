package mcp

import (
	"encoding/json"
	"net/http"
	"strings"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// toolError — тело ошибки инструмента: тот же code, что у REST, HTTP-статус и подсказка агенту.
type toolError struct {
	Error   string         `json:"error"`
	Status  int            `json:"status"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Hint    string         `json:"hint,omitempty"`
}

// jsonResult сериализует значение в текстовый контент инструмента.
func jsonResult(value any) (*mcp.CallToolResult, any, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil, nil
}

// rawResult возвращает уже сериализованный JSON без повторного разбора.
func rawResult(raw json.RawMessage) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil, nil
}

// errorResult переводит ошибку use case в результат инструмента с isError.
// Неожиданный сбой логируется, а агенту уходит безопасный internal_error без причины.
func errorResult(logger *zap.Logger, tool string, err error) (*mcp.CallToolResult, any, error) {
	return errorResultWithHint(logger, tool, err, "")
}

// errorResultWithHint — errorResult с дополнительной подсказкой инструмента (например, поле версии формата).
func errorResultWithHint(logger *zap.Logger, tool string, err error, extra string) (*mcp.CallToolResult, any, error) {
	status := domainerrors.HTTPStatusOf(err)
	body := toolError{Error: domainerrors.CodeOf(err), Status: status, Message: domainerrors.SafeMessageOf(err), Details: domainerrors.DetailsOf(err)}
	if status >= http.StatusInternalServerError {
		logger.Error("mcp tool failed", zap.String("tool", tool), zap.Error(err))
		body = toolError{Error: "internal_error", Status: status, Message: "Внутренняя ошибка сервера"}
	}
	body.Hint = hintFor(body.Error, status)
	if extra != "" && status < http.StatusInternalServerError {
		body.Hint = strings.TrimSpace(body.Hint + " " + extra)
	}
	raw, marshalErr := json.MarshalIndent(body, "", "  ")
	if marshalErr != nil {
		return nil, nil, marshalErr
	}
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil, nil
}

// hintFor подсказывает агенту следующий шаг для частых ошибок.
func hintFor(code string, status int) string {
	switch code {
	case "revision_conflict":
		return "Документ изменился: перечитайте его (endge_get_document) и повторите с новой revision."
	case "expected_revision_invalid":
		return "Передайте текущую revision документа из поля revision."
	case "head_sequence_conflict":
		return "Появились новые изменения или коммиты: запросите endge_commit_plan заново."
	case "workspace_required":
		return "Передайте workspace: identity из endge_list_workspaces."
	case "access_token_workspace_forbidden":
		return "Токен выпущен для одного workspace: не передавайте workspace или укажите workspace токена (endge_status)."
	case "access_token_scope_insufficient":
		return "Токен только для чтения: попросите пользователя выпустить токен с правом записи («Для агента» в Configurator)."
	case "access_token_delete_forbidden":
		return "Токен не разрешает удаление: попросите пользователя выпустить токен с флагом delete."
	case "access_token_commit_forbidden":
		return "Токен не разрешает коммиты: покажите пользователю endge_commit_plan и попросите закоммитить в Configurator."
	case "access_token_owner_admin_required":
		return "Коммит токеном доступен, только если его владелец — Admin этого workspace: покажите пользователю endge_commit_plan и попросите закоммитить в Configurator."
	case "validation_error":
		return "Входные данные не прошли проверку REST-контракта: исправьте поля из details (схема <коллекция>.CreateRequest/PatchRequest или commit.CreateRequest справочника API)."
	case "facet_identity_required":
		return "Для collection=facet-documents передайте facetIdentity: identity фасета из endge_list_documents с collection=facets."
	case "access_token_admin_forbidden":
		return "Операция требует роли Admin workspace и недоступна токену: покажите пользователю endge_commit_plan и попросите выполнить её в Configurator."
	case "collection_unsupported":
		return "Используйте одну из коллекций из описания параметра collection."
	case "precondition_required":
		return "Передайте текущую revision документа (поле revision из endge_get_document)."
	}
	switch status {
	case http.StatusUnauthorized:
		return "Токен недействителен, истёк или отозван: попросите пользователя выпустить новый."
	case http.StatusForbidden:
		return "У пользователя нет нужной роли в этом workspace."
	case http.StatusNotFound:
		return "Проверьте collection и identity (endge_list_documents)."
	}
	return ""
}

// summaryFields — краткий вид документа без source и meta.
var summaryFields = []string{"identity", "displayName", "description", "revision", "active", "folderIdentity", "facetIdentity", "sourceVersion", "modelVersion", "icon", "color", "position", "documentCount", "managedBy", "updatedAt", "deletedAt"}

// summarize оставляет краткие поля документа; в snapshot workspace revision и время лежат в state.
func summarize(item map[string]any) map[string]any {
	merged := map[string]any{}
	if state, ok := item["state"].(map[string]any); ok {
		for key, value := range state {
			merged[key] = value
		}
	}
	for key, value := range item {
		merged[key] = value
	}
	result := map[string]any{}
	for _, key := range summaryFields {
		value, ok := merged[key]
		if !ok || value == nil || value == "" {
			continue
		}
		result[key] = value
	}
	return result
}
