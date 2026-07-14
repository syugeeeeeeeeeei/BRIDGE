# ANCHOR コンポーネント規則

**対象package:** `src/bridge/anchor`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

ANCHORは中断・再開可能な主探索Sessionを所有する。

- HypothesisとRegion
- corridor、portal、hub仮説
- 局所探索状態とfrontier
- Candidate、Upper Bound、Lower Bound
- Checkpoint、停滞証拠、次操作提案
- Snapshot／Resume
- Handoff Resultの検証付き適用

## 2. 必須Session API

- `NewSession`
- `Step`
- `Snapshot`
- `Resume`
- `Progress`
- `Result`
- `Finished`
- `Cancel`

一括`Solve`を提供する場合もSession adapterとして実装する。

## 3. 禁止事項

- portfolio全体予算の所有
- worker管理、BOLTS選択、終了方針
- BOLTSの直接起動
- 未証明Exactの主張
- ULTRASOUNDへの依存

## 4. 不変条件

- `Step`はgrantを超過しない
- 同一Work割当列では連続実行とSnapshot/Resumeが論理的に一致する
- first pathだけを理由に強制終了しない
- HypothesisごとにWork、Candidate、Bound、状態を保持する
- Handoff適用時にPath、距離、Scope、Evidenceを検証する

## 5. 必須テスト

- Step境界値テスト
- Snapshot/Resume同値性テスト
- 仮説状態遷移テスト
- first path後改善テスト
- Handoff適用negative test

## Adaptive Fast Path方針

- ANCHORはWeighted A*系Heuristicを部品として使用する。
- 固定複数Sessionを常時実行してはならない。
- 単一主Sessionで開始し、Candidate発見後はMode契約に従って早期返却する。
- Heuristic計算、観測、Region管理が探索Workを不必要に増幅してはならない。
