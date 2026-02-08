# syntax=docker/dockerfile:1


FROM golang:1.25-alpine AS builder
WORKDIR /app

# Install git (for go mod)
RUN apk add git

# Copy go mod and sum files first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the source code
COPY . .

# Build the worker binary
RUN go build -o worker ./cmd/worker

# --- Runtime image ---
FROM alpine:3.18
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/worker ./worker

# Set environment variables (can be overridden by docker-compose)
ENV GIN_MODE=release

# Run the worker
CMD ["./worker"]
