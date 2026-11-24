ARG APP_DIR='/app'
ARG PLATFORM='image'
ARG VERSION='develop'

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

# App directory
ARG APP_DIR
ARG TARGETOS
ARG TARGETARCH
ARG PLATFORM
ARG VERSION
WORKDIR $APP_DIR

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/

# Build the application
RUN \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build \
    -ldflags "-X github.com/getmiroir/trading-service/internal/version.Platform=${PLATFORM} -X github.com/getmiroir/trading-service/internal/version.Version=${VERSION}" \
    -a \
    -installsuffix cgo \
    -o copy-service \
    ./cmd/copy-service

# Final stage
FROM alpine:latest

# App directory
ARG APP_DIR
WORKDIR $APP_DIR
ENV WAILS_APP_SETTINGS_FILE=$APP_DIR/settings.yaml

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build $APP_DIR/copy-service .

# Set the entrypoint
ENTRYPOINT ["./copy-service"]
