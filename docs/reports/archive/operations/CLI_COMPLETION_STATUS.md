# BRIDGE CLI 新仕様 実装状況

## 今回修正した事項

- 旧`--request`依存をCLI本体・開発タスクから削除
- version情報を`src/buildinfo`へ一元化し、build時注入に対応
- HTTP ServerへContent-Type検証、複数JSON拒否、request ID、適切なエラー分類、ready状態、キャンセル・期限超過分類を追加
- Server設定ひな形を生成後に正式loaderで再検証
- Server設定の日本語コメントを主要な全制限値へ追加
- `bridge artifact evaluate <zip>`でZIP内`result.json`を直接評価可能に変更
- ZIP生成時のファイルディスクリプタ保持を修正
- `bridge serve show --resolved`と`BRIDGE_SERVER_*`環境変数による設定解決を追加
- Python・TypeScript SDKの完全version一致を廃止し、CapabilitiesとSchema IDによる互換判定へ変更
- Python・TypeScript SDKのObservation旧名称を正式名称へ正規化
- Python・TypeScript SDKへLocal Solver TransportとHTTP Server Transportを実装
- Python・TypeScript SDKからローカルBRIDGE Serverを起動・readiness確認・停止できる管理APIを追加
- SDK同梱バイナリとmanifestを0.15.0へ更新

## 検証済み

- `go test ./...`
- `go vet ./...`
- `go test -race ./src/products/cli/cmd/bridge ./src/products/server`
- Python SDKテスト
- TypeScript SDK build・test

## 厳格な完了条件に対する残課題

以下は今回の修正後も未完了であり、CLI新仕様全体を完全完了とは判定しない。

- CLIは責務別Fileへ物理分割済みだが、`app`、`commands`、`config`、`output`等のPackage境界への分離は未完了
- 公開Go契約型を`src/contracts`へ分離し、GATEで明示変換する構造
- CLI全コマンド共通のMachine modeおよび構造化stderr
- Scenarioプリセットの実験系列化（特にscalability・anytime）
- 全Shell向けの完全な補完生成
- OpenAPI文書
- 全OSでの実行CI、fuzz、golden test、文書例自動検証
- Artifact manifestの全ファイルhash・完了状態を含む厳格化

この文書は未達項目を隠さず、次の実装単位を明確にするための状態記録である。
