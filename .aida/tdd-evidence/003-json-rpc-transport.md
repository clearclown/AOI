# TDD Evidence: JSON-RPC Transport

## Feature: HTTP Server with JSON-RPC Endpoints
- Component: `backend/internal/protocol`
- Date: 2026-01-28

## RED Phase
Test written first: `TestHealthEndpoint`
```go
func TestHealthEndpoint(t *testing.T) {
    server := NewServer(nil, nil)
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    server.ServeHTTP(w, req)
    if w.Code != 200 {
        t.Fatalf("expected 200, got %d", w.Code)
    }
    var resp aoi.HealthResponse
    json.NewDecoder(w.Body).Decode(&resp)
    if resp.Status != "OK" {
        t.Fatalf("expected OK, got %s", resp.Status)
    }
}
```
Result: FAIL - `NewServer` undefined, no HTTP handler implemented.

## GREEN Phase
Minimal implementation:
```go
type Server struct {
    mux      *http.ServeMux
    registry *identity.AgentRegistry
    acl      *acl.ACLManager
}

func NewServer(reg *identity.AgentRegistry, aclMgr *acl.ACLManager) *Server {
    s := &Server{mux: http.NewServeMux(), registry: reg, acl: aclMgr}
    s.mux.HandleFunc("/health", s.handleHealth)
    return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(aoi.HealthResponse{Status: "OK"})
}
```
Result: PASS

## REFACTOR Phase
- Added `/api/agents` endpoint (GET/POST)
- Added `/api/query` endpoint (POST)
- Added method validation (405 responses)
- Added Content-Type header setting
- Added error handling for invalid JSON
- Added dependency injection for registry and ACL

## Final Test Count: 19 tests
## Coverage: 89.5%
