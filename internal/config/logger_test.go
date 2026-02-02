package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNewLogger_Defaults(t *testing.T) {
	v := viper.New()
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	logger, err := NewLogger(v)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_DebugLevel(t *testing.T) {
	v := viper.New()
	v.Set("logging.level", "debug")
	v.Set("logging.format", "json")

	logger, err := NewLogger(v)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_ConsoleFormat(t *testing.T) {
	v := viper.New()
	v.Set("logging.level", "warn")
	v.Set("logging.format", "console")

	logger, err := NewLogger(v)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	v := viper.New()
	v.Set("logging.level", "banana")
	v.Set("logging.format", "json")

	_, err := NewLogger(v)
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}

func TestNewLogger_InvalidFormat(t *testing.T) {
	v := viper.New()
	v.Set("logging.level", "info")
	v.Set("logging.format", "xml")

	_, err := NewLogger(v)
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}
