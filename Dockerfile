FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the server and client applications
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/a2a-server ./cmd/a2a-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/a2a-client ./cmd/a2a-client

# Create a minimal image for the final applications
FROM alpine:3.18

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binaries from the builder stage
COPY --from=builder /app/bin/a2a-server /app/bin/a2a-server
COPY --from=builder /app/bin/a2a-client /app/bin/a2a-client

# Create directories for configuration and plugins
RUN mkdir -p /app/config /app/plugins

# Set the entrypoint to the server by default
ENTRYPOINT ["/app/bin/a2a-server"]

# Default command line arguments
CMD ["--config", "/app/config/server.json"]
