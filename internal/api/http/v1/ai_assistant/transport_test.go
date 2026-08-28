package ai_assistant

import (
	"encoding/json"
	"testing"
)

func TestLegacyRunPayloadRemainsValidJSONContract(t *testing.T) {
	var request RunRequest
	if err := json.Unmarshal([]byte(`{"requestId":"19f831a0-9387-42e9-9ea1-51e393ac97bc","modelProfileId":"d6418b52-edfd-4af5-8769-64c771fb25c1","prompt":"hello"}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.InteractionID != "" || request.ReplyToClarificationID != "" || request.SelectedCandidateID != "" {
		t.Fatalf("legacy payload unexpectedly created linkage: %#v", request)
	}
}

func TestRunPayloadAcceptsClarificationLinkage(t *testing.T) {
	var request RunRequest
	if err := json.Unmarshal([]byte(`{"requestId":"19f831a0-9387-42e9-9ea1-51e393ac97bc","modelProfileId":"d6418b52-edfd-4af5-8769-64c771fb25c1","prompt":"Sample","interactionId":"6ba182d8-e5cf-46f0-823a-b5d66c8d2a13","replyToClarificationId":"303e1111-6848-4a6a-9f30-9406b630df47","selectedCandidateId":"candidate-1"}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.InteractionID == "" || request.ReplyToClarificationID == "" || request.SelectedCandidateID != "candidate-1" {
		t.Fatalf("clarification linkage was lost: %#v", request)
	}
}
