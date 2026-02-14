package config

type IrAuth struct {
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	AuthFile     string
}

var (
	EnableTelemetry   bool
	TelemetryEndpoint string
	LogConfig         string
	LogLevel          string
	OtelOutput        string // output for otel-logger (stdout, grpc)
	IrAuthConfig      IrAuth
)
