package dispatch

import (
	"time"

	"github.com/HerbHall/subnetree/internal/ca"
)

// DispatchConfig holds configuration for the Dispatch module.
type DispatchConfig struct {
	GRPCAddr              string        `mapstructure:"grpc_addr"`
	AgentTimeout          time.Duration `mapstructure:"agent_timeout"`
	EnrollmentTokenExpiry time.Duration `mapstructure:"enrollment_token_expiry"`
	CAConfig              ca.Config     `mapstructure:"ca"`
}

// DefaultConfig returns the default Dispatch configuration.
// CA paths are empty by default; set them in config to enable mTLS cert issuance.
func DefaultConfig() DispatchConfig {
	return DispatchConfig{
		GRPCAddr:              ":9090",
		AgentTimeout:          5 * time.Minute,
		EnrollmentTokenExpiry: 24 * time.Hour,
		CAConfig: ca.Config{
			Validity:     ca.DefaultValidity,
			Organization: ca.DefaultOrganization,
		},
	}
}
