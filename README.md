# Custom OpenTelemetry Collector with GELF Receiver

This project implements a custom OpenTelemetry Collector with a GELF (Graylog Extended Log Format) receiver. The GELF receiver allows you to collect logs from applications that send GELF messages via TCP or UDP and convert them to OpenTelemetry log format.

## Features

### GELF Receiver Features
- ✅ **Protocol Support**: TCP and UDP (configurable)
- ✅ **Default Port**: 12201 (GELF standard, configurable)
- ✅ **Compression**: GZIP compression support
- ✅ **Message Validation**: Validates required GELF fields
- ✅ **Chunking**: UDP chunk reassembly support (GELF specification)
- ✅ **Error Handling**: Comprehensive error logging
- ✅ **Performance**: Supports OpenTelemetry batch processor
- ✅ **Field Mapping**: Proper mapping from GELF to OpenTelemetry log format

### GELF Specification Compliance
- Supports GELF version 1.1
- Required fields: `version`, `host`, `short_message`
- Optional fields: `full_message`, `timestamp`, `level`, `facility`, `line`, `file`
- Additional fields (prefixed with `_`) are preserved as log attributes
- Syslog level mapping to OpenTelemetry severity levels

## Quick Start

### Build the Collector
```bash
# Build the OpenTelemetry Collector with GELF receiver
make build-collector
```

### Run the Collector
```bash
# Start the collector
make run
```

### Test GELF Messages
```bash
# Send test GELF messages and view output
make test-gelf
```

## Configuration

The GELF receiver supports the following configuration options in your `collector-config.yaml`:

```yaml
receivers:
  gelf:
    listen_address: "0.0.0.0:12201"    # Address to listen on
    protocol: "both"                   # "tcp", "udp", or "both" 
    use_compression: true              # Enable GZIP compression
    max_message_size: 8192             # Maximum message size in bytes
    read_timeout: "30s"                # TCP read timeout
    write_timeout: "30s"               # TCP write timeout  
    chunk_timeout: "5s"                # UDP chunk reassembly timeout
```

### Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `listen_address` | Network address to listen on | `0.0.0.0:12201` | No |
| `protocol` | Transport protocol: `tcp`, `udp`, or `both` | `both` | No |
| `use_compression` | Enable GZIP compression support | `true` | No |
| `max_message_size` | Maximum message size in bytes | `8192` | No |
| `read_timeout` | Read timeout for TCP connections | `30s` | No |
| `write_timeout` | Write timeout for TCP connections | `30s` | No |
| `chunk_timeout` | Timeout for UDP chunk reassembly | `5s` | No |

## Sending GELF Messages

### UDP Example
```bash
echo -n '{"version": "1.1", "host": "web-server", "short_message": "User login", "level": 6, "_user_id": 1234}' | nc -u localhost 12201
```

### TCP Example  
```bash
echo -n -e '{"version": "1.1", "host": "api-server", "short_message": "API request", "level": 6, "_endpoint": "/api/users"}'"\0" | nc localhost 12201
```

### GELF Message Format

Required fields:
- `version`: GELF spec version (e.g., "1.1")
- `host`: Source hostname/application
- `short_message`: Short descriptive message

Optional fields:
- `full_message`: Long message with details/backtrace
- `timestamp`: UNIX timestamp (with decimals for milliseconds)
- `level`: Syslog level (0-7)
- `facility`: Syslog facility
- `line`: Source code line number
- `file`: Source file name

Additional fields:
- Any field prefixed with `_` (except `_id`)
- Will be mapped to log attributes without the `_` prefix

### Syslog Level Mapping

| GELF Level | Syslog Level | OpenTelemetry Severity |
|------------|--------------|------------------------|
| 0 | EMERGENCY | FATAL4 |
| 1 | ALERT | FATAL3 |
| 2 | CRITICAL | FATAL2 |
| 3 | ERROR | ERROR |
| 4 | WARNING | WARN |
| 5 | NOTICE | INFO2 |
| 6 | INFO | INFO |
| 7 | DEBUG | DEBUG |

## Architecture

```
GELF Application → TCP/UDP:12201 → GELF Receiver → Batch Processor → Debug Exporter
```

### Components

1. **GELF Receiver** (`gelfreceiver/`)
   - `config.go`: Configuration structure and validation
   - `factory.go`: OpenTelemetry receiver factory
   - `receiver.go`: Main receiver logic with TCP/UDP listeners
   - `gelf.go`: GELF message parsing and processing
   - `chunks.go`: UDP chunk reassembly logic

2. **Collector Configuration** (`collector-config.yaml`)
   - Receiver, processor, and exporter pipeline configuration

3. **Builder Configuration** (`builder-config.yaml`)
   - OpenTelemetry Collector Builder configuration

## Example Output

When a GELF message is received, it's converted to OpenTelemetry format:

```
ResourceLog #0
Resource SchemaURL: 
Resource attributes:
     -> host.name: Str(web-server)
     -> facility: Str(auth)
ScopeLogs #0
InstrumentationScope gelf-receiver 1.0.0
LogRecord #0
Timestamp: 2025-06-22 21:16:46.188364917 +0000 UTC
SeverityText: INFO
SeverityNumber: Info(9)
Body: Str(User login successful)
Attributes:
     -> user_id: Double(1234)
     -> session_id: Str(abc123)
     -> full_message: Str(User authentication completed successfully)
```

## Commands

### Build Commands
```bash
# Build the collector
make build-collector

# Clean build artifacts
make clean
```

### Run Commands
```bash
# Run the collector
make run

# Test with sample GELF messages
make test-gelf
```

### Docker Commands
```bash
# Build Docker image
make build

# Run in Docker (with port mapping)
make run-docker
```

## Development

### Project Structure
```
custom-otel-collector-builder/
├── builder-config.yaml          # OCB configuration
├── collector-config.yaml        # Collector runtime configuration
├── gelfreceiver/               # GELF receiver implementation
│   ├── config.go              # Configuration
│   ├── factory.go             # Receiver factory
│   ├── receiver.go            # Main receiver logic
│   ├── gelf.go                # GELF processing
│   └── chunks.go              # UDP chunking
├── otelcol-dev/               # Generated collector binary
├── test-gelf.sh               # Test script
├── Dockerfile                 # Docker build
└── Makefile                  # Build automation
```

### Adding Features

To extend the GELF receiver:

1. **Configuration**: Add options to `config.go`
2. **Processing**: Modify logic in `gelf.go`
3. **Networking**: Update handlers in `receiver.go`
4. **Testing**: Add test cases to `test-gelf.sh`

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```
   Error: bind: address already in use
   ```
   - Solution: Change `listen_address` port or stop conflicting service

2. **Invalid GELF Message**
   ```
   Failed to parse GELF message: missing required field: version
   ```
   - Solution: Ensure GELF messages include required fields

3. **Compression Issues**
   - Set `use_compression: false` if having GZIP issues
   - Check sender GZIP implementation

### Debug Mode

Enable debug logging in `collector-config.yaml`:
```yaml
service:
  telemetry:
    logs:
      level: debug
```

## License

This project is provided as-is for educational and development purposes.
