# Backend Player Task

## Mission
You are the Backend Player. Your mission is to create a **working Go backend** for the AOI protocol following **strict TDD** principles.

## Current Context
- Project: AOI (Agent Operational Interconnect)
- Working Directory: /home/ablaze/Projects/AOI
- Specifications:
  - `/home/ablaze/Projects/AOI/.aida/specs/aoi-protocol-requirements.md`
  - `/home/ablaze/Projects/AOI/.aida/specs/aoi-protocol-design.md`
  - `/home/ablaze/Projects/AOI/.aida/specs/aoi-protocol-tasks.md`

## Your Deliverables

### 1. Directory Structure
Create this structure in `/home/ablaze/Projects/AOI/backend/`:

```
backend/
├── cmd/
│   └── aoi-agent/
│       └── main.go
├── internal/
│   ├── identity/      # Tailscale identity (mock OK for MVP)
│   │   ├── identity.go
│   │   └── identity_test.go
│   ├── protocol/      # JSON-RPC protocol
│   │   ├── transport.go
│   │   ├── transport_test.go
│   │   ├── handlers.go
│   │   └── handlers_test.go
│   ├── secretary/     # Secretary agent logic
│   │   ├── secretary.go
│   │   └── secretary_test.go
│   ├── context/       # Context management
│   │   ├── monitor.go
│   │   └── monitor_test.go
│   └── acl/           # Access control
│       ├── acl.go
│       └── acl_test.go
├── pkg/
│   └── aoi/           # Public API types
│       ├── types.go
│       └── types_test.go
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 2. Core Components (MVP - Keep it Simple!)

#### A. Identity Layer (internal/identity/)
- Mock Tailscale integration
- Agent identity struct with ID, role, capabilities
- Simple agent registry (in-memory map)
- **TESTS FIRST**: Test agent registration, discovery

#### B. Protocol Layer (internal/protocol/)
- HTTP server listening on localhost:8080
- `/health` endpoint returning `{"status": "ok"}`
- JSON-RPC handler stub (just parse and echo for MVP)
- **TESTS FIRST**: Test health endpoint, JSON-RPC parsing

#### C. Secretary Agent (internal/secretary/)
- Secretary struct with identity and config
- Basic query handler (returns mock response)
- **TESTS FIRST**: Test secretary initialization, query handling

#### D. ACL Manager (internal/acl/)
- Simple rule struct (agent, resource, permission)
- CheckPermission() function
- **TESTS FIRST**: Test permission allow/deny logic

#### E. Context Monitor (internal/context/)
- Stub for future MCP integration
- Basic context snapshot struct
- **TESTS FIRST**: Test context snapshot creation

#### F. Public API (pkg/aoi/)
- Common types: AgentIdentity, Message, Query, Task
- **TESTS FIRST**: Test type marshaling/unmarshaling

### 3. TDD Protocol (MANDATORY!)

For EACH component:
1. **RED**: Write failing test first
2. **GREEN**: Write minimal code to pass
3. **REFACTOR**: Clean up while tests pass
4. **REPEAT**: For each function

Example:
```go
// Step 1: Write test FIRST (it will fail)
func TestAgentRegistry_Register(t *testing.T) {
    registry := NewAgentRegistry()
    agent := &AgentIdentity{ID: "test-agent", Role: "engineer"}

    err := registry.Register(agent)
    assert.NoError(t, err)

    found, err := registry.GetAgent("test-agent")
    assert.NoError(t, err)
    assert.Equal(t, "test-agent", found.ID)
}

// Step 2: Implement just enough to pass
type AgentRegistry struct {
    agents map[string]*AgentIdentity
}

func (r *AgentRegistry) Register(agent *AgentIdentity) error {
    r.agents[agent.ID] = agent
    return nil
}

// Step 3: Run test - it should pass now
// Step 4: Refactor if needed
```

### 4. Minimum Viable Implementation

**DO NOT over-engineer!** This is MVP. Focus on:
- ✅ Working server that starts
- ✅ Health endpoint returns 200 OK
- ✅ Basic agent registration
- ✅ Stub handlers for future features
- ✅ At least 5 test files
- ✅ All tests pass

**DO NOT implement** (save for later):
- ❌ Real Tailscale integration (use mock)
- ❌ Real MCP integration (use stub)
- ❌ Database persistence (use in-memory)
- ❌ Complex query parsing (return mock data)
- ❌ Real task execution (return success)

### 5. Quality Gates

Before you declare completion, verify:
1. ✅ `cd /home/ablaze/Projects/AOI/backend && go mod init github.com/aoi-protocol/aoi`
2. ✅ `go mod tidy`
3. ✅ `go build ./...` succeeds
4. ✅ `go test ./...` passes (minimum 5 test files)
5. ✅ `go run cmd/aoi-agent/main.go` starts server
6. ✅ `curl localhost:8080/health` returns `{"status":"ok"}`

### 6. File Checklist

Create these files (in order):
1. `backend/go.mod` - Module definition
2. `backend/pkg/aoi/types.go` - Core types
3. `backend/pkg/aoi/types_test.go` - Type tests
4. `backend/internal/identity/identity.go` - Identity logic
5. `backend/internal/identity/identity_test.go` - Identity tests
6. `backend/internal/acl/acl.go` - ACL logic
7. `backend/internal/acl/acl_test.go` - ACL tests
8. `backend/internal/protocol/transport.go` - HTTP server
9. `backend/internal/protocol/transport_test.go` - Transport tests
10. `backend/internal/secretary/secretary.go` - Secretary
11. `backend/internal/secretary/secretary_test.go` - Secretary tests
12. `backend/internal/context/monitor.go` - Context stub
13. `backend/internal/context/monitor_test.go` - Monitor tests
14. `backend/cmd/aoi-agent/main.go` - Entry point
15. `backend/Makefile` - Build automation
16. `backend/README.md` - Documentation

### 7. Example main.go

```go
package main

import (
    "log"
    "net/http"

    "github.com/aoi-protocol/aoi/internal/protocol"
)

func main() {
    log.Println("Starting AOI Agent...")

    server := protocol.NewServer()

    log.Println("Listening on :8080")
    if err := server.Start(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

### 8. Makefile

```makefile
.PHONY: build test run clean

build:
	go build -o bin/aoi-agent ./cmd/aoi-agent

test:
	go test -v ./...

run:
	go run ./cmd/aoi-agent

clean:
	rm -rf bin/
```

## Success Criteria

You are DONE when:
- [ ] All 16 files created
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes with 5+ test files
- [ ] Server starts and health check works
- [ ] Code is clean and well-commented
- [ ] README.md documents how to run

## Notes
- Use `testing` package for tests
- Use `net/http` for server
- Use `encoding/json` for JSON
- No external dependencies for MVP (standard library only)
- Keep it simple - this is a prototype!

## Start Here
1. Read the specs first
2. Create backend/ directory
3. Follow TDD: Test → Code → Test
4. Verify quality gates
5. Report completion

Good luck! 🚀
