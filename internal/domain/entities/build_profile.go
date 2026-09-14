package entities

import (
	"encoding/json"
	"time"
)

const (
	BuildProfileVisibilityShared  = "shared"
	BuildProfileVisibilityPrivate = "private"
	BuildProfileSettingsVersion   = 1
)

type BuildProfileTopologyNode struct {
	Node    string `json:"node"`
	Runtime string `json:"runtime"`
}

type BuildProfileSettings struct {
	IncludeAST        bool                       `json:"includeAst" default:"false"`
	FileFormat        string                     `json:"fileFormat" enums:"gzip,json" default:"gzip"`
	BuildScope        string                     `json:"buildScope"`
	Contexts          string                     `json:"contexts"`
	Diagnostics       string                     `json:"diagnostics"`
	DebuggerStructure string                     `json:"debuggerStructure"`
	Topology          []BuildProfileTopologyNode `json:"topology"`
}

type BuildProfile struct {
	ID                  string          `json:"id"`
	Identity            string          `json:"identity"`
	WorkspaceID         string          `json:"-"`
	DisplayName         string          `json:"displayName"`
	Visibility          string          `json:"visibility"`
	OwnerUserID         string          `json:"-"`
	OwnerLogin          string          `json:"ownerLogin,omitempty"`
	SettingsVersion     int             `json:"settingsVersion"`
	Settings            json.RawMessage `json:"settings" swaggertype:"object"`
	Revision            int             `json:"revision"`
	OwnedByMe           bool            `json:"ownedByMe"`
	CanManage           bool            `json:"canManage"`
	CanChangeVisibility bool            `json:"canChangeVisibility"`
	CreatedBy           Actor           `json:"createdBy"`
	UpdatedBy           Actor           `json:"updatedBy"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
}
