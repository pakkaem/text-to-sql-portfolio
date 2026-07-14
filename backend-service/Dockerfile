# Step 1: Build the Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install certificates and git
RUN apk add --no-cache ca-certificates git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Handle gitignored seed files gracefully
# If seed-data.sql doesn't exist (e.g., in CI or Railway git build), touch a blank file so COPY succeeds
RUN if [ -f seed-data.sql ]; then cp seed-data.sql /app/seed-data.sql; else touch /app/seed-data.sql; fi
RUN if [ -f seed-extra.sql ]; then cp seed-extra.sql /app/seed-extra.sql; else touch /app/seed-extra.sql; fi

# Compile statically (since modernc.org/sqlite is a pure-Go driver, CGO_ENABLED=0 is fully supported)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o backend .

# Step 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata curl

# Create a non-root user for security
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy binary & seeds
COPY --from=builder /app/backend /app/backend
COPY --from=builder /app/seed-data.sql /app/seed-data.sql
COPY --from=builder /app/seed-extra.sql /app/seed-extra.sql

# Set correct file ownership for runtime
RUN chown -R appuser:appgroup /app

USER appuser

# Expose port (default 8080)
EXPOSE 8080

# Run the app
CMD ["/app/backend"]
