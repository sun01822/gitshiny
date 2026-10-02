// Package config holds application configuration.
package config

import (
	"os"
	"strings"
)

// Config is the runtime configuration.
type Config struct {
	// Format is the default output format: text, json or csv.
	Format string
}

// Load reads configuration from the environment.
//
//	GITSHINY_FORMAT   default output format (text|json|csv)
func Load() Config {
	f := strings.ToLower(strings.TrimSpace(os.Getenv("GITSHINY_FORMAT")))
	if f == "" {
		f = "text"
	}
	return Config{Format: f}
}
