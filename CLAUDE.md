# AOI (Agent Operational Interconnect) Protocol

## プロジェクト概要

AOIは、AIエージェント同士が人間を介さずに直接通信・協調するためのプロトコルおよびシステム。
「人間をミドルウェアから解放し、意思決定に集中させる」ことがビジョン。

### コアコンセプト

- **A2A (Agent-to-Agent)**: PMのAIとエンジニアのAIが直接対話
- **秘書AI (Secretary Agent)**: 各ユーザーに配備される外交官的レイヤー
- **Privacy First**: Tailscaleによる閉域網通信、MCPベースのアクセス制御

## アーキテクチャ

### 3レイヤー構成

1. **Identity Layer**: Tailscale Service Identityによる相互認証
2. **Interaction Layer**: A2A形式の「提案・合意・実行」メッセージング
3. **Context Layer**: MCP (Model Context Protocol)によるエディタ状態の構造化共有

### 秘書AIの役割分担

| ロール | 責務 |
|--------|------|
| PM秘書AI | 進捗管理、仕様の構造化、優先順位の交渉 |
| エンジニア秘書AI | コンテキスト要約、割り込み遮断、作業AI操作代行 |

## 技術スタック

- **通信**: libp2p または JSON-RPC over HTTPS
- **ネットワーク**: Tailscale (TS-Auth, ACLs)
- **コンテキストアクセス**: MCP (Model Context Protocol)
- **エージェント制御**: LangGraph または AutoGen

## 開発フェーズ

| Phase | 目標 |
|-------|------|
| MVP | Tailscale上でエディタログの要約共有（進捗実況） |
| Operation | PMからエンジニアAIへの操作機能（テスト実行、ドキュメント生成） |
| Ecosystem | 複数社・複数プロジェクト間のエージェント外交プロトコル標準化 |

## 重要なドキュメント

- `docs/企画書.md`: プロジェクトのビジョンと背景
- `docs/要件定義書.md`: 機能要件・非機能要件の詳細

## 開発ガイドライン

### セキュリティ原則

- ソースコードそのものは外部に渡さない（解析結果やBooleanのみ応答）
- フォルダ/プロジェクト単位のACLによるスコープ限定
- 全通信はVPN経由、パブリックエンドポイント非公開

### 設計原則

- 非同期ファースト: 重い解析は非同期処理、完了時にWebhook/AOI Notify
- Human-in-the-loop: AI間合意は人間が最終承認するUI設計
- 監査可能性: 秘書AI間の通信・操作を自然言語で振り返れるログ

## コマンド例

```bash
# 秘書エージェント起動
aoi-agent start --mode engineer --context ./my-project

# PMからエンジニアへのクエリ
aoi-query --to eng-suzuki "認証機能の実装進捗と、現在のブロック要因を抽出せよ"
```

## 用語集

| 用語 | 説明 |
|------|------|
| Doer | 実作業を行うAI（Cursor, ClaudeCode等） |
| Secretary Agent | DoerとネットワークをつなぐAI層 |
| AOI Protocol | エージェント間通信のためのセキュア規格 |
| Context Mirroring | 作業ログのリアルタイム監視・要約・インデックス化 |
| HitL | Human-in-the-loop、人間による最終承認 |
