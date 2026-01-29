# TDD Evidence: Docker Deployment

## Feature: Docker Compose Setup
- Component: `docker-compose.yml`, `Dockerfile`s
- Date: 2026-01-28

## RED Phase
Verification test: Health check after container startup
```bash
docker compose up -d
curl -f http://localhost:8080/health
# Expected: {"status":"OK"}
# Result: FAIL - No docker-compose.yml exists
```

## GREEN Phase
Created `docker-compose.yml`:
```yaml
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    depends_on:
      backend:
        condition: service_healthy
```

Created `backend/Dockerfile` (multi-stage Go build):
```dockerfile
FROM docker.io/library/golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o aoi-agent ./cmd/aoi-agent

FROM docker.io/library/alpine:3.19
COPY --from=builder /app/aoi-agent /usr/local/bin/
CMD ["aoi-agent"]
```

Created `frontend/Dockerfile` (Node build + Nginx serve):
```dockerfile
FROM docker.io/library/node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM docker.io/library/nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
```

Result: PASS - `{"status":"OK"}`

## REFACTOR Phase
- Added nginx.conf for frontend routing
- Added .dockerignore files
- Added bridge network for service communication
- Added Podman-compatible image paths (docker.io/library/)
- Added health check interval/timeout configuration

## Verification
```
docker compose build  → PASS
docker compose up -d  → PASS
curl localhost:8080/health → {"status":"OK"}
docker compose down   → PASS
```
