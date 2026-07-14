# BRIDGE コンポーネント規則索引

**対象版:** v0.15.0以降  
**状態:** 規範文書索引

| コンポーネント | 規則 | 主責務 |
|---|---|---|
| CORE | `src/bridge/core/COMPONENT_RULE.md` | 共有型、Evidence、Handoff、Termination、Work v2 |
| GATE | `src/bridge/gate/COMPONENT_RULE.md` | 公開境界、状態と証明の損失なき伝播 |
| TRUSS | `src/bridge/truss/COMPONENT_RULE.md` | online epoch、Budget、Task、Handoff、Evidence、終了判定 |
| ANCHOR | `src/bridge/anchor/COMPONENT_RULE.md` | 中断・再開可能なHypothesis Session |
| BOLTS | `src/bridge/bolts/COMPONENT_RULE.md` | Capabilityベースの局所・補助solver |
| BEARING | `src/bridge/bearing/COMPONENT_RULE.md` | 非干渉の型付きevent契約 |
| ULTRASOUND | `src/bridge/ultrasound/COMPONENT_RULE.md` | 観測、replay、Work・Anytime・再利用分析 |
| TRAFFIC | `src/bridge/traffic/COMPONENT_RULE.md` | benchmark、統計、fail-closedデータ検証 |
| HEALTHY | `src/bridge/healthy/COMPONENT_RULE.md` | 保存済みartifactの整合監査と再構成 |

上位規範は`docs/ARCHITECTURE_RULE.md`、用語の正本は`docs/WORD_DEFINITION.md`である。
