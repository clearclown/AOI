import { useState, useEffect, useRef, useCallback } from 'react'
import { apiClient, AuditEntry as APIAuditEntry, AuditEventType, AuditStats } from '../../services/api'

export interface AuditEntry {
  id: string
  timestamp: string
  fromAgent: string
  toAgent: string
  eventType: string
  summary: string
  success?: boolean
  errorMsg?: string
}

interface AuditLogProps {
  entries?: AuditEntry[]
  autoRefresh?: boolean
  refreshInterval?: number
  initialCount?: number
}

const EVENT_TYPES: AuditEventType[] = [
  'query',
  'execute',
  'approval',
  'notify',
  'context_read',
  'mcp_call',
  'agent_join',
  'agent_leave',
]

export default function AuditLog({
  entries: externalEntries,
  autoRefresh = true,
  refreshInterval = 5000,
  initialCount = 50,
}: AuditLogProps) {
  const [internalEntries, setInternalEntries] = useState<AuditEntry[]>([])
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [eventTypeFilter, setEventTypeFilter] = useState<string>('all')
  const [autoScroll, setAutoScroll] = useState<boolean>(true)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [stats, setStats] = useState<AuditStats | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const entries = externalEntries ?? internalEntries
  const isUsingAPI = externalEntries === undefined

  const fetchEntries = useCallback(async () => {
    if (!isUsingAPI) return
    try {
      setIsLoading(true)
      setError(null)
      const data = await apiClient.getRecentAuditEntries(initialCount)
      setInternalEntries(
        data.map((e: APIAuditEntry) => ({
          id: e.id,
          timestamp: e.timestamp,
          fromAgent: e.fromAgent,
          toAgent: e.toAgent,
          eventType: e.eventType,
          summary: e.summary,
          success: e.success,
          errorMsg: e.errorMsg,
        }))
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch audit entries')
    } finally {
      setIsLoading(false)
    }
  }, [isUsingAPI, initialCount])

  const fetchStats = useCallback(async () => {
    if (!isUsingAPI) return
    try {
      const data = await apiClient.getAuditStats()
      setStats(data)
    } catch {
      // Stats are optional, ignore errors
    }
  }, [isUsingAPI])

  useEffect(() => {
    if (isUsingAPI) {
      fetchEntries()
      fetchStats()
    }
  }, [isUsingAPI, fetchEntries, fetchStats])

  useEffect(() => {
    if (isUsingAPI && autoRefresh) {
      const interval = setInterval(() => {
        fetchEntries()
        fetchStats()
      }, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [isUsingAPI, autoRefresh, refreshInterval, fetchEntries, fetchStats])

  // Auto-scroll to latest entry when new entries arrive
  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [entries, autoScroll])

  if (isLoading && entries.length === 0) {
    return <div>Loading...</div>
  }

  if (error) {
    return (
      <div>
        <p style={{ color: '#721c24' }}>Error: {error}</p>
        <button onClick={fetchEntries}>Retry</button>
      </div>
    )
  }

  if (entries.length === 0) {
    return <div>No audit entries</div>
  }

  // Get unique event types for filter dropdown
  const uniqueTypes = Array.from(new Set(entries.map((e) => e.eventType)))
  const eventTypes = ['all', ...EVENT_TYPES.filter((t) => uniqueTypes.includes(t))]

  // Filter entries based on search and event type
  const filteredEntries = entries.filter((entry) => {
    const matchesSearch =
      searchQuery === '' ||
      entry.summary.toLowerCase().includes(searchQuery.toLowerCase()) ||
      entry.fromAgent.toLowerCase().includes(searchQuery.toLowerCase()) ||
      entry.toAgent.toLowerCase().includes(searchQuery.toLowerCase())

    const matchesType = eventTypeFilter === 'all' || entry.eventType === eventTypeFilter

    return matchesSearch && matchesType
  })

  const getEventColor = (eventType: string, success?: boolean) => {
    if (success === false) return '#dc3545'
    switch (eventType) {
      case 'query':
        return '#007bff'
      case 'execute':
        return '#28a745'
      case 'approval':
        return '#ffc107'
      case 'notify':
        return '#17a2b8'
      case 'agent_join':
        return '#6f42c1'
      case 'agent_leave':
        return '#6c757d'
      default:
        return '#007bff'
    }
  }

  return (
    <div>
      <h2>Audit Log</h2>

      {stats && (
        <div
          style={{
            marginBottom: '15px',
            padding: '10px',
            backgroundColor: '#f8f9fa',
            borderRadius: '4px',
            fontSize: '0.9em',
          }}
        >
          <strong>Stats:</strong> {stats.totalEntries} total entries | {stats.successCount} success |{' '}
          {stats.failureCount} failures
        </div>
      )}

      <div style={{ marginBottom: '15px', display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
        <input
          type="text"
          placeholder="Search entries..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          style={{ padding: '8px', flex: 1, maxWidth: '300px' }}
        />

        <select
          value={eventTypeFilter}
          onChange={(e) => setEventTypeFilter(e.target.value)}
          style={{ padding: '8px' }}
        >
          {eventTypes.map((type) => (
            <option key={type} value={type}>
              {type === 'all' ? 'All Events' : type}
            </option>
          ))}
        </select>

        <label style={{ display: 'flex', alignItems: 'center', gap: '5px', fontSize: '0.9em' }}>
          <input type="checkbox" checked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)} />
          Auto-scroll
        </label>

        {isUsingAPI && (
          <button onClick={fetchEntries} disabled={isLoading} style={{ padding: '8px 12px' }}>
            {isLoading ? 'Refreshing...' : 'Refresh'}
          </button>
        )}

        <span style={{ fontSize: '0.9em', color: '#666' }}>
          {filteredEntries.length} of {entries.length} entries
        </span>
      </div>

      <div ref={containerRef} style={{ maxHeight: '500px', overflowY: 'auto' }}>
        {filteredEntries.map((entry) => (
          <div
            key={entry.id}
            style={{
              borderLeft: `3px solid ${getEventColor(entry.eventType, entry.success)}`,
              paddingLeft: '10px',
              marginBottom: '15px',
            }}
          >
            <p style={{ fontSize: '0.9em', color: '#666' }}>{new Date(entry.timestamp).toLocaleString()}</p>
            <p>
              <strong>{entry.fromAgent}</strong> → <strong>{entry.toAgent}</strong>
            </p>
            <p>{entry.summary}</p>
            {entry.success === false && entry.errorMsg && (
              <p style={{ color: '#dc3545', fontSize: '0.9em' }}>Error: {entry.errorMsg}</p>
            )}
            <span
              style={{
                fontSize: '0.8em',
                color: 'white',
                backgroundColor: getEventColor(entry.eventType, entry.success),
                padding: '2px 6px',
                borderRadius: '3px',
              }}
            >
              {entry.eventType}
            </span>
            {entry.success === false && (
              <span
                style={{
                  fontSize: '0.8em',
                  color: 'white',
                  backgroundColor: '#dc3545',
                  padding: '2px 6px',
                  borderRadius: '3px',
                  marginLeft: '5px',
                }}
              >
                FAILED
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
