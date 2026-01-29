# AOI Protocol - Technical Design Specification

**Project**: AOI (Agent Operational Interconnect)
**Version**: 1.0.0
**Date**: 2026-01-28
**Status**: Approved

## 1. Architecture Overview

### 1.1 System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                        Tailscale VPN Network                     │
│                                                                   │
│  ┌─────────────────────────┐      ┌─────────────────────────┐  │
│  │   PM's Machine          │      │   Engineer's Machine     │  │
│  │                         │      │                          │  │
│  │  ┌──────────────────┐  │      │  ┌──────────────────┐   │  │
│  │  │  PM Secretary    │◄─┼──────┼─►│ Eng Secretary    │   │  │
│  │  │  Agent           │  │ AOI  │  │ Agent            │   │  │
│  │  └────────┬─────────┘  │ Msg  │  └────────┬─────────┘   │  │
│  │           │             │      │           │              │  │
│  │  ┌────────▼─────────┐  │      │  ┌────────▼──────────┐  │  │
│  │  │  AOI Protocol    │  │      │  │  AOI Protocol     │  │  │
│  │  │  Layer           │  │      │  │  Layer            │  │  │
│  │  └──────────────────┘  │      │  └───────────────────┘  │  │
│  │                         │      │           │              │  │
│  │  ┌──────────────────┐  │      │  ┌────────▼──────────┐  │  │
│  │  │  Approval UI     │  │      │  │  MCP Bridge       │  │  │
│  │  │  (Web)           │  │      │  │                   │  │  │
│  │  └──────────────────┘  │      │  └────────┬──────────┘  │  │
│  │                         │      │           │MCP           │  │
│  └─────────────────────────┘      │  ┌────────▼──────────┐  │  │
│                                    │  │  Work AI          │  │  │
│                                    │  │  (Cursor/Claude)  │  │  │
│                                    │  └───────────────────┘  │  │
│                                    └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Three-Layer Architecture

#### Layer 1: Identity Layer
**Purpose**: Authentication, authorization, and agent discovery
**Components**:
- Agent Identity Registry
- Tailscale Authentication Integration
- Capability Manifest Publisher
- Access Control Manager

#### Layer 2: Interaction Layer
**Purpose**: Message exchange and protocol handling
**Components**:
- JSON-RPC Transport
- Message Handlers (Discovery, Query, Task, Notify)
- Message Queue and Retry Logic
- Rate Limiter

#### Layer 3: Context Layer
**Purpose**: Work AI integration and context management
**Components**:
- MCP Bridge (Work AI ↔ Secretary)
- Context Monitor and Indexer
- Query Translator
- Task Executor

### 1.3 Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      Secretary Agent Process                     │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    Identity Layer                           │ │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐   │ │
│  │  │   Registry   │ │ Tailscale    │ │   ACL Manager    │   │ │
│  │  │   Service    │ │ Auth         │ │                  │   │ │
│  │  └──────────────┘ └──────────────┘ └──────────────────┘   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                   Interaction Layer                         │ │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐   │ │
│  │  │  JSON-RPC    │ │   Message    │ │   Rate Limiter   │   │ │
│  │  │  Transport   │ │   Handlers   │ │   & Retry        │   │ │
│  │  └──────────────┘ └──────────────┘ └──────────────────┘   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    Context Layer                            │ │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐   │ │
│  │  │  MCP Bridge  │ │   Context    │ │    Interrupt     │   │ │
│  │  │              │ │   Monitor    │ │    Controller    │   │ │
│  │  └──────────────┘ └──────────────┘ └──────────────────┘   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                     Storage Layer                           │ │
│  │  ┌──────────────┐ ┌──────────────┐                         │ │
│  │  │   SQLite     │ │   Config     │                         │ │
│  │  │   (Audit)    │ │   Manager    │                         │ │
│  │  └──────────────┘ └──────────────┘                         │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## 2. Detailed Component Design

### 2.1 Identity Layer

#### 2.1.1 Agent Identity Registry

**Purpose**: Maintain registry of active agents on the network

**Data Store**: In-memory cache + SQLite persistence

**Operations**:
```typescript
class AgentRegistry {
  // Register this agent
  async register(identity: AgentIdentity): Promise<void>

  // Discover other agents
  async discover(filter?: AgentFilter): Promise<AgentRegistryEntry[]>

  // Update agent status
  async updateStatus(agentId: AgentId, status: AgentStatus): Promise<void>

  // Remove offline agents (TTL-based)
  async prune(ttlSeconds: number): Promise<void>

  // Get specific agent
  async getAgent(agentId: AgentId): Promise<AgentRegistryEntry | null>
}

interface AgentFilter {
  role?: AgentRole;
  contexts?: ContextReference[];
  capabilities?: ActionCapability[];
}
```

**Discovery Protocol**:
1. On startup, agent broadcasts `aoi.discover` with `announce=true`
2. Other agents receive broadcast, add to registry
3. Agents send heartbeat every 60 seconds
4. If no heartbeat for 180 seconds, agent marked offline

**Storage Schema**:
```sql
CREATE TABLE agent_registry (
  id TEXT PRIMARY KEY,
  identity JSON NOT NULL,
  endpoint TEXT NOT NULL,
  status TEXT NOT NULL,
  last_seen TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_registry_status ON agent_registry(status);
CREATE INDEX idx_registry_last_seen ON agent_registry(last_seen);
```

#### 2.1.2 Tailscale Authentication

**Purpose**: Integrate with Tailscale for network-level authentication

**Implementation**:
```typescript
class TailscaleAuth {
  // Get local node information
  async getLocalNode(): Promise<TailscaleNode>

  // Verify remote node is in same tailnet
  async verifyNode(nodeId: string): Promise<boolean>

  // Get node IP address
  async getNodeIP(nodeId: string): Promise<string>

  // Check if Tailscale is running
  async healthCheck(): Promise<boolean>
}

interface TailscaleNode {
  id: string;           // Tailscale Node ID
  name: string;         // Hostname
  ipv4: string;         // 100.64.x.x address
  ipv6: string;         // fd7a:... address
  online: boolean;
  lastSeen: string;
}
```

**Integration Method**:
- Use Tailscale CLI: `tailscale status --json`
- Parse JSON output for node information
- Alternative: Tailscale HTTP API (future)

**Security Flow**:
```
1. Client connects to server Tailscale IP
2. TLS handshake (Tailscale provides encryption)
3. Server reads client Tailscale IP from connection
4. Server queries Tailscale for node info
5. If node in same tailnet → authenticated
6. If node not in tailnet → reject connection
```

#### 2.1.3 Access Control Manager

**Purpose**: Enforce ACL rules for context access

**Implementation**:
```typescript
class AclManager {
  // Load ACL configuration
  constructor(config: AclConfig)

  // Check if agent has permission
  async checkPermission(
    request: PermissionCheckRequest
  ): Promise<PermissionCheckResult>

  // Add dynamic rule (runtime)
  async addRule(rule: AccessRule): Promise<void>

  // Remove rule
  async removeRule(ruleId: string): Promise<void>

  // List all rules for an agent
  async getRulesForAgent(agentId: AgentId): Promise<AccessRule[]>
}
```

**Permission Evaluation Algorithm**:
```typescript
function evaluatePermission(
  agentId: AgentId,
  resource: Resource,
  action: Action
): PermissionCheckResult {
  // 1. Find all rules matching the agent (by ID or role)
  const matchingRules = findMatchingRules(agentId, resource);

  // 2. Sort by specificity (most specific first)
  //    file > folder > repository > wildcard
  const sortedRules = sortBySpecificity(matchingRules);

  // 3. First matching rule wins
  for (const rule of sortedRules) {
    if (rule.permission === PermissionLevel.ADMIN) return ALLOW;
    if (action === "read" && rule.permission >= PermissionLevel.READ) return ALLOW;
    if (action === "write" && rule.permission >= PermissionLevel.WRITE) return ALLOW;
  }

  // 4. Fall back to default permission
  return config.default_permission === PermissionLevel.NONE ? DENY : ALLOW;
}
```

**Rule Matching**:
- Exact match: `/src/auth/jwt.ts` matches file exactly
- Glob match: `/src/auth/*` matches all files in folder
- Repository match: `acme/webapp` matches repository
- Wildcard: `*` matches any agent (use with caution)

### 2.2 Interaction Layer

#### 2.2.1 JSON-RPC Transport

**Purpose**: Handle HTTP transport for JSON-RPC messages

**Implementation**:
```typescript
class JsonRpcTransport {
  private server: https.Server;
  private handlers: Map<string, MessageHandler>;

  // Start listening on Tailscale IP
  async listen(port: number): Promise<void>

  // Send request to remote agent
  async send(
    target: AgentId,
    request: JsonRpcRequest
  ): Promise<JsonRpcResponse>

  // Register method handler
  registerHandler(method: string, handler: MessageHandler): void

  // Shutdown
  async close(): Promise<void>
}

type MessageHandler = (
  params: unknown,
  context: RequestContext
) => Promise<unknown>;

interface RequestContext {
  from: AgentId;        // Caller identity (from Tailscale IP)
  timestamp: string;
  requestId: string;
}
```

**Request Flow**:
```
Client                         Server
  │                              │
  ├─ POST /aoi/v1/rpc ──────────►│
  │  (JSON-RPC request)          │
  │                              ├─ Authenticate via Tailscale
  │                              ├─ Parse JSON-RPC
  │                              ├─ Route to handler
  │                              ├─ Execute handler
  │                              ├─ Build response
  │◄───────── JSON-RPC response ─┤
  │                              │
```

**Error Handling**:
- Network errors → Retry with exponential backoff
- Timeout errors → Return TIMEOUT error code
- Protocol errors → Return appropriate JSON-RPC error
- Application errors → Log and return INTERNAL_ERROR

**Transport Security**:
- HTTPS only (TLS 1.3)
- Self-signed certificate (Tailscale already encrypts)
- No authentication headers (Tailscale IP is identity)

#### 2.2.2 Message Handlers

**Purpose**: Implement business logic for each message type

**Discovery Handler**:
```typescript
class DiscoveryHandler {
  async handle(
    params: DiscoveryParams,
    context: RequestContext
  ): Promise<DiscoveryResult> {
    if (params.announce) {
      // Register the announcing agent
      await registry.register({
        id: params.agent_id,
        capabilities: params.capabilities,
        // ... other fields from context
      });
    }

    // Return list of known agents
    const agents = await registry.discover();
    return { agents };
  }
}
```

**Query Handler**:
```typescript
class QueryHandler {
  async handle(
    params: QueryParams,
    context: RequestContext
  ): Promise<QueryResult> {
    // 1. Validate requester has permission
    const permitted = await acl.checkPermission({
      agent_id: params.from,
      action: "read",
      resource_type: "repository",
      resource_identifier: params.context_scope[0]
    });

    if (!permitted.allowed) {
      throw new AoiError(
        AoiErrorCode.PERMISSION_DENIED,
        "Access denied to requested context"
      );
    }

    // 2. Parse query
    const parsedQuery = await queryParser.parse(params.query);

    // 3. Search indexed context
    const results = await contextIndex.search(parsedQuery);

    // 4. Generate summary
    const summary = await summarizer.generate(results, params.query);

    // 5. Apply content filtering (zero-knowledge)
    const filtered = await contentFilter.sanitize(summary, permitted);

    // 6. Return structured result
    return {
      summary: filtered.summary,
      progress: filtered.progress,
      blockers: filtered.blockers,
      context_refs: filtered.refs,
      completed: true
    };
  }
}
```

**Task Execution Handler**:
```typescript
class TaskExecutionHandler {
  async handle(
    params: TaskExecuteParams,
    context: RequestContext
  ): Promise<TaskExecuteResult> {
    // 1. Check permission
    await acl.checkPermission({
      agent_id: params.from,
      action: "execute",
      resource_type: "repository",
      resource_identifier: params.context_scope[0]
    });

    // 2. Check if approval required
    if (config.ui.approval_required) {
      const approved = await approvalService.requestApproval({
        type: "task_execution",
        requester: params.from,
        task: params.task_type,
        params: params.task_params,
        timeout: config.ui.approval_timeout_sec
      });

      if (!approved) {
        throw new AoiError(
          AoiErrorCode.APPROVAL_TIMEOUT,
          "Task execution not approved"
        );
      }
    }

    // 3. Create task and queue
    const taskId = await taskQueue.enqueue(params);

    // 4. If async, return immediately
    if (params.async) {
      return {
        task_id: taskId,
        status: "pending"
      };
    }

    // 5. Otherwise, wait for completion
    const result = await taskQueue.wait(taskId, params.timeout);
    return result;
  }
}
```

#### 2.2.3 Rate Limiter and Retry

**Rate Limiter**:
```typescript
class RateLimiter {
  private limits: Map<AgentId, TokenBucket>;

  // Check if request allowed
  async checkLimit(agentId: AgentId): Promise<boolean> {
    const bucket = this.getBucket(agentId);
    return bucket.consume(1);
  }

  // Get remaining quota
  async getRemainingQuota(agentId: AgentId): Promise<number> {
    const bucket = this.getBucket(agentId);
    return bucket.tokens;
  }
}

class TokenBucket {
  constructor(
    private capacity: number,    // Max tokens
    private refillRate: number   // Tokens per second
  ) {}

  consume(tokens: number): boolean {
    this.refill();
    if (this.tokens >= tokens) {
      this.tokens -= tokens;
      return true;
    }
    return false;
  }

  private refill(): void {
    const now = Date.now();
    const elapsed = (now - this.lastRefill) / 1000;
    const newTokens = elapsed * this.refillRate;
    this.tokens = Math.min(this.capacity, this.tokens + newTokens);
    this.lastRefill = now;
  }
}
```

**Retry Logic**:
```typescript
class RetryPolicy {
  constructor(
    private maxRetries: number = 3,
    private baseDelay: number = 1000  // milliseconds
  ) {}

  async execute<T>(
    fn: () => Promise<T>,
    retryableErrors: ErrorCode[]
  ): Promise<T> {
    let lastError: Error;

    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        return await fn();
      } catch (error) {
        lastError = error;

        // Check if error is retryable
        if (!this.isRetryable(error, retryableErrors)) {
          throw error;
        }

        // Last attempt, don't sleep
        if (attempt === this.maxRetries) {
          throw error;
        }

        // Exponential backoff
        const delay = this.baseDelay * Math.pow(2, attempt);
        await sleep(delay);
      }
    }

    throw lastError;
  }

  private isRetryable(error: Error, codes: ErrorCode[]): boolean {
    return codes.includes(error.code);
  }
}
```

### 2.3 Context Layer

#### 2.3.1 MCP Bridge

**Purpose**: Translate between AOI protocol and MCP protocol

**Implementation**:
```typescript
class McpBridge {
  private mcpClient: McpClient;
  private adapter: WorkAiAdapter;

  // Connect to work AI via MCP
  async connect(endpoint: string): Promise<void>

  // Query work AI context
  async queryContext(query: ContextQuery): Promise<ContextQueryResult>

  // Execute task via work AI
  async executeTask(
    taskType: string,
    params: Record<string, unknown>
  ): Promise<TaskResult>

  // Subscribe to work AI events
  async subscribeEvents(
    handler: (event: EditorEvent) => void
  ): Promise<void>
}
```

**MCP Client**:
```typescript
class McpClient {
  // Initialize MCP connection
  async initialize(transport: McpTransport): Promise<void>

  // List available tools
  async listTools(): Promise<Tool[]>

  // Invoke tool
  async invokeTool(name: string, args: unknown): Promise<unknown>

  // Read resource
  async readResource(uri: string): Promise<ResourceContent>

  // List resources
  async listResources(): Promise<Resource[]>
}
```

**Work AI Adapters**:
```typescript
// Base adapter interface
interface WorkAiAdapter {
  name: string;

  // Translate AOI task to work AI command
  translateTask(
    taskType: string,
    params: Record<string, unknown>
  ): McpToolInvocation;

  // Parse work AI response
  parseTaskResult(mcpResult: unknown): TaskResult;

  // Extract context from work AI
  extractContext(): Promise<ContextSnapshot>;
}

// Cursor adapter
class CursorAdapter implements WorkAiAdapter {
  name = "cursor";

  translateTask(taskType: string, params: any): McpToolInvocation {
    switch (taskType) {
      case "run_tests":
        return {
          tool: "execute_command",
          arguments: {
            command: `npm test ${params.test_suite || ""}`,
            cwd: params.cwd
          }
        };

      case "check_api":
        return {
          tool: "read_file",
          arguments: {
            path: params.api_file
          }
        };

      default:
        throw new Error(`Unknown task type: ${taskType}`);
    }
  }

  // ... other methods
}

// ClaudeCode adapter (similar structure)
class ClaudeCodeAdapter implements WorkAiAdapter {
  // ... implementation
}
```

#### 2.3.2 Context Monitor and Indexer

**Context Monitor**:
```typescript
class ContextMonitor {
  // Start monitoring editor events
  async start(): Promise<void>

  // Stop monitoring
  async stop(): Promise<void>

  // Register event handler
  onEvent(handler: (event: EditorEvent) => void): void

  // Get current snapshot
  async getSnapshot(): Promise<ContextSnapshot>
}
```

**Event Collection**:
- Subscribe to MCP resource change notifications
- Poll work AI for state changes (if notifications unavailable)
- Capture file edits, chat messages, diagnostics, test runs
- Apply content redaction before storing

**Context Indexer**:
```typescript
class ContextIndexer {
  private db: Database;  // SQLite FTS5 (full-text search)

  // Index an event
  async index(event: EditorEvent): Promise<void>

  // Search indexed context
  async search(query: ContextQuery): Promise<IndexedContext[]>

  // Get context by ID
  async get(id: string): Promise<IndexedContext | null>

  // Cleanup old entries (retention policy)
  async prune(retentionDays: number): Promise<void>
}
```

**Index Schema**:
```sql
CREATE VIRTUAL TABLE context_index USING fts5(
  id UNINDEXED,
  type,
  primary_key,
  content,
  summary,
  tags,
  timestamp UNINDEXED,
  metadata UNINDEXED
);

CREATE INDEX idx_context_type ON context_index(type);
CREATE INDEX idx_context_timestamp ON context_index(timestamp);
```

**Indexing Pipeline**:
```
Editor Event
    ↓
Content Redaction (remove secrets)
    ↓
AI Summarization (generate summary)
    ↓
Tag Extraction (file types, keywords)
    ↓
Store in SQLite FTS5
```

#### 2.3.3 Interrupt Controller

**Purpose**: Decide whether to respond, buffer, or escalate queries

**Implementation**:
```typescript
class InterruptController {
  // Evaluate incoming query
  async evaluateQuery(
    query: QueryParams
  ): Promise<InterruptDecision>

  // Buffer query for later
  async bufferQuery(query: QueryParams): Promise<void>

  // Get buffered queries
  async getBuffered(): Promise<QueryParams[]>

  // Clear buffer
  async clearBuffer(): Promise<void>
}

enum InterruptDecision {
  AUTO_RESPOND = "auto_respond",     // Answer from cache
  BUFFER = "buffer",                 // Queue for later
  ESCALATE = "escalate"              // Interrupt human
}

interface InterruptDecisionContext {
  query: QueryParams;
  focusMode: boolean;                // Is user in focus hours?
  currentActivity: string;           // What user is doing
  queryPriority: Priority;
  confidence: number;                // Can we answer accurately?
}
```

**Decision Algorithm**:
```typescript
function evaluateInterrupt(
  context: InterruptDecisionContext
): InterruptDecision {
  // Priority override
  if (context.queryPriority === "high") {
    return InterruptDecision.ESCALATE;
  }

  // Outside focus hours → escalate by default
  if (!context.focusMode) {
    return InterruptDecision.ESCALATE;
  }

  // Can we answer confidently?
  if (context.confidence >= config.auto_respond_threshold) {
    return InterruptDecision.AUTO_RESPOND;
  }

  // Low priority + uncertain → buffer
  if (context.queryPriority === "low") {
    return InterruptDecision.BUFFER;
  }

  // Default: escalate
  return InterruptDecision.ESCALATE;
}
```

**Confidence Calculation**:
- Recent context match (last 1 hour) → 0.9 confidence
- Recent context match (last 24 hours) → 0.7 confidence
- Keyword match only → 0.5 confidence
- No match → 0.2 confidence

### 2.4 Storage Layer

#### 2.4.1 Audit Log (SQLite)

**Schema**:
```sql
CREATE TABLE audit_log (
  id TEXT PRIMARY KEY,
  timestamp TIMESTAMP NOT NULL,
  event_type TEXT NOT NULL,
  from_agent TEXT,
  to_agent TEXT,
  method TEXT,
  params JSON,
  result JSON,
  error JSON,
  duration_ms INTEGER,
  metadata JSON
);

CREATE INDEX idx_audit_timestamp ON audit_log(timestamp DESC);
CREATE INDEX idx_audit_agents ON audit_log(from_agent, to_agent);
CREATE INDEX idx_audit_event_type ON audit_log(event_type);
```

**Log Operations**:
```typescript
class AuditLog {
  // Log an event
  async log(entry: AuditLogEntry): Promise<void>

  // Query logs
  async query(filter: AuditLogFilter): Promise<AuditLogEntry[]>

  // Export logs
  async export(format: "json" | "csv"): Promise<string>

  // Prune old logs
  async prune(retentionDays: number): Promise<void>
}
```

**Logged Events**:
- All incoming/outgoing messages
- Permission checks (allowed/denied)
- Task executions (start, complete, fail)
- Configuration changes
- Agent startup/shutdown

## 3. Data Flow Diagrams

### 3.1 Query Flow (PM asks Engineer for progress)

```
PM Agent                  Network              Eng Secretary           MCP Bridge            Work AI
   │                         │                       │                     │                    │
   │ aoi.query              │                       │                     │                    │
   ├────────────────────────►                       │                     │                    │
   │                         ├─ Tailscale Auth ────►                     │                    │
   │                         ├─ Rate Limit Check ──►│                     │                    │
   │                         ├─ ACL Check ──────────►                     │                    │
   │                         │                       ├─ Interrupt Eval ──►│                    │
   │                         │                       │ (AUTO_RESPOND)      │                    │
   │                         │                       ├─ Query Index ──────►│                    │
   │                         │                       ├─ (if needed) ───────┼─ queryContext ───►│
   │                         │                       │                     │◄───────────────────┤
   │                         │                       ├─ Generate Summary ──┤                    │
   │                         │                       ├─ Filter Content ────┤                    │
   │◄──────── response ──────┤◄──────────────────────┤                     │                    │
   │                         │                       │                     │                    │
   │                         │                       ├─ Log to Audit ──────►                    │
   │                         │                       │                     │                    │
```

### 3.2 Task Execution Flow (PM requests test run)

```
PM Agent         Network      Eng Secretary    Approval UI    MCP Bridge    Work AI
   │                │               │                │             │            │
   │ task.execute  │               │                │             │            │
   ├───────────────►               │                │             │            │
   │                ├─ Auth+ACL ──►│                │             │            │
   │                │               ├─ Approval? ───►            │            │
   │                │               │                ├─ Show ─────►           │
   │                │               │                │ Dialog     │            │
   │                │               │◄─ Approved ────┤            │            │
   │                │               ├─ Queue Task ──►│            │            │
   │◄─ task_id ─────┤◄──────────────┤                │            │            │
   │                │               │                │            │            │
   │                │               ├─ Execute ──────────────────►│            │
   │                │               │                │            ├─ invoke ──►│
   │                │               │                │            │ tool       │
   │                │               │                │            │◄───────────┤
   │                │               │◄────────────────────────────┤            │
   │◄─ notify ──────┤◄────────────── (task complete)│            │            │
   │ (task done)    │               │                │            │            │
```

### 3.3 Context Mirroring Flow

```
Work AI          MCP Bridge        Context Monitor      Indexer        Query Handler
   │                  │                   │                 │                │
   │ (user edits     │                   │                 │                │
   │  file)          │                   │                 │                │
   ├─ notify ────────►                   │                 │                │
   │                  ├─ capture event ──►                 │                │
   │                  │                   ├─ redact ───────►                │
   │                  │                   ├─ summarize ────►                │
   │                  │                   ├─ index ─────────►               │
   │                  │                   │                 │                │
   │ ... later ...    │                   │                 │                │
   │                  │                   │◄─ query ────────┤                │
   │                  │                   │  (recent files) │                │
   │                  │                   ├─ search ────────►               │
   │                  │                   ├◄─ results ──────┤                │
   │                  │                   ├─ return ────────►                │
```

## 4. Configuration Design

### 4.1 Configuration File Format

**Location**: `~/.config/aoi/config.json` or `/etc/aoi/config.json`

**Full Schema**: See `/home/ablaze/Projects/AOI/.aida/artifacts/designs/structure.md` section "Configuration File Structure"

**Validation**: JSON Schema validation on load

**Hot Reload**: Support SIGHUP to reload config without restart

### 4.2 Environment Variables

- `AOI_CONFIG_PATH`: Override default config path
- `TAILSCALE_AUTH_KEY`: Tailscale authentication key
- `AOI_LOG_LEVEL`: Override log level (debug, info, warn, error)
- `AOI_DATA_DIR`: Override data directory (default: `~/.local/share/aoi`)

## 5. Deployment Design

### 5.1 Installation Methods

**Method 1: Binary Distribution**
```bash
curl -fsSL https://aoi-protocol.dev/install.sh | sh
```

**Method 2: Package Managers**
```bash
# macOS
brew install aoi-agent

# Linux (apt)
apt install aoi-agent

# Linux (snap)
snap install aoi-agent
```

**Method 3: Source**
```bash
git clone https://github.com/aoi-protocol/aoi
cd aoi
pnpm install
pnpm build
pnpm link --global
```

### 5.2 System Service

**systemd Unit** (Linux):
```ini
[Unit]
Description=AOI Secretary Agent
After=network.target tailscaled.service
Wants=tailscaled.service

[Service]
Type=simple
User=%i
ExecStart=/usr/local/bin/aoi-agent start --mode engineer
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=default.target
```

**launchd** (macOS):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>dev.aoi-protocol.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/aoi-agent</string>
    <string>start</string>
    <string>--mode</string>
    <string>engineer</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
```

### 5.3 Update Mechanism

**Auto-update Strategy**:
1. Check GitHub releases for new version (daily)
2. Download new binary to temp location
3. Verify signature (GPG)
4. Prompt user to restart agent
5. On restart, replace old binary with new

**Manual Update**:
```bash
aoi-agent update
```

## 6. Testing Strategy

### 6.1 Unit Tests

**Coverage Target**: 80%+

**Test Framework**: Jest

**Test Organization**:
```
packages/aoi-protocol/tests/
  ├── identity/
  │   ├── registry.test.ts
  │   └── auth.test.ts
  ├── messages/
  │   ├── discovery.test.ts
  │   ├── query.test.ts
  │   └── task.test.ts
  └── transport/
      └── jsonrpc.test.ts
```

### 6.2 Integration Tests

**Test Scenarios**:
1. End-to-end query flow (PM → Engineer)
2. Task execution with approval
3. Context mirroring and indexing
4. ACL enforcement
5. Rate limiting
6. Retry logic
7. Offline mode

**Test Setup**:
- Mock Tailscale network (localhost)
- Mock MCP server (simulated work AI)
- SQLite in-memory database
- Dockerized environment for isolated testing

### 6.3 Security Tests

**Penetration Testing Scenarios**:
1. Attempt unauthorized context access
2. Attempt agent impersonation
3. Attempt rate limit bypass
4. Attempt to extract source code via crafted queries
5. SQL injection in audit log
6. XSS in web UI

**Tools**:
- OWASP ZAP for web UI testing
- Custom scripts for protocol testing

### 6.4 Performance Tests

**Load Testing**:
- 10 agents, 100 queries/minute for 1 hour
- Measure latency (p50, p95, p99)
- Measure memory usage
- Measure CPU usage

**Tools**: k6 or Artillery

## 7. Monitoring and Observability

### 7.1 Metrics

**Key Metrics**:
- Query latency (histogram)
- Query success rate (counter)
- Agent count (gauge)
- Message rate (counter)
- Error rate by type (counter)
- ACL deny rate (counter)
- Approval timeout rate (counter)

**Export Format**: Prometheus metrics endpoint

### 7.2 Logging

**Log Levels**:
- ERROR: Unrecoverable errors
- WARN: Recoverable errors, degraded operation
- INFO: Normal operation events
- DEBUG: Detailed debugging information

**Log Format**: Structured JSON

**Log Example**:
```json
{
  "timestamp": "2026-01-28T17:00:00Z",
  "level": "info",
  "component": "query_handler",
  "message": "Query processed successfully",
  "agent_from": "pm-tanaka",
  "agent_to": "eng-suzuki",
  "query_id": "req-123",
  "latency_ms": 245
}
```

### 7.3 Tracing

**Distributed Tracing**: OpenTelemetry

**Trace Spans**:
- HTTP request (transport layer)
- Permission check (ACL)
- Context search (indexer)
- MCP query (bridge)
- Task execution (work AI)

## 8. Security Considerations

### 8.1 Threat Model

**Threats**:
1. Malicious agent on Tailscale network
2. Compromised agent certificate
3. Man-in-the-middle (MITM) attack
4. Data exfiltration via crafted queries
5. Denial of service (DoS)
6. Audit log tampering

**Mitigations**:
1. Tailscale network isolation + ACL
2. Certificate revocation list (future)
3. Tailscale encryption (WireGuard)
4. Query validation + content filtering + ACL
5. Rate limiting + circuit breakers
6. Write-only audit log + cryptographic hashing

### 8.2 Cryptography

**TLS Configuration**:
- TLS 1.3 only
- Strong cipher suites (ECDHE, AES-GCM)
- Self-signed certificates (Tailscale handles encryption)

**Future Enhancement**: E2E encryption with agent-specific keys

### 8.3 Compliance

**GDPR**:
- User can request data export
- User can request data deletion
- Audit log includes consent records

**Audit Trail Integrity**:
- Each log entry includes hash of previous entry (blockchain-lite)
- Periodic checkpoint with timestamp (prevent backfill attacks)

## 9. Future Enhancements

**Phase 2 (Planned)**:
- libp2p transport for P2P communication (no central discovery)
- Agent-to-agent negotiation protocols (priority, scheduling)
- Multi-agent task coordination (e.g., PM + QA + Eng collaboration)

**Phase 3 (Vision)**:
- Cross-company agent federation
- Agent marketplace (pluggable behaviors)
- Verifiable compute (cryptographic proof of execution)

---

**Document Control**
Version: 1.0.0
Last Updated: 2026-01-28
Approved By: AIDA Leader-Spec
Status: APPROVED FOR IMPLEMENTATION
