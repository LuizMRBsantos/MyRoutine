# ─── Build stage ─────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install security updates
RUN apk update && apk upgrade && apk add --no-cache ca-certificates git

WORKDIR /app

# Download dependencies first (layer cache optimization)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.Version=$(git describe --tags --always)" \
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
