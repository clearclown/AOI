import { useState, useEffect, useRef, useCallback } from 'react'
import { apiClient, type H2ASession, type H2ASendResult } from '../../services/api'
import { useWebSocket, WS_MESSAGE_TYPES, type H2AOutputPayload, type WSMessage } from '../../hooks/useWebSocket'

interface TalkToAgentProps {
  currentUserId?: string
}

interface OutputLine {
  id: string
  text: string
  isStreaming?: boolean
  timestamp: string
}

type SendMode = 'send' | 'stream'

export default function TalkToAgent({ currentUserId = 'user' }: TalkToAgentProps) {
  const [sessions, setSessions] = useState<H2ASession[]>([])
  const [targetAgentId, setTargetAgentId] = useState('')
  const [fromUser, setFromUser] = useState(currentUserId)
  const [command, setCommand] = useState('')
  const [sendMode, setSendMode] = useState<SendMode>('send')
  const [output, setOutput] = useState<OutputLine[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [streamId, setStreamId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Session registration form
  const [showRegister, setShowRegister] = useState(false)
  const [regAgentId, setRegAgentId] = useState('')
  const [regSessionName, setRegSessionName] = useState('')
  const [regPaneName, setRegPaneName] = useState('')
  const [regLoading, setRegLoading] = useState(false)
  const [regError, setRegError] = useState<string | null>(null)

  const outputEndRef = useRef<HTMLDivElement>(null)
  const lineIdRef = useRef(0)

  const ws = useWebSocket({ autoConnect: true })

  const nextId = () => String(++lineIdRef.current)

  const appendOutput = useCallback((text: string, isStreaming = false) => {
    setOutput(prev => [...prev, {
      id: nextId(),
      text,
      isStreaming,
      timestamp: new Date().toISOString(),
    }])
  }, [])

  // Scroll to bottom when output changes
  useEffect(() => {
    outputEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [output])

  // Load registered sessions on mount
  useEffect(() => {
    loadSessions()
  }, [])

  // Subscribe to h2a_output messages
  useEffect(() => {
    if (!targetAgentId) return

    const handler = (msg: WSMessage<H2AOutputPayload>) => {
      const payload = msg.payload
      if (!payload) return
      if (payload.agent_id !== targetAgentId) return

      if (payload.is_complete) {
        setStreamId(null)
        setIsLoading(false)
        appendOutput('[ストリーム完了]')
        return
      }
      if (payload.output) {
        setOutput(prev => {
          // Replace last streaming line or append
          const last = prev[prev.length - 1]
          if (last?.isStreaming) {
            return [...prev.slice(0, -1), { ...last, text: payload.output, timestamp: new Date().toISOString() }]
          }
          return [...prev, { id: nextId(), text: payload.output, isStreaming: true, timestamp: new Date().toISOString() }]
        })
      }
    }

    ws.addMessageHandler(WS_MESSAGE_TYPES.H2A_OUTPUT as Parameters<typeof ws.addMessageHandler>[0], handler as Parameters<typeof ws.addMessageHandler>[1])
    ws.subscribe([`h2a:${targetAgentId}`])

    return () => {
      ws.removeMessageHandler(WS_MESSAGE_TYPES.H2A_OUTPUT as Parameters<typeof ws.removeMessageHandler>[0], handler as Parameters<typeof ws.removeMessageHandler>[1])
    }
  }, [targetAgentId, ws, appendOutput])

  async function loadSessions() {
    try {
      const result = await apiClient.h2aListSessions()
      setSessions(result.sessions ?? [])
    } catch {
      // Backend may not be running yet; ignore silently
    }
  }

  async function handleSend() {
    if (!targetAgentId || !command.trim()) return
    setIsLoading(true)
    setError(null)

    try {
      if (sendMode === 'stream') {
        appendOutput(`> ${command}`)
        const result = await apiClient.h2aStream(targetAgentId, fromUser, command, 500)
        setStreamId(result.stream_id)
        setCommand('')
        // Output arrives via WebSocket h2a_output messages; isLoading cleared on is_complete
      } else {
        appendOutput(`> ${command}`)
        const result: H2ASendResult = await apiClient.h2aSend(targetAgentId, fromUser, command, true)
        setCommand('')
        if (result.output) {
          appendOutput(result.output)
        }
        setIsLoading(false)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      setIsLoading(false)
    }
  }

  async function handleStop() {
    if (!streamId) return
    try {
      await apiClient.h2aStop(streamId)
      setStreamId(null)
      setIsLoading(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleRegister() {
    if (!regAgentId || !regSessionName) return
    setRegLoading(true)
    setRegError(null)
    try {
      await apiClient.h2aRegisterSession(regAgentId, regSessionName, regPaneName || undefined)
      setRegAgentId('')
      setRegSessionName('')
      setRegPaneName('')
      setShowRegister(false)
      await loadSessions()
    } catch (err) {
      setRegError(err instanceof Error ? err.message : String(err))
    } finally {
      setRegLoading(false)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>Talk to Agent</h2>
        <button onClick={() => setShowRegister(v => !v)} style={btnStyle('secondary')}>
          {showRegister ? '▲ セッション登録を閉じる' : '+ tmuxセッション登録'}
        </button>
      </div>

      {/* Session registration panel */}
      {showRegister && (
        <div style={panelStyle}>
          <h3 style={{ margin: '0 0 12px' }}>tmuxセッション登録</h3>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '8px', marginBottom: '8px' }}>
            <input
              placeholder="Agent ID (例: eng-suzuki)"
              value={regAgentId}
              onChange={e => setRegAgentId(e.target.value)}
              style={inputStyle}
            />
            <input
              placeholder="セッション名 (例: claude-eng)"
              value={regSessionName}
              onChange={e => setRegSessionName(e.target.value)}
              style={inputStyle}
            />
            <input
              placeholder="ペーン名 (省略可)"
              value={regPaneName}
              onChange={e => setRegPaneName(e.target.value)}
              style={inputStyle}
            />
          </div>
          {regError && <p style={errorStyle}>{regError}</p>}
          <button
            onClick={handleRegister}
            disabled={regLoading || !regAgentId || !regSessionName}
            style={btnStyle('primary')}
          >
            {regLoading ? '登録中...' : '登録'}
          </button>
        </div>
      )}

      {/* Registered sessions list */}
      {sessions.length > 0 && (
        <div style={{ fontSize: '13px', color: '#666' }}>
          登録済みセッション: {sessions.map(s => (
            <span
              key={s.agent_id}
              onClick={() => setTargetAgentId(s.agent_id)}
              style={{
                cursor: 'pointer',
                padding: '2px 8px',
                marginRight: '6px',
                borderRadius: '12px',
                background: targetAgentId === s.agent_id ? '#1976d2' : '#e3f2fd',
                color: targetAgentId === s.agent_id ? '#fff' : '#1976d2',
                display: 'inline-block',
              }}
            >
              {s.agent_id}
            </span>
          ))}
          <button onClick={loadSessions} style={{ marginLeft: '8px', fontSize: '12px', cursor: 'pointer' }}>↺</button>
        </div>
      )}

      {/* Send form */}
      <div style={panelStyle}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px', marginBottom: '8px' }}>
          <div>
            <label style={labelStyle}>送信先エージェントID</label>
            <input
              placeholder="例: eng-suzuki"
              value={targetAgentId}
              onChange={e => setTargetAgentId(e.target.value)}
              style={inputStyle}
            />
          </div>
          <div>
            <label style={labelStyle}>送信者ID (自分のID)</label>
            <input
              placeholder="例: pm-tanaka"
              value={fromUser}
              onChange={e => setFromUser(e.target.value)}
              style={inputStyle}
            />
          </div>
        </div>

        <div style={{ marginBottom: '8px' }}>
          <label style={labelStyle}>コマンド / 指示</label>
          <textarea
            placeholder="例: テストを実行してエラーを報告して (Ctrl+Enter で送信)"
            value={command}
            onChange={e => setCommand(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={3}
            style={{ ...inputStyle, width: '100%', resize: 'vertical', fontFamily: 'monospace' }}
          />
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer', fontSize: '14px' }}>
            <input
              type="checkbox"
              checked={sendMode === 'stream'}
              onChange={e => setSendMode(e.target.checked ? 'stream' : 'send')}
            />
            リアルタイムストリーミング
          </label>

          {streamId ? (
            <button onClick={handleStop} style={btnStyle('danger')}>■ ストリーム停止</button>
          ) : (
            <button
              onClick={handleSend}
              disabled={isLoading || !targetAgentId || !command.trim()}
              style={btnStyle('primary')}
            >
              {isLoading ? '送信中...' : '送信 (Ctrl+Enter)'}
            </button>
          )}

          <button onClick={() => setOutput([])} style={btnStyle('secondary')}>出力クリア</button>
        </div>

        {error && <p style={errorStyle}>{error}</p>}
      </div>

      {/* Output display */}
      <div style={{
        background: '#1e1e1e',
        color: '#d4d4d4',
        borderRadius: '6px',
        padding: '16px',
        minHeight: '200px',
        maxHeight: '500px',
        overflowY: 'auto',
        fontFamily: 'monospace',
        fontSize: '13px',
        lineHeight: '1.5',
      }}>
        {output.length === 0 ? (
          <span style={{ color: '#666' }}>出力はここに表示されます...</span>
        ) : (
          output.map(line => (
            <div key={line.id} style={{
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
              color: line.text.startsWith('>') ? '#569cd6' : line.isStreaming ? '#9cdcfe' : '#d4d4d4',
            }}>
              {line.text}
            </div>
          ))
        )}
        <div ref={outputEndRef} />
      </div>

      {/* WebSocket status */}
      <div style={{ fontSize: '12px', color: '#999', textAlign: 'right' }}>
        WebSocket: {ws.status}
        {streamId && <span style={{ marginLeft: '8px', color: '#4caf50' }}>● ストリーミング中 ({streamId.slice(0, 8)}...)</span>}
      </div>
    </div>
  )
}

// Styles
const panelStyle: React.CSSProperties = {
  border: '1px solid #ddd',
  borderRadius: '6px',
  padding: '16px',
  background: '#fafafa',
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px',
  borderRadius: '4px',
  border: '1px solid #ccc',
  fontSize: '14px',
  boxSizing: 'border-box',
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '12px',
  fontWeight: 600,
  marginBottom: '4px',
  color: '#555',
}

const errorStyle: React.CSSProperties = {
  color: '#d32f2f',
  fontSize: '13px',
  margin: '8px 0 0',
}

function btnStyle(variant: 'primary' | 'secondary' | 'danger'): React.CSSProperties {
  const base: React.CSSProperties = {
    padding: '8px 16px',
    borderRadius: '4px',
    border: 'none',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 500,
  }
  switch (variant) {
    case 'primary': return { ...base, background: '#1976d2', color: '#fff' }
    case 'secondary': return { ...base, background: '#e0e0e0', color: '#333' }
    case 'danger': return { ...base, background: '#d32f2f', color: '#fff' }
  }
}
