> Status: Historical  
> This document is retained for development history and is not normative for the current implementation.

# BRIDGE 責務分離仕様

本書は旧Python版で確立した責務分離を、現行Go architectureへ引き継ぐための設計記録である。

| 層 | 所有責務 |
|---|---|
| CORE | 中立的な値型 |
| GATE | 外部境界 |
| TRUSS | 判断とportfolio制御 |
| ANCHOR | 主探索 |
| BOLTS | 補助solver |
| BEARING | 観測契約 |
| ULTRASOUND | 観測・保存・分析 |
| TRAFFIC | 試験・benchmark |

旧来の「制御と探索と観測の混在」を禁止し、Go package境界で依存方向を固定する。
