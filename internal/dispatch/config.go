package dispatch

import "time"

// DispatchConfig holds configuration for the Dispatch module.
type DispatchConfig struct {
	GRPCAddr              string        `mapstructure:"grpc_addr"`
	AgentTimeout          time.Duration `mapstructure:"agent_timeout"`
	EnrollmentTokenExpiry time.Duration `mapstructure:"enrollment_token_expiry"`
}

// DefaultConfig returns the default Dispatch configuration.
func DefaultConfig() DispatchConfig {
	return DispatchConfig{
		GRPCAddr:              ":9090",
		AgentTimeout:          5 * time.Minute,
		EnrollmentTokenExpiry: 24 * time.Hour,
	}
}
