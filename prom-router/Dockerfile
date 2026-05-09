# Build Stage
FROM golang:1.24-alpine AS builder

# Set environment variables for go build
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -ldflags="-w -s" -o bin/prom-router ./cmd/prom-router

# Run Stage
FROM alpine:3.19

WORKDIR /app

# Install root certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/bin/prom-router /app/prom-router

EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/app/prom-router"]
