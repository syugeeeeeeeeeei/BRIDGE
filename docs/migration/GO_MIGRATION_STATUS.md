# Go移植状況

**対象版:** v0.5.0

## 完了済み

- Go package構成
- CORE/GATE/TRUSS/ANCHOR/BOLTS/BEARING/ULTRASOUND/TRAFFICの分離
- 決定論的Graph順序とpriority queue
- semantic Work Action
- Dijkstra、双方向Dijkstra、A*、reachability
- ANCHOR主要strategy
- deterministic benchmark
- Python-Go semantic parity
- 75 paired casesによる研究準備性評価

## 継続改善

- sessionのpause/resume/snapshot
- memory budgetの実強制
- CSRとworkspace再利用
- typed event vocabularyの完全化
- 実並列corridor scheduling
- Python参照版のWork定義統一

## 判定

現行Go版はPython版のbit-for-bit複製ではないが、正解性、品質、再現性、傾向相関の基準を満たし、比較研究へ投入できる移植段階である。
