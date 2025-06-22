# Makefile for building and running a custom OpenTelemetry Collector Docker image with GELF receiver
IMAGE_NAME=custom-otel-collector
IMAGE_TAG=$(shell git describe --tags --abbrev=0 $(shell git rev-list --tags --max-count=1) 2>/dev/null || echo 1.0.0)
PLATFORMS=linux/amd64,linux/arm64
BUILDER=mybuilder

.PHONY: all setup build-collector build run clean test-gelf

all: build-collector

setup:
	@if ! docker buildx inspect $(BUILDER) >/dev/null 2>&1; then \
		docker buildx create --name $(BUILDER) --use; \
	else \
		docker buildx use $(BUILDER); \
	fi
	docker run --rm --privileged tonistiigi/binfmt --install all

# Build the OpenTelemetry Collector with GELF receiver
build-collector:
	@echo "Building OpenTelemetry Collector with GELF receiver..."
	cd gelfreceiver && go mod tidy
	cd otelcol-dev && env GOWORK=off go build -o otelcol-dev .
	@echo "✅ OpenTelemetry Collector with GELF receiver built successfully!"

# Run the collector
run:
	@echo "Starting OpenTelemetry Collector with GELF receiver..."
	./otelcol-dev/otelcol-dev --config=collector-config.yaml

# Test the GELF receiver with sample messages
test-gelf:
	@echo "Testing GELF receiver with sample messages..."
	@echo "Starting collector in background..."
	@./otelcol-dev/otelcol-dev --config=collector-config.yaml > collector-test.log 2>&1 &
	@echo $$! > collector.pid
	@sleep 3
	@echo "Sending test GELF messages..."
	@./test-gelf.sh
	@sleep 2
	@echo "Stopping collector..."
	@kill `cat collector.pid` 2>/dev/null || true
	@rm -f collector.pid
	@echo "✅ Test completed! Check collector-test.log for output"

build: setup
	docker buildx build --load \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--platform=$(PLATFORMS) .

run-docker:
	docker run -it --rm -p 4317:4317 -p 4318:4318 -p 12201:12201/udp -p 12201:12201/tcp \
		--name otelcol $(IMAGE_NAME):$(IMAGE_TAG)

clean:
	@rm -f collector-test.log collector-output.log collector.pid
	@rm -rf otelcol-dev/otelcol-dev
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true

commit:
	@if [ -z "$(m)" ]; then \
		echo "Error: Please provide a commit message with make commit m='your message'"; \
		exit 1; \
	fi
	git add .
	git commit -m "$(m)"

release:
	git add .
	git commit -m "Update image to $(IMAGE_TAG)"
	@current_tag=$(IMAGE_TAG); \
	major=$$(echo $$current_tag | cut -d. -f1); \
	minor=$$(echo $$current_tag | cut -d. -f2); \
	patch=$$(echo $$current_tag | cut -d. -f3); \
	if [ -z "$$major" ] || [ -z "$$minor" ] || [ -z "$$patch" ]; then \
		echo "Invalid tag format, skipping tag increment."; \
	else \
		new_patch=$$(($$patch + 1)); \
		new_tag="$$major.$$minor.$$new_patch"; \
		git tag -a $$new_tag -m "Release version $$new_tag"; \
		git push origin $$new_tag; \
	fi
	git push origin main
