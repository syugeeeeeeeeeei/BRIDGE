# BRIDGE Documentation Index

> Status: Normative  
> Applies To: BRIDGE v0.15.x  
> Owner: BRIDGE maintainers

## 文書の区分

- `Normative`: 現行仕様の正本です。
- `Informative`: 利用方法や補足説明です。
- `Historical`: 過去の判断・実装記録であり、現行仕様の根拠には使用しません。
- `Deprecated`: 参照禁止または廃止予定です。

## 優先順位

1. `src/contracts/json-schema/`の機械検証可能なSchema
2. `docs/contracts/`の規範契約
3. `docs/ARCHITECTURE_RULE.md`
4. `docs/WORD_DEFINITION.md`
5. 各コンポーネントの`COMPONENT_RULE.md`
6. `docs/project-knowledge/`の長期原則・判断軸
7. `docs/operations/`の利用手順
8. `docs/reports/`の履歴・評価資料

SchemaとMarkdown契約の不一致は許容しません。不一致を検出した場合は、Producerの実装、Schema、契約文書を同一変更で修正します。

## 現行の規範文書

- [Architecture Rule](ARCHITECTURE_RULE.md)
- [Terminology](WORD_DEFINITION.md)
- [Benchmark Scenario Contract](contracts/BENCHMARK_SCENARIO_CONTRACT.md)
- [Benchmark Artifact Contract](contracts/BENCHMARK_ARTIFACT_CONTRACT.md)
- [Trace Artifact Contract](contracts/TRACE_ARTIFACT_CONTRACT.md)
- [Graph Snapshot Contract](contracts/GRAPH_SNAPSHOT_CONTRACT.md)
- [Simulation Artifact Contract](contracts/SIMULATION_ARTIFACT_CONTRACT.md)
- [HEALTHY Result Contract](contracts/HEALTHY_RESULT_CONTRACT.md)
- [Documentation Rule](repository/DOCUMENTATION_RULE.md)
- [Component Rule Index](components/BRIDGE_COMPONENT_RULE_INDEX.md)
- [Responsibility Refactoring Rules](architecture/RESPONSIBILITY_REFACTORING_RULES.md)
- [Project Knowledge Index](project-knowledge/00_PROJECT_KNOWLEDGE_INDEX.md)
- [Integrated Project Knowledge](project-knowledge/BRIDGE_PROJECT_KNOWLEDGE.md)

## 利用手順

- [CLI Usage](operations/CLI_USAGE.md)
- [Benchmark Operation](operations/BENCHMARK_OPERATION.md)
- [Simulator Operation](operations/SIMULATOR_OPERATION.md)
- [Development Operation](operations/DEVELOPMENT_OPERATION.md)

## 履歴資料

`docs/reports/current/`には現行版の状態説明だけを置き、`docs/reports/archive/`配下は履歴資料として隔離します。archiveは現行実装の仕様判断には使用しません。

## SDK

- [SDKドキュメント](sdk/README.md)
