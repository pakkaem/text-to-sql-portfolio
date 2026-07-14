FROM python:3.11-slim

WORKDIR /app

# Install curl for health checking
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create a non-privileged user with a home directory
RUN useradd -u 10001 -m appuser

# Copy requirements file first to utilize Docker's cache layers
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application
COPY . .

# Change ownership of /app to the non-root user
RUN chown -R appuser:appuser /app

USER appuser

# Set default env variables (can be overridden at runtime)
ENV PORT=8000
ENV AI_MODEL_MODE=groq

# Expose the API port
EXPOSE 8000

# Start uvicorn server, binding to 0.0.0.0 and reading the PORT environment variable
CMD uvicorn app:app --host 0.0.0.0 --port ${PORT}
