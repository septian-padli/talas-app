FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies (git needed for some modules)
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
RUN go build -o main ./cmd/api

# Final Stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .
# We don't copy .env here because we use docker-compose env_file or environment

EXPOSE 3002

CMD ["./main"]
