# Backend Player Instructions - AOI Protocol

## Your Mission
Create a working Go backend for the AOI (Agent Operational Interconnect) protocol following TDD principles.

## Working Directory
`/home/ablaze/Projects/AOI/backend`

## Required Structure
```
backend/
├── cmd/
│   └── aoi-agent/
│       └── main.go           # Entry point
├── internal/
│   ├── identity/             # Agent identity and registry
│   │   ├── identity.go
│   │   └── identity_test.go
│   ├── protocol/             # JSON-RPC protocol
│   │   ├── transport.go
│   │   └── transport_test.go
│   ├── secretary/            # Secretary agent logic
│   │   ├── secretary.go
│   │   └── secretary_test.go
│   └── acl/                  # Access control
│       ├── acl.go
│       └── acl_test.go
├── pkg/
│   └── aoi/                  # Public API types
│       ├── types.go
│       └── types_test.go
├── go.mod
├── go.sum
└── Makefile
```

## TDD Protocol (MANDATORY)
For EACH feature:
1. **RED**: Write failing test first
2. **GREEN**: Write minimal code to pass test
3. **REFACTOR**: Clean up while tests pass

Example workflow:
```bash
# 1. Write test
echo 'func TestHealthEndpoint(t *testing.T) { ... }' >> internal/protocol/transport_test.go

# 2. Run test (should fail)
go test ./internal/protocol -v

# 3. Implement feature
echo 'func HealthEndpoint() { ... }' >> internal/protocol/transport.go

# 4. Run test (should pass)
go test ./internal/protocol -v

# 5. Refactor if needed
```

## Core Components to Implement

### 1. Identity Management (internal/identity/)
**Types needed:**
- AgentIdentity struct (id, role, owner, capabilities)
- AgentRole enum (PM, Engineer)
- AgentRegistry interface

**Functions to implement:**
- NewRegistry() - create registry
- Register(identity AgentIdentity) - register agent
- GetAgent(id string) - fetch agent
- ListAgents() - list all agents

**Tests required (identity_test.go):**
- TestRegisterAgent - register should work
- TestGetAgent - fetch should return correct agent
- TestListAgents - list should return all agents
- TestDuplicateRegister - duplicate should error

### 2. JSON-RPC Protocol (internal/protocol/)
**Types needed:**
- JSONRPCRequest struct
- JSONRPCResponse struct
- Transport interface

**Functions to implement:**
- NewTransport(addr string) - create HTTP transport
- HandleRequest(method string, handler func) - register handler
- ServeHTTP(w, r) - HTTP handler
- HealthEndpoint() - return 200 OK

**Tests required (transport_test.go):**
- TestJSONRPCParsing - parse valid request
- TestJSONRPCValidation - reject invalid request
- TestHealthEndpoint - /health returns 200
- TestMethodRouting - route to correct handler

### 3. Secretary Agent (internal/secretary/)
**Types needed:**
- Secretary struct (identity, registry, handlers)
- QueryRequest struct
- QueryResponse struct

**Functions to implement:**
- NewSecretary(identity AgentIdentity) - create secretary
- HandleQuery(req QueryRequest) - handle query
- Start() - start agent
- Shutdown() - graceful shutdown

**Tests required (secretary_test.go):**
- TestSecretaryCreation - create secretary
- TestHandleQuery - handle simple query
- TestStartShutdown - lifecycle works

### 4. Access Control (internal/acl/)
**Types needed:**
- ACL struct
- Permission enum (None, Read, Write, Admin)
- Rule struct (agent, resource, permission)

**Functions to implement:**
- NewACL() - create ACL
- AddRule(rule Rule) - add permission rule
- CheckPermission(agent, resource, action) - check if allowed

**Tests required (acl_test.go):**
- TestAddRule - add rule works
- TestCheckPermission - permission check works
- TestDenyTakesPrecedence - deny overrides allow

### 5. Main Entry Point (cmd/aoi-agent/main.go)
```go
package main

import (
    "flag"
    "log"
    "net/http"

    "aoi/internal/identity"
    "aoi/internal/protocol"
    "aoi/internal/secretary"
)

func main() {
    addr := flag.String("addr", "0.0.0.0:8080", "Listen address")
    role := flag.String("role", "engineer", "Agent role (pm/engineer)")
    flag.Parse()

    // Create identity
    agentID := identity.AgentIdentity{
        ID:   "eng-local",
        Role: identity.RoleEngineer,
        Owner: "local-user",
    }

    // Create secretary
    sec := secretary.NewSecretary(agentID)

    // Create transport
    transport := protocol.NewTransport(*addr)

    // Register health endpoint
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // Start server
    log.Printf("Starting AOI agent on %s", *addr)
    log.Fatal(http.ListenAndServe(*addr, nil))
}
```

## Initialization Steps

### Step 1: Create Go Module
```bash
cd /home/ablaze/Projects/AOI
mkdir -p backend
cd backend
go mod init aoi
```

### Step 2: Create Directory Structure
```bash
mkdir -p cmd/aoi-agent
mkdir -p internal/{identity,protocol,secretary,acl}
mkdir -p pkg/aoi
```

### Step 3: Create First Test (TDD START)
```bash
# Create identity test first
cat > internal/identity/identity_test.go << 'EOF'
package identity

import "testing"

func TestRegisterAgent(t *testing.T) {
    registry := NewRegistry()
    agent := AgentIdentity{
        ID:   "eng-test",
        Role: RoleEngineer,
        Owner: "test-user",
    }

    err := registry.Register(agent)
    if err != nil {
        t.Fatalf("Register failed: %v", err)
    }

    retrieved, err := registry.GetAgent("eng-test")
    if err != nil {
        t.Fatalf("GetAgent failed: %v", err)
    }

    if retrieved.ID != agent.ID {
        t.Errorf("Expected ID %s, got %s", agent.ID, retrieved.ID)
    }
}
EOF

# Run test (should fail - no implementation yet)
go test ./internal/identity -v
```

### Step 4: Implement to Pass Test
```bash
# Create implementation
cat > internal/identity/identity.go << 'EOF'
package identity

import "errors"

type AgentRole string

const (
    RolePM       AgentRole = "pm"
    RoleEngineer AgentRole = "engineer"
)

type AgentIdentity struct {
    ID    string
    Role  AgentRole
    Owner string
}

type Registry struct {
    agents map[string]AgentIdentity
}

func NewRegistry() *Registry {
    return &Registry{
        agents: make(map[string]AgentIdentity),
    }
}

func (r *Registry) Register(agent AgentIdentity) error {
    if _, exists := r.agents[agent.ID]; exists {
        return errors.New("agent already registered")
    }
    r.agents[agent.ID] = agent
    return nil
}

func (r *Registry) GetAgent(id string) (AgentIdentity, error) {
    agent, exists := r.agents[id]
    if !exists {
        return AgentIdentity{}, errors.New("agent not found")
    }
    return agent, nil
}
EOF

# Run test (should pass now)
go test ./internal/identity -v
```

## Quality Gates (YOU MUST PASS)

### Gate 1: All Tests Pass
```bash
cd /home/ablaze/Projects/AOI/backend
go test ./... -v
```
**Required**: Minimum 5 test files, all tests passing

### Gate 2: Build Succeeds
```bash
cd /home/ablaze/Projects/AOI/backend
go build ./...
go build -o aoi-agent ./cmd/aoi-agent
```
**Required**: No compilation errors

### Gate 3: Binary Runs
```bash
./aoi-agent &
sleep 2
curl http://localhost:8080/health
# Should return: OK
killall aoi-agent
```

## Completion Criteria
- [ ] Directory structure created
- [ ] go.mod initialized
- [ ] At least 5 test files (*_test.go)
- [ ] All tests pass
- [ ] Binary builds successfully
- [ ] Health endpoint works
- [ ] Code follows Go best practices

## Tips
- Start with identity package (simplest)
- Use in-memory storage (no SQLite for prototype)
- Mock Tailscale (just return hardcoded node info)
- Keep it simple - working prototype over perfect design
- Run tests frequently: `go test ./... -v`
- Use `go fmt ./...` to format code

## When You're Done
Respond with:
```
✅ Backend Implementation Complete

Test Results:
- identity_test.go: PASS (4 tests)
- protocol_test.go: PASS (4 tests)
- secretary_test.go: PASS (3 tests)
- acl_test.go: PASS (4 tests)
- types_test.go: PASS (2 tests)

Total: 5 test files, 17 tests, all passing

Build: SUCCESS
Health Check: PASS (http://localhost:8080/health returns OK)
```
