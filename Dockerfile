# Build Stage
FROM golang:1.24-alpine AS builder

# Set environment variables for go build (GOARCH is auto-detected)
ENV CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -ldflags="-w -s" -o bin/tsdb-query-gateway ./cmd

# Run Stage
FROM alpine:3.19

WORKDIR /app

# Install root certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D -u 1000 appuser

# Copy the binary from the builder stage
COPY --from=builder --chown=appuser:appuser /app/bin/tsdb-query-gateway /app/tsdb-query-gateway
USER appuser

EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/app/tsdb-query-gateway"]
