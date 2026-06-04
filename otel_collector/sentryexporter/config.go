package sentryexporter

import "fmt"

// Config holds the configuration for the Sentry exporter.
type Config struct {
	// DSN is the Sentry Data Source Name for your project.
	DSN string `mapstructure:"dsn"`
}

func (c *Config) Validate() error {
	if c.DSN == "" {
		return fmt.Errorf("dsn is required")
	}
	return nil
}
