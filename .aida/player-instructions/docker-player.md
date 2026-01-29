# Docker Player Instructions - AOI Protocol

## Your Mission
Create Docker deployment configuration for the AOI protocol backend and frontend.

## Working Directory
`/home/ablaze/Projects/AOI`

## Files to Create

### 1. docker-compose.yml (Root)
```yaml
version: '3.8'

services:
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: aoi-backend
    ports:
      - "8080:8080"
    environment:
      - AOI_PORT=8080
      - AOI_ROLE=engineer
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s
    networks:
      - aoi-network

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: aoi-frontend
    ports:
      - "3000:80"
    depends_on:
      backend:
        condition: service_healthy
    environment:
      - VITE_API_URL=http://backend:8080
    networks:
      - aoi-network

networks:
  aoi-network:
    driver: bridge
```

### 2. backend/Dockerfile
```dockerfile
# Use Podman-compatible official Go image
FROM docker.io/library/golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o aoi-agent ./cmd/aoi-agent

# Use minimal base image for runtime
FROM docker.io/library/alpine:latest

# Install curl for health checks
RUN apk --no-cache add curl

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
```

### 3. frontend/Dockerfile
```dockerfile
# Build stage
FROM docker.io/library/node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the app
RUN npm run build

# Production stage
FROM docker.io/library/nginx:alpine

# Copy built files from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

### 4. frontend/nginx.conf
```nginx
server {
    listen 80;
    server_name localhost;

    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API requests to backend
    location /api {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "OK\n";
        add_header Content-Type text/plain;
    }
}
```

### 5. .dockerignore (Root)
```
# Git
.git
.gitignore

# Node
node_modules
npm-debug.log

# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
vendor/

# IDE
.idea
.vscode
*.swp
*.swo

# Build outputs
dist
build
*.out

# OS
.DS_Store
Thumbs.db

# AIDA
.aida
```

## Implementation Steps

### Step 1: Create docker-compose.yml
```bash
cd /home/ablaze/Projects/AOI

cat > docker-compose.yml << 'EOF'
[paste docker-compose.yml content here]
EOF
```

### Step 2: Create Backend Dockerfile
```bash
cd /home/ablaze/Projects/AOI/backend

cat > Dockerfile << 'EOF'
[paste backend/Dockerfile content here]
EOF
```

### Step 3: Create Frontend Dockerfile
```bash
cd /home/ablaze/Projects/AOI/frontend

cat > Dockerfile << 'EOF'
[paste frontend/Dockerfile content here]
EOF

cat > nginx.conf << 'EOF'
[paste nginx.conf content here]
EOF
```

### Step 4: Create .dockerignore
```bash
cd /home/ablaze/Projects/AOI

cat > .dockerignore << 'EOF'
[paste .dockerignore content here]
EOF
```

## Testing the Docker Setup

### Test 1: Build Images
```bash
cd /home/ablaze/Projects/AOI

# Build both services
docker compose build

# Expected output:
# [+] Building 45.2s (25/25) FINISHED
# => [backend] ...
# => [frontend] ...
```

### Test 2: Start Services
```bash
# Start in detached mode
docker compose up -d

# Check status
docker compose ps

# Expected output:
# NAME              IMAGE               COMMAND                  SERVICE    STATUS
# aoi-backend       aoi-backend         "./aoi-agent -addr..."   backend    Up (healthy)
# aoi-frontend      aoi-frontend        "nginx -g 'daemon of..."  frontend   Up
```

### Test 3: Health Checks
```bash
# Test backend health
curl http://localhost:8080/health
# Expected: OK

# Test frontend
curl http://localhost:3000
# Expected: HTML content

# Check logs
docker compose logs backend
docker compose logs frontend
```

### Test 4: Cleanup
```bash
# Stop services
docker compose down

# Remove volumes (if any)
docker compose down -v
```

## Quality Gates (YOU MUST PASS)

### Gate 1: Build Succeeds
```bash
cd /home/ablaze/Projects/AOI
docker compose build
# Must exit with code 0
```

### Gate 2: Services Start
```bash
docker compose up -d
docker compose ps
# Both services must show "Up" status
# Backend must show "(healthy)" after health check
```

### Gate 3: Health Check Works
```bash
curl -f http://localhost:8080/health
# Must return 200 OK
```

### Gate 4: Frontend Accessible
```bash
curl -f http://localhost:3000
# Must return HTML (200 OK)
```

### Gate 5: Logs Clean
```bash
docker compose logs | grep -i error
# Should not show critical errors
```

## Troubleshooting Guide

### Issue: Build Fails for Backend
**Problem**: `go mod download` fails
**Solution**: Check that backend/go.mod exists
```bash
cd backend
go mod init aoi
go mod tidy
```

### Issue: Build Fails for Frontend
**Problem**: `npm ci` fails
**Solution**: Check that frontend/package.json exists
```bash
cd frontend
npm install
```

### Issue: Backend Container Exits
**Problem**: Binary crashes on startup
**Solution**: Check backend logs
```bash
docker compose logs backend
# Look for panic or error messages
```

### Issue: Health Check Fails
**Problem**: Health endpoint not responding
**Solution**: Check if backend is listening on correct port
```bash
docker compose exec backend ps aux
docker compose exec backend netstat -tlnp
```

### Issue: Frontend Shows 502
**Problem**: Can't connect to backend
**Solution**: Check network connectivity
```bash
docker compose exec frontend ping backend
docker compose exec frontend curl http://backend:8080/health
```

## Podman Compatibility Notes

If using Podman instead of Docker:
```bash
# Use podman-compose instead
podman-compose build
podman-compose up -d
podman-compose ps
```

All base images use `docker.io/library/` prefix for Podman compatibility.

## Completion Criteria
- [ ] docker-compose.yml created
- [ ] backend/Dockerfile created
- [ ] frontend/Dockerfile created
- [ ] frontend/nginx.conf created
- [ ] .dockerignore created
- [ ] All images build successfully
- [ ] All services start and stay healthy
- [ ] Health checks pass
- [ ] No critical errors in logs

## When You're Done
Respond with:
```
✅ Docker Deployment Complete

Build Results:
- Backend image: SUCCESS (size: ~XXX MB)
- Frontend image: SUCCESS (size: ~XXX MB)

Service Status:
- aoi-backend: Up (healthy)
- aoi-frontend: Up

Health Checks:
- Backend: PASS (http://localhost:8080/health → OK)
- Frontend: PASS (http://localhost:3000 → HTML)

Logs: No critical errors

Cleanup: Run `docker compose down` to stop services
```
