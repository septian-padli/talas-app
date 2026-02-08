FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Install git (for go mod)
RUN apk add  git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the API binary
RUN go build -o api ./cmd/api
# Build Seeder binary
RUN go build -o seeder ./cmd/seeder

# --- Runtime image ---
FROM alpine:3.18
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/api ./api
COPY --from=builder /app/seeder ./seeder

# Copy any config files if needed (optional)
# COPY config.yaml ./

# Set environment variables (can be overridden by docker-compose)
ENV GIN_MODE=release

# Run the API
CMD ["./api"]
