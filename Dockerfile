# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first (better layer caching)
COPY go.mod go.sum ./

# Allow Go to download the required toolchain version
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source code
COPY . .

# Generate templ templates and build
RUN go run github.com/a-h/templ/cmd/templ generate
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: Runtime (minimal image)
FROM alpine:3.21

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary and required assets
COPY --from=builder /app/server .
COPY --from=builder /app/data ./data
COPY --from=builder /app/internal/web/static ./internal/web/static

EXPOSE 8080

CMD ["./server"]
