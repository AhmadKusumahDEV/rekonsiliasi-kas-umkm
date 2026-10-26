package models

type LivenessResponse struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Time          string `json:"time"`
}

type ReadinessResponse struct {
	Status  string                 `json:"status"`
	Service string                 `json:"service"`
	Version string                 `json:"version"`
	Checks  map[string]HealthCheck `json:"checks"`
}

type HealthCheck struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Message   string `json:"message,omitempty"`
}
