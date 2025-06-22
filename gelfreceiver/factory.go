package gelfreceiver

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	// typeStr is the unique identifier for the GELF receiver
	typeStr = "gelf"

	// stability level for this receiver
	stability = component.StabilityLevelAlpha
)

var (
	componentType = component.MustNewType(typeStr)
)

// NewFactory creates a factory for the GELF receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		componentType,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, stability),
	)
}

// createDefaultConfig creates the default configuration for the GELF receiver
func createDefaultConfig() component.Config {
	return &Config{
		ListenAddress:  "0.0.0.0:12201",
		Protocol:       "both", // Support both TCP and UDP by default
		UseCompression: true,   // Enable GZIP compression by default
		MaxMessageSize: 8192,   // Default to 8192 bytes (common Graylog limit)
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		ChunkTimeout:   5 * time.Second, // GELF spec maximum
	}
}

// createLogsReceiver creates a logs receiver based on provided config
func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	if consumer == nil {
		return nil, errors.New("nil next consumer")
	}

	gelfCfg := cfg.(*Config)

	// Create the GELF receiver instance
	r := &gelfReceiver{
		config:   gelfCfg,
		settings: set,
		consumer: consumer,
	}

	return r, nil
}
