# Multi-stage Dockerfile for Isolate Panel
# Stage 1: Go builder
FROM golang:1.23-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Copy Go module files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Stage 2: Node.js builder
FROM node:22-alpine AS node-builder

WORKDIR /app

# Copy package files
COPY frontend/package*.json ./
RUN npm install

# Stage 3: Runtime
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite \
    supervisor \
    go \
    nodejs \
    npm

# Copy Go toolchain from go-builder
COPY --from=go-builder /usr/local/go /usr/local/go
COPY --from=go-builder /go/bin/air /usr/local/bin/air

# Set Go environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"

# Create necessary directories
RUN mkdir -p /app/backend /app/frontend /app/data /var/log/supervisor

# Copy supervisord config
COPY docker/supervisord.conf /etc/supervisord.conf

# Expose ports
EXPOSE 8080 5173

# Start supervisord
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
