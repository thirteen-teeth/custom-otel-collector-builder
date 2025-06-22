package gelfreceiver

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// GELFMessage represents a GELF message structure
type GELFMessage struct {
	Version      string                 `json:"version"`
	Host         string                 `json:"host"`
	ShortMessage string                 `json:"short_message"`
	FullMessage  string                 `json:"full_message,omitempty"`
	Timestamp    float64                `json:"timestamp,omitempty"`
	Level        int                    `json:"level,omitempty"`
	Facility     string                 `json:"facility,omitempty"`
	Line         int                    `json:"line,omitempty"`
	File         string                 `json:"file,omitempty"`
	Additional   map[string]interface{} `json:"-"` // Will be populated during parsing
}

// UDPProcessor handles UDP GELF messages
type UDPProcessor struct {
	config   *Config
	consumer consumer.Logs
	logger   *zap.Logger
}

// NewUDPProcessor creates a new UDP processor
func NewUDPProcessor(config *Config, consumer consumer.Logs, logger *zap.Logger) *UDPProcessor {
	return &UDPProcessor{
		config:   config,
		consumer: consumer,
		logger:   logger,
	}
}

// Process handles a UDP packet and processes GELF messages
func (p *UDPProcessor) Process(ctx context.Context, data []byte, addr *net.UDPAddr, assembler *ChunkAssembler) error {
	// Check if this is a chunked message
	if len(data) >= 12 && data[0] == 0x1e && data[1] == 0x0f {
		// This is a chunked message
		complete, err := assembler.AddChunk(data)
		if err != nil {
			return fmt.Errorf("failed to process chunk: %w", err)
		}

		if complete == nil {
			// Message not complete yet
			return nil
		}

		// Use the complete message
		data = complete
	}

	// Process the message
	return p.processMessage(ctx, data)
}

// processMessage processes a single GELF message
func (p *UDPProcessor) processMessage(ctx context.Context, data []byte) error {
	// Parse GELF message
	gelfMsg, err := parseGELFMessage(data, p.config.UseCompression)
	if err != nil {
		return fmt.Errorf("failed to parse GELF message: %w", err)
	}

	// Convert to OpenTelemetry logs
	logs := convertGELFToOTelLogs(gelfMsg)

	// Send to consumer
	return p.consumer.ConsumeLogs(ctx, logs)
}

// parseGELFMessage parses a GELF message from raw data
func parseGELFMessage(data []byte, useCompression bool) (*GELFMessage, error) {
	var reader io.Reader = bytes.NewReader(data)

	// Try to decompress if compression is enabled
	if useCompression {
		// Try GZIP decompression
		if gzReader, err := gzip.NewReader(reader); err == nil {
			defer gzReader.Close()
			reader = gzReader
		} else {
			// If decompression fails, use raw data
			reader = bytes.NewReader(data)
		}
	}

	// Read the data
	jsonData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read message data: %w", err)
	}

	// Parse JSON
	var rawMsg map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawMsg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract standard fields
	msg := &GELFMessage{
		Additional: make(map[string]interface{}),
	}

	for key, value := range rawMsg {
		switch key {
		case "version":
			if v, ok := value.(string); ok {
				msg.Version = v
			}
		case "host":
			if v, ok := value.(string); ok {
				msg.Host = v
			}
		case "short_message":
			if v, ok := value.(string); ok {
				msg.ShortMessage = v
			}
		case "full_message":
			if v, ok := value.(string); ok {
				msg.FullMessage = v
			}
		case "timestamp":
			if v, ok := value.(float64); ok {
				msg.Timestamp = v
			}
		case "level":
			if v, ok := value.(float64); ok {
				msg.Level = int(v)
			}
		case "facility":
			if v, ok := value.(string); ok {
				msg.Facility = v
			}
		case "line":
			if v, ok := value.(float64); ok {
				msg.Line = int(v)
			}
		case "file":
			if v, ok := value.(string); ok {
				msg.File = v
			}
		default:
			// Additional fields (those starting with _)
			if len(key) > 1 && key[0] == '_' {
				msg.Additional[key[1:]] = value // Remove the _ prefix
			}
		}
	}

	// Validate required fields
	if msg.Version == "" {
		return nil, fmt.Errorf("missing required field: version")
	}
	if msg.Host == "" {
		return nil, fmt.Errorf("missing required field: host")
	}
	if msg.ShortMessage == "" {
		return nil, fmt.Errorf("missing required field: short_message")
	}

	return msg, nil
}

// convertGELFToOTelLogs converts a GELF message to OpenTelemetry logs
func convertGELFToOTelLogs(gelfMsg *GELFMessage) plog.Logs {
	logs := plog.NewLogs()

	// Create resource logs
	resourceLogs := logs.ResourceLogs().AppendEmpty()

	// Set resource attributes
	resourceAttrs := resourceLogs.Resource().Attributes()
	resourceAttrs.PutStr("host.name", gelfMsg.Host)

	if gelfMsg.Facility != "" {
		resourceAttrs.PutStr("facility", gelfMsg.Facility)
	}

	// Create scope logs
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("gelf-receiver")
	scopeLogs.Scope().SetVersion("1.0.0")

	// Create log record
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Set timestamp
	if gelfMsg.Timestamp > 0 {
		// Convert UNIX timestamp to nanoseconds
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(int64(gelfMsg.Timestamp), int64((gelfMsg.Timestamp-float64(int64(gelfMsg.Timestamp)))*1e9))))
	} else {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	// Set severity
	logRecord.SetSeverityNumber(convertGELFLevelToOTel(gelfMsg.Level))
	logRecord.SetSeverityText(getSeverityText(gelfMsg.Level))

	// Set body
	logRecord.Body().SetStr(gelfMsg.ShortMessage)

	// Set attributes
	attrs := logRecord.Attributes()

	if gelfMsg.FullMessage != "" {
		attrs.PutStr("full_message", gelfMsg.FullMessage)
	}

	if gelfMsg.File != "" {
		attrs.PutStr("source.file", gelfMsg.File)
	}

	if gelfMsg.Line > 0 {
		attrs.PutInt("source.line", int64(gelfMsg.Line))
	}

	// Add additional fields
	for key, value := range gelfMsg.Additional {
		switch v := value.(type) {
		case string:
			attrs.PutStr(key, v)
		case float64:
			attrs.PutDouble(key, v)
		case bool:
			attrs.PutBool(key, v)
		case int:
			attrs.PutInt(key, int64(v))
		case int64:
			attrs.PutInt(key, v)
		default:
			// Convert to string as fallback
			attrs.PutStr(key, fmt.Sprintf("%v", v))
		}
	}

	return logs
}

// convertGELFLevelToOTel converts GELF syslog levels to OpenTelemetry severity numbers
func convertGELFLevelToOTel(gelfLevel int) plog.SeverityNumber {
	switch gelfLevel {
	case 0: // Emergency
		return plog.SeverityNumberFatal4
	case 1: // Alert
		return plog.SeverityNumberFatal3
	case 2: // Critical
		return plog.SeverityNumberFatal2
	case 3: // Error
		return plog.SeverityNumberError
	case 4: // Warning
		return plog.SeverityNumberWarn
	case 5: // Notice
		return plog.SeverityNumberInfo2
	case 6: // Informational
		return plog.SeverityNumberInfo
	case 7: // Debug
		return plog.SeverityNumberDebug
	default:
		return plog.SeverityNumberInfo // Default to info
	}
}
