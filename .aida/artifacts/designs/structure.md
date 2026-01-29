# Phase 2: System Structure & Design

## Directory Structure

```
aoi-protocol/
├── .aida/                          # AIDA pipeline artifacts
│   ├── artifacts/
│   ├── specs/
│   ├── state/
│   └── results/
│
├── docs/                           # Documentation
│   ├── 企画書.md
│   ├── 要件定義書.md
│   ├── api/                        # API documentation
│   └── architecture/               # Architecture diagrams
│
├── packages/                       # Monorepo structure
│   │
│   ├── aoi-protocol/              # Core protocol implementation
│   │   ├── src/
│   │   │   ├── transport/         # Network transport layer
│   │   │   │   ├── jsonrpc.ts     # JSON-RPC implementation
│   │   │   │   ├── tailscale.ts   # Tailscale integration
│   │   │   │   └── types.ts
│   │   │   ├── messages/          # Message types and handlers
│   │   │   │   ├── discovery.ts
│   │   │   │   ├── query.ts
│   │   │   │   ├── response.ts
│   │   │   │   ├── notify.ts
│   │   │   │   └── types.ts
│   │   │   ├── identity/          # Identity and auth
│   │   │   │   ├── registry.ts
│   │   │   │   ├── auth.ts
│   │   │   │   └── types.ts
│   │   │   └── index.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── tsconfig.json
│   │
│   ├── aoi-secretary/             # Secretary agent implementation
│   │   ├── src/
│   │   │   ├── secretary.ts       # Main secretary agent
│   │   │   ├── context/           # Context management
│   │   │   │   ├── monitor.ts     # Log monitoring
│   │   │   │   ├── indexer.ts     # Context indexing
│   │   │   │   ├── mirror.ts      # Context mirroring
│   │   │   │   └── types.ts
│   │   │   ├── interrupt/         # Interrupt control
│   │   │   │   ├── filter.ts      # Request filtering
│   │   │   │   ├── priority.ts    # Priority evaluation
│   │   │   │   └── buffer.ts      # Request buffering
│   │   │   ├── delegation/        # Task delegation
│   │   │   │   ├── executor.ts    # Task execution
│   │   │   │   ├── translator.ts  # Command translation
│   │   │   │   └── types.ts
│   │   │   ├── acl/               # Access control
│   │   │   │   ├── manager.ts     # ACL management
│   │   │   │   ├── rules.ts       # Permission rules
│   │   │   │   └── types.ts
│   │   │   └── index.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── tsconfig.json
│   │
│   ├── aoi-mcp-bridge/            # MCP integration layer
│   │   ├── src/
│   │   │   ├── bridge.ts          # MCP <-> AOI bridge
│   │   │   ├── adapters/          # Editor adapters
│   │   │   │   ├── cursor.ts
│   │   │   │   ├── claudecode.ts
│   │   │   │   └── base.ts
│   │   │   ├── protocols/         # MCP protocol handlers
│   │   │   │   ├── context.ts
│   │   │   │   ├── tools.ts
│   │   │   │   └── types.ts
│   │   │   └── index.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── tsconfig.json
│   │
│   ├── aoi-cli/                   # Command-line interface
│   │   ├── src/
│   │   │   ├── commands/
│   │   │   │   ├── start.ts       # Start secretary agent
│   │   │   │   ├── query.ts       # Send query
│   │   │   │   ├── config.ts      # Configuration
│   │   │   │   └── status.ts      # Status check
│   │   │   ├── config/            # Configuration management
│   │   │   └── index.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── tsconfig.json
│   │
│   └── aoi-ui/                    # Human-in-the-loop UI
│       ├── src/
│       │   ├── components/
│       │   │   ├── ApprovalDialog.tsx
│       │   │   ├── AuditTimeline.tsx
│       │   │   ├── AgentStatus.tsx
│       │   │   └── ConfigPanel.tsx
│       │   ├── api/               # API client
│       │   ├── stores/            # State management
│       │   └── App.tsx
│       ├── public/
│       ├── package.json
│       └── vite.config.ts
│
├── config/                         # Shared configuration
│   ├── aoi.schema.json            # Config schema
│   └── aoi.example.json           # Example config
│
├── scripts/                        # Build and deployment scripts
│   ├── setup-tailscale.sh
│   ├── build.sh
│   └── deploy.sh
│
├── package.json                    # Root package.json (workspace)
├── tsconfig.json                   # Root TypeScript config
├── .gitignore
├── LICENSE
└── README.md
```

## Data Schemas

### 1. Agent Identity Schema

```typescript
// packages/aoi-protocol/src/identity/types.ts

/**
 * Unique identifier for an agent instance
 */
export type AgentId = string; // Format: "{role}-{username}" e.g., "pm-tanaka", "eng-suzuki"

/**
 * Agent role type
 */
export enum AgentRole {
  PM = "pm",           // Project Manager secretary
  ENGINEER = "engineer", // Engineer secretary
  QA = "qa",           // QA secretary (future)
  DESIGN = "design"    // Design secretary (future)
}

/**
 * Action capability types
 */
export enum ActionCapability {
  READ = "context.read",           // Read context/code
  WRITE = "context.write",         // Modify context (rare, high permission)
  EXECUTE = "task.execute",        // Execute tasks
  QUERY = "query.send",            // Send queries to other agents
  RESPOND = "query.respond"        // Respond to queries
}

/**
 * Context reference - identifies a specific context
 */
export interface ContextReference {
  type: "repository" | "issue" | "folder" | "file";
  identifier: string;  // e.g., "owner/repo", "ISSUE-123", "/src/auth"
  metadata?: Record<string, unknown>;
}

/**
 * Agent capability manifest
 */
export interface AgentCapabilities {
  actions: ActionCapability[];
  contexts: ContextReference[];
  maxConcurrentQueries?: number;
  supportedLanguages?: string[];
}

/**
 * Complete agent identity
 */
export interface AgentIdentity {
  id: AgentId;
  role: AgentRole;
  owner: string;                    // Human owner username
  tailscaleNodeId: string;          // Tailscale node identifier
  capabilities: AgentCapabilities;
  registeredAt: string;             // ISO 8601 timestamp
  lastSeenAt: string;               // ISO 8601 timestamp
  metadata?: {
    version?: string;
    hostname?: string;
    [key: string]: unknown;
  };
}

/**
 * Agent registry entry (for discovery)
 */
export interface AgentRegistryEntry {
  identity: AgentIdentity;
  endpoint: string;                 // Tailscale IP or hostname
  status: "online" | "offline" | "busy";
}
```

### 2. AOI Message Schemas

```typescript
// packages/aoi-protocol/src/messages/types.ts

/**
 * Base JSON-RPC 2.0 message structure
 */
export interface JsonRpcBase {
  jsonrpc: "2.0";
  id?: string | number | null;
}

/**
 * JSON-RPC Request
 */
export interface JsonRpcRequest extends JsonRpcBase {
  method: string;
  params?: unknown;
  id: string | number;
}

/**
 * JSON-RPC Response (Success)
 */
export interface JsonRpcSuccess extends JsonRpcBase {
  result: unknown;
  id: string | number;
}

/**
 * JSON-RPC Response (Error)
 */
export interface JsonRpcError extends JsonRpcBase {
  error: {
    code: number;
    message: string;
    data?: unknown;
  };
  id: string | number | null;
}

/**
 * JSON-RPC Notification (no response expected)
 */
export interface JsonRpcNotification extends JsonRpcBase {
  method: string;
  params?: unknown;
}

// ===== AOI-Specific Message Types =====

/**
 * Discovery message parameters
 */
export interface DiscoveryParams {
  agent_id: AgentId;
  capabilities: AgentCapabilities;
  announce: boolean;  // true = announce presence, false = query for others
}

/**
 * Discovery result
 */
export interface DiscoveryResult {
  agents: AgentRegistryEntry[];
}

/**
 * Query message parameters
 */
export interface QueryParams {
  from: AgentId;
  to: AgentId;
  query: string;                    // Natural language query
  context_scope: string[];          // Which contexts query can access
  priority?: "low" | "normal" | "high";
  async?: boolean;                  // If true, expect notify later
  timeout?: number;                 // Timeout in seconds
}

/**
 * Context reference in results
 */
export interface ContextRefData {
  type: "commit" | "file" | "line" | "symbol" | "issue";
  ref: string;                      // e.g., "commit/abc123", "file/auth.ts:45-120"
  summary?: string;
}

/**
 * Query result
 */
export interface QueryResult {
  summary: string;                  // High-level answer
  progress?: number;                // 0-100 for progress queries
  blockers?: string[];              // List of blocking issues
  context_refs?: ContextRefData[];  // References to specific code/commits
  metadata?: Record<string, unknown>;
  completed: boolean;               // true if answer is complete, false if async pending
}

/**
 * Notify message parameters (async completion)
 */
export interface NotifyParams {
  from: AgentId;
  to: AgentId;
  event: string;                    // Event type e.g., "task.completed", "context.updated"
  related_query_id?: string;        // If related to previous query
  data: unknown;                    // Event-specific data
  timestamp: string;                // ISO 8601
}

/**
 * Task execution request
 */
export interface TaskExecuteParams {
  from: AgentId;
  to: AgentId;
  task_type: string;                // e.g., "run_tests", "generate_docs", "check_api"
  task_params: Record<string, unknown>;
  context_scope: string[];
  async: boolean;                   // If true, will send notify when done
}

/**
 * Task execution result
 */
export interface TaskExecuteResult {
  task_id: string;
  status: "completed" | "failed" | "pending";
  output?: unknown;
  error?: string;
}

/**
 * Error codes (JSON-RPC standard + custom)
 */
export enum AoiErrorCode {
  PARSE_ERROR = -32700,
  INVALID_REQUEST = -32600,
  METHOD_NOT_FOUND = -32601,
  INVALID_PARAMS = -32602,
  INTERNAL_ERROR = -32603,

  // AOI custom errors (application level)
  PERMISSION_DENIED = -32001,       // Agent lacks permission
  CONTEXT_NOT_FOUND = -32002,       // Requested context unavailable
  AGENT_NOT_FOUND = -32003,         // Target agent not in registry
  AGENT_BUSY = -32004,              // Agent cannot respond now
  TIMEOUT = -32005,                 // Query timed out
  RATE_LIMIT = -32006               // Too many queries
}
```

### 3. Context Mirroring Data Schema

```typescript
// packages/aoi-secretary/src/context/types.ts

/**
 * Editor event types
 */
export enum EditorEventType {
  FILE_OPENED = "file.opened",
  FILE_EDITED = "file.edited",
  FILE_SAVED = "file.saved",
  CHAT_MESSAGE = "chat.message",
  CHAT_RESPONSE = "chat.response",
  DIAGNOSTIC = "diagnostic",          // Errors, warnings
  TEST_RUN = "test.run",
  COMMAND_EXECUTED = "command.executed"
}

/**
 * Editor event data
 */
export interface EditorEvent {
  type: EditorEventType;
  timestamp: string;                  // ISO 8601
  file_path?: string;
  content?: string;                   // For edits, messages
  metadata?: Record<string, unknown>;
}

/**
 * Context snapshot - represents current work state
 */
export interface ContextSnapshot {
  timestamp: string;
  active_files: string[];             // Currently open files
  recent_changes: {
    file: string;
    lines_added: number;
    lines_removed: number;
    summary: string;
  }[];
  recent_chat: {
    role: "user" | "assistant";
    content: string;
    timestamp: string;
  }[];
  active_errors: {
    file: string;
    line: number;
    message: string;
    severity: "error" | "warning" | "info";
  }[];
  recent_tests: {
    name: string;
    status: "passed" | "failed" | "skipped";
    duration_ms: number;
  }[];
  focus_summary: string;              // AI-generated summary of current focus
}

/**
 * Indexed context - searchable representation
 */
export interface IndexedContext {
  id: string;
  type: "file" | "symbol" | "conversation" | "task";
  primary_key: string;                // File path, symbol name, etc.
  content: string;                    // Searchable text
  summary: string;                    // AI-generated summary
  tags: string[];                     // Searchable tags
  timestamp: string;
  metadata?: Record<string, unknown>;
}

/**
 * Context query (internal to secretary)
 */
export interface ContextQuery {
  query: string;                      // Natural language or keywords
  filters?: {
    types?: string[];
    date_range?: [string, string];
    tags?: string[];
  };
  limit?: number;
}

/**
 * Context query result
 */
export interface ContextQueryResult {
  results: IndexedContext[];
  total: number;
  query_time_ms: number;
}
```

### 4. Access Control Schema

```typescript
// packages/aoi-secretary/src/acl/types.ts

/**
 * Permission level
 */
export enum PermissionLevel {
  NONE = "none",
  READ = "read",
  WRITE = "write",
  ADMIN = "admin"
}

/**
 * Scope type - what the permission applies to
 */
export type ScopeType = "repository" | "folder" | "file" | "issue";

/**
 * Access rule
 */
export interface AccessRule {
  id: string;
  agent_id: AgentId | "*";            // Specific agent or wildcard
  agent_role?: AgentRole;             // Or by role
  scope_type: ScopeType;
  scope_pattern: string;              // e.g., "owner/repo", "/src/*", "ISSUE-*"
  permission: PermissionLevel;
  expires_at?: string;                // Optional expiration
  metadata?: {
    reason?: string;
    granted_by?: string;
    [key: string]: unknown;
  };
}

/**
 * ACL configuration
 */
export interface AclConfig {
  default_permission: PermissionLevel; // Default if no rule matches
  rules: AccessRule[];
  audit_mode: boolean;                 // If true, log but don't enforce
}

/**
 * Permission check request
 */
export interface PermissionCheckRequest {
  agent_id: AgentId;
  action: "read" | "write" | "execute";
  resource_type: ScopeType;
  resource_identifier: string;
}

/**
 * Permission check result
 */
export interface PermissionCheckResult {
  allowed: boolean;
  matched_rule?: AccessRule;
  reason?: string;
}
```

## API Contracts

### JSON-RPC Endpoints

All endpoints use JSON-RPC 2.0 over HTTPS (Tailscale-secured).

**Base URL**: `https://{tailscale-hostname}:8443/aoi/v1/rpc`

#### 1. `aoi.discover` - Agent Discovery

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.discover",
  "params": {
    "agent_id": "pm-tanaka",
    "capabilities": {
      "actions": ["query.send", "context.read"],
      "contexts": [
        {"type": "repository", "identifier": "acme/webapp"}
      ]
    },
    "announce": true
  },
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "agents": [
      {
        "identity": {
          "id": "eng-suzuki",
          "role": "engineer",
          "owner": "suzuki",
          "tailscaleNodeId": "nABCD1234",
          "capabilities": {
            "actions": ["query.respond", "task.execute", "context.read"],
            "contexts": [
              {"type": "repository", "identifier": "acme/webapp"}
            ]
          },
          "registeredAt": "2026-01-28T10:00:00Z",
          "lastSeenAt": "2026-01-28T16:45:00Z"
        },
        "endpoint": "100.64.0.5:8443",
        "status": "online"
      }
    ]
  },
  "id": 1
}
```

#### 2. `aoi.query` - Send Query

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.query",
  "params": {
    "from": "pm-tanaka",
    "to": "eng-suzuki",
    "query": "認証機能の実装進捗と、現在のブロック要因を抽出せよ",
    "context_scope": ["acme/webapp"],
    "priority": "normal",
    "async": false,
    "timeout": 30
  },
  "id": 2
}
```

**Response (Synchronous)**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "summary": "JWT認証の実装は完了しています。現在リフレッシュトークン機能を実装中で、進捗は75%です。",
    "progress": 75,
    "blockers": [
      "Redis接続設定が未確定のため、トークンストレージの実装が保留"
    ],
    "context_refs": [
      {
        "type": "file",
        "ref": "src/auth/jwt.ts:45-120",
        "summary": "JWT生成・検証ロジック"
      },
      {
        "type": "commit",
        "ref": "abc123def456",
        "summary": "JWT実装のコミット"
      }
    ],
    "completed": true
  },
  "id": 2
}
```

**Response (Asynchronous - immediate)**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "summary": "クエリを受理しました。解析中です。",
    "completed": false,
    "metadata": {
      "estimated_completion": "2026-01-28T17:00:00Z"
    }
  },
  "id": 2
}
```

**Later Notification**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.notify",
  "params": {
    "from": "eng-suzuki",
    "to": "pm-tanaka",
    "event": "query.completed",
    "related_query_id": "2",
    "data": {
      "summary": "詳細な解析結果...",
      "progress": 75,
      "blockers": ["..."]
    },
    "timestamp": "2026-01-28T17:00:00Z"
  }
}
```

#### 3. `aoi.task.execute` - Execute Task

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.task.execute",
  "params": {
    "from": "pm-tanaka",
    "to": "eng-suzuki",
    "task_type": "run_tests",
    "task_params": {
      "test_suite": "auth",
      "coverage": true
    },
    "context_scope": ["acme/webapp"],
    "async": true
  },
  "id": 3
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "task_id": "task-789",
    "status": "pending",
    "message": "タスクをキューに追加しました"
  },
  "id": 3
}
```

**Completion Notification**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.notify",
  "params": {
    "from": "eng-suzuki",
    "to": "pm-tanaka",
    "event": "task.completed",
    "related_query_id": "3",
    "data": {
      "task_id": "task-789",
      "status": "completed",
      "output": {
        "tests_run": 45,
        "tests_passed": 43,
        "tests_failed": 2,
        "coverage": 87.5,
        "duration_ms": 5432
      }
    },
    "timestamp": "2026-01-28T17:05:00Z"
  }
}
```

#### 4. `aoi.notify` - Send Notification

**Notification (No response expected)**:
```json
{
  "jsonrpc": "2.0",
  "method": "aoi.notify",
  "params": {
    "from": "eng-suzuki",
    "to": "pm-tanaka",
    "event": "context.updated",
    "data": {
      "context": "acme/webapp",
      "change_type": "major_commit",
      "summary": "認証機能のリファクタリング完了"
    },
    "timestamp": "2026-01-28T17:10:00Z"
  }
}
```

#### 5. Error Responses

**Example - Permission Denied**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32001,
    "message": "Permission denied",
    "data": {
      "reason": "Agent pm-tanaka does not have read access to context acme/internal-secrets",
      "required_permission": "context.read",
      "requested_scope": "acme/internal-secrets"
    }
  },
  "id": 4
}
```

**Example - Agent Not Found**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32003,
    "message": "Agent not found",
    "data": {
      "requested_agent": "eng-unknown",
      "suggestion": "Check agent registry with aoi.discover"
    }
  },
  "id": 5
}
```

## Configuration File Structure

```json
// config/aoi.example.json
{
  "version": "1.0",
  "agent": {
    "id": "eng-suzuki",
    "role": "engineer",
    "owner": "suzuki"
  },
  "network": {
    "tailscale": {
      "enabled": true,
      "auth_key": "${TAILSCALE_AUTH_KEY}",
      "tailnet": "example.com"
    },
    "listen_port": 8443,
    "discovery_interval_sec": 60
  },
  "contexts": [
    {
      "type": "repository",
      "identifier": "acme/webapp",
      "local_path": "/home/suzuki/projects/webapp"
    }
  ],
  "acl": {
    "default_permission": "none",
    "rules": [
      {
        "id": "allow-pm-read",
        "agent_role": "pm",
        "scope_type": "repository",
        "scope_pattern": "acme/*",
        "permission": "read",
        "metadata": {
          "reason": "PMs can read all project contexts"
        }
      }
    ],
    "audit_mode": false
  },
  "secretary": {
    "work_ai": {
      "type": "cursor",
      "mcp_endpoint": "stdio:///usr/local/bin/cursor-mcp"
    },
    "interrupt_control": {
      "enabled": true,
      "focus_mode_hours": [9, 17],
      "auto_respond_threshold": 0.8
    },
    "context_mirroring": {
      "enabled": true,
      "index_interval_sec": 30,
      "retention_days": 30
    }
  },
  "ui": {
    "enabled": true,
    "port": 3000,
    "approval_required": true
  },
  "logging": {
    "level": "info",
    "audit_log_path": "/var/log/aoi/audit.log"
  }
}
```

## Implementation Notes

### Technology Choices

1. **Language**: TypeScript
   - Type safety for protocol messages
   - Wide ecosystem support
   - Excellent tooling

2. **Runtime**: Node.js
   - Cross-platform (Linux, macOS, Windows)
   - Mature networking libraries
   - Easy integration with MCP

3. **Transport**: HTTPS with JSON-RPC
   - Simple debugging and monitoring
   - Wide language support for clients
   - TLS encryption (over Tailscale tunnel)

4. **UI Framework**: React + Vite
   - Fast development iteration
   - Component reusability
   - Modern tooling

5. **Storage**: SQLite for local state
   - Embedded, no separate database server
   - ACID compliance for audit logs
   - Simple deployment

### Build and Deployment

- **Monorepo**: Uses pnpm workspaces for package management
- **Build**: TypeScript compiled to CommonJS
- **Distribution**: Single binary via pkg or equivalent
- **Updates**: Self-update mechanism via GitHub releases

### Testing Strategy

- **Unit Tests**: Jest for all packages
- **Integration Tests**: Test AOI protocol message exchange
- **E2E Tests**: Playwright for UI testing
- **Security Tests**: ACL enforcement, Tailscale auth validation

## Next Phase Requirements

Phase 3 (Alignment) will verify:
- Message schema completeness
- ACL enforcement in all code paths
- Error handling coverage
- Human-in-the-loop integration points
- Zero-knowledge privacy guarantees
