# architecture_spec_v0.0.1 実装報告

## 実装済み

- `core`: `WorkBudget`, `Deadline`, `SolverTask`, `SolverProgress`, `SharedSearchState`, `CancellationToken`
- `bearing`: `SearchObserver`, `NullObserver`, `SafeObserver`, schema version
- `anchor`: `MainSolverPort`, `AnchorSession`, PIER互換ANCHOR adapter
- `bolts`: `BoltSolverPort`, capability schema, Bidirectional Dijkstra/A* adapters
- `truss`: portfolio budget所有、task生成、ANCHOR/BOLTS orchestration、shared upper bound、終了判定
- `gate`: public input normalizationとTRUSSのみへの委譲
- `ultrasound`: 開発専用InMemory BEARING adapter
- `traffic`: scenario/run recordの公開schema
- 旧`CABLE`はTRUSS互換facadeへ変更
- `route(..., mode=fast|balanced|quality|exact)`はGATE→TRUSS経由へ変更

## 契約テスト

- 本番層からULTRASOUND/TRAFFICへの依存禁止
- GATEからANCHOR/BOLTSへの直接依存禁止
- ULTRASOUND ON/OFFでpath、distance、work、論理solver trace一致
- GATE→TRUSS公開契約
- portfolio budgetの記録と違反検出

## 現時点の制約

- 既存PIER、Dijkstra、A*はone-shot solverであり、真の途中停止可能な`run_slice`ではない。
- 明示budgetを超えた場合は`budget_violation`として検出するが、legacy solver内部を途中で強制停止できない。
- deadlineもtask起動前後で評価するsoft deadlineであり、solver内部のhard interruptionではない。
- pause/resumeはinterfaceと互換stateのみで、探索frontierの完全な永続化は未実装。
- ANCHORのDetourPort、BOLTS registry、lower-bound専用solver、exact certification専用sessionは次段階。
- ULTRASOUNDはin-memory event収集まで。JSONL/binary、snapshot、seek、replayは未実装。
- TRAFFICはschemaまで。generator、runner、baseline pairing、CI verdict、stress/soakは未実装。

## 検証結果

`pytest -q`: 20 passed

既存の公開APIとlegacy solver modeは維持されています。
