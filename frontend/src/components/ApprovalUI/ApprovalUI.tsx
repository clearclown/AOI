import { useState, useEffect, useCallback } from 'react'
import { apiClient, ApprovalRequest as APIApprovalRequest, ApprovalStatus } from '../../services/api'

export interface ApprovalRequest {
  id: string
  requester: string
  taskType: string
  description?: string
  params: Record<string, unknown>
  status?: ApprovalStatus
  timestamp?: string
  createdAt?: string
  expiresAt?: string
}

interface ApprovalUIProps {
  requests?: ApprovalRequest[]
  onApprove?: (id: string) => void
  onDeny?: (id: string, reason?: string) => void
  currentUser?: string
  autoRefresh?: boolean
  refreshInterval?: number
}

export default function ApprovalUI({
  requests: externalRequests,
  onApprove,
  onDeny,
  currentUser = 'user',
  autoRefresh = true,
  refreshInterval = 5000,
}: ApprovalUIProps) {
  const [internalRequests, setInternalRequests] = useState<ApprovalRequest[]>([])
  const [loadingStates, setLoadingStates] = useState<Record<string, 'approving' | 'denying' | null>>({})
  const [denyReasons, setDenyReasons] = useState<Record<string, string>>({})
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const requests = externalRequests ?? internalRequests
  const isUsingAPI = externalRequests === undefined

  const fetchRequests = useCallback(async () => {
    if (!isUsingAPI) return
    try {
      setIsLoading(true)
      setError(null)
      const data = await apiClient.listApprovalRequests('pending')
      setInternalRequests(data.map((req: APIApprovalRequest) => ({
        id: req.id,
        requester: req.requester,
        taskType: req.taskType,
        description: req.description,
        params: req.params,
        status: req.status,
        createdAt: req.createdAt,
        expiresAt: req.expiresAt,
      })))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch requests')
    } finally {
      setIsLoading(false)
    }
  }, [isUsingAPI])

  useEffect(() => {
    if (isUsingAPI) {
      fetchRequests()
    }
  }, [isUsingAPI, fetchRequests])

  useEffect(() => {
    if (isUsingAPI && autoRefresh) {
      const interval = setInterval(fetchRequests, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [isUsingAPI, autoRefresh, refreshInterval, fetchRequests])
  const [feedback, setFeedback] = useState<Record<string, { type: 'success' | 'error'; message: string }>>({})

  const clearFeedback = (id: string) => {
    setTimeout(() => {
      setFeedback((prev) => {
        const newFeedback = { ...prev }
        delete newFeedback[id]
        return newFeedback
      })
    }, 3000)
  }

  const handleApprove = async (id: string) => {
    setLoadingStates((prev) => ({ ...prev, [id]: 'approving' }))
    setFeedback((prev) => {
      const newFeedback = { ...prev }
      delete newFeedback[id]
      return newFeedback
    })

    try {
      if (isUsingAPI) {
        await apiClient.approveRequest(id, currentUser)
        setInternalRequests((prev) => prev.filter((r) => r.id !== id))
      }
      if (onApprove) {
        onApprove(id)
      }
      setFeedback((prev) => ({ ...prev, [id]: { type: 'success', message: 'Approved successfully' } }))
      clearFeedback(id)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to approve'
      setFeedback((prev) => ({ ...prev, [id]: { type: 'error', message: errorMessage } }))
    } finally {
      setLoadingStates((prev) => ({ ...prev, [id]: null }))
    }
  }

  const handleDeny = async (id: string) => {
    setLoadingStates((prev) => ({ ...prev, [id]: 'denying' }))
    setFeedback((prev) => {
      const newFeedback = { ...prev }
      delete newFeedback[id]
      return newFeedback
    })

    try {
      const reason = denyReasons[id] || 'No reason provided'
      if (isUsingAPI) {
        await apiClient.denyRequest(id, currentUser, reason)
        setInternalRequests((prev) => prev.filter((r) => r.id !== id))
      }
      if (onDeny) {
        onDeny(id, reason)
      }
      setFeedback((prev) => ({ ...prev, [id]: { type: 'success', message: 'Denied successfully' } }))
      clearFeedback(id)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to deny'
      setFeedback((prev) => ({ ...prev, [id]: { type: 'error', message: errorMessage } }))
    } finally {
      setLoadingStates((prev) => ({ ...prev, [id]: null }))
    }
  }

  if (isLoading && requests.length === 0) {
    return <div>Loading...</div>
  }

  if (error) {
    return (
      <div>
        <p style={{ color: '#721c24' }}>Error: {error}</p>
        <button onClick={fetchRequests}>Retry</button>
      </div>
    )
  }

  if (requests.length === 0) {
    return <div>No pending approvals</div>
  }

  const getTimestamp = (req: ApprovalRequest) => {
    const ts = req.timestamp || req.createdAt
    return ts ? new Date(ts).toLocaleString() : 'Unknown'
  }

  return (
    <div>
      <h2>Pending Approvals</h2>
      {isUsingAPI && (
        <button onClick={fetchRequests} style={{ marginBottom: '10px' }} disabled={isLoading}>
          {isLoading ? 'Refreshing...' : 'Refresh'}
        </button>
      )}
      {requests.map((req) => {
        const loading = loadingStates[req.id]
        const feedbackMsg = feedback[req.id]

        return (
          <div key={req.id} style={{ border: '1px solid #ccc', padding: '10px', margin: '10px 0' }}>
            <p>
              <strong>Requester:</strong> {req.requester}
            </p>
            <p>
              <strong>Task:</strong> {req.taskType}
            </p>
            {req.description && (
              <p>
                <strong>Description:</strong> {req.description}
              </p>
            )}
            <p>
              <strong>Time:</strong> {getTimestamp(req)}
            </p>
            {req.expiresAt && (
              <p style={{ fontSize: '0.9em', color: '#666' }}>
                Expires: {new Date(req.expiresAt).toLocaleString()}
              </p>
            )}

            {feedbackMsg && (
              <div
                style={{
                  padding: '8px',
                  marginTop: '10px',
                  marginBottom: '10px',
                  borderRadius: '4px',
                  backgroundColor: feedbackMsg.type === 'success' ? '#d4edda' : '#f8d7da',
                  color: feedbackMsg.type === 'success' ? '#155724' : '#721c24',
                  fontSize: '0.9em',
                }}
              >
                {feedbackMsg.message}
              </div>
            )}

            <div style={{ marginTop: '10px' }}>
              <input
                type="text"
                placeholder="Deny reason (optional)"
                value={denyReasons[req.id] || ''}
                onChange={(e) => setDenyReasons((prev) => ({ ...prev, [req.id]: e.target.value }))}
                style={{ padding: '5px', marginRight: '10px', width: '200px' }}
                data-testid={`deny-reason-${req.id}`}
              />
            </div>

            <div style={{ marginTop: '10px' }}>
              <button
                onClick={() => handleApprove(req.id)}
                disabled={!!loading}
                style={{
                  opacity: loading ? 0.6 : 1,
                  cursor: loading ? 'not-allowed' : 'pointer',
                  backgroundColor: '#28a745',
                  color: 'white',
                  border: 'none',
                  padding: '8px 16px',
                  borderRadius: '4px',
                }}
                data-testid={`approve-${req.id}`}
              >
                {loading === 'approving' ? 'Approving...' : 'Approve'}
              </button>
              <button
                onClick={() => handleDeny(req.id)}
                disabled={!!loading}
                style={{
                  marginLeft: '10px',
                  opacity: loading ? 0.6 : 1,
                  cursor: loading ? 'not-allowed' : 'pointer',
                  backgroundColor: '#dc3545',
                  color: 'white',
                  border: 'none',
                  padding: '8px 16px',
                  borderRadius: '4px',
                }}
                data-testid={`deny-${req.id}`}
              >
                {loading === 'denying' ? 'Denying...' : 'Deny'}
              </button>
            </div>
          </div>
        )
      })}
    </div>
  )
}
