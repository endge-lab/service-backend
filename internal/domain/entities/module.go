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
	Identity      string               `json:"identity"`
	Version       string               `json:"version"`
	FormatVersion int                  `json:"formatVersion"`
	HostAPI       int                  `json:"hostApi"`
	Contributes   *ModuleContributions `json:"contributes,omitempty"`
}

type ModuleContributions struct {
	Renderers []ModuleRendererDescriptor `json:"renderers,omitempty"`
}

type ModuleRendererDescriptor struct {
	Primitive string `json:"primitive"`
	ID        string `json:"id"`
	Label     string `json:"label"`
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
