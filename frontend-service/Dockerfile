# Stage 1: Build SvelteKit application
FROM node:20-alpine AS builder

WORKDIR /app

# Copy dependency files first
COPY package*.json ./

# Install dependencies (including devDependencies for building)
RUN npm ci

# Copy configuration and source files
COPY svelte.config.js vite.config.js jsconfig.json ./
COPY src/ ./src/
COPY static/ ./static/

# Build application (SvelteKit outputs to ../build, which is /build because WORKDIR is /app)
RUN npm run build

# Stage 2: Production runtime
FROM node:20-alpine AS runner

WORKDIR /app

# Create a non-root user for security
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

# Copy production package.json and install production dependencies
COPY --from=builder /app/package*.json ./
RUN npm ci --omit=dev

# Copy build files frombuilder stage
COPY --from=builder /build ./build

# Set permissions for the non-root user
RUN chown -R appuser:appgroup /app

USER appuser

# Configure runtime port (default for adapter-node is 3000)
ENV PORT=3000
EXPOSE 3000

# Start SvelteKit server
CMD ["node", "build/index.js"]
