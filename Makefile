IMAGE_NAME=custom-otel-collector
IMAGE_TAG=$(shell git describe --tags --abbrev=0 $(shell git rev-list --tags --max-count=1) 2>/dev/null || echo 1.0.0)
PLATFORMS=linux/amd64,linux/arm64
BUILDER=mybuilder

.PHONY: all setup build run clean

all: build

setup:
	@if ! docker buildx inspect $(BUILDER) >/dev/null 2>&1; then \
		docker buildx create --name $(BUILDER) --use; \
	else \
		docker buildx use $(BUILDER); \
	fi
	docker run --rm --privileged tonistiigi/binfmt --install all

build: setup
	docker buildx build --load \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--platform=$(PLATFORMS) .

run:
	docker run -it --rm -p 4317:4317 -p 4318:4318 \
		--name otelcol $(IMAGE_NAME):$(IMAGE_TAG)

clean:
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true

commit:
	git add .
	git commit -m "Update image to $(IMAGE_TAG)"
	# Increment patch version
	@current_tag=$(IMAGE_TAG); \
	IFS='.' read -r major minor patch <<EOF $$current_tag; \
EOF
	if [ -z "$$major" ] || [ -z "$$minor" ] || [ -z "$$patch" ]; then \
		echo "Invalid tag format, skipping tag increment."; \
	else \
		new_patch=$$(($$patch + 1)); \
		new_tag="$$major.$$minor.$$new_patch"; \
		git tag -a $$new_tag -m "Release version $$new_tag"; \
		git push origin $$new_tag; \
	fi
	git push origin main


###