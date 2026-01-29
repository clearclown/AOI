# Docker Player Task

## Mission
You are the Docker Player. Your mission is to create **Docker/Podman compatible containerization** for the AOI project.

## Current Context
- Project: AOI (Agent Operational Interconnect)
- Working Directory: /home/ablaze/Projects/AOI
- Backend: Go application at /home/ablaze/Projects/AOI/backend
- Frontend: React application at /home/ablaze/Projects/AOI/frontend

## Your Deliverables

### 1. File Structure

```
/home/ablaze/Projects/AOI/
├── docker-compose.yml
├── backend/
│   └── Dockerfile
└── frontend/
    └── Dockerfile
```

### 2. Backend Dockerfile

Create `/home/ablaze/Projects/AOI/backend/Dockerfile`:

```dockerfile
# Use official Go image - Podman compatible
FROM docker.io/library/golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/aoi-agent ./cmd/aoi-agent

# Final stage - minimal image
FROM docker.io/library/alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/aoi-agent .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./aoi-agent"]
```

**Key Points**:
- Multi-stage build for smaller image
- Use `docker.io/library/` prefix (Podman compatible)
- CGO_ENABLED=0 for static binary
- Health check on /health endpoint
- Alpine for minimal footprint

### 3. Frontend Dockerfile

Create `/home/ablaze/Projects/AOI/frontend/Dockerfile`:

```dockerfile
# Build stage - Podman compatible
FROM docker.io/library/node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build for production
RUN npm run build

# Production stage - nginx
FROM docker.io/library/nginx:alpine

# Copy built files to nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx config (optional, create if needed)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
```

### 4. Frontend Nginx Config

Create `/home/ablaze/Projects/AOI/frontend/nginx.conf`:

```nginx
server {
    listen 80;
    server_name localhost;

    root /usr/share/nginx/html;
    index index.html;

    # Serve static files
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API calls to backend
    location /api {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
```

### 5. Docker Compose File

Create `/home/ablaze/Projects/AOI/docker-compose.yml`:

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
      - AOI_LOG_LEVEL=info
      - AOI_PORT=8080
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
    networks:
      - aoi-network
    restart: unless-stopped

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
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost/"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
    networks:
      - aoi-network
    restart: unless-stopped

networks:
  aoi-network:
    driver: bridge
```

**Key Points**:
- Version 3.8 (good compatibility)
- Health checks with depends_on
- Bridge network for inter-service communication
- Restart policy for resilience
- Port mapping: 8080 (backend), 3000 (frontend)

### 6. Docker Ignore Files

Create `/home/ablaze/Projects/AOI/backend/.dockerignore`:

```
# Git
.git
.gitignore

# IDE
.vscode
.idea

# Build artifacts
bin/
*.exe
*.test

# Testing
*_test.go
coverage.*

# Documentation
README.md
docs/

# Docker
Dockerfile
.dockerignore
```

Create `/home/ablaze/Projects/AOI/frontend/.dockerignore`:

```
# Dependencies
node_modules
package-lock.json

# Build artifacts
dist
build

# Testing
coverage
*.test.tsx
*.test.ts
vitest.config.ts

# Git
.git
.gitignore

# IDE
.vscode
.idea

# Misc
README.md
.env
.env.local
```

### 7. Quality Gates

Before you declare completion, verify:

1. ✅ **Build Backend**:
   ```bash
   cd /home/ablaze/Projects/AOI
   docker compose build backend
   # OR for Podman:
   podman-compose build backend
   ```

2. ✅ **Build Frontend**:
   ```bash
   docker compose build frontend
   ```

3. ✅ **Start Services**:
   ```bash
   docker compose up -d
   # Wait 10 seconds for startup
   sleep 10
   ```

4. ✅ **Check Backend Health**:
   ```bash
   curl -f http://localhost:8080/health
   # Should return: {"status":"ok"}
   ```

5. ✅ **Check Frontend Health**:
   ```bash
   curl -f http://localhost:3000/
   # Should return HTML
   ```

6. ✅ **Check Service Status**:
   ```bash
   docker compose ps
   # Both services should show "healthy"
   ```

7. ✅ **Check Logs** (no critical errors):
   ```bash
   docker compose logs backend
   docker compose logs frontend
   ```

8. ✅ **Stop Services**:
   ```bash
   docker compose down
   ```

### 8. Troubleshooting

If builds fail:
- Check backend builds first: `cd backend && go build ./...`
- Check frontend builds first: `cd frontend && npm run build`
- Verify file paths in Dockerfiles
- Check for missing dependencies

If health checks fail:
- Verify backend /health endpoint works
- Check port mappings (8080, 3000)
- Review container logs
- Verify nginx config for frontend

### 9. File Checklist

Create these files:
1. `/home/ablaze/Projects/AOI/docker-compose.yml`
2. `/home/ablaze/Projects/AOI/backend/Dockerfile`
3. `/home/ablaze/Projects/AOI/backend/.dockerignore`
4. `/home/ablaze/Projects/AOI/frontend/Dockerfile`
5. `/home/ablaze/Projects/AOI/frontend/nginx.conf`
6. `/home/ablaze/Projects/AOI/frontend/.dockerignore`

### 10. Testing Commands

Create a simple test script `/home/ablaze/Projects/AOI/test-docker.sh`:

```bash
#!/bin/bash
set -e

echo "==> Building images..."
docker compose build

echo "==> Starting services..."
docker compose up -d

echo "==> Waiting for services to be healthy..."
sleep 15

echo "==> Testing backend health..."
curl -f http://localhost:8080/health || (echo "Backend health check failed" && exit 1)

echo "==> Testing frontend..."
curl -f http://localhost:3000/ || (echo "Frontend health check failed" && exit 1)

echo "==> Checking service status..."
docker compose ps

echo "==> All tests passed!"
echo "==> Cleaning up..."
docker compose down

echo "==> Done!"
```

Make it executable:
```bash
chmod +x /home/ablaze/Projects/AOI/test-docker.sh
```

## Success Criteria

You are DONE when:
- [ ] All 6 files created
- [ ] `docker compose build` succeeds
- [ ] `docker compose up -d` starts services
- [ ] Backend health check responds
- [ ] Frontend loads in browser
- [ ] Both services show "healthy" status
- [ ] Test script passes

## Dependencies

**WAIT** for Backend and Frontend players to complete before starting!

You need:
- ✅ Backend with working health endpoint
- ✅ Frontend with successful build
- ✅ Both projects have go.mod and package.json

## Notes
- Use `docker.io/library/` for Podman compatibility
- Multi-stage builds keep images small
- Health checks enable proper orchestration
- Alpine images are lightweight
- Nginx serves static files efficiently

## Start Here
1. WAIT for backend and frontend to complete
2. Create Dockerfiles for both services
3. Create docker-compose.yml
4. Test build process
5. Verify health checks
6. Report completion

Good luck! 🐳
