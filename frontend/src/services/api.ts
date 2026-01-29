const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

interface JSONRPCRequest {
  jsonrpc: '2.0';
  method: string;
  params?: Record<string, unknown>;
  id: number | string;
}

interface JSONRPCResponse<T = unknown> {
  jsonrpc: '2.0';
  result?: T;
  error?: { code: number; message: string; data?: unknown };
  id: number | string;
}

export interface Agent {
  id: string;
  role: 'engineer' | 'pm' | 'secretary' | string;
  owner: string;
  status: 'online' | 'offline' | 'busy';
  capabilities: string[];
  lastSeen: string;
  endpoint: string;
}

export interface QueryResult {
  answer: string;
  confidence: number;
  sources: string[];
}

export interface AgentStatus {
  id: string;
  role: string;
  uptime: number;
  queriesHandled: number;
  connectedAgents: number;
}

// Approval types
export type ApprovalStatus = 'pending' | 'approved' | 'denied' | 'expired';

export interface ApprovalRequest {
  id: string;
  requester: string;
  taskType: string;
  description: string;
  params: Record<string, unknown>;
  status: ApprovalStatus;
  createdAt: string;
  updatedAt: string;
  expiresAt: string;
  approvedBy?: string;
  deniedBy?: string;
  denyReason?: string;
}

// Audit types
export type AuditEventType =
  | 'query'
  | 'execute'
  | 'approval'
  | 'notify'
  | 'context_read'
  | 'mcp_call'
  | 'agent_join'
  | 'agent_leave';

export interface AuditEntry {
  id: string;
  timestamp: string;
  eventType: AuditEventType;
  fromAgent: string;
  toAgent: string;
  summary: string;
  details?: Record<string, unknown>;
  success: boolean;
  errorMsg?: string;
}

export interface AuditQueryParams {
  fromAgent?: string;
  toAgent?: string;
  eventType?: AuditEventType;
  searchTerm?: string;
  startTime?: string;
  endTime?: string;
  successOnly?: boolean;
  limit?: number;
  offset?: number;
  sortDescending?: boolean;
}

export interface AuditQueryResult {
  entries: AuditEntry[];
  totalCount: number;
  offset: number;
  limit: number;
}

export interface AuditStats {
  totalEntries: number;
  maxEntries: number;
  eventTypeCounts: Record<string, number>;
  successCount: number;
  failureCount: number;
}

class AOIClient {
  private baseUrl: string;
  private requestId = 0;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = baseUrl;
  }

  private async rpc<T>(method: string, params?: Record<string, unknown>): Promise<T> {
    const id = ++this.requestId;
    const response = await fetch(`${this.baseUrl}/api/v1/rpc`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ jsonrpc: '2.0', method, params, id } as JSONRPCRequest),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data: JSONRPCResponse<T> = await response.json();
    if (data.error) throw new Error(data.error.message);
    return data.result as T;
  }

  async getHealth(): Promise<{ status: string }> {
    const res = await fetch(`${this.baseUrl}/health`);
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}: ${res.statusText}`);
    }
    return res.json();
  }

  async discoverAgents(): Promise<Agent[]> {
    return this.rpc<Agent[]>('aoi.discover');
  }

  async queryAgent(agentId: string, query: string): Promise<QueryResult> {
    return this.rpc<QueryResult>('aoi.query', { agent_id: agentId, query });
  }

  async getStatus(): Promise<AgentStatus> {
    return this.rpc<AgentStatus>('aoi.status');
  }

  // Approval API methods
  async createApprovalRequest(
    requester: string,
    taskType: string,
    description: string,
    params: Record<string, unknown>
  ): Promise<ApprovalRequest> {
    return this.rpc<ApprovalRequest>('aoi.approval.create', {
      requester,
      taskType,
      description,
      params,
    });
  }

  async getApprovalRequest(id: string): Promise<ApprovalRequest> {
    return this.rpc<ApprovalRequest>('aoi.approval.get', { id });
  }

  async listApprovalRequests(status?: ApprovalStatus): Promise<ApprovalRequest[]> {
    return this.rpc<ApprovalRequest[]>('aoi.approval.list', { status });
  }

  async approveRequest(id: string, approvedBy: string): Promise<ApprovalRequest> {
    return this.rpc<ApprovalRequest>('aoi.approval.approve', { id, approvedBy });
  }

  async denyRequest(id: string, deniedBy: string, reason: string): Promise<ApprovalRequest> {
    return this.rpc<ApprovalRequest>('aoi.approval.deny', { id, deniedBy, reason });
  }

  // Audit API methods
  async logAuditEntry(
    eventType: AuditEventType,
    fromAgent: string,
    toAgent: string,
    summary: string,
    details?: Record<string, unknown>,
    success: boolean = true,
    errorMsg: string = ''
  ): Promise<AuditEntry> {
    return this.rpc<AuditEntry>('aoi.audit.log', {
      eventType,
      fromAgent,
      toAgent,
      summary,
      details,
      success,
      errorMsg,
    });
  }

  async getAuditEntry(id: string): Promise<AuditEntry> {
    return this.rpc<AuditEntry>('aoi.audit.get', { id });
  }

  async searchAuditEntries(params: AuditQueryParams): Promise<AuditQueryResult> {
    return this.rpc<AuditQueryResult>('aoi.audit.search', params);
  }

  async getRecentAuditEntries(count: number = 50): Promise<AuditEntry[]> {
    return this.rpc<AuditEntry[]>('aoi.audit.recent', { count });
  }

  async getAuditStats(): Promise<AuditStats> {
    return this.rpc<AuditStats>('aoi.audit.stats');
  }
}

export const apiClient = new AOIClient();
