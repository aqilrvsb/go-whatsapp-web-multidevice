############################
# STEP 1 build executable binary
############################
FROM golang:1.23-alpine3.20 AS builder
RUN apk update && apk add --no-cache gcc musl-dev gcompat
WORKDIR /whatsapp
COPY ./src .

# Fetch dependencies.
RUN go mod download
# Build the binary with optimizations
RUN go build -a -ldflags="-w -s" -o /app/whatsapp

#############################
## STEP 2 build a smaller image
#############################
FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates
WORKDIR /app

# Copy compiled from builder.
COPY --from=builder /app/whatsapp /app/whatsapp

# Create necessary directories
RUN mkdir -p /app/storages /app/statics/qrcode /app/statics/media /app/statics/senditems

# Expose port (Railway will override this with PORT env var)
EXPOSE 3000

# Run the binary.
ENTRYPOINT ["/app/whatsapp"]
CMD [ "rest" ]