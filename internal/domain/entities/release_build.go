package entities

import "encoding/json"

// ReleaseBuildMetadata is an immutable account of effective build inputs, not a live profile.
type ReleaseBuildMetadata struct {
	Version         int                  `json:"version"`
	ProgramID       string               `json:"programId"`
	CompilerVersion string               `json:"compilerVersion"`
	Profile         *ReleaseBuildProfile `json:"profile,omitempty"`
	Runtime         string               `json:"runtime"`
	Scope           string               `json:"scope"`
	ContextMode     string               `json:"contextMode"`
	Context         json.RawMessage      `json:"context" swaggertype:"object"`
	IncludeAST      bool                 `json:"includeAst"`
	FileFormat      string               `json:"fileFormat"`
	SizeBytes       int64                `json:"sizeBytes"`
	Checksum        string               `json:"checksum"`
}

type ReleaseBuildProfile struct {
	Identity    string `json:"identity"`
	DisplayName string `json:"displayName"`
	Revision    int    `json:"revision"`
}
