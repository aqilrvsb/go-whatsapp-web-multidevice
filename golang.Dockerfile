# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /build/src
RUN go build -a -ldflags="-w -s" -o /app/whatsapp

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/whatsapp .

# Copy static files if any
COPY --from=builder /build/src/views ./views
COPY --from=builder /build/src/public ./public

# Create necessary directories
RUN mkdir -p /app/storages /app/sessions && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 3000

# Run the application
CMD ["./whatsapp"]
