//go:build e2e

package e2e_test

import (
	"github.com/gofiber/fiber/v2"
	"net/http"
	"testing"
)

func TestMockUnavailableHTTPAndViewerPolicy(t *testing.T) {
	f := newAIAPIFixture(t)
	for _, route := range []struct {
		method, path string
		body         any
		status       int
	}{
		{http.MethodGet, "/capabilities", nil, 200},
		{http.MethodPost, "/generate", map[string]any{"schema": map[string]any{}, "generation": map[string]any{"count": 1}}, 503},
		{http.MethodPost, "/streams", map[string]any{"schema": map[string]any{}, "generation": map[string]any{}, "stream": map[string]any{}}, 503},
		{http.MethodGet, "/streams/missing", nil, 404},
		{http.MethodGet, "/streams/missing/events", nil, 404},
		{http.MethodPatch, "/streams/missing", map[string]any{"paused": true}, 404},
		{http.MethodPost, "/streams/missing/keepalive", nil, 404},
		{http.MethodDelete, "/streams/missing", nil, 404},
	} {
		path := "/api/v1/mock-data" + route.path
		anonymous := perform(t, f.app, route.method, path, route.body, nil)
		assertStatus(t, anonymous, fiber.StatusUnauthorized)
		anonymous.Body.Close()
		outsider := perform(t, f.app, route.method, path, route.body, f.outsider)
		assertStatus(t, outsider, fiber.StatusForbidden)
		outsider.Body.Close()
		viewer := perform(t, f.app, route.method, path, route.body, f.viewer)
		assertStatus(t, viewer, route.status)
		if route.path == "/capabilities" {
			body := decodeObject(t, viewer)
			if body["available"] != false || body["canRun"] != false {
				t.Fatal(body)
			}
		} else {
			viewer.Body.Close()
		}
	}
	version := perform(t, f.app, http.MethodGet, "/version", nil, nil)
	assertStatus(t, version, 200)
	body := decodeObject(t, version)
	services, ok := body["services"].([]any)
	if !ok || len(services) != 2 {
		t.Fatalf("known service metadata lost: %v", body)
	}
	found := false
	for _, raw := range services {
		item := raw.(map[string]any)
		if item["service"] == "service_mock_generator" {
			found = true
			if item["status"] != "unavailable" {
				t.Fatal(item)
			}
		}
	}
	if !found {
		t.Fatal("Mock row missing")
	}
	// Existing Workbench endpoint is still reachable while Mock is absent.
	response := perform(t, f.app, http.MethodGet, "/api/v1/ai/provider-adapters", nil, f.viewer)
	assertStatus(t, response, 200)
	response.Body.Close()
}
