# Phase 1: Requirements Extraction & Architecture

## Document Analysis Summary

### Source Documents Analyzed
1. **README.md** - Project overview and quick start guide
2. **docs/企画書.md** - Project proposal with vision and roadmap
3. **docs/要件定義書.md** - Detailed requirements specification

## Core Feature Extraction

### 1. AOI Protocol (Agent-to-Agent Communication)

**Purpose**: Enable direct AI-to-AI communication without human intermediation

**Key Capabilities**:
- **Identity Management**: Tailscale Node ID or machine-specific certificate-based authentication
- **Capability Discovery**: Agents advertise their available contexts (repository names, issue numbers) and executable actions (Read/Write/Execute)
- **Semantic Query**: Natural language queries structured as JSON-RPC for information exchange
- **Message Types**:
  - Query: Request information from another agent
  - Response: Reply to queries with structured data
  - Notify: Asynchronous notifications for task completion

**Communication Patterns**:
- PM AI queries Engineer AI for progress updates and blockers
- Engineer AI responds with summarized context without human intervention
- Asynchronous task delegation with completion notifications

### 2. Secretary Agent System

The system deploys two types of secretary agents:

#### 2.1 PM Secretary AI (Project Manager's Agent)
**Responsibilities**:
- Progress management and monitoring
- Specification structuring and documentation
- Priority negotiation with other agents
- Query orchestration to gather project status

**Workflow**:
1. Monitors project goals and timelines
2. Queries engineer agents for progress
3. Structures responses into reports
4. Negotiates priorities when conflicts arise

#### 2.2 Engineer Secretary AI (Developer's Agent)
**Responsibilities**:
- Work context summarization and indexing
- Interrupt control and filtering
- Proxy operation of work AI (Cursor/ClaudeCode)
- Deep Work protection

**Core Functions**:

**A. Context Mirroring**:
- Real-time monitoring of editor logs (chat history, code changes, errors)
- Automatic summarization and indexing for external queries
- Maintains structured representation of current work state

**B. Interrupt Control**:
- Evaluates external queries against current focus state
- Decides between:
  - AI Proxy Response: Auto-reply with existing context
  - Buffering: Queue request for later
  - Human Escalation: Critical decisions only

**C. Task Delegation**:
- Receives task requests from other agents (e.g., "check API spec")
- Translates to commands for local work AI
- Executes in background, returns results asynchronously

### 3. Tailscale-Based Secure Networking

**Security Model**: "Complete Closed Network Communication"

**Implementation**:
- All AOI traffic routed through Tailscale VPN
- No public endpoints exposed
- Node-to-node authentication via Tailscale identity
- Network-level access control via Tailscale ACLs

**Benefits**:
- Zero-trust network architecture
- Automatic mesh networking
- Encrypted peer-to-peer communication
- Easy team member onboarding (invite to tailnet)

### 4. MCP-Based Context Sharing

**Purpose**: Structured, secure access to editor internal state

**Access Control**:
- **Zero-Knowledge Approach**: Share analysis results, not source code
- **Folder/Project-level ACL**: Granular permission system
- **Scope Limitation**: Define what contexts (repositories, folders) each agent can access

**Context Types**:
- Code structure and architecture
- Recent changes and diff summaries
- Error logs and debugging state
- Task progress and completion status
- API specifications and documentation

**Privacy Guarantees**:
- Source code remains local unless explicitly shared
- Queries return boolean or summary results only
- Audit trail of all context accesses

## High-Level Architecture (3-Layer Model)

### Layer 1: Identity Layer
**Technology**: Tailscale Service Identity

**Components**:
- Agent Identity Registry
  - Unique ID per agent instance
  - Role identification (PM/Engineer/Other)
  - Capability manifest
- Authentication Service
  - Tailscale-based node authentication
  - Certificate validation
- Access Control Manager
  - Permission rules (ACL)
  - Scope definitions (what contexts are accessible)

**Flow**:
1. Agent starts → Registers with identity service
2. Publishes capability manifest (available contexts, actions)
3. Other agents discover via service registry
4. Authentication occurs via Tailscale before any communication

### Layer 2: Interaction Layer
**Technology**: A2A (Agent-to-Agent) Messaging

**Protocol**: JSON-RPC over HTTPS (Tailscale-secured)

**Message Types**:

1. **Discovery Message**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.discover",
  "params": {
    "agent_id": "eng-suzuki",
    "capabilities": ["context.read", "task.execute"],
    "contexts": ["project/auth-service", "issue/AUTH-123"]
  }
}
```

2. **Query Message**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.query",
  "params": {
    "from": "pm-tanaka",
    "to": "eng-suzuki",
    "query": "認証機能の実装進捗と、現在のブロック要因を抽出せよ",
    "context_scope": ["project/auth-service"]
  },
  "id": "req-001"
}
```

3. **Response Message**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "summary": "JWT実装完了、リフレッシュトークン機能は実装中",
    "progress": 75,
    "blockers": ["Redis接続設定が未確定"],
    "context_refs": ["commit/abc123", "file/auth.ts:45-120"]
  },
  "id": "req-001"
}
```

4. **Notify Message** (Async):
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.notify",
  "params": {
    "from": "eng-suzuki",
    "to": "pm-tanaka",
    "event": "task.completed",
    "task_id": "AUTH-123",
    "result_summary": "認証機能のテスト完了、カバレッジ95%"
  }
}
```

**Interaction Patterns**:
- **Request-Response**: Synchronous queries with immediate response
- **Request-Async-Notify**: Heavy analysis tasks with later notification
- **Broadcast**: Announce status changes to multiple agents
- **Negotiation**: Multi-round exchanges for priority conflicts

### Layer 3: Context Layer
**Technology**: MCP (Model Context Protocol)

**Purpose**: Bridge between work AI (Cursor/ClaudeCode) and Secretary Agent

**Architecture**:
```
Work AI (Cursor/ClaudeCode)
    ↕ [MCP Interface]
Secretary Agent (Context Manager)
    ↕ [AOI Protocol]
Remote Secretary Agents
```

**Context Manager Components**:

1. **Monitor Service**:
   - Watches editor events (file edits, chat messages, diagnostics)
   - Captures code execution results
   - Logs error states

2. **Indexing Service**:
   - Builds searchable index of work context
   - Maintains summary representations
   - Tracks temporal sequence (what happened when)

3. **Query Handler**:
   - Processes AOI queries from remote agents
   - Transforms to MCP queries for work AI
   - Filters results based on ACL

4. **Task Executor**:
   - Receives task delegation requests
   - Interfaces with work AI to execute tasks
   - Returns results via AOI notify

**Data Flow Example**:
```
PM Agent: "What's the status of authentication?"
    ↓ AOI Query
Engineer Secretary: Checks ACL → Query approved
    ↓ MCP Query
Work AI: Analyzes codebase, recent commits, tests
    ↓ MCP Response
Engineer Secretary: Summarizes + filters sensitive data
    ↓ AOI Response
PM Agent: Receives "JWT complete, refresh token in progress"
```

## Architecture Decisions

### Communication Protocol Choice
**Decision**: JSON-RPC over HTTPS (via Tailscale)

**Rationale**:
- Simple, human-readable format for debugging
- Wide language support for implementation
- HTTPS provides transport security (layered on Tailscale)
- JSON-RPC 2.0 has built-in error handling
- Alternative (libp2p) deferred to Phase 3 for complex P2P scenarios

### Agent Deployment Model
**Decision**: Sidecar Architecture

**Rationale**:
- Secretary agent runs as separate process on user's machine
- Communicates with work AI via MCP (local IPC)
- Communicates with remote agents via AOI (network)
- Allows work AI to remain unchanged (no modifications to Cursor/ClaudeCode)
- Easy updates and versioning of secretary agent

### State Management
**Decision**: Event Sourcing for Audit Log

**Rationale**:
- All agent interactions stored as immutable events
- Enables full audit trail reconstruction
- Supports Human-in-the-Loop replay and approval
- Facilitates debugging and compliance

## Key Design Constraints

1. **Privacy First**: No source code leaves machine without explicit user approval
2. **Human Override**: All AI-to-AI agreements can be vetoed by humans
3. **Non-Blocking**: Engineer work continues even if PM agent is offline
4. **Graceful Degradation**: System functions (reduced capability) if Tailscale unavailable
5. **Auditable**: Every action traceable to originating agent and human owner

## Success Metrics

- **Meeting Reduction**: 90% reduction in progress sync meetings
- **Context Switch Reduction**: Engineers interrupted <3 times per day (vs current ~20)
- **Response Time**: Agent queries answered in <10 seconds
- **Accuracy**: Query responses match human expert answers 85%+ of the time

## Next Phase Dependencies

Phase 2 (Structure) requires:
- Detailed message schema definitions
- Data models for agent identity, capabilities, contexts
- API contract specifications for all endpoints
- Directory structure for codebase organization
