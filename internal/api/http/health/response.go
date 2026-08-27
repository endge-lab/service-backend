package health

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"service-backend"`
	Version string `json:"version" example:"1.0.0"`
	Env     string `json:"env" example:"development"`
}

type VersionResponse struct {
	Service  string                     `json:"service" example:"service-backend"`
	Version  string                     `json:"version" example:"1.0.0"`
	Env      string                     `json:"env" example:"development"`
	Services []ConnectedServiceResponse `json:"services"`
}

type ConnectedServiceResponse struct {
	Service string `json:"service" example:"service_ai_workbench"`
	Version string `json:"version,omitempty" example:"0.4.0"`
	Env     string `json:"env,omitempty" example:"development"`
	Status  string `json:"status" enums:"available,unavailable" example:"available"`
}
