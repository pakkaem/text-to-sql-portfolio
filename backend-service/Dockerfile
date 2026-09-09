# Step 1: Build the Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install certificates and git
RUN apk add --no-cache ca-certificates git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code + seed data
COPY . .

# Compile statically
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o backend .

# Step 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata curl

# Create a non-root user for security
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy binary & seed
COPY --from=builder /app/backend /app/backend
COPY --from=builder /app/seed-data.sql /app/seed-data.sql

# Set correct file ownership for runtime
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["/app/backend"]