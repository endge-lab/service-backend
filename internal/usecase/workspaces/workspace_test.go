package workspaces

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type repositoryStub struct{ created, current *entities.RWorkspace }

var _ ports.WorkspacesRepository = (*repositoryStub)(nil)

func (s *repositoryStub) Create(_ context.Context, v *entities.RWorkspace) (*entities.RWorkspace, error) {
	copy := *v
	copy.ID = uuid.New()
	s.created = &copy
	return &copy, nil
}
func (s *repositoryStub) List(context.Context) ([]*entities.RWorkspace, error) { return nil, nil }
func (s *repositoryStub) GetByIdentity(context.Context, string) (*entities.RWorkspace, error) {
	return s.current, nil
}
func (s *repositoryStub) Update(_ context.Context, v *entities.RWorkspace) (*entities.RWorkspace, error) {
	copy := *v
	s.current = &copy
	return &copy, nil
}
func newService(r *repositoryStub) *Workspace {
	return NewWorkspaceService(WorkspaceParams{Repository: r, Observability: observability.NewCore(otel.Tracer("test"), zap.NewNop())})
}

func TestCreateUsesSystemDefaultConfiguration(t *testing.T) {
	r := &repositoryStub{}
	got, err := newService(r).Create(context.Background(), CreateWorkspaceInput{Identity: "default", DisplayName: "Default"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Configuration.DefaultLocale != "ru" || got.Configuration.DefaultTheme != "light" || len(got.Configuration.Locales) != 2 || len(got.Configuration.Themes) != 2 {
		t.Fatalf("unexpected default: %+v", got.Configuration)
	}
}
func TestCreateRejectsInvalidConfiguration(t *testing.T) {
	c := entities.DefaultEndgeConfiguration()
	c.DefaultTheme = "missing"
	_, err := newService(&repositoryStub{}).Create(context.Background(), CreateWorkspaceInput{Identity: "default", DisplayName: "Default", Configuration: &c})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateConfigurationInvariants(t *testing.T) {
	profile := "main-profile"
	tests := []struct {
		name   string
		change func(*entities.EndgeConfiguration)
	}{
		{name: "variable name is required", change: func(c *entities.EndgeConfiguration) {
			c.Vars = []entities.EndgeVariable{{DefaultValue: stringPtr("https://api.example.com")}}
		}},
		{name: "duplicate variable names", change: func(c *entities.EndgeConfiguration) {
			c.Vars = []entities.EndgeVariable{{Name: "API_URL"}, {Name: "API_URL"}}
		}},
		{name: "duplicate locale codes", change: func(c *entities.EndgeConfiguration) { c.Locales[1].Code = "ru" }},
		{name: "fallback locale does not exist", change: func(c *entities.EndgeConfiguration) { c.FallbackLocale = "de" }},
		{name: "invalid locale direction", change: func(c *entities.EndgeConfiguration) { c.Locales[0].Direction = "up" }},
		{name: "duplicate theme identities", change: func(c *entities.EndgeConfiguration) { c.Themes[1].Identity = "light" }},
		{name: "duplicate adapter ids", change: func(c *entities.EndgeConfiguration) { c.SFCAdapterIDs = []string{"native-vue", "native-vue"} }},
		{name: "default adapter does not exist", change: func(c *entities.EndgeConfiguration) { c.DefaultSFCAdapterID = "unknown" }},
		{name: "invalid sse mode", change: func(c *entities.EndgeConfiguration) { c.SSE = &entities.EndgeSSEConfiguration{AuthMode: "unexpected"} }},
		{name: "profile sse without profile identity", change: func(c *entities.EndgeConfiguration) {
			c.SSE = &entities.EndgeSSEConfiguration{AuthMode: entities.SSEAuthModeProfile}
		}},
		{name: "profile sse with profile identity", change: func(c *entities.EndgeConfiguration) {
			c.SSE = &entities.EndgeSSEConfiguration{AuthMode: entities.SSEAuthModeProfile, AuthProfileIdentity: &profile}
		}},
		{name: "manual sse", change: func(c *entities.EndgeConfiguration) {
			c.SSE = &entities.EndgeSSEConfiguration{AuthMode: entities.SSEAuthModeManual}
		}},
		{name: "none sse", change: func(c *entities.EndgeConfiguration) {
			c.SSE = &entities.EndgeSSEConfiguration{AuthMode: entities.SSEAuthModeNone}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := entities.DefaultEndgeConfiguration()
			test.change(&configuration)
			err := validateConfiguration(configuration)
			valid := test.name == "profile sse with profile identity" || test.name == "manual sse" || test.name == "none sse"
			if valid && err != nil {
				t.Fatalf("validateConfiguration() error = %v", err)
			}
			if !valid && err == nil {
				t.Fatal("validateConfiguration() error = nil, want validation error")
			}
		})
	}
}

func stringPtr(value string) *string { return &value }
func TestUpdateReplacesConfigurationWithoutMerge(t *testing.T) {
	old := entities.DefaultEndgeConfiguration()
	current := &entities.RWorkspace{ID: uuid.New(), Identity: "default", DisplayName: "Old", Configuration: old}
	r := &repositoryStub{current: current}
	next := entities.DefaultEndgeConfiguration()
	next.Vars = []entities.EndgeVariable{{Name: "API_URL"}}
	got, err := newService(r).Update(context.Background(), UpdateWorkspaceInput{Identity: "default", Configuration: &next})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Configuration.Vars) != 1 || got.Configuration.Vars[0].Name != "API_URL" {
		t.Fatalf("configuration was not replaced: %+v", got.Configuration.Vars)
	}
}
