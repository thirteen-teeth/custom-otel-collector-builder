# Custom OpenTelemetry Collector Builder

This project builds a custom OpenTelemetry Collector with a GELF (Graylog Extended Log Format) receiver. It provides a complete build environment with Docker support, version management, and testing utilities.

## Features

### Build System
- ✅ **Docker Support**: Multi-platform builds (linux/amd64, linux/arm64)
- ✅ **Version Management**: Synchronized IMAGE_TAG and git tags
- ✅ **Testing**: Automated GELF receiver testing
- ✅ **CI/CD Ready**: Complete build and release pipeline

## Quick Start

### Build the Collector
```bash
# Build the OpenTelemetry Collector with GELF receiver
make build-collector
```

### Run the Collector
```bash
# Start the collector locally
make run
```

### Test GELF Messages
```bash
# Send test GELF messages and view output
make test-gelf
```

### Build Docker Image
```bash
# Build multi-platform Docker image
make build
```

### Run in Docker
```bash
# Run the collector in Docker
make run-docker
```

## Configuration

The collector uses two main configuration files:

### Builder Configuration (`builder-config.yaml`)
Defines which components to include in the custom collector:

```yaml
dist:
  name: otelcol-custom
  description: Custom OpenTelemetry Collector with GELF receiver
  output_path: ./otelcol-custom
  otelcol_version: 0.128.0

receivers:
  - gomod: github.com/thirteen-teeth/otel-gelf-receiver v1.0.0
    path: ./gelfreceiver

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.128.0
  - gomod: go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.128.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/loggingexporter v0.128.0
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.128.0
```

### Collector Configuration (`collector-config.yaml`)
Defines how the collector processes data:

```yaml
receivers:
  gelf:
    listen_address: "0.0.0.0:12201"
    protocol: "both"
    use_compression: true

processors:
  batch:
  memory_limiter:
    limit_mib: 512

exporters:
  logging:
    loglevel: debug

service:
  pipelines:
    logs:
      receivers: [gelf]
      processors: [memory_limiter, batch]
      exporters: [logging]
```

## Available Commands

### Build Commands
```bash
make build-collector    # Build the collector binary
make build             # Build Docker image
make setup             # Setup Docker buildx
```

### Run Commands
```bash
make run              # Run collector locally
make run-docker       # Run collector in Docker
make test-gelf        # Test GELF receiver
```

### Maintenance Commands
```bash
make clean            # Clean build artifacts
make help             # Show all available commands
```

## Docker Usage

### Build Docker Image
```bash
make build
```

### Run in Docker
```bash
make run-docker
```

### Custom Configuration
Mount your own configuration files:
```bash
docker run -it --rm \
  -p 4317:4317 -p 4318:4318 -p 12201:12201/udp -p 12201:12201/tcp \
  -v $(pwd)/my-config.yaml:/otelcol/collector-config.yaml \
  custom-otel-collector:1.0.8
```

## Project Structure

```
├── builder-config.yaml      # Collector builder configuration
├── collector-config.yaml    # Runtime collector configuration  
├── Dockerfile              # Multi-stage Docker build
├── Makefile                # Build automation
├── version.sh              # Version management script
├── test-gelf.sh            # GELF testing script
└── gelfreceiver/           # GELF receiver module (git submodule)
```
```

## Testing

### Test GELF Receiver
```bash
# Run automated tests
make test-gelf
```

### Manual Testing
```bash
# Start collector in background
make run &

# Send test UDP message
echo -n '{"version": "1.1", "host": "test-server", "short_message": "Test message"}' | nc -u localhost 12201

# Send test TCP message  
echo -n -e '{"version": "1.1", "host": "test-server", "short_message": "Test message"}'"\0" | nc localhost 12201
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```
   Error: bind: address already in use
   ```
   - Solution: Change port in `collector-config.yaml` or stop conflicting service

2. **Build Failures**
   ```
   Error: failed to build collector
   ```
   - Solution: Ensure Go 1.24.4+ is installed and `gelfreceiver` module is present

3. **Docker Build Issues**
   - Run `make setup` to configure Docker buildx
   - Check Docker daemon is running

### Debug Mode

Enable debug logging in `collector-config.yaml`:
```yaml
exporters:
  logging:
    loglevel: debug
```

## Version Management

This project includes a comprehensive version management system to keep the `IMAGE_TAG` in the Makefile synchronized with git tags.

### Quick Reference

```bash
# Check if IMAGE_TAG matches latest git tag
make check-sync

# Sync IMAGE_TAG with latest git tag
make sync-from-git

# Increment versions
make increment-patch    # 1.0.1 → 1.0.2
make increment-minor    # 1.0.1 → 1.1.0
make increment-major    # 1.0.1 → 2.0.0

# Release workflow
make quick-release      # Increment patch and release in one command
make release           # Create release with current IMAGE_TAG
```

### Version Script

The `version.sh` script provides additional utilities:

```bash
# Check current version status
./version.sh status

# Set IMAGE_TAG to specific version
./version.sh set 1.2.3

# Sync IMAGE_TAG with latest git tag
./version.sh sync-from-git

# Check if versions are in sync (exit 0 if sync)
./version.sh check

# Show what next versions would be
./version.sh next-patch
./version.sh next-minor
./version.sh next-major
```

### Recommended Workflow

1. **Check current status**: `./version.sh status`
2. **Make your changes**: Edit code, update configs, etc.
3. **Increment version**: `make increment-patch` (or minor/major as needed)
4. **Release**: `make release` (commits, tags, and pushes)

Or use the quick release command:
```bash
# Make changes, then:
make quick-release  # Increments patch version and releases
```

## License

This project is provided as-is for educational and development purposes.
