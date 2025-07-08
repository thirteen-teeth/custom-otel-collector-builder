# Makefile for building and running a custom OpenTelemetry Collector Docker image with GELF receiver
IMAGE_NAME=custom-otel-collector
IMAGE_TAG=1.0.7
PLATFORMS=linux/amd64,linux/arm64
BUILDER=mybuilder

.PHONY: all setup build-collector build run clean test-gelf help

all: build-collector ## Build the OpenTelemetry Collector (default target)

help: ## Show this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Version Management:"
	@echo "  make check-sync              # Check if IMAGE_TAG matches git tag"
	@echo "  make sync-from-git           # Set IMAGE_TAG to match latest git tag"
	@echo "  make increment-patch         # Increment patch version (1.0.1 → 1.0.2)"
	@echo "  make increment-minor         # Increment minor version (1.0.1 → 1.1.0)"
	@echo "  make increment-major         # Increment major version (1.0.1 → 2.0.0)"
	@echo "  make quick-release           # Increment patch and release in one command"
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

check-sync: ## Check if IMAGE_TAG matches the latest git tag
	@latest_tag=$$(git describe --tags --abbrev=0 2>/dev/null || echo "none"); \
	if [ "$$latest_tag" = "none" ]; then \
		echo "❌ No git tags found. Current IMAGE_TAG: $(IMAGE_TAG)"; \
		exit 1; \
	elif [ "$$latest_tag" != "$(IMAGE_TAG)" ]; then \
		echo "❌ IMAGE_TAG ($(IMAGE_TAG)) does not match latest git tag ($$latest_tag)"; \
		exit 1; \
	else \
		echo "✅ IMAGE_TAG ($(IMAGE_TAG)) matches latest git tag ($$latest_tag)"; \
	fi

sync-from-git: ## Set IMAGE_TAG to match the latest git tag
	@latest_tag=$$(git describe --tags --abbrev=0 2>/dev/null || echo "none"); \
	if [ "$$latest_tag" = "none" ]; then \
		echo "❌ No git tags found. Cannot sync IMAGE_TAG."; \
		exit 1; \
	fi; \
	echo "Setting IMAGE_TAG to match latest git tag: $$latest_tag"; \
	sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$$latest_tag/" Makefile; \
	echo "✅ IMAGE_TAG updated to $$latest_tag"

increment-patch: ## Increment the patch version in IMAGE_TAG
	@current_tag=$(IMAGE_TAG); \
	major=$$(echo $$current_tag | cut -d. -f1); \
	minor=$$(echo $$current_tag | cut -d. -f2); \
	patch=$$(echo $$current_tag | cut -d. -f3); \
	if [ -z "$$major" ] || [ -z "$$minor" ] || [ -z "$$patch" ]; then \
		echo "❌ Invalid tag format: $$current_tag"; \
		exit 1; \
	fi; \
	new_patch=$$(($$patch + 1)); \
	new_tag="$$major.$$minor.$$new_patch"; \
	echo "Incrementing IMAGE_TAG from $$current_tag to $$new_tag"; \
	sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$$new_tag/" Makefile; \
	echo "✅ IMAGE_TAG updated to $$new_tag"

increment-minor: ## Increment the minor version in IMAGE_TAG and reset patch to 0
	@current_tag=$(IMAGE_TAG); \
	major=$$(echo $$current_tag | cut -d. -f1); \
	minor=$$(echo $$current_tag | cut -d. -f2); \
	patch=$$(echo $$current_tag | cut -d. -f3); \
	if [ -z "$$major" ] || [ -z "$$minor" ] || [ -z "$$patch" ]; then \
		echo "❌ Invalid tag format: $$current_tag"; \
		exit 1; \
	fi; \
	new_minor=$$(($$minor + 1)); \
	new_tag="$$major.$$new_minor.0"; \
	echo "Incrementing IMAGE_TAG from $$current_tag to $$new_tag"; \
	sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$$new_tag/" Makefile; \
	echo "✅ IMAGE_TAG updated to $$new_tag"

increment-major: ## Increment the major version in IMAGE_TAG and reset minor and patch to 0
	@current_tag=$(IMAGE_TAG); \
	major=$$(echo $$current_tag | cut -d. -f1); \
	minor=$$(echo $$current_tag | cut -d. -f2); \
	patch=$$(echo $$current_tag | cut -d. -f3); \
	if [ -z "$$major" ] || [ -z "$$minor" ] || [ -z "$$patch" ]; then \
		echo "❌ Invalid tag format: $$current_tag"; \
		exit 1; \
	fi; \
	new_major=$$(($$major + 1)); \
	new_tag="$$new_major.0.0"; \
	echo "Incrementing IMAGE_TAG from $$current_tag to $$new_tag"; \
	sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$$new_tag/" Makefile; \
	echo "✅ IMAGE_TAG updated to $$new_tag"

release: ## Create a release commit, tag, and push to origin
	@echo "Creating release for version $(IMAGE_TAG)..."
	@if git diff --quiet && git diff --cached --quiet; then \
		echo "❌ No changes to commit. Make your changes first."; \
		exit 1; \
	fi
	git add .
	git commit -m "Release version $(IMAGE_TAG)"
	git tag -a $(IMAGE_TAG) -m "Release version $(IMAGE_TAG)"
	git push origin $(IMAGE_TAG)
	git push origin main
	@echo "✅ Released version $(IMAGE_TAG) and pushed to origin"

quick-release: increment-patch release ## Increment patch version and release in one command
