# =============================================================================
# Poly-Go Dockerfile — Multi-stage build
# =============================================================================

# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /poly .

# Stage 2: Runtime
FROM alpine:latest

# Git is needed for poly's git-related tools
RUN apk add --no-cache git ca-certificates

COPY --from=builder /poly /poly

ENTRYPOINT ["/poly"]
