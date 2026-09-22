package mock_data

import "encoding/json"

type RelationSource struct {
	Path     string `json:"path"`
	UniqueBy string `json:"uniqueBy,omitempty"`
}
type RelationTarget struct {
	Path string `json:"path"`
}
type RelationMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}
type Relation struct {
	ID        string            `json:"id"`
	Source    RelationSource    `json:"source"`
	Target    RelationTarget    `json:"target"`
	Selection string            `json:"selection" enums:"single,random,round-robin"`
	Mappings  []RelationMapping `json:"mappings"`
}
type StreamGenerationOptions struct {
	Seed                        *string    `json:"seed,omitempty"`
	Profile                     string     `json:"profile,omitempty"`
	OptionalPropertyProbability *float64   `json:"optionalPropertyProbability,omitempty"`
	DefaultArrayLength          *int       `json:"defaultArrayLength,omitempty"`
	Relations                   []Relation `json:"relations,omitempty"`
}
type GenerationOptions struct {
	StreamGenerationOptions
	Count int `json:"count" minimum:"1" maximum:"1000"`
}
type GenerationRequest struct {
	Schema     json.RawMessage   `json:"schema" swaggertype:"object"`
	Generation GenerationOptions `json:"generation"`
}
type StreamParameters struct {
	IntervalMs      int   `json:"intervalMs" minimum:"100" maximum:"60000"`
	ItemsPerMessage int   `json:"itemsPerMessage" minimum:"1" maximum:"100"`
	EmitImmediately *bool `json:"emitImmediately,omitempty"`
}
type StreamRequest struct {
	Schema     json.RawMessage         `json:"schema" swaggertype:"object"`
	Generation StreamGenerationOptions `json:"generation"`
	Stream     StreamParameters        `json:"stream"`
}
type StreamPatch struct {
	IntervalMs      *int  `json:"intervalMs,omitempty"`
	ItemsPerMessage *int  `json:"itemsPerMessage,omitempty"`
	Paused          *bool `json:"paused,omitempty"`
}
type GenerationMetadata struct {
	Count   int    `json:"count"`
	Seed    string `json:"seed"`
	Profile string `json:"profile"`
}
type GenerationResponse struct {
	Items []json.RawMessage  `json:"items" swaggertype:"array,object"`
	Meta  GenerationMetadata `json:"meta"`
}
