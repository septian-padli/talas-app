# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git (for go mod)
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the worker binary
RUN go build -o worker ./cmd/worker

# --- Runtime image ---
FROM alpine:3.18
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/worker ./worker

# Copy any config files if needed (optional)
# COPY config.yaml ./

# Set environment variables (can be overridden by docker-compose)
ENV GIN_MODE=release

# Run the worker
CMD ["./worker"]
