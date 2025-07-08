# Makefile for building and running a custom OpenTelemetry Collector Docker image with GELF receiver
IMAGE_NAME=custom-otel-collector
IMAGE_TAG=1.0.5
PLATFORMS=linux/amd64,linux/arm64
BUILDER=mybuilder

.PHONY: all setup build-collector build run clean test-gelf help

all: build-collector ## Build the OpenTelemetry Collector (default target)

help: ## Show this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Usage examples:"
	@echo "  make help                    # Show this help"
	@echo "  make build-collector         # Build the collector"
	@echo "  make run                     # Run the collector locally"
	@echo "  make test-gelf               # Test GELF receiver"
	@echo "  make build                   # Build Docker image"
	@echo "  make commit m='message'      # Commit with message"
	@echo "  make release                 # Create release and push"

setup: ## Set up Docker buildx for multi-platform builds
	@if ! docker buildx inspect $(BUILDER) >/dev/null 2>&1; then \
		docker buildx create --name $(BUILDER) --use; \
	else \
		docker buildx use $(BUILDER); \
	fi
	docker run --rm --privileged tonistiigi/binfmt --install all

# Build the OpenTelemetry Collector with GELF receiver
build-collector: ## Build the OpenTelemetry Collector with GELF receiver
	@echo "Building OpenTelemetry Collector with GELF receiver..."
	cd gelfreceiver && go mod tidy
	cd otelcol-custom && env GOWORK=off go build -o otelcol-custom .
	@echo "✅ OpenTelemetry Collector with GELF receiver built successfully!"

# Run the collector
run: ## Run the OpenTelemetry Collector locally
	@echo "Starting OpenTelemetry Collector with GELF receiver..."
	./otelcol-custom/otelcol-custom --config=collector-config.yaml

# Test the GELF receiver with sample messages
test-gelf: ## Test the GELF receiver with sample messages
	@echo "Testing GELF receiver with sample messages..."
	@echo "Starting collector in background..."
	@./otelcol-custom/otelcol-custom --config=collector-config.yaml > collector-test.log 2>&1 &
	@echo $$! > collector.pid
	@sleep 3
	@echo "Sending test GELF messages..."
	@./test-gelf.sh
	@sleep 2
	@echo "Stopping collector..."
	@kill `cat collector.pid` 2>/dev/null || true
	@rm -f collector.pid
	@echo "✅ Test completed! Check collector-test.log for output"

build: setup ## Build Docker image for multiple platforms
	docker buildx build --load \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--platform=$(PLATFORMS) .

run-docker: ## Run the collector in a Docker container
	docker run -it --rm -p 4317:4317 -p 4318:4318 -p 12201:12201/udp -p 12201:12201/tcp \
		--name otelcol $(IMAGE_NAME):$(IMAGE_TAG)

clean: ## Clean up build artifacts and Docker images
	@rm -f collector-test.log collector-output.log collector.pid
	@rm -rf otelcol-custom/otelcol-custom
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true

commit: ## Commit changes with a message (use: make commit m='your message')
	@if [ -z "$(m)" ]; then \
		echo "Error: Please provide a commit message with make commit m='your message'"; \
		exit 1; \
	fi
	git add .
	git commit -m "$(m)"

increment-tag: ## Increment the patch version in IMAGE_TAG
	@if [ -z "$(IMAGE_TAG)" ]; then \
		echo "Error: IMAGE_TAG is not set. Please set it before running this target."; \
		exit 1; \
	fi
	@current_tag=$(IMAGE_TAG); \
	major=$$(echo $$current_tag | cut -d. -f1); \
	minor=$$(echo $$current_tag | cut -d. -f2); \
	patch=$$(echo $$current_tag | cut -d. -f3); \
	if [ -z "$$major" ] || [ -z "$$minor" ] || [ -z "$$patch" ]; then \
		echo "Invalid tag format, skipping tag increment."; \
	else \
		new_patch=$$(($$patch + 1)); \
		new_tag="$$major.$$minor.$$new_patch"; \
		echo "Incrementing tag from $$current_tag to $$new_tag"; \
		sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$$new_tag/" Makefile; \
		echo "New IMAGE_TAG is $$new_tag"; \
	fi

release: ## Create a release commit, tag, and push to origin
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
