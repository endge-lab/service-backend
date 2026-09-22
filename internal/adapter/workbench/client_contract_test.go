package workbench

import (
	"testing"

	workbenchpb "github.com/endge-lab/service-backend/internal/adapter/workbenchpb"
)

func TestClarificationRequiredEventMapping(t *testing.T) {
	event := runEventFromProto(&workbenchpb.RunResponse{
		Type:          workbenchpb.RunEventType_RUN_EVENT_TYPE_CLARIFICATION_REQUIRED,
		InteractionId: "interaction-1",
		Clarification: &workbenchpb.Clarification{
			Id: "clarification-1", InteractionId: "interaction-1", TaskId: "task-1", Slot: "entity", Question: "Choose",
			Candidates: []*workbenchpb.ClarificationCandidate{{CandidateId: "candidate-1", DocumentType: "compositions", Identity: "sample", DisplayName: "Sample"}},
		},
	})
	if event.Type != "clarification_required" || event.InteractionID != "interaction-1" || event.Clarification == nil {
		t.Fatalf("unexpected event mapping: %#v", event)
	}
	if len(event.Clarification.Candidates) != 1 || event.Clarification.Candidates[0].CandidateID != "candidate-1" {
		t.Fatalf("unsafe or incomplete candidate mapping: %#v", event.Clarification)
	}
}
