package health

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
	Env     string `json:"env"`
}

type VersionResponse struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Env     string `json:"env"`
}
