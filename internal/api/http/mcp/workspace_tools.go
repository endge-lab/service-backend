package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type workspaceCreateInput struct {
	Workspace map[string]any `json:"workspace" jsonschema:"Тело REST POST /api/v1/workspaces: обязательны identity и displayName"`
}

type workspaceUpdateInput struct {
	Workspace string         `json:"workspace" jsonschema:"Identity рабочего пространства из endge_list_workspaces"`
	Revision  int            `json:"revision" jsonschema:"Текущая revision из endge_list_workspaces"`
	Changes   map[string]any `json:"changes" jsonschema:"Поля REST PATCH /api/v1/workspaces/{identity}, например displayName или description"`
}

type workspaceArchiveInput struct {
	Workspace string `json:"workspace" jsonschema:"Identity рабочего пространства"`
	Revision  int    `json:"revision" jsonschema:"Текущая revision из endge_list_workspaces или endge_list_workspace_archive"`
}

type workspaceSummary struct {
	Identity    string     `json:"identity"`
	DisplayName string     `json:"displayName"`
	Revision    int        `json:"revision"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

func summarizeWorkspace(value entities.Workspace) workspaceSummary {
	return workspaceSummary{Identity: value.Identity, DisplayName: value.DisplayName, Revision: value.Revision, DeletedAt: value.DeletedAt}
}

func (t *tools) registerWorkspaceTools(server *mcp.Server) {
	if t.allowed.workspace == workspaceToolsPlatform {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "endge_create_workspace",
			Description: "Создать workspace с начальным состоянием. Владелец токена должен быть Platform Admin.",
			Annotations: mutating("Создать рабочее пространство", false),
		}, t.createWorkspace)
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_update_workspace",
		Description: "Изменить настройки workspace с проверкой revision. Требуется роль Admin в этом workspace.",
		Annotations: mutating("Изменить рабочее пространство", false),
	}, t.updateWorkspace)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_list_workspace_archive",
		Description: "Перечислить архивные workspace, которыми владелец токена может управлять, с revision для восстановления.",
		Annotations: readOnly("Архив рабочих пространств"),
	}, t.listWorkspaceArchive)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_archive_workspace",
		Description: "Мягко удалить workspace с проверкой revision, сохранив документы и назначения доступа. Требуется роль Admin в этом workspace.",
		Annotations: mutating("Архивировать рабочее пространство", true),
	}, t.archiveWorkspace)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "endge_restore_workspace",
		Description: "Восстановить workspace из архива с проверкой revision. Требуется роль Admin в этом workspace.",
		Annotations: mutating("Восстановить рабочее пространство", false),
	}, t.restoreWorkspace)
}

func (t *tools) createWorkspace(ctx context.Context, _ *mcp.CallToolRequest, in workspaceCreateInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_create_workspace"
	body, err := json.Marshal(in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	input, err := t.contracts.WorkspaceCreate(body)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	value, err := t.useCases.Workspaces.Create(ctx, input)
	return t.workspaceResult(tool, value, err)
}

func (t *tools) updateWorkspace(ctx context.Context, _ *mcp.CallToolRequest, in workspaceUpdateInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_update_workspace"
	body, err := json.Marshal(in.Changes)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	input, err := t.contracts.WorkspacePatch(body)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	value, err := t.useCases.Workspaces.Patch(ctx, strings.TrimSpace(in.Workspace), input, in.Revision)
	return t.workspaceResult(tool, value, err)
}

func (t *tools) listWorkspaceArchive(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	const tool = "endge_list_workspace_archive"
	items, err := t.useCases.Workspaces.ListArchive(ctx)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	archive := make([]workspaceSummary, 0, len(items))
	for _, item := range items {
		archive = append(archive, summarizeWorkspace(item.Workspace))
	}
	return jsonResult(map[string]any{"workspaces": archive})
}

func (t *tools) archiveWorkspace(ctx context.Context, _ *mcp.CallToolRequest, in workspaceArchiveInput) (*mcp.CallToolResult, any, error) {
	value, err := t.useCases.Workspaces.Delete(ctx, in.Workspace, in.Revision)
	return t.workspaceResult("endge_archive_workspace", value, err)
}

func (t *tools) restoreWorkspace(ctx context.Context, _ *mcp.CallToolRequest, in workspaceArchiveInput) (*mcp.CallToolResult, any, error) {
	value, err := t.useCases.Workspaces.Restore(ctx, in.Workspace, in.Revision)
	return t.workspaceResult("endge_restore_workspace", value, err)
}

func (t *tools) workspaceResult(tool string, value *entities.Workspace, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return jsonResult(summarizeWorkspace(*value))
}
