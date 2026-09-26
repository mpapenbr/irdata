package config

import "github.com/mpapenbr/irdata/auth"

var (
	EnableTelemetry   bool
	TelemetryEndpoint string
	LogConfig         string
	LogLevel          string
	OtelOutput        string // output for otel-logger (stdout, grpc)
	CacheDir          string
	NumWorkers        int
	IrAuthConfig      auth.AuthConfig
)
