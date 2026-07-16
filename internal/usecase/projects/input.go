package projects

type CreateProjectInput struct {
	Identity    string
	DisplayName string
	Description *string
	Active      bool
	Meta        map[string]any
}

type UpdateProjectInput struct {
	Identity    string
	DisplayName string
	Description *string
	Active      bool
	Meta        map[string]any
}
