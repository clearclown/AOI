# Use Podman-compatible official Go image
FROM docker.io/library/golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod ./
COPY go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o aoi-agent ./cmd/aoi-agent

# Use minimal base image for runtime
FROM docker.io/library/alpine:latest

# Install wget for health checks
RUN apk --no-cache add wget

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/aoi-agent .

# Expose port
EXPOSE 8080

# Set environment variables
ENV AOI_PORT=8080
ENV AOI_ROLE=engineer

# Run the application
CMD ["./aoi-agent", "-addr", "0.0.0.0:8080"]
