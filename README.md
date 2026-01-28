AOI Protocol & Nexus Secretary

"AI同士が話し、AIがAIを操作する。人間はミドルウェアではない。"

AOI (Agent Operational Interconnect) は、異なる役割を持つAIエージェント（PMエージェント、エンジニアエージェントなど）が、人間を介さずにセキュアにコンテキストを同期し、作業を調整するための通信プロトコルです。

🚀 特徴

A2A (Agent-to-Agent) Direct Logic: PMのAIがエンジニアのAI（Cursor/ClaudeCode）に直接話しかけ、進捗を確認し、タスクを依頼します。

Privacy First: Tailscaleによる閉域網通信と、MCPベースの厳格なアクセス制御。

Asynchronous Coordination: 会議やチャットでの割り込みを排除し、秘書AIがバックグラウンドで調整を完結させます。

Educational Bridge: AI同士の操作内容を人間に分かりやすく解説し、人間の「理解の遅れ」を補完します。

🛠 プロトコル構成

AOIは以下の3つのレイヤーで構成されます。

Identity Layer: Tailscale Service Identity による相互認証。

Interaction Layer: A2A形式による「提案・合意・実行」のメッセージング。

Context Layer: MCP (Model Context Protocol) を用いた、エディタ内部状態の構造化共有。

📦 導入方法（コンセプト）

Tailscaleのセットアップ: チームメンバーを同一テールネットに招待。

秘書エージェントの起動: 各マシンでAOIサイドカーを起動。

aoi-agent start --mode engineer --context ./my-project


権限設定: PMのAIに対して、どのリポジトリの要約を許可するかを aoi.config.json で定義。

コミュニケーションの開始: PMのAIからクエリを送信。

# PM AIからの内部コマンド例
aoi-query --to eng-suzuki "認証機能の実装進捗と、現在のブロック要因を抽出せよ"


📄 ライセンス

This project is licensed under the MIT License.