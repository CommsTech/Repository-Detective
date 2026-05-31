# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gitea-bugbot .

# Final stage
FROM alpine:latest

# Install runtime dependencies and deterministic scanner tools
RUN apk --no-cache add ca-certificates tzdata wget curl bash shellcheck \
    && curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin \
    && curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.55.2 \
    && wget -qO- https://github.com/astral-sh/ruff/releases/download/v0.4.8/ruff-x86_64-unknown-linux-musl.tar.gz | tar xz -C /usr/local/bin ruff

# Create non-root user
RUN addgroup -g 1001 -S bugbot && \
    adduser -u 1001 -S bugbot -G bugbot

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/gitea-bugbot .

# Copy configuration
COPY --from=builder /app/config ./config

# Change ownership to non-root user
RUN chown -R bugbot:bugbot /app

# Switch to non-root user
USER bugbot

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./gitea-bugbot"]
