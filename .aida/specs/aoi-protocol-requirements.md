# AOI Protocol - Requirements Specification

**Project**: AOI (Agent Operational Interconnect)
**Version**: 1.0.0
**Date**: 2026-01-28
**Status**: Approved

## Executive Summary

AOI (Agent Operational Interconnect) is a communication protocol that enables AI agents to communicate, coordinate, and operate each other directly without human intermediation. The system deploys "secretary agents" that act as diplomatic representatives for their human owners, handling progress inquiries, context sharing, and task delegation while protecting deep work time and maintaining strict privacy controls.

**Core Innovation**: Eliminate humans as middleware in AI-to-AI communication, reducing meeting overhead by 90% while maintaining security and oversight through Human-in-the-Loop approval systems.

## 1. Functional Requirements

### FR-1: Agent-to-Agent Communication Protocol

#### FR-1.1: Identity Management
**Priority**: MUST
**Description**: Each agent must have a unique, verifiable identity tied to a human owner and secured by Tailscale network identity.

**Requirements**:
- Agent ID format: `{role}-{username}` (e.g., "pm-tanaka", "eng-suzuki")
- Agent roles: PM (Project Manager), Engineer, QA, Design
- Tailscale Node ID integration for network-level authentication
- Agent capability manifest advertising available contexts and actions
- Version declaration for protocol compatibility checking

**Acceptance Criteria**:
- Agent can register identity on startup
- Agent can discover other agents via registry query
- Identity includes owner (human username) for audit trail
- Capability manifest includes actions (read/write/execute) and contexts (repositories, issues)

#### FR-1.2: Semantic Query Protocol
**Priority**: MUST
**Description**: Agents must be able to send natural language queries and receive structured responses.

**Requirements**:
- Query format: Natural language string with context scope specification
- Support for synchronous queries (immediate response)
- Support for asynchronous queries (notify when complete)
- Query priority levels: low, normal, high
- Timeout specification for query expiration
- Structured response format with summary, progress, blockers, context references

**Acceptance Criteria**:
- PM agent can query "What's the progress on authentication feature?"
- Engineer agent responds with structured data (progress percentage, blockers, file references)
- Async queries return immediately with "processing" status, followed by notification when complete
- Query fails gracefully if target agent offline or timeout exceeded

#### FR-1.3: Task Delegation
**Priority**: MUST
**Description**: Agents must be able to request other agents to execute tasks on their local work AI.

**Requirements**:
- Task types: run_tests, generate_docs, check_api, analyze_code
- Task parameters as flexible JSON object
- Async execution model with completion notification
- Task status tracking: pending, running, completed, failed
- Result payload with task-specific output

**Acceptance Criteria**:
- PM agent can request "Run authentication tests with coverage"
- Engineer secretary translates request to work AI command
- Work AI executes tests in background
- Engineer secretary returns results (tests passed/failed, coverage percentage)
- PM agent receives notification when task completes

#### FR-1.4: Event Notification
**Priority**: MUST
**Description**: Agents must be able to broadcast asynchronous notifications about context changes or task completions.

**Requirements**:
- Event types: task.completed, context.updated, agent.status_changed
- Notification routing: one-to-one and one-to-many
- Event timestamp and originating agent identification
- Related query/task ID for correlation
- Flexible data payload per event type

**Acceptance Criteria**:
- Engineer agent notifies PM when major commit happens
- Notification includes event type, timestamp, agent IDs, summary
- PM agent can correlate notification to previous query
- System supports rate limiting to prevent notification spam

### FR-2: Secretary Agent Functions

#### FR-2.1: Context Mirroring
**Priority**: MUST
**Description**: Secretary agent monitors work AI (Cursor/ClaudeCode) activity in real-time and maintains an indexed, searchable representation of current work context.

**Requirements**:
- Monitor editor events: file open/edit/save, chat messages, diagnostics, test runs
- Index context into searchable database
- Generate AI summaries of current focus
- Track temporal sequence (timeline of events)
- Retention policy (default 30 days)
- Content redaction for sensitive data (API keys, passwords)

**Acceptance Criteria**:
- Secretary detects when user edits file `auth.ts`
- Secretary indexes file content with AI-generated summary
- External query "What's the engineer working on?" returns "Implementing JWT refresh token logic"
- Sensitive strings (API keys) are redacted before indexing
- Context query response includes relevant file references without exposing source code

#### FR-2.2: Interrupt Control
**Priority**: MUST
**Description**: Secretary agent filters incoming queries and decides whether to auto-respond, buffer for later, or interrupt the human.

**Requirements**:
- Focus mode detection (configurable hours, e.g., 9am-5pm)
- Query priority evaluation
- Auto-respond capability using indexed context
- Request buffering queue
- Human escalation for high-priority or ambiguous queries
- Configurable auto-respond confidence threshold

**Acceptance Criteria**:
- During focus mode, low-priority query auto-responded without interrupting engineer
- High-priority query triggers desktop notification for human review
- Secretary maintains buffer of queued queries, presents to human when focus mode ends
- Auto-respond confidence below threshold → escalate to human
- Interrupt control can be disabled per user preference

#### FR-2.3: Task Delegation to Work AI
**Priority**: MUST
**Description**: Secretary receives task requests from remote agents and executes them via local work AI.

**Requirements**:
- Translate AOI task requests to work AI commands
- MCP interface to work AI (Cursor/ClaudeCode)
- Async task execution with progress tracking
- Result collection and formatting
- Error handling and retry logic
- Task approval mechanism (human-in-the-loop)

**Acceptance Criteria**:
- PM agent sends "run tests" task request
- Engineer secretary checks ACL permissions
- If approved automatically (or human approves), secretary invokes work AI via MCP
- Work AI executes tests, returns results
- Secretary formats results and notifies PM agent
- Human can review task history in audit log

### FR-3: Security and Access Control

#### FR-3.1: Zero-Knowledge Context Sharing
**Priority**: MUST
**Description**: Source code never leaves the local machine without explicit user approval. Only analysis results, summaries, and metadata are shared.

**Requirements**:
- Query responses contain summaries, not source code
- Context references provide file path and line numbers, not content
- Boolean responses for yes/no queries (e.g., "Is feature X implemented?")
- Explicit approval required for any operation that shares code content
- Content sanitization to remove sensitive data before sharing

**Acceptance Criteria**:
- Query "Is authentication implemented?" returns "Yes, JWT-based auth in auth.ts:45-120" (not the code itself)
- Query "What's blocking progress?" returns summary of blockers, not code snippets
- If PM requests code content explicitly, human approval dialog appears
- Sensitive patterns (API keys) are redacted even in metadata

#### FR-3.2: Folder and Project-Level ACL
**Priority**: MUST
**Description**: Fine-grained access control determines which agents can access which contexts.

**Requirements**:
- Scope types: repository, folder, file, issue
- Permission levels: none, read, write, admin
- Pattern matching for scope (e.g., "/src/*", "acme/webapp", "ISSUE-*")
- Rule-based ACL with precedence (deny takes precedence)
- Per-agent and per-role rules
- Optional rule expiration
- Audit mode for testing ACL rules without enforcement

**Acceptance Criteria**:
- PM agents have read access to all project repositories by default
- Engineer agent can restrict PM access to specific folders (e.g., exclude /internal)
- Query to unauthorized context returns "Permission denied" error
- ACL rules can be configured via config file
- Audit log records all permission checks and decisions

#### FR-3.3: Closed Network Communication
**Priority**: MUST
**Description**: All AOI communication occurs exclusively over Tailscale VPN with no public endpoints.

**Requirements**:
- Tailscale integration for network transport
- Agent listens only on Tailscale IP (100.64.0.0/10 range)
- No DNS exposure, no public IP exposure
- Tailscale ACLs for network-level access control
- Pre-flight check on startup to ensure Tailscale is active
- Graceful degradation if Tailscale unavailable (offline mode)

**Acceptance Criteria**:
- Agent startup fails if Tailscale not running
- Agent binds to Tailscale IP only (e.g., 100.64.0.5:8443)
- External network scan shows no open ports on public interface
- All agent-to-agent traffic encrypted by Tailscale
- If Tailscale disconnects, agent enters offline mode and queues messages

### FR-4: Human-in-the-Loop

#### FR-4.1: Approval UI
**Priority**: MUST
**Description**: Humans can review and approve AI-to-AI agreements before execution.

**Requirements**:
- Approval dialog for sensitive operations (task execution, code sharing)
- Configurable approval requirement (always, sometimes, never)
- Approval timeout (default 5 minutes)
- Approval history and audit trail
- Approval can be granted/denied with optional reason

**Acceptance Criteria**:
- Task execution request triggers approval dialog if `ui.approval_required=true`
- Dialog shows: requesting agent, task type, parameters, affected contexts
- Human can approve (task executes), deny (task fails with error), or timeout (auto-deny)
- Approval decision logged to audit trail
- PM agent receives notification of approval decision

#### FR-4.2: Audit Log and Timeline
**Priority**: MUST
**Description**: All agent interactions recorded in human-readable audit log for transparency and compliance.

**Requirements**:
- Event sourcing: all messages and actions stored immutably
- Natural language summaries of agent interactions
- Timeline view in UI showing chronological events
- Search and filter by agent, date range, event type
- Export capability (JSON, CSV)
- Retention policy (configurable, default 90 days)

**Acceptance Criteria**:
- User opens audit log UI and sees timeline of agent interactions
- Timeline shows: "PM agent pm-tanaka queried eng-suzuki about authentication progress"
- Click on entry shows full message payload and response
- User can search for "authentication" and find all related interactions
- Audit log survives agent restart (persisted to disk)

### FR-5: Technology Integration

#### FR-5.1: MCP (Model Context Protocol) Integration
**Priority**: MUST
**Description**: Secretary agent interfaces with work AI (Cursor/ClaudeCode) via standardized MCP protocol.

**Requirements**:
- MCP client implementation
- Adapters for Cursor and ClaudeCode
- Context query translation (AOI query → MCP query)
- Tool invocation for task execution
- Error handling for MCP protocol errors
- Fallback if MCP unavailable

**Acceptance Criteria**:
- Secretary can query Cursor for current file context via MCP
- Secretary can request ClaudeCode to run tests via MCP tool invocation
- MCP connection established on secretary startup
- If MCP connection fails, secretary logs error and disables work AI integration

#### FR-5.2: Tailscale Network Integration
**Priority**: MUST
**Description**: Leverage Tailscale for secure mesh networking and identity management.

**Requirements**:
- Tailscale SDK or CLI integration
- Automatic node discovery within tailnet
- Use Tailscale Node ID for agent authentication
- Support for Tailscale ACLs
- Health check for Tailscale connectivity
- Auto-reconnect on network interruption

**Acceptance Criteria**:
- Agent startup queries Tailscale for local node ID
- Agent discovers other agents via Tailscale network
- Agent-to-agent communication uses Tailscale IPs
- Tailscale ACLs enforced at network layer
- If Tailscale connection lost, agent retries connection every 30 seconds

## 2. Non-Functional Requirements

### NFR-1: Performance

#### NFR-1.1: Response Time
**Priority**: SHOULD
**Requirement**: Synchronous queries respond within 10 seconds for 95% of requests.
**Rationale**: Maintain fluid conversation flow between agents.

#### NFR-1.2: Async Task Execution
**Priority**: MUST
**Requirement**: Heavy analysis tasks (e.g., full codebase scan) execute asynchronously with progress updates.
**Rationale**: Prevent blocking caller while long-running tasks execute.

#### NFR-1.3: Context Indexing
**Priority**: SHOULD
**Requirement**: Context changes indexed within 30 seconds of occurrence.
**Rationale**: Ensure queries return up-to-date information.

### NFR-2: Reliability

#### NFR-2.1: Message Delivery
**Priority**: SHOULD
**Requirement**: At-least-once message delivery with automatic retry (up to 3 attempts, exponential backoff).
**Rationale**: Network interruptions should not lose critical messages.

#### NFR-2.2: Fault Tolerance
**Priority**: SHOULD
**Requirement**: System degrades gracefully if components fail (e.g., MCP unavailable, Tailscale disconnected).
**Rationale**: Partial functionality better than complete failure.

#### NFR-2.3: Data Persistence
**Priority**: MUST
**Requirement**: Audit logs and indexed context survive agent restart.
**Rationale**: Compliance and debugging require persistent records.

### NFR-3: Scalability

#### NFR-3.1: Agent Count
**Priority**: SHOULD
**Requirement**: Support 10-20 agents per tailnet without performance degradation.
**Rationale**: Typical team size is 5-10 people, each with 1-2 agents.

#### NFR-3.2: Message Throughput
**Priority**: SHOULD
**Requirement**: Handle 100 messages per minute per agent.
**Rationale**: Accounts for burst traffic during high coordination periods.

#### NFR-3.3: Rate Limiting
**Priority**: MUST
**Requirement**: Enforce rate limits (default 10 queries per minute per agent) to prevent abuse.
**Rationale**: Prevent denial-of-service from malicious or buggy agents.

### NFR-4: Usability

#### NFR-4.1: Configuration Simplicity
**Priority**: MUST
**Requirement**: Agent configured via single JSON file with sensible defaults.
**Rationale**: Reduce setup complexity for end users.

#### NFR-4.2: CLI Usability
**Priority**: SHOULD
**Requirement**: CLI commands follow standard conventions (e.g., `aoi-agent start --help`).
**Rationale**: Familiar interface reduces learning curve.

#### NFR-4.3: UI Accessibility
**Priority**: SHOULD
**Requirement**: Web UI accessible, keyboard navigable, screen reader compatible.
**Rationale**: Inclusive design for all users.

### NFR-5: Maintainability

#### NFR-5.1: Code Quality
**Priority**: SHOULD
**Requirement**: TypeScript with strict typing, 80%+ test coverage.
**Rationale**: Type safety and tests reduce bugs and ease refactoring.

#### NFR-5.2: Modularity
**Priority**: MUST
**Requirement**: Clear separation of concerns (protocol, secretary, UI as separate packages).
**Rationale**: Independent development and testing of components.

#### NFR-5.3: Documentation
**Priority**: MUST
**Requirement**: API documentation, architecture diagrams, and user guides.
**Rationale**: Enable onboarding and future maintenance.

## 3. Constraints

### Technical Constraints
- **TC-1**: Must run on Linux, macOS, and Windows
- **TC-2**: Requires Node.js 18+ runtime
- **TC-3**: Requires Tailscale installed and configured
- **TC-4**: Work AI must support MCP protocol (Cursor, ClaudeCode)

### Business Constraints
- **BC-1**: Open-source (MIT License)
- **BC-2**: No external dependencies on paid services
- **BC-3**: Privacy-first: no telemetry without explicit opt-in

### Regulatory Constraints
- **RC-1**: GDPR compliance for audit logs (user can delete their data)
- **RC-2**: Audit trail must be tamper-evident for compliance

## 4. Success Metrics

### Primary Metrics
1. **Meeting Reduction**: 90% reduction in synchronous progress meetings
2. **Interrupt Reduction**: Engineer interruptions reduced from 20/day to <3/day
3. **Context Switch Time**: 80% reduction in time spent explaining context

### Secondary Metrics
4. **Query Accuracy**: Agent responses match human expert answers 85%+ of the time
5. **Adoption Rate**: 80% of team using AOI within 3 months of deployment
6. **User Satisfaction**: NPS score >50

## 5. Acceptance Criteria (System-Level)

**The AOI system is considered complete when**:

1. ✅ PM agent can discover engineer agents on the network
2. ✅ PM agent can query engineer for progress on specific feature
3. ✅ Engineer secretary auto-responds with accurate context summary
4. ✅ PM agent can request task execution (e.g., run tests)
5. ✅ Engineer secretary executes task via work AI and returns results
6. ✅ Human can review and approve sensitive operations via UI
7. ✅ All interactions recorded in searchable audit log
8. ✅ ACL prevents unauthorized context access
9. ✅ Source code never transmitted without explicit approval
10. ✅ System functions over Tailscale VPN with no public exposure

## 6. Out of Scope (Phase 1)

The following are explicitly excluded from the initial release:

- **Multi-team coordination**: Cross-company agent negotiation (Phase 3)
- **Advanced AI features**: Multi-agent planning, autonomous decision-making
- **Mobile apps**: iOS/Android secretary agents
- **Integration with non-MCP editors**: VSCode, IntelliJ without MCP support
- **Blockchain/verifiable compute**: Cryptographic proof of agent actions

## 7. Glossary

- **A2A**: Agent-to-Agent communication
- **ACL**: Access Control List
- **AOI**: Agent Operational Interconnect (this protocol)
- **Context**: The current state of work (code, conversations, errors, tests)
- **HitL**: Human-in-the-Loop (approval and oversight mechanism)
- **MCP**: Model Context Protocol (standardized AI-editor interface)
- **Secretary Agent**: AI agent that represents a human in agent-to-agent negotiations
- **Tailnet**: Tailscale private network
- **Work AI**: The AI that directly edits code (Cursor, ClaudeCode, etc.)
- **Zero-Knowledge**: Sharing analysis results without revealing source data

## 8. References

- Project Proposal: `/home/ablaze/Projects/AOI/docs/企画書.md`
- Requirements Definition: `/home/ablaze/Projects/AOI/docs/要件定義書.md`
- Extraction Artifact: `/home/ablaze/Projects/AOI/.aida/artifacts/requirements/extraction.md`
- Structure Design: `/home/ablaze/Projects/AOI/.aida/artifacts/designs/structure.md`
- Alignment Verification: `/home/ablaze/Projects/AOI/.aida/artifacts/alignment.md`

---

**Document Control**
Version: 1.0.0
Last Updated: 2026-01-28
Approved By: AIDA Leader-Spec
Status: APPROVED FOR IMPLEMENTATION
