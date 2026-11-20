package user_state

import (
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	state "github.com/endge-lab/service-backend/internal/usecase/user_state"
	"github.com/gofiber/fiber/v2"
)

type contextsResponse struct {
	Items   []state.Scope      `json:"items"`
	Current *state.Scope       `json:"current"`
	Catalog []state.Definition `json:"catalog"`
}

type resetRequest struct {
	Partition    string `json:"partition"`
	User         string `json:"userId,omitempty"`
	Confirmation string `json:"confirmation"`
}

type resetResponse struct {
	Deleted int64 `json:"deleted"`
}

func groups(c *fiber.Ctx) []string {
	identity, _ := middleware.IdentityFromContext(c.UserContext())
	return identity.Groups
}

func selections(c *fiber.Ctx) (map[string]string, error) {
	raw := c.Query("facets", "{}")
	var result map[string]string
	if len(raw) > 4096 || json.Unmarshal([]byte(raw), &result) != nil || result == nil {
		return nil, domainerrors.InvalidInput("user_state_context_invalid", "Expected a facet selection object")
	}
	return result, nil
}

func decode(c *fiber.Ctx, target any) error {
	if len(c.Body()) > 64*1024 || json.Unmarshal(c.Body(), target) != nil {
		return domainerrors.InvalidInput("user_state_body_invalid", "Expected a settings request of at most 64 KiB")
	}
	return nil
}

func send(c *fiber.Ctx, data any, err error) error {
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(data)
}
