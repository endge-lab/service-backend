package http

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestResponseRedactsSSEManualToken(t *testing.T) {
	token := "secret"
	result := response(&entities.RWorkspace{Configuration: entities.EndgeConfiguration{SSE: &entities.EndgeSSEConfiguration{ManualToken: &token}}})
	if result.Configuration.SSE == nil {
		t.Fatal("SSE must remain in response")
	}
	if result.Configuration.SSE.ManualToken != nil {
		t.Fatal("manual token must be redacted")
	}
}
