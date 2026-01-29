# AOI Protocol (Agent Operational Interconnect)

*"AI同士が話し、AIがAIを操作する。人間はミドルウェアではない。"*

AOI は、異なる役割を持つAIエージェント（PMエージェント、エンジニアエージェントなど）が、人間を介さずにセキュアにコンテキストを同期し、作業を調整するための通信プロトコルおよびシステムです。

## 特徴

| 特徴 | 説明 |
|------|------|
| **A2A (Agent-to-Agent)** | PMのAIがエンジニアのAIに直接話しかけ、進捗を確認し、タスクを依頼 |
| **Privacy First** | Tailscaleによる閉域網通信と、MCPベースの厳格なアクセス制御 |
| **Asynchronous** | 会議やチャットでの割り込みを排除し、秘書AIがバックグラウンドで調整 |
| **Real-time** | WebSocketによるリアルタイム通知とステータス更新 |
| **Human-in-the-loop** | AI間合意は人間が最終承認するUI設計 |

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────┐
│                        AOI Protocol                             │
├─────────────────────────────────────────────────────────────────┤
│  Identity Layer    │  Tailscale Service Identity + ACL         │
├─────────────────────────────────────────────────────────────────┤
│  Interaction Layer │  JSON-RPC 2.0 + WebSocket                 │
├─────────────────────────────────────────────────────────────────┤
│  Context Layer     │  MCP (Model Context Protocol) Bridge      │
└─────────────────────────────────────────────────────────────────┘
```

### コンポーネント

- **Backend (Go)**: JSON-RPC 2.0 API、WebSocket、通知システム
- **Frontend (React)**: Dashboard、Audit Log、Approval UI
- **Context Monitor**: ファイル監視、アクティビティ追跡
- **MCP Bridge**: Model Context Protocol 統合
- **Tailscale**: VPN閉域網、ノード認証

## クイックスタート

### Docker で起動

```bash
git clone https://github.com/aoi-protocol/aoi.git
cd aoi
docker compose up -d
```

アプリケーションにアクセス:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Health Check: http://localhost:8080/health

### ローカル開発

#### Backend

```bash
cd backend
go run ./cmd/aoi-agent
```

#### Frontend

```bash
cd frontend
npm install
npm run dev
```

## API

### REST Endpoints

| エンドポイント | メソッド | 説明 |
|--------------|---------|------|
| `/health` | GET | ヘルスチェック |
| `/api/v1/agents` | GET | エージェント一覧 |
| `/api/v1/agents/:id` | GET | エージェント詳細 |
| `/api/v1/context` | GET | コンテキスト概要 |
| `/api/v1/context/history` | GET | コンテキスト履歴 |

### JSON-RPC 2.0 Methods

| メソッド | 説明 |
|---------|------|
| `aoi.discover` | エージェント発見 |
| `aoi.query` | エージェントへのクエリ |
| `aoi.execute` | タスク実行 |
| `aoi.notify` | 通知送信 |
| `aoi.status` | ステータス取得 |
| `aoi.context` | コンテキスト取得 |

### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  // message.type: 'agent_update' | 'audit_entry' | 'notification'
};
```

## 設定

`backend/aoi.config.json`:

```json
{
  "agent": {
    "id": "eng-suzuki",
    "role": "engineer",
    "mode": "production"
  },
  "network": {
    "listenAddr": ":8080",
    "tlsCert": "/path/to/cert.pem",
    "tlsKey": "/path/to/key.pem"
  },
  "acl": {
    "enabled": true,
    "defaultPermission": "read"
  },
  "context": {
    "watchPaths": ["./src", "./lib"],
    "defaultTTL": "1h"
  },
  "tailscale": {
    "enabled": true,
    "apiURL": "http://localhost:3000",
    "aclPolicy": "tag:pm"
  }
}
```

## テスト

```bash
# Backend テスト
cd backend && go test ./...

# Frontend Unit テスト
cd frontend && npm test

# E2E テスト
cd frontend && npx playwright test

# Docker 環境で E2E
docker run --rm --network aoi_aoi-network \
  -v $(pwd)/frontend:/work -w /work \
  -e E2E_BASE_URL=http://aoi-frontend \
  mcr.microsoft.com/playwright:v1.58.0 \
  npx playwright test
```

## テストカバレッジ

| カテゴリ | テスト数 |
|---------|---------|
| Backend (Go) | 294 |
| Frontend (Unit) | 199 |
| E2E (Playwright) | 61 |
| **合計** | **554** |

## ディレクトリ構成

```
aoi/
├── backend/
│   ├── cmd/aoi-agent/    # エントリーポイント
│   ├── internal/
│   │   ├── acl/          # アクセス制御
│   │   ├── config/       # 設定管理
│   │   ├── context/      # コンテキスト監視
│   │   ├── identity/     # エージェント管理
│   │   ├── mcp/          # MCP ブリッジ
│   │   ├── notify/       # 通知システム
│   │   ├── protocol/     # HTTP + WebSocket
│   │   ├── secretary/    # クエリ処理
│   │   └── tailscale/    # Tailscale 統合
│   └── pkg/aoi/          # 共通型定義
├── frontend/
│   ├── src/
│   │   ├── components/   # React コンポーネント
│   │   ├── hooks/        # カスタムフック
│   │   ├── services/     # API クライアント
│   │   └── types/        # TypeScript 型
│   └── e2e/              # Playwright E2E テスト
├── docs/
│   ├── 企画書.md
│   └── 要件定義書.md
└── docker-compose.yml
```

## 秘書AIの役割

| ロール | 責務 |
|--------|------|
| **PM秘書AI** | 進捗管理、仕様の構造化、優先順位の交渉 |
| **エンジニア秘書AI** | コンテキスト要約、割り込み遮断、作業AI操作代行 |

## 開発フェーズ

| Phase | 状態 | 目標 |
|-------|------|------|
| MVP | ✅ 完了 | Tailscale上でエディタログの要約共有 |
| Operation | ✅ 完了 | PMからエンジニアAIへの操作機能 |
| Ecosystem | 進行中 | 複数社・複数プロジェクト間の標準化 |

## ライセンス

MIT License

## 貢献

Issue や Pull Request をお待ちしています。
