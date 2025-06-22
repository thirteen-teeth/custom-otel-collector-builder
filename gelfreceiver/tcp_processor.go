package gelfreceiver

import (
	"bufio"
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

// TCPProcessor handles TCP GELF messages
type TCPProcessor struct {
	config   *Config
	consumer consumer.Logs
	logger   *zap.Logger
}

// NewTCPProcessor creates a new TCP processor
func NewTCPProcessor(config *Config, consumer consumer.Logs, logger *zap.Logger) *TCPProcessor {
	return &TCPProcessor{
		config:   config,
		consumer: consumer,
		logger:   logger,
	}
}

// Process handles a TCP connection and processes GELF messages
func (p *TCPProcessor) Process(ctx context.Context, conn net.Conn) error {
	// Set read timeout
	if err := conn.SetReadDeadline(time.Now().Add(p.config.ReadTimeout)); err != nil {
		p.logger.Warn("Failed to set read deadline", zap.Error(err))
	}

	scanner := bufio.NewScanner(conn)

	// Set maximum scan token size to our max message size
	buf := make([]byte, p.config.MaxMessageSize)
	scanner.Buffer(buf, p.config.MaxMessageSize)

	// Custom split function for null-terminated messages
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Look for null terminator
		if i := bytes.IndexByte(data, 0); i >= 0 {
			return i + 1, data[0:i], nil
		}

		// If we're at EOF, return whatever we have
		if atEOF {
			return len(data), data, nil
		}

		// Need more data
		return 0, nil, nil
	})

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			messageData := scanner.Bytes()
			if len(messageData) == 0 {
				continue
			}

			// Process the message
			if err := p.processMessage(ctx, messageData); err != nil {
				p.logger.Error("Failed to process TCP message",
					zap.Error(err),
					zap.String("remote_addr", conn.RemoteAddr().String()),
					zap.Int("message_size", len(messageData)))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// processMessage processes a single GELF message
func (p *TCPProcessor) processMessage(ctx context.Context, data []byte) error {
	// Try to decompress if it looks like GZIP
	var messageData []byte
	var err error

	if p.config.UseCompression && len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		// This looks like GZIP
		messageData, err = p.decompressGZIP(data)
		if err != nil {
			p.logger.Debug("Failed to decompress GZIP data, trying as plain text", zap.Error(err))
			messageData = data
		}
	} else {
		messageData = data
	}

	// Parse GELF message
	gelfMsg, err := p.parseGELFMessage(messageData)
	if err != nil {
		return err
	}

	// Convert to OpenTelemetry log format
	logs := p.convertToOTelLogs(gelfMsg)

	// Send to consumer
	return p.consumer.ConsumeLogs(ctx, logs)
}

// decompressGZIP decompresses GZIP data
func (p *TCPProcessor) decompressGZIP(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// parseGELFMessage parses a GELF JSON message
func (p *TCPProcessor) parseGELFMessage(data []byte) (map[string]interface{}, error) {
	var gelfMsg map[string]interface{}
	if err := json.Unmarshal(data, &gelfMsg); err != nil {
		return nil, err
	}

	// Validate required GELF fields
	if version, ok := gelfMsg["version"].(string); !ok || version == "" {
		return nil, fmt.Errorf("missing or invalid version field")
	}

	if host, ok := gelfMsg["host"].(string); !ok || host == "" {
		return nil, fmt.Errorf("missing or invalid host field")
	}

	if shortMessage, ok := gelfMsg["short_message"].(string); !ok || shortMessage == "" {
		return nil, fmt.Errorf("missing or invalid short_message field")
	}

	return gelfMsg, nil
}

// convertToOTelLogs converts GELF message to OpenTelemetry logs
func (p *TCPProcessor) convertToOTelLogs(gelfMsg map[string]interface{}) plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Set basic GELF fields
	if shortMessage, ok := gelfMsg["short_message"].(string); ok {
		logRecord.Body().SetStr(shortMessage)
	}

	// Set timestamp
	if timestamp, ok := gelfMsg["timestamp"].(float64); ok {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(int64(timestamp), 0)))
	} else {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	// Set severity level
	if level, ok := gelfMsg["level"].(float64); ok {
		logRecord.SetSeverityNumber(plog.SeverityNumber(level))
		logRecord.SetSeverityText(getSeverityText(int(level)))
	}

	// Set attributes
	attrs := logRecord.Attributes()

	// Standard GELF fields
	if host, ok := gelfMsg["host"].(string); ok {
		attrs.PutStr("gelf.host", host)
	}

	if version, ok := gelfMsg["version"].(string); ok {
		attrs.PutStr("gelf.version", version)
	}

	if fullMessage, ok := gelfMsg["full_message"].(string); ok {
		attrs.PutStr("gelf.full_message", fullMessage)
	}

	if facility, ok := gelfMsg["facility"].(string); ok {
		attrs.PutStr("gelf.facility", facility)
	}

	if line, ok := gelfMsg["line"].(float64); ok {
		attrs.PutInt("gelf.line", int64(line))
	}

	if file, ok := gelfMsg["file"].(string); ok {
		attrs.PutStr("gelf.file", file)
	}

	// Additional fields (those starting with _)
	for key, value := range gelfMsg {
		if len(key) > 1 && key[0] == '_' && key != "_id" {
			switch v := value.(type) {
			case string:
				attrs.PutStr("gelf."+key[1:], v)
			case float64:
				attrs.PutDouble("gelf."+key[1:], v)
			case bool:
				attrs.PutBool("gelf."+key[1:], v)
			default:
				// Convert to string as fallback
				attrs.PutStr("gelf."+key[1:], fmt.Sprintf("%v", v))
			}
		}
	}

	// Set resource attributes
	resourceAttrs := resourceLogs.Resource().Attributes()
	if host, ok := gelfMsg["host"].(string); ok {
		resourceAttrs.PutStr("host.name", host)
	}

	return logs
}

// getSeverityText converts numeric syslog level to text
func getSeverityText(level int) string {
	switch level {
	case 0:
		return "EMERGENCY"
	case 1:
		return "ALERT"
	case 2:
		return "CRITICAL"
	case 3:
		return "ERROR"
	case 4:
		return "WARNING"
	case 5:
		return "NOTICE"
	case 6:
		return "INFO"
	case 7:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}
