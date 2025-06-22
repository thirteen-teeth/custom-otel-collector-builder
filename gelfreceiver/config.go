package gelfreceiver

import (
	"errors"
	"fmt"
	"net"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	// ListenAddress is the address to listen on (default: "0.0.0.0:12201")
	ListenAddress string `mapstructure:"listen_address"`

	// Protocol specifies the transport protocol: "tcp", "udp", or "both" (default: "both")
	Protocol string `mapstructure:"protocol"`

	// UseCompression enables GZIP compression support (default: true)
	UseCompression bool `mapstructure:"use_compression"`

	// MaxMessageSize sets the maximum size for incoming messages in bytes (default: 8192)
	MaxMessageSize int `mapstructure:"max_message_size"`

	// ReadTimeout sets the read timeout for TCP connections (default: 30s)
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// WriteTimeout sets the write timeout for TCP connections (default: 30s)
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// ChunkTimeout sets the timeout for UDP chunk reassembly (default: 5s)
	ChunkTimeout time.Duration `mapstructure:"chunk_timeout"`
}

var _ component.Config = (*Config)(nil)
var _ confmap.Unmarshaler = (*Config)(nil)

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.ListenAddress == "" {
		return errors.New("listen_address cannot be empty")
	}

	// Validate that the address can be parsed
	host, port, err := net.SplitHostPort(cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("invalid listen_address format: %w", err)
	}

	// Validate host
	if host != "" {
		if ip := net.ParseIP(host); ip == nil {
			// Try to resolve hostname
			if _, err := net.LookupHost(host); err != nil {
				return fmt.Errorf("invalid host in listen_address: %w", err)
			}
		}
	}

	// Validate port
	if port == "" {
		return errors.New("port must be specified in listen_address")
	}

	// Validate protocol
	switch cfg.Protocol {
	case "tcp", "udp", "both":
		// Valid protocols
	default:
		return fmt.Errorf("invalid protocol '%s': must be 'tcp', 'udp', or 'both'", cfg.Protocol)
	}

	// Validate message size
	if cfg.MaxMessageSize <= 0 {
		return errors.New("max_message_size must be greater than 0")
	}

	if cfg.MaxMessageSize > 65536 {
		return errors.New("max_message_size cannot exceed 65536 bytes (UDP limit)")
	}

	// Validate timeouts
	if cfg.ReadTimeout <= 0 {
		return errors.New("read_timeout must be greater than 0")
	}

	if cfg.WriteTimeout <= 0 {
		return errors.New("write_timeout must be greater than 0")
	}

	if cfg.ChunkTimeout <= 0 {
		return errors.New("chunk_timeout must be greater than 0")
	}

	if cfg.ChunkTimeout > 5*time.Second {
		return errors.New("chunk_timeout cannot exceed 5 seconds (GELF specification limit)")
	}

	return nil
}

// Unmarshal implements confmap.Unmarshaler to handle custom unmarshaling
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	// First unmarshal normally
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}

	// Handle duration parsing if they were provided as strings
	if timeoutStr := conf.Get("read_timeout"); timeoutStr != nil {
		if str, ok := timeoutStr.(string); ok {
			if duration, err := time.ParseDuration(str); err == nil {
				cfg.ReadTimeout = duration
			}
		}
	}

	if timeoutStr := conf.Get("write_timeout"); timeoutStr != nil {
		if str, ok := timeoutStr.(string); ok {
			if duration, err := time.ParseDuration(str); err == nil {
				cfg.WriteTimeout = duration
			}
		}
	}

	if timeoutStr := conf.Get("chunk_timeout"); timeoutStr != nil {
		if str, ok := timeoutStr.(string); ok {
			if duration, err := time.ParseDuration(str); err == nil {
				cfg.ChunkTimeout = duration
			}
		}
	}

	return nil
}
