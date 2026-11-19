package entities

import "time"

type Module struct {
	ID          string `json:"id"`
	Identity    string `json:"identity"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Revision    int64  `json:"revision"`
	UpdatedBy   string `json:"updatedBy"`
}

type ModuleManifest struct {
	Identity      string `json:"identity"`
	Version       string `json:"version"`
	FormatVersion int    `json:"formatVersion"`
	HostAPI       int    `json:"hostApi"`
}

type ModuleVersion struct {
	Version     string         `json:"version"`
	Manifest    ModuleManifest `json:"manifest"`
	SHA256      string         `json:"sha256"`
	HasCSS      bool           `json:"hasCss"`
	Size        int            `json:"size"`
	PublishedBy string         `json:"publishedBy"`
	PublishedAt time.Time      `json:"publishedAt"`
	Archive     []byte         `json:"-"`
}

type ModuleDetail struct {
	Module
	Versions []ModuleVersion `json:"versions"`
}
