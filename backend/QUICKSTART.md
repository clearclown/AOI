# AOI Backend Quick Start Guide

## Building the Agent

```bash
cd backend
go build -o aoi-agent ./cmd/aoi-agent
```

## Running with Default Configuration

```bash
./aoi-agent
```

This will start the agent with:
- ID: `default-agent`
- Role: `engineer`
- Listen Address: `0.0.0.0:8080`
- TLS: Disabled

## Running with Custom Configuration

### 1. Create a config file

```bash
cp aoi.config.example.json aoi.config.json
```

Edit `aoi.config.json` to customize:

```json
{
  "agent": {
    "id": "my-pm-agent",
    "role": "pm",
    "owner": "john-doe"
  },
  "network": {
    "listen_addr": "0.0.0.0:8080",
    "tls_enabled": false
  },
  "acl": {
    "rules": [
      {
        "agent_id": "trusted-agent",
        "resource": "/api/query",
        "permission": "read"
      }
    ]
  },
  "context": {
    "watch_paths": [".", "./docs"],
    "index_interval": "5m"
  }
}
```

### 2. Run with config file

```bash
./aoi-agent --config aoi.config.json
```

### 3. Override config with flags

```bash
./aoi-agent --config aoi.config.json --role engineer --addr 0.0.0.0:9090
```

## Testing the Agent

### Check Health (REST API)

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"ok"}
```

### List Agents (REST API)

```bash
curl http://localhost:8080/api/agents
```

### Query Agent (REST API - Legacy)

```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is the status?",
    "from_agent": "test-agent"
  }'
```

## Using JSON-RPC 2.0

### Discover Agents

```bash
curl -X POST http://localhost:8080/api/v1/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "aoi.discover",
    "id": 1
  }'
```

Expected response:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "agents": [...],
    "count": 1
  },
  "id": 1
}
```

### Query Agent (JSON-RPC)

```bash
curl -X POST http://localhost:8080/api/v1/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "aoi.query",
    "params": {
      "query": "What is the project status?",
      "from_agent": "pm-agent",
      "context_scope": "project",
      "metadata": {
        "priority": "high"
      }
    },
    "id": 2
  }'
```

Expected response:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "answer": "Engineer Summary: Technical analysis complete...",
    "confidence": 0.90,
    "sources": ["codebase", "technical_docs"],
    "metadata": {
      "priority": "high"
    }
  },
  "id": 2
}
```

### Execute Task

```bash
curl -X POST http://localhost:8080/api/v1/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "aoi.execute",
    "params": {
      "id": "task-123",
      "type": "analyze",
      "parameters": {
        "target": "src/",
        "depth": 3
      },
      "async": true,
      "timeout": 300
    },
    "id": 3
  }'
```

Expected response:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "task_id": "task-123",
    "status": "completed",
    "output": "Task executed successfully"
  },
  "id": 3
}
```

### Send Notification

```bash
curl -X POST http://localhost:8080/api/v1/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "aoi.notify",
    "params": {
      "type": "status_update",
      "from": "agent-1",
      "to": "agent-2",
      "message": "Build completed successfully",
      "data": {
        "build_id": "12345",
        "duration": "2m30s"
      }
    },
    "id": 4
  }'
```

### Get Agent Status

```bash
curl -X POST http://localhost:8080/api/v1/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "aoi.status",
    "params": {
      "agent_id": "optional-agent-id"
    },
    "id": 5
  }'
```

## Agent Roles

The agent behavior changes based on its role:

### PM Secretary (`role: "pm"`)
- Returns project status summaries
- Tracks progress and blockers
- Sources: `project_status.md`, `roadmap.md`
- Confidence: 0.85

### Engineer Secretary (`role: "engineer"`)
- Returns technical context summaries
- Provides codebase insights
- Sources: `codebase`, `technical_docs`
- Confidence: 0.90

### QA Secretary (`role: "qa"`)
- Returns test coverage metrics
- Provides quality assurance data
- Sources: `test_results`, `bug_reports`
- Confidence: 0.88

### Design Secretary (`role: "design"`)
- Returns UI/UX status
- Provides design system info
- Sources: `design_specs`, `ui_mockups`
- Confidence: 0.87

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests for specific package
go test ./internal/config -v
go test ./internal/notify -v
go test ./internal/protocol -v

# Run with coverage
go test ./... -cover
```

## Development

### Project Structure

```
backend/
├── cmd/aoi-agent/          # Main entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── notify/             # Notification system
│   ├── protocol/           # Transport (REST + JSON-RPC)
│   ├── secretary/          # Query processing
│   ├── acl/                # Access control
│   └── identity/           # Agent registry
└── pkg/aoi/                # Public types
```

### Adding a New JSON-RPC Method

1. Add handler in `internal/protocol/transport.go`:

```go
func (s *Server) handleMyMethod(w http.ResponseWriter, req *JSONRPCRequest) {
    var params MyParams
    if err := json.Unmarshal(req.Params, &params); err != nil {
        s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
        return
    }

    result := processMyMethod(params)
    s.sendJSONRPCSuccess(w, req.ID, result)
}
```

2. Register in `handleJSONRPC`:

```go
case "aoi.mymethod":
    s.handleMyMethod(w, &req)
```

3. Add tests in `internal/protocol/transport_test.go`

## Troubleshooting

### Port Already in Use

```bash
# Use a different port
./aoi-agent --addr 0.0.0.0:9090
```

### Config File Not Found

```bash
# Specify full path
./aoi-agent --config /path/to/aoi.config.json

# Or use defaults
./aoi-agent
```

### JSON-RPC Error Codes

- `-32700`: Parse error - Check JSON syntax
- `-32600`: Invalid request - Missing `jsonrpc: "2.0"`
- `-32601`: Method not found - Check method name
- `-32602`: Invalid params - Check parameter structure
- `-32603`: Internal error - Check server logs

## Next Steps

1. Review `/home/ablaze/Projects/AOI/backend/IMPLEMENTATION.md` for detailed documentation
2. Explore the example config: `aoi.config.example.json`
3. Run tests to understand behavior: `go test ./... -v`
4. Implement your custom query processing logic
5. Add custom JSON-RPC methods as needed

## Support

For issues or questions:
1. Check test files for usage examples
2. Review IMPLEMENTATION.md for architecture details
3. Examine transport.go for JSON-RPC implementation
