# GATE コンポーネント規則

**対象package:** `src/bridge/gate`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

- 公開入力の検証と既定値適用
- 外部IDとNodeIDの変換
- TRUSS公開APIの呼出し
- 公開結果、終了状態、証明状態、エラーの損失なき変換
- 公開API境界のend-to-end時間計測

## 2. 禁止事項

- ANCHORまたはBOLTSの直接起動
- solver選択、budget配分、quality終了判定
- Collector、Sink、Trace保存先の生成
- 間接条件から証明状態を再推定すること

`budget_exhausted=false`、`error_code==""`、`found=true`だけを根拠に、`search_completed`、`reachability_proven`、`optimality_proven`を生成してはならない。

## 3. 不変条件

- 内部`TerminationStatus`と公開状態が矛盾しない
- Reachabilityの成功をOptimalityへ昇格しない
- pathが存在する場合、そのpath自体を到達可能性の証拠として扱えるが、最適性の証拠にはしない
- Observerの有無で結果を変更しない

## 4. 必須テスト

- 証明状態伝播テスト
- ReachabilityとOptimalityの分離テスト
- invalid request mappingテスト
- timing contractテスト
