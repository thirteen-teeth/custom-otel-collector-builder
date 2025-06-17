```
# Enable Docker multi-arch builds
docker run --rm --privileged tonistiigi/binfmt --install all
docker buildx create --name mybuilder --use

# Build the Docker image as Linux AMD and ARM,
# and loads the build result to "docker images"
docker buildx build --load \
  -t custom-otel-collector:1.0.0 \
  --platform=linux/amd64,linux/arm64 .

# Test the newly-built image
docker run -it --rm -p 4317:4317 -p 4318:4318 \
    --name otelcol custom-otel-collector:1.0.0
```
