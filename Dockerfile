# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /app/src
RUN go build -o /app/main .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy templates and static files
COPY --from=builder /app/src/views ./views
COPY --from=builder /app/src/public ./public

# Create directories for storage
RUN mkdir -p /app/storages /app/sessions

# Expose port
EXPOSE 3000

# Run the binary
CMD ["./main"]
