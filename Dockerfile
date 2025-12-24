# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=${VERSION}" -o pm-agents ./cmd/pm-agents

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates git

# Create non-root user
RUN adduser -D -u 1000 pmuser
USER pmuser

WORKDIR /home/pmuser

# Copy binary from builder
COPY --from=builder /app/pm-agents /usr/local/bin/pm-agents

ENTRYPOINT ["pm-agents"]
