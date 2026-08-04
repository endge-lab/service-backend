package health

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"service-backend"`
	Version string `json:"version" example:"1.0.0"`
	Env     string `json:"env" example:"development"`
}

type VersionResponse struct {
	Service string `json:"service" example:"service-backend"`
	Version string `json:"version" example:"1.0.0"`
	Env     string `json:"env" example:"development"`
}
