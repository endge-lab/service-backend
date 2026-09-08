package mock_data

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	errs "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/gofiber/fiber/v2"
	"io"
	"strings"
	"time"
)

type Handler struct{ usecase UseCase }

func NewHandler(u UseCase) *Handler { return &Handler{usecase: u} }
func (h *Handler) body(c *fiber.Ctx) (json.RawMessage, error) {
	if len(c.Body()) > h.usecase.RequestLimit() {
		return nil, errs.New("request.too_large", "Request exceeds byte limit", 413)
	}
	if !bytes.HasPrefix(bytes.TrimSpace(c.Body()), []byte("{")) || !json.Valid(c.Body()) {
		return nil, errs.InvalidInput("request.invalid", "Expected a JSON object")
	}
	return append(json.RawMessage(nil), c.Body()...), nil
}
func respondValue(c *fiber.Ctx, value any, err error) error {
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	return c.JSON(value)
}

// Capabilities returns generator availability and effective limits.
// @Summary Получить возможности Mock Generator
// @Description Доступно Workspace Viewer. Контекст пользователя и workspace проверяется backend; генератор вызывается по OIDC gRPC. Сессии доступны только создателю.
// @ID getMockCapabilities
// @Tags Mock Generator
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/capabilities [get]
func (h *Handler) Capabilities(c *fiber.Ctx) error {
	r, e := h.usecase.Capabilities(c.UserContext())
	return respondValue(c, r, e)
}

// Generate generates independent JSON roots.
// @Summary Сгенерировать JSON по JSON Schema Draft 2020-12
// @Description Доступно Workspace Viewer. Контекст пользователя и workspace проверяется backend; генератор вызывается по OIDC gRPC. Сессии доступны только создателю.
// @ID generateMockData
// @Tags Mock Generator
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param request body GenerationRequest true "Schema and generation options"
// @Success 200 {object} GenerationResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 413 {object} shared.ErrorResponse
// @Failure 429 {object} shared.ErrorResponse
// @Failure 502 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Failure 504 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/generate [post]
func (h *Handler) Generate(c *fiber.Ctx) error {
	raw, e := h.body(c)
	if e != nil {
		return respond.WriteErrorResponse(c, e)
	}
	r, e := h.usecase.Generate(c.UserContext(), raw)
	return respondValue(c, r, e)
}

// Create prepares a stream without generating data until subscribed.
// @Summary Создать сессию генерации
// @Description Доступно Workspace Viewer. Контекст пользователя и workspace проверяется backend; генератор вызывается по OIDC gRPC. Сессии доступны только создателю.
// @ID createMockStream
// @Tags Mock Generator
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param request body StreamRequest true "Schema, generation options and stream parameters"
// @Success 201 {object} entities.MockStream
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 413 {object} shared.ErrorResponse
// @Failure 429 {object} shared.ErrorResponse
// @Failure 502 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Failure 504 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/streams [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	raw, e := h.body(c)
	if e != nil {
		return respond.WriteErrorResponse(c, e)
	}
	r, e := h.usecase.Create(c.UserContext(), raw)
	if e != nil {
		return respond.WriteErrorResponse(c, e)
	}
	r.EventsURL = string(c.Path()) + "/" + r.ID + "/events"
	return c.Status(201).JSON(r)
}

// Get returns an owned stream.
// @Summary Получить состояние сессии
// @Description Доступно Workspace Viewer. Контекст пользователя и workspace проверяется backend; генератор вызывается по OIDC gRPC. Сессии доступны только создателю.
// @ID getMockStream
// @Tags Mock Generator
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Stream ID"
// @Success 200 {object} entities.MockStream
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/streams/{id} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	r, e := h.usecase.Get(c.UserContext(), c.Params("id"))
	return respondValue(c, r, e)
}

// Update changes parameters between batches without resetting PRNG.
// @Summary Изменить параметры или приостановить поток
// @Description Доступно Workspace Viewer. Контекст пользователя и workspace проверяется backend; генератор вызывается по OIDC gRPC. Сессии доступны только создателю.
// @ID updateMockStream
// @Tags Mock Generator
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Stream ID"
// @Param request body StreamPatch true "Mutable parameters"
// @Success 200 {object} entities.MockStream
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/streams/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	raw, e := h.body(c)
	if e != nil {
		return respond.WriteErrorResponse(c, e)
	}
	r, e := h.usecase.Update(c.UserContext(), c.Params("id"), raw)
	return respondValue(c, r, e)
}

// KeepAlive renews an active stream lease; data does not renew it.
// @Summary Продлить сессию на 180 секунд
// @Description Доступно Workspace Viewer. Контекст пользователя и workspace проверяется backend; генератор вызывается по OIDC gRPC. Сессии доступны только создателю.
// @ID keepAliveMockStream
// @Tags Mock Generator
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Stream ID"
// @Success 200 {object} entities.MockStream
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/streams/{id}/keepalive [post]
func (h *Handler) KeepAlive(c *fiber.Ctx) error {
	r, e := h.usecase.KeepAlive(c.UserContext(), c.Params("id"))
	return respondValue(c, r, e)
}

// Stop cancels both sides of the stream.
// @Summary Остановить сессию
// @Description Доступно Workspace Viewer. Контекст пользователя и workspace проверяется backend; генератор вызывается по OIDC gRPC. Сессии доступны только создателю.
// @ID stopMockStream
// @Tags Mock Generator
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Stream ID"
// @Success 204
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/streams/{id} [delete]
func (h *Handler) Stop(c *fiber.Ctx) error {
	if e := h.usecase.Stop(c.UserContext(), c.Params("id")); e != nil {
		return respond.WriteErrorResponse(c, e)
	}
	return c.SendStatus(204)
}

// Events relays an authenticated gRPC subscription as standard SSE message events.
// @Summary Подключиться к SSE генерации
// @Description Единственная подписка. Message envelope: started/data/completed/failed. Нет replay. KeepAlive обязателен даже при поступлении данных.
// @ID subscribeMockStream
// @Tags Mock Generator
// @Produce text/event-stream
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Stream ID"
// @Success 200 {string} string "SSE message events"
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Failure 504 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/mock-data/streams/{id}/events [get]
func (h *Handler) Events(c *fiber.Ctx) error {
	if c.Get("Last-Event-ID") != "" {
		return respond.WriteErrorResponse(c, errs.Conflict("stream.replay_unsupported", "Stream replay is unsupported"))
	}
	sub, err := h.usecase.Subscribe(c.UserContext(), c.Params("id"))
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	streamID := strings.Clone(c.Params("id"))
	conn := c.Context().Conn()
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache, no-transform")
	c.Set("X-Accel-Buffering", "no")
	c.Context().SetConnectionClose()
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		defer sub.Close()
		write := func(raw []byte) error {
			if conn != nil {
				if err := conn.SetWriteDeadline(time.Now().Add(h.usecase.WriteTimeout())); err != nil {
					return err
				}
			}
			if _, err := writer.Write(raw); err != nil {
				return err
			}
			return writer.Flush()
		}
		type packet struct {
			data []byte
			err  error
		}
		messages := make(chan packet)
		go func() {
			for {
				event, err := sub.Recv()
				if err != nil {
					select {
					case messages <- packet{err: err}:
					case <-sub.Context().Done():
					}
					return
				}
				raw, err := json.Marshal(event)
				if err == nil && !h.usecase.ReserveBytes(int64(len(raw))) {
					err = errs.New("stream.backpressure", "Backend buffer budget exhausted", 429)
				}
				if err != nil {
					select {
					case messages <- packet{err: err}:
					case <-sub.Context().Done():
					}
					return
				}
				select {
				case messages <- packet{data: raw}:
				case <-sub.Context().Done():
					h.usecase.ReleaseBytes(int64(len(raw)))
					return
				}
			}
		}()
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-sub.Context().Done():
				return
			case <-heartbeat.C:
				if write([]byte(": heartbeat\n\n")) != nil {
					return
				}
			case p := <-messages:
				if p.err != nil {
					event := entities.MockEvent{Type: "failed", StreamID: streamID, ErrorCode: errs.CodeOf(p.err), ErrorMessage: errs.SafeMessageOf(p.err)}
					if p.err == io.EOF || errs.CodeOf(p.err) == "stream.stopped" {
						event.Type = "completed"
						event.ErrorCode = ""
						event.ErrorMessage = ""
					}
					raw, _ := json.Marshal(event)
					_ = write(append(append([]byte("data: "), raw...), []byte("\n\n")...))
					return
				}
				err := func() error {
					defer h.usecase.ReleaseBytes(int64(len(p.data)))
					return write(append(append([]byte("data: "), p.data...), []byte("\n\n")...))
				}()
				if err != nil {
					return
				}
			}
		}
	})
	return nil
}
