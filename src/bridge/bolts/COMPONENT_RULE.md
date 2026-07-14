# BOLTS コンポーネント規則

**対象package:** `src/bridge/bolts`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

BOLTSはCapabilityベースの局所・補助solver群を所有する。

- `CONNECT_CHECKPOINTS`
- `ESCAPE_REGION`
- `REPAIR_SEGMENT`
- `PROVE_UNREACHABLE`
- `TIGHTEN_BOUND`
- `CERTIFY_CANDIDATE`
- Dijkstra、双方向Dijkstra、A*、Weighted A*、Reachability

## 2. 禁止事項

- portfolio scheduling
- ANCHORの継続判断
- 他solverの連鎖起動
- GATEへの直接公開
- Reachability結果のOptimality昇格

## 3. Reachability契約

Reachability Solverは到達可能性または到達不能を判定する。重み付き最短路を保証しない。

- path発見: `reachability_proven=true`, `optimality_proven=false`
- 完全探索による到達不能: `search_completed=true`, `reachability_proven=true`, `optimality_proven=false`
- 予算不足・cancel: 証明未完了

全終了経路でsolver timingを記録する。

## 4. Evidence規則

- EvidenceはCapabilityの実行事実とScopeに基づいて生成する
- empirical値をadmissible lower bound、unreachable、exactへ昇格しない
- `GeneratedWork`は実際のledgerと一致させる
- Scope外での再利用を許可しない

## 5. 必須テスト

- 全Capability契約テスト
- Reachability証明意味論テスト
- 全終了経路Timingテスト
- Work grant境界テスト
- Evidence validation negative test
