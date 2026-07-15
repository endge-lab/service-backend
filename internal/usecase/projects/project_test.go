package projects

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/usecase/adapters"
)

func TestNormalizeAndValidateCreateProjectInput(t *testing.T) {
	t.Run("normalizes valid input", func(t *testing.T) {
		input := adapters.CreateProjectInput{
			Identity:    " demo-project ",
			DisplayName: " Demo Project ",
		}

		if err := normalizeAndValidateCreateProjectInput(&input); err != nil {
			t.Fatalf("normalizeAndValidateCreateProjectInput() error = %v", err)
		}
		if input.Identity != "demo-project" {
			t.Fatalf("identity = %q, want %q", input.Identity, "demo-project")
		}
		if input.DisplayName != "Demo Project" {
			t.Fatalf("display name = %q, want %q", input.DisplayName, "Demo Project")
		}
		if input.Meta == nil {
			t.Fatal("meta must be initialized")
		}
	})

	t.Run("rejects empty identity", func(t *testing.T) {
		input := adapters.CreateProjectInput{Identity: " ", DisplayName: "Demo Project"}

		if err := normalizeAndValidateCreateProjectInput(&input); err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("rejects empty display name", func(t *testing.T) {
		input := adapters.CreateProjectInput{Identity: "demo-project", DisplayName: " "}

		if err := normalizeAndValidateCreateProjectInput(&input); err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestNormalizeAndValidateUpdateProjectInput(t *testing.T) {
	input := adapters.UpdateProjectInput{
		Identity:    " demo-project ",
		DisplayName: " Demo Project ",
	}

	if err := normalizeAndValidateUpdateProjectInput(&input); err != nil {
		t.Fatalf("normalizeAndValidateUpdateProjectInput() error = %v", err)
	}
	if input.Identity != "demo-project" {
		t.Fatalf("identity = %q, want %q", input.Identity, "demo-project")
	}
	if input.DisplayName != "Demo Project" {
		t.Fatalf("display name = %q, want %q", input.DisplayName, "Demo Project")
	}
	if input.Meta == nil {
		t.Fatal("meta must be initialized")
	}
}
