# BRIDGE 実運用想定ベンチマーク — Windows実行手順

## 前提

- Go 1.22以上をインストール
- PowerShellを使用
- BRIDGE Go v0.9.1 Operational BenchmarkのZIPを展開済み

## 1. 全テスト

```powershell
cd .\BRIDGE_Go_v0.9.1

go test .\...
```

詳細表示:

```powershell
go test -v .\...
```

キャッシュを使わず再実行:

```powershell
go test -count=1 .\...
```

## 2. 実運用想定ベンチマーク

今回と同じ条件です。

```powershell
go run .\cmd\bridge-operational-benchmark `
  --sizes 100,500,1000,5000,10000,20000 `
  --seeds 1,2,3 `
  | Out-File -Encoding utf8 .\bridge_operational_benchmark_raw.csv
```

PowerShell 7でBOMなしUTF-8を使う場合:

```powershell
go run .\cmd\bridge-operational-benchmark `
  --sizes 100,500,1000,5000,10000,20000 `
  --seeds 1,2,3 `
  | Set-Content -Encoding utf8NoBOM .\bridge_operational_benchmark_raw.csv
```

## 3. 実行時間も測定する

```powershell
Measure-Command {
  go run .\cmd\bridge-operational-benchmark `
    --sizes 100,500,1000,5000,10000,20000 `
    --seeds 1,2,3 `
    | Out-File -Encoding utf8 .\bridge_operational_benchmark_raw.csv
}
```

## 4. 軽量確認

```powershell
go run .\cmd\bridge-operational-benchmark `
  --sizes 100,1000,5000 `
  --seeds 1 `
  | Out-File -Encoding utf8 .\bridge_operational_smoke.csv
```

## 5. より厳密な反復

seedを10種類へ増やします。

```powershell
go run .\cmd\bridge-operational-benchmark `
  --sizes 100,500,1000,5000,10000,20000 `
  --seeds 1,2,3,4,5,6,7,8,9,10 `
  | Out-File -Encoding utf8 .\bridge_operational_benchmark_10seeds.csv
```

## 6. CSVの簡易確認

```powershell
$data = Import-Csv .\bridge_operational_benchmark_raw.csv
$data.Count
$data | Group-Object solver | Select-Object Name, Count
```

BRIDGEだけを表示:

```powershell
$data | Where-Object solver -eq "bridge" |
  Select-Object requested_nodes, topology, seed, found, distance_ratio, total_work, time_ms |
  Format-Table -AutoSize
```

平均実行時間:

```powershell
$data |
  Where-Object exact_found -eq "true" |
  Group-Object solver |
  ForEach-Object {
    [PSCustomObject]@{
      Solver = $_.Name
      MeanTimeMS = ($_.Group | Measure-Object time_ms -Average).Average
      MeanWork = ($_.Group | Measure-Object total_work -Average).Average
    }
  } | Format-Table -AutoSize
```

## 7. 既存の5,000ノード固定ベンチマーク

```powershell
go run .\cmd\bridge-topology-5000 |
  Out-File -Encoding utf8 .\bridge_topology_5000.csv
```

## 注意

- `go run`にはコンパイル時間が含まれます。アルゴリズムの`time_ms`列には各探索の内部計測値が記録されます。
- Windows Defender、電源プラン、CPUブースト、バックグラウンド処理により実行時間は変動します。
- 時間比較は平均だけでなく中央値、p95、複数seedを使用してください。
- 離散Workは実行環境の影響を受けにくいため、主要比較指標として併用してください。
