# CORE コンポーネント規則

**対象package:** `src/bridge/core`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

COREは共有値型と中立契約だけを所有する。

- Graph、NodeID、Edge
- RouteRequest、RouteResult
- Hypothesis、Region、Checkpoint
- Evidence、ProofClass
- HandoffRequest、HandoffResult
- TerminationStatus
- Work Model v2、WorkMetrics、Budget、Bounds

## 2. 禁止事項

- solver選択、task scheduling、予算配分
- 証明の生成または昇格
- trace保存、CLI/JSON I/O
- algorithm固有heuristic
- 他のBRIDGE componentへの依存

## 3. 不変条件

- `TerminationStatus`は排他的である
- EvidenceはScope、生成元、Generated Work、ProofClassを持つ
- empirical Evidenceをproofとして扱わない
- Work Model versionは`2.0`である
- 公開契約にsolver private stateを漏らさない

## 4. 必須テスト

- 型とvalidationの単体テスト
- Evidence不正昇格negative test
- TerminationStatus整合テスト
- Work Model version・Actionテスト
- architecture dependency test

## 5. 関連文書

- `docs/ARCHITECTURE_RULE.md`
- `docs/WORD_DEFINITION.md`
