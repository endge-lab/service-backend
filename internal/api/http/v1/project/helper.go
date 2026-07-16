package project

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/gofiber/fiber/v2"
)

func NewProjectResponse(project *entities.RProject) *ProjectResponse {
	if project == nil {
		return nil
	}

	return &ProjectResponse{
		ID:          project.ID,
		Identity:    project.Identity,
		DisplayName: project.DisplayName,
		Description: project.Description,
		Active:      project.Active,
		DeletedAt:   project.DeletedAt,
		Meta:        project.Meta,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}

func NewProjectsListResponse(projects []*entities.RProject) ProjectsListResponse {
	items := make([]*ProjectResponse, 0, len(projects))
	for _, item := range projects {
		items = append(items, NewProjectResponse(item))
	}

	return ProjectsListResponse{Items: items}
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}
