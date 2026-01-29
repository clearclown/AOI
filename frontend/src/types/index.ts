export interface Agent {
  id: string;
  role: 'engineer' | 'pm' | 'secretary' | string;
  owner: string;
  status: 'online' | 'offline' | 'busy';
  capabilities: string[];
  lastSeen: string;
  endpoint: string;
}

export interface AuditEntry {
  id: string;
  from: string;
  to: string;
  eventType: 'query' | 'response' | 'error' | 'approval' | 'task';
  summary: string;
  timestamp: string;
  metadata?: Record<string, string>;
}

export interface ApprovalRequest {
  id: string;
  requester: string;
  taskType: string;
  description: string;
  params?: Record<string, unknown>;
  timestamp: string;
  status: 'pending' | 'approved' | 'denied';
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

export interface Notification {
  id: string;
  type: string;
  from: string;
  to: string;
  message: string;
  timestamp: string;
}
