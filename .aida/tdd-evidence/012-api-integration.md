# TDD Evidence: API Integration Tests

## Feature: HTTP API Endpoint Testing
- Component: `backend/internal/protocol`
- Date: 2026-01-28

## RED Phase
Test: `TestAgentsEndpoint_POST`
```go
func TestAgentsEndpoint_POST(t *testing.T) {
    server := NewServer(identity.NewAgentRegistry(), acl.NewACLManager())
    body := `{"id":"agent-1","role":"engineer"}`
    req := httptest.NewRequest("POST", "/api/agents", strings.NewReader(body))
    w := httptest.NewRecorder()
    server.ServeHTTP(w, req)
    if w.Code != 201 {
        t.Fatalf("expected 201, got %d", w.Code)
    }
}
```
Result: FAIL - `/api/agents` POST handler not implemented.

## GREEN Phase
```go
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        agents := s.registry.List()
        json.NewEncoder(w).Encode(agents)
    case "POST":
        var agent aoi.AgentIdentity
        if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
            http.Error(w, "invalid json", 400)
            return
        }
        s.registry.Register(&agent)
        w.WriteHeader(201)
        json.NewEncoder(w).Encode(agent)
    default:
        http.Error(w, "method not allowed", 405)
    }
}
```
Result: PASS

## REFACTOR Phase
- Added Content-Type validation
- Added error response formatting
- Added method not allowed (405) for unsupported methods
- Added empty body handling
- Added invalid JSON error handling
- Added query endpoint with metadata support

## Tests in this cycle:
- `TestAgentsEndpoint_GET` / `GET_Empty` / `POST` / `POST_InvalidJSON` / `POST_EmptyBody`
- `TestAgentsEndpoint_PUT_NotAllowed` / `DELETE_NotAllowed`
- `TestQueryEndpoint_POST` / `POST_InvalidJSON` / `GET_NotAllowed`
- `TestQueryEndpoint_EmptyQuery` / `WithMetadata`
- `TestServer_UnknownRoute` (404)

## Final: 13 API integration tests, all passing
