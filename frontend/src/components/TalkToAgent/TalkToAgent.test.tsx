import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import TalkToAgent from './TalkToAgent'

// ── Mock apiClient ─────────────────────────────────────────────────────────
vi.mock('../../services/api', () => ({
  apiClient: {
    h2aListSessions: vi.fn().mockResolvedValue({ sessions: [], count: 0 }),
    h2aRegisterSession: vi.fn().mockResolvedValue({ status: 'registered', agent_id: 'eng-test' }),
    h2aSend: vi.fn().mockResolvedValue({ status: 'sent', output: 'hello from tmux' }),
    h2aStream: vi.fn().mockResolvedValue({ status: 'streaming', stream_id: 'stream-123', topic: 'h2a:eng-test' }),
    h2aStop: vi.fn().mockResolvedValue({ status: 'stopped' }),
  },
}))

// ── Mock useWebSocket ──────────────────────────────────────────────────────
vi.mock('../../hooks/useWebSocket', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../hooks/useWebSocket')>()
  return {
    ...actual,
    useWebSocket: vi.fn(() => ({
      status: 'connected',
      lastMessage: null,
      sendMessage: vi.fn(),
      subscribe: vi.fn(),
      unsubscribe: vi.fn(),
      connect: vi.fn(),
      disconnect: vi.fn(),
      addMessageHandler: vi.fn(),
      removeMessageHandler: vi.fn(),
    })),
  }
})

import { apiClient } from '../../services/api'

describe('TalkToAgent', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(apiClient.h2aListSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ sessions: [], count: 0 })
    ;(apiClient.h2aSend as ReturnType<typeof vi.fn>).mockResolvedValue({ status: 'sent', output: 'hello from tmux' })
    ;(apiClient.h2aStream as ReturnType<typeof vi.fn>).mockResolvedValue({ status: 'streaming', stream_id: 'stream-123', topic: 'h2a:eng-test' })
    ;(apiClient.h2aStop as ReturnType<typeof vi.fn>).mockResolvedValue({ status: 'stopped' })
    ;(apiClient.h2aRegisterSession as ReturnType<typeof vi.fn>).mockResolvedValue({ status: 'registered', agent_id: 'eng-test' })
  })

  it('タイトルが表示される', async () => {
    render(<TalkToAgent />)
    expect(screen.getByText('Talk to Agent')).toBeInTheDocument()
  })

  it('初期ロード時にセッション一覧を取得する', async () => {
    render(<TalkToAgent />)
    await waitFor(() => {
      expect(apiClient.h2aListSessions).toHaveBeenCalledTimes(1)
    })
  })

  it('セッション一覧が表示される', async () => {
    ;(apiClient.h2aListSessions as ReturnType<typeof vi.fn>).mockResolvedValue({
      sessions: [
        { agent_id: 'eng-suzuki', session_name: 'claude-eng', registered_at: '2026-01-01T00:00:00Z' },
      ],
      count: 1,
    })
    render(<TalkToAgent />)
    await waitFor(() => {
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
    })
  })

  it('コマンドを入力して送信できる', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent currentUserId="pm-tanaka" />)

    await user.type(screen.getByPlaceholderText(/例: eng-suzuki/), 'eng-test')
    await user.type(screen.getByPlaceholderText(/テストを実行/), 'echo hello')

    const sendBtn = screen.getByText('送信 (Ctrl+Enter)')
    await user.click(sendBtn)

    await waitFor(() => {
      expect(apiClient.h2aSend).toHaveBeenCalledWith('eng-test', 'pm-tanaka', 'echo hello', true)
    })
  })

  it('送信成功時に出力が表示される', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent currentUserId="pm-tanaka" />)

    await user.type(screen.getByPlaceholderText(/例: eng-suzuki/), 'eng-test')
    await user.type(screen.getByPlaceholderText(/テストを実行/), 'echo hello')
    await user.click(screen.getByText('送信 (Ctrl+Enter)'))

    await waitFor(() => {
      expect(screen.getByText('hello from tmux')).toBeInTheDocument()
    })
  })

  it('エラー時にエラーメッセージが表示される', async () => {
    ;(apiClient.h2aSend as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('ACL denied'))
    const user = userEvent.setup()
    render(<TalkToAgent currentUserId="eng-suzuki" />)

    await user.type(screen.getByPlaceholderText(/例: eng-suzuki/), 'eng-yamada')
    await user.type(screen.getByPlaceholderText(/テストを実行/), 'ls')
    await user.click(screen.getByText('送信 (Ctrl+Enter)'))

    await waitFor(() => {
      expect(screen.getByText('ACL denied')).toBeInTheDocument()
    })
  })

  it('ストリームモードに切り替えられる', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent />)
    const checkbox = screen.getByRole('checkbox')
    await user.click(checkbox)
    expect(checkbox).toBeChecked()
  })

  it('ストリームモードで送信するとh2aStreamが呼ばれる', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent currentUserId="pm-tanaka" />)

    await user.click(screen.getByRole('checkbox')) // stream mode on
    await user.type(screen.getByPlaceholderText(/例: eng-suzuki/), 'eng-test')
    await user.type(screen.getByPlaceholderText(/テストを実行/), 'watch tests')
    await user.click(screen.getByText('送信 (Ctrl+Enter)'))

    await waitFor(() => {
      expect(apiClient.h2aStream).toHaveBeenCalledWith('eng-test', 'pm-tanaka', 'watch tests', 500)
    })
  })

  it('出力クリアボタンで出力がリセットされる', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent currentUserId="pm-tanaka" />)

    // Send something first
    await user.type(screen.getByPlaceholderText(/例: eng-suzuki/), 'eng-test')
    await user.type(screen.getByPlaceholderText(/テストを実行/), 'echo hello')
    await user.click(screen.getByText('送信 (Ctrl+Enter)'))
    await waitFor(() => screen.getByText('hello from tmux'))

    await user.click(screen.getByText('出力クリア'))
    expect(screen.queryByText('hello from tmux')).not.toBeInTheDocument()
  })

  it('tmuxセッション登録パネルを開閉できる', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent />)

    const toggleBtn = screen.getByText('+ tmuxセッション登録')
    await user.click(toggleBtn)
    expect(screen.getByText('tmuxセッション登録')).toBeInTheDocument()

    await user.click(screen.getByText('▲ セッション登録を閉じる'))
    expect(screen.queryByText('tmuxセッション登録')).not.toBeInTheDocument()
  })

  it('セッション登録フォームが機能する', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent />)

    await user.click(screen.getByText('+ tmuxセッション登録'))
    await user.type(screen.getByPlaceholderText(/Agent ID/), 'eng-test')
    await user.type(screen.getByPlaceholderText(/セッション名/), 'test-session')
    await user.click(screen.getByText('登録'))

    await waitFor(() => {
      expect(apiClient.h2aRegisterSession).toHaveBeenCalledWith('eng-test', 'test-session', undefined)
    })
  })

  it('Ctrl+Enter でコマンドを送信できる', async () => {
    const user = userEvent.setup()
    render(<TalkToAgent currentUserId="pm-tanaka" />)

    await user.type(screen.getByPlaceholderText(/例: eng-suzuki/), 'eng-test')
    const textarea = screen.getByPlaceholderText(/テストを実行/)
    await user.type(textarea, 'ctrl enter test')
    fireEvent.keyDown(textarea, { key: 'Enter', ctrlKey: true })

    await waitFor(() => {
      expect(apiClient.h2aSend).toHaveBeenCalled()
    })
  })

  it('WebSocket接続状態が表示される', () => {
    render(<TalkToAgent />)
    expect(screen.getByText(/WebSocket: connected/)).toBeInTheDocument()
  })
})
