# ─── Build stage ─────────────────────────────────────────────
# Runs on the build machine's own architecture; Go cross-compiles for the
# target (e.g. arm64 for AWS Graviton) without emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

# Install security updates
RUN apk update && apk upgrade && apk add --no-cache ca-certificates

WORKDIR /app

# Download dependencies first (layer cache optimization)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# TARGETARCH is filled in by Docker (amd64, arm64...). VERSION is the git
# commit, passed by CI: docker build --build-arg VERSION=$(git rev-parse --short HEAD)
ARG TARGETARCH
ARG VERSION=dev

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -trimpath \
    -ldflags="-w -s -X github.com/myroutine/backend/internal/buildinfo.Version=${VERSION}" \
    -o /app/bin/api ./cmd/api

# ─── Final stage — minimal image ─────────────────────────────
# Using scratch (empty image) — ~5MB final size, no shell = more secure
FROM scratch

# Copy SSL certificates for HTTPS calls (Claude API etc.)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /app/bin/api /api

# Run as non-root (numeric UID for scratch)
USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/api"]
