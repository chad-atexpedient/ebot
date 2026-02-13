# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first (for better Docker layer caching)
COPY go.mod go.sum* ./
COPY apiclient/ ./apiclient/
COPY logger/ ./logger/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application — fail hard on errors, no silent fallbacks
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /ebot ./cmd/ebot

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies including curl for healthcheck
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary from builder
COPY --from=builder /ebot /app/ebot

# Copy any config files if they exist
COPY --from=builder /app/api/ /app/api/ 2>/dev/null || true

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Health check using curl (more reliable than wget)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -sf http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/ebot"]
CMD ["serve"]
