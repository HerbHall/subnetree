package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a configured Zap logger from Viper settings.
// Reads "logging.level" (debug, info, warn, error; default "info")
// and "logging.format" (json, console; default "json").
func NewLogger(v *viper.Viper) (*zap.Logger, error) {
	level := v.GetString("logging.level")
	format := v.GetString("logging.format")

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	var cfg zap.Config
	switch format {
	case "console":
		cfg = zap.NewDevelopmentConfig()
	case "json", "":
		cfg = zap.NewProductionConfig()
	default:
		return nil, fmt.Errorf("invalid log format %q: must be \"json\" or \"console\"", format)
	}

	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	return cfg.Build()
}
