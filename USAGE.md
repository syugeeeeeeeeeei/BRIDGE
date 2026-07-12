# BRIDGE 使用ガイド

## 1. 本書の目的

本書は、Go版BRIDGEを次の用途で使用するための手順をまとめたものです。

- Goテスト、race detector、静的検査の実行
- Python参照版との互換性・研究準備性評価
- ベンチマーク出力の読み方と評価方法
- CLI経路探索ツールとしての利用
- Goプログラムからのライブラリ利用
- 他言語・外部プログラムとの連携

現行の本番実装はGoです。旧Python版は`others/legacy/bridge_py`に参照実装として保存されています。

---

## 2. 前提環境

### 2.1 必須環境

- Go 1.22以降
- Git

Python-Go互換性評価も実行する場合は、次も必要です。

- Python 3.11以降
- `pytest`

### 2.2 リポジトリルート

以降のコマンドは、`go.mod`が存在するBRIDGEのルートディレクトリで実行します。

````bash
cd BRIDGE
````

### 2.3 初期確認

````bash
go version
go env GOMOD
go mod download
````

`go env GOMOD`がBRIDGEの`go.mod`を示していれば、実行位置は正しい状態です。

---

## 3. テストの実行方法

## 3.1 Goテスト一式

````bash
go test ./...
````

このコマンドは、次のテストを含むGoパッケージ全体を検証します。

- COREの型・Graph・Work不変条件
- GATEの公開API
- TRUSSの依存規則と予算管理
- ANCHORの探索・決定論性
- TRAFFICのベンチマーク・トポロジー生成
- observer有効・無効時の非干渉性

詳細なテスト名と実行時間を表示する場合は、次を使用します。

````bash
go test -v ./...
````

キャッシュを使用せず再実行する場合は、次を使用します。

````bash
go test -count=1 ./...
````

## 3.2 race detector

並行処理時のデータ競合を検査します。

````bash
go test -race ./...
````

BRIDGEのworker並列化やobserver変更を行った場合は、通常テストに加えて必ず実行してください。

## 3.3 静的検査

````bash
go vet ./...
````

主に次の問題を検出します。

- 不正な書式指定
- 到達不能または疑わしいコード
- interfaceやcopy lockに関する問題
-標準ライブラリ利用上の誤り

## 3.4 個別パッケージのテスト

例としてANCHORのみを検証する場合は、次を使用します。

````bash
go test -v ./src/bridge/anchor
````

TRAFFICのみを検証する場合は、次を使用します。

````bash
go test -v ./src/bridge/traffic
````

特定のテスト関数だけを実行する場合は、`-run`を指定します。

````bash
go test -v ./src/bridge/anchor -run TestAnchor
````

## 3.5 Python参照版とのsemantic parity

小規模な固定ケースについて、Python版とGo版の意味的一致を検証します。

````bash
python tests/compatibility/verify.py
````

比較対象は主に次の項目です。

- 経路
- 距離
- 到達可否
- exactフラグ
- quality certification

正常時の出力例です。

````text
Python-Go semantic parity: 10 results matched
````

この検証は小規模Golden Caseの一致確認です。全トポロジーや性能傾向の評価には、次節の研究準備性評価を使用します。

## 3.6 研究準備性評価

Python版とGo版を同一条件で実行し、移植完了基準を定量判定します。

````bash
python tests/compatibility/evaluate_research_readiness.py
````

このコマンドは内部で次を実行します。

1. Go版の研究用ベンチマークを実行
2. Python参照版の対応ベンチマークを実行
3. topology、node数、seed、modeでpaired caseを照合
4. 距離品質、到達可否、Work傾向、網羅率を集計
5. 移植完了しきい値を判定

結果は標準出力にJSONとして表示され、次のファイルにも保存されます。

````text
docs/reports/GO_MIGRATION_READINESS.json
````

終了コードは次の意味です。

| 終了コード | 意味 |
|---:|---|
| `0` | 全必須しきい値を満たした |
| `2` | 一つ以上の必須しきい値を満たさなかった |
| その他 | 実行環境、ビルド、Python依存関係等のエラー |

---

## 4. ベンチマークの実行方法

## 4.1 決定論的ベンチマーク

````bash
go run ./cmd/bridge-benchmark --repetitions 5 > benchmark.csv
````

同一ケースを複数回実行し、経路・距離・Work・solver trace等が一致することを確認します。

デフォルトでは実時間をCSVへ含めません。これは、OSスケジューラ、CPU負荷、GC等による非決定的な変動を結果比較から除外するためです。

主なオプションは次のとおりです。

| オプション | 既定値 | 説明 |
|---|---:|---|
| `--repetitions` | `5` | 同一ケースの反復回数 |
| `--seed` | `1` | グラフ生成seed |
| `--include-timing` | `false` | `time_ms`をCSVへ追加 |

実時間も測定する場合は、次を使用します。

````bash
go run ./cmd/bridge-benchmark \
  --repetitions 5 \
  --include-timing \
  > benchmark_with_timing.csv
````

## 4.2 研究用ベンチマーク

Go版のみの研究用データを生成する場合は、次を使用します。

````bash
go run ./cmd/bridge-research --output go_research.json
````

現行の評価行列は次の構成です。

- トポロジー: open、wall、U-shape、cul-de-sac、disconnected
- 規模: 100、225、400、625、900ノード
- seed: 1、2、3
- mode: balanced
- 比較基準: exact mode

合計75ケースを出力します。

---

## 5. テスト出力と評価方法

## 5.1 `go test`の評価

正常時は各パッケージに対して`ok`が表示されます。

````text
ok   github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core
ok   github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/anchor
ok   github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic
````

`FAIL`が一つでも含まれる場合、現行変更は統合可能な状態ではありません。

## 5.2 決定論的ベンチマークCSV

出力列は次のとおりです。

| 列 | 意味 |
|---|---|
| `nodes` | グラフのノード数 |
| `edges` | グラフの辺数 |
| `seed` | グラフ生成seed |
| `mode` | BRIDGEの実行mode |
| `found` | 経路が見つかったか |
| `distance` | 経路距離 |
| `total_work` | 意味的探索Actionの総数 |
| `work_expanded_nodes` | 展開したノード数 |
| `scheduled_steps` | 現在のworker条件で必要なStep数 |
| `result_sha256` | 非決定的値を除いた結果ダイジェスト |
| `repeatability_runs` | 一致確認に使用した実行回数 |
| `time_ms` | `--include-timing`指定時のみ。実行時間 |

評価時は、次を確認します。

- 同一入力で`result_sha256`が変化しない
- `found=true`の場合に`distance`が有限である
- `total_work`がWork Budgetを超えていない
- 逐次実行では`scheduled_steps`が`total_work`以下である
- exact modeが期待する到達可否と最短距離を返す

`time_ms`は完全一致を求めません。複数回計測し、中央値、p95、ばらつきで評価してください。

## 5.3 研究準備性JSON

`GO_MIGRATION_READINESS.json`の主要項目は次のとおりです。

### `metrics`

| 指標 | 意味 | 現行の完了基準 |
|---|---|---:|
| `valid_path_rate` | 返却経路の妥当率 | `1.000` |
| `connected_found_rate` | 接続グラフで経路を発見した率 | `0.990以上` |
| `mean_distance_ratio` | 平均の`BRIDGE距離 / exact距離` | `1.05以下` |
| `p95_distance_ratio` | Distance Ratioの95パーセンタイル | `1.15以下` |
| `worst_distance_ratio` | 最悪Distance Ratio | `1.35以下` |
| `topology_coverage` | Python-Goでpairedにできたケース率 | `0.90以上` |
| `found_agreement` | Python-Goの到達可否一致率 | `0.99以上` |
| `distance_ratio_spearman` | 品質傾向の順位相関 | 参考指標 |
| `work_spearman` | 問題難易度・Work傾向の順位相関 | 参考指標 |
| `trend_correlation` | 距離品質相関とWork相関の平均 | `0.70以上` |

### `checks`

各しきい値の合否がbooleanで記録されます。

````json
{
  "checks": {
    "valid_path_rate": true,
    "connected_found_rate": true,
    "mean_distance_ratio": true,
    "p95_distance_ratio": true,
    "worst_distance_ratio": true,
    "trend_correlation": true,
    "topology_coverage": true,
    "found_agreement": true
  }
}
````

### `migration_complete`

````json
{
  "migration_complete": true
}
````

`true`は、定義済みの研究投入基準をすべて満たしたことを示します。Python版との全フィールド完全一致を示す値ではありません。

## 5.4 WorkとStepの解釈

- `Work`: 実行された意味的探索Actionの総数
- `Logical Step`: 無制限の並列資源を仮定した依存深度
- `Scheduled Step`: 指定worker数で必要となる実行段階数

逐次実行では、原則としてWorkとScheduled Stepは一致します。複数corridor等を並列化した場合、Workは合算され、Stepは圧縮されます。

実行時間、メモリ使用量、GC、I/OはWorkへ含めません。これらは別の実装性能指標です。

---

## 6. CLI経路探索ツールとしての使用方法

## 6.1 現行CLIの対象

現行の`cmd/bridge`は、指定した幅・高さの重み付きgridを生成し、左上ノードから右下ノードまで探索する簡易CLIです。

任意の外部グラフファイルを読み込むCLIは、現時点では実装されていません。任意グラフを扱う場合は、後述のGo API連携を使用してください。

## 6.2 基本実行

````bash
go run ./cmd/bridge \
  --width 20 \
  --height 20 \
  --mode balanced
````

デフォルト値は次のとおりです。

| オプション | 既定値 | 説明 |
|---|---:|---|
| `--width` | `20` | gridの横幅 |
| `--height` | `20` | gridの高さ |
| `--mode` | `balanced` | `fast`、`balanced`、`quality`、`exact` |
| `--work-budget` | `0` | `0`は自動予算 |

## 6.3 modeの選択

### `fast`

first pathの早期発見を重視します。品質保証よりも応答性を優先する用途向けです。

````bash
go run ./cmd/bridge --width 100 --height 100 --mode fast
````

### `balanced`

探索速度、Work、経路品質の均衡を取る標準modeです。

````bash
go run ./cmd/bridge --width 100 --height 100 --mode balanced
````

### `quality`

初期経路取得後の改善や品質認証を重視します。

````bash
go run ./cmd/bridge --width 100 --height 100 --mode quality
````

### `exact`

BOLTSのexact solverを使用し、最短距離を求めます。品質比較用baselineにも使用します。

````bash
go run ./cmd/bridge --width 100 --height 100 --mode exact
````

## 6.4 Work Budgetの指定

````bash
go run ./cmd/bridge \
  --width 100 \
  --height 100 \
  --mode balanced \
  --work-budget 50000
````

Work Budgetが小さすぎる場合、次の状態になり得ます。

- 経路未発見
- `budget_exhausted=true`
- fallbackやcertificationを実行できない
- exact性を証明できない

## 6.5 バイナリとしてビルド

Linux/macOS:

````bash
go build -o bridge ./cmd/bridge
./bridge --width 50 --height 50 --mode balanced
````

Windows PowerShell:

````powershell
go build -o bridge.exe ./cmd/bridge
.\bridge.exe --width 50 --height 50 --mode balanced
````

## 6.6 CLI出力

結果はJSONで標準出力へ出力されます。

主なフィールドは次のとおりです。

| フィールド | 意味 |
|---|---|
| `path` | `NodeID`の経路配列 |
| `distance` | 経路距離 |
| `found` | 経路発見の成否 |
| `exact` | exact solverによる結果か |
| `solver_name` | 採用されたsolver |
| `work.total_actions` | 総Work |
| `work.scheduled_steps` | Scheduled Step |
| `quality_certified` | 品質が認証されたか |
| `certified_ratio` | 認証済み上界比 |
| `first_path_work` | 最初の経路発見までのWork |
| `fallback_used` | fallback系taskを使用したか |
| `budget_exhausted` | Work Budgetを使い切ったか |
| `deadline_exceeded` | deadlineを超えたか |
| `solver_trace` | TRUSSが実行したtask履歴 |

ファイルへ保存する場合はリダイレクトします。

````bash
go run ./cmd/bridge \
  --width 50 \
  --height 50 \
  --mode quality \
  > result.json
````

---

## 7. Goプログラムから利用する方法

## 7.1 基本構成

外部Goプログラムは、GATEの公開APIだけを呼び出してください。

````text
外部プログラム → src/bridge/gate → bridge/truss → ANCHOR / BOLTS
````

ANCHOR、BOLTS、TRUSSのprivate実装を直接呼び出す利用方法は、正式な外部APIではありません。

## 7.2 モジュールの取得

BRIDGEを公開Gitリポジトリから利用する場合は、利用側プロジェクトで次を実行します。

````bash
go get github.com/syugeeeeeeeeeei/BRIDGE
````

ローカルのBRIDGEを使用する場合は、利用側`go.mod`へ`replace`を設定します。

````go
module example.com/my-route-app

go 1.22

require github.com/syugeeeeeeeeeei/BRIDGE v0.0.0

replace github.com/syugeeeeeeeeeei/BRIDGE => ../BRIDGE
````

## 7.3 最小コード例

````go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
)

func main() {
    graph := core.NewAdjacencyGraph(4, false)

    must(graph.AddEdge(0, 1, 1.0))
    must(graph.AddEdge(1, 2, 1.0))
    must(graph.AddEdge(2, 3, 1.0))
    must(graph.AddEdge(0, 3, 10.0))

    request := core.RouteRequest{
        Source:  0,
        Target:  3,
        Mode:    core.ModeBalanced,
        Workers: 1,
        Seed:    1,
    }

    result, err := gate.New(nil).Route(context.Background(), graph, request)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("found=%v distance=%v path=%v work=%d\n",
        result.Found,
        result.Distance,
        result.Path,
        result.TotalWork(),
    )
}

func must(err error) {
    if err != nil {
        panic(err)
    }
}
````

期待される経路は、概ね次です。

````text
0 → 1 → 2 → 3
````

## 7.4 有向グラフ

````go
graph := core.NewAdjacencyGraph(4, true)
_ = graph.AddEdge(0, 1, 1)
_ = graph.AddEdge(1, 2, 1)
_ = graph.AddEdge(2, 3, 1)
````

有向グラフでは、逆方向の辺は自動追加されません。

## 7.5 座標の設定

ANCHORのgeometric系戦略で利用できる座標を設定します。

````go
_ = graph.SetPosition(0, core.Point{X: 0, Y: 0})
_ = graph.SetPosition(1, core.Point{X: 1, Y: 0})
_ = graph.SetPosition(2, core.Point{X: 2, Y: 0})
_ = graph.SetPosition(3, core.Point{X: 3, Y: 0})
````

座標は必須ではありません。座標がない場合は、利用可能な非幾何戦略へ切り替わります。

## 7.6 Work Budget

````go
budget := uint64(10000)
request.WorkBudget = &budget
````

`WorkBudget`はポインタです。`nil`の場合はTRUSSが自動予算を使用します。

## 7.7 Deadline

Go APIでは`context.Context`による取消と、`RouteRequest`のdeadline指定を利用できます。

````go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

result, err := gate.New(nil).Route(ctx, graph, request)
````

`DeadlineMS`を使用する場合です。

````go
deadlineMS := 100.0
request.DeadlineMS = &deadlineMS
````

外部Goプログラムでは、原則として`context.WithTimeout`を優先してください。

## 7.8 Quality Target

````go
maxRatio := 1.05
request.Mode = core.ModeQuality
request.MaxSuboptimality = &maxRatio
````

これは、認証可能な場合に「最適距離の1.05倍以内」を品質目標として表します。

## 7.9 結果の検証

返却された経路がGraph上で有効か確認できます。

````go
if result.Found {
    distance := core.PathDistance(graph, result.Path)
    if math.IsInf(distance, 1) {
        log.Fatal("invalid path")
    }
}
````

研究・本番利用では、少なくとも次を監視してください。

- `Found`
- `Distance`
- `ErrorCode`
- `BudgetExhausted`
- `DeadlineExceeded`
- `QualityCertified`
- `TotalWork()`
- `Work.ScheduledSteps`

---

## 8. ULTRASOUND observerを利用する方法

## 8.1 observerなし

本番の既定構成です。

````go
runner := gate.New(nil)
````

`nil`は非観測構成として扱われます。

## 8.2 メモリobserver

開発・テストでイベントを収集する場合は、`src/bridge/ultrasound`の公開observerを使用します。

実際の型とconstructorは`src/bridge/ultrasound/observer.go`を確認してください。

利用時も、外部プログラムはobserverをGATEへ注入します。

````text
ULTRASOUND observer → gate.New(observer) → Route(...)
````

observerの有効・無効によって、経路、距離、Work、tie-breakingが変化してはいけません。

---

## 9. 他言語・外部プログラムと連携する方法

## 9.1 推奨方式

現行版にはRESTサーバー、gRPCサーバー、C ABI、共有ライブラリ用APIは標準搭載されていません。

外部連携は、次のいずれかで行います。

1. GoアプリケーションへBRIDGEを直接組み込む
2. BRIDGEを呼び出す専用Goサービスを作成する
3. JSON LinesまたはRPCを受け付けるラッパーCLIを作成する

本格運用では、2のサービス方式を推奨します。

## 9.2 HTTPサービスの最小例

次は、外部プログラムからHTTP/JSONで利用するための簡易ラッパー例です。

````go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
)

type edgeInput struct {
    From   core.NodeID `json:"from"`
    To     core.NodeID `json:"to"`
    Weight float64     `json:"weight"`
}

type routeInput struct {
    Nodes    int               `json:"nodes"`
    Directed bool              `json:"directed"`
    Edges    []edgeInput       `json:"edges"`
    Request  core.RouteRequest `json:"request"`
}

func main() {
    runner := gate.New(nil)

    http.HandleFunc("/route", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "POST required", http.StatusMethodNotAllowed)
            return
        }

        var input routeInput
        if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        graph := core.NewAdjacencyGraph(input.Nodes, input.Directed)
        for _, edge := range input.Edges {
            if err := graph.AddEdge(edge.From, edge.To, edge.Weight); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
        }

        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        result, err := runner.Route(ctx, graph, input.Request)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(result)
    })

    log.Fatal(http.ListenAndServe(":8080", nil))
}
````

リクエスト例です。

````json
{
  "schema_version": "bridge.route.request.v2",
  "graph": {
    "type": "inline",
    "directed": false,
    "nodes": [
      {"id": 0},
      {"id": 1},
      {"id": 2},
      {"id": 3}
    ],
    "edges": [
      {"from": 0, "to": 1, "weight": 1.0},
      {"from": 1, "to": 2, "weight": 1.0},
      {"from": 2, "to": 3, "weight": 1.0},
      {"from": 0, "to": 3, "weight": 10.0}
    ]
  },
  "route": {
    "source": 0,
    "target": 3,
    "route_mode": "balanced",
    "logical_worker_count": 1,
    "seed": 1
  },
  "observation_config": {
    "level": "off"
  }
}
````

呼び出し例です。

````bash
curl -X POST http://localhost:8080/route \
  -H "Content-Type: application/json" \
  --data-binary @request.json
````

## 9.3 Pythonからの連携

上記HTTPサービスを利用する場合、Pythonからは標準HTTPクライアントで呼び出せます。

````python
import json
import urllib.request

payload = {
    "schema_version": "bridge.route.request.v2",
    "graph": {
        "type": "inline",
        "directed": False,
        "nodes": [
            {"id": 0},
            {"id": 1},
            {"id": 2},
            {"id": 3},
        ],
        "edges": [
            {"from": 0, "to": 1, "weight": 1.0},
            {"from": 1, "to": 2, "weight": 1.0},
            {"from": 2, "to": 3, "weight": 1.0},
            {"from": 0, "to": 3, "weight": 10.0},
        ],
    },
    "route": {
        "source": 0,
        "target": 3,
        "route_mode": "balanced",
        "logical_worker_count": 1,
        "seed": 1,
    },
    "observation_config": {
        "level": "off",
    },
}

request = urllib.request.Request(
    "http://localhost:8080/route",
    data=json.dumps(payload).encode("utf-8"),
    headers={"Content-Type": "application/json"},
    method="POST",
)

with urllib.request.urlopen(request) as response:
    result = json.load(response)

print(result["path"])
print(result["path_cost"])
````

## 9.4 Node.js / TypeScriptからの連携

````typescript
const response = await fetch("http://localhost:8080/route", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    schema_version: "bridge.route.request.v2",
    graph: {
      type: "inline",
      directed: false,
      nodes: [{ id: 0 }, { id: 1 }, { id: 2 }, { id: 3 }],
      edges: [
        { from: 0, to: 1, weight: 1.0 },
        { from: 1, to: 2, weight: 1.0 },
        { from: 2, to: 3, weight: 1.0 },
        { from: 0, to: 3, weight: 10.0 },
      ],
    },
    route: {
      source: 0,
      target: 3,
      route_mode: "balanced",
      logical_worker_count: 1,
      seed: 1,
    },
    observation_config: {
      level: "off",
    },
  }),
});

if (!response.ok) {
  throw new Error(await response.text());
}

const result = await response.json();
console.log(result.path, result.distance);
````

## 9.5 連携時のNodeID管理

BRIDGE内部の`NodeID`は`uint32`です。

外部システムで文字列IDを使用する場合は、GATEへ渡す前に整数へ変換し、結果返却時に元のIDへ戻してください。

例です。

````text
"Tokyo"  → 0
"Nagoya" → 1
"Osaka"  → 2
````

対応表は外部アプリケーション側で保持します。

## 9.6 大規模グラフを扱う際の注意

現行の`AdjacencyGraph`は、構築しやすい可変隣接リスト実装です。

大規模・高頻度運用では、次を事前に評価してください。

- グラフ構築時間
- peak RSS
- queryごとのallocation
- GC停止時間
- 同時query数
- observerのオーバーヘッド
- Work Budgetとdeadlineの設定

静的な大規模グラフでは、将来のCSR実装または専用immutable Graph実装を使用する方が効率的です。

---

## 10. 実運用前の推奨確認

実運用へ投入する前に、最低限次を実行してください。

````bash
go test -count=1 ./...
go test -race ./...
go vet ./...
python tests/compatibility/verify.py
python tests/compatibility/evaluate_research_readiness.py
````

さらに、実際に利用するグラフに対して次を確認してください。

- 到達可能なqueryで`Found`が安定してtrueになる
- exact baselineに対するDistance Ratio
- p50、p95、p99 latency
- peak memory
- WorkとScheduled Step
- Budget Exhausted率
- Deadline Exceeded率
- 同一入力のrepeatability
- worker数変更時の性能と決定論性

研究用途では、グラフ、query、seed、mode、Work Budget、BRIDGE version、Go version、OS、CPUを実験結果と同時に保存してください。

---

## 9. ULTRASOUND Traceの保存とReplay

Traceは探索の棋譜であり、外部の視覚シミュレーターが`events.jsonl`だけを読み込んで探索状態を再構成できます。

### 9.1 Traceを記録する

Windows PowerShell:

````powershell
go run .\cmd\bridge-ultrasound record `
  --width 70 `
  --height 70 `
  --seed 1 `
  --route-mode balanced `
  --ultrasound-mode trace `
  --output .\ultrasound-runs `
  --run-id grid-4900-seed-1
````

出力:

````text
ultrasound-runs/grid-4900-seed-1/
├── manifest.json
├── events.jsonl
├── metrics.json
├── result.json
└── validation.json
````

### 9.2 Traceを検証する

````powershell
go run .\cmd\bridge-ultrasound validate `
  --input .\ultrasound-runs\grid-4900-seed-1
````

### 9.3 Replay状態をJSONへ書き出す

````powershell
go run .\cmd\bridge-ultrasound replay `
  --input .\ultrasound-runs\grid-4900-seed-1 `
  --output .\ultrasound-runs\grid-4900-seed-1\replay_state.json
````

`replay_state.json`にはfrontier、展開済みnode、評価済みedge、distance、parent、candidate path、Workおよび最終component状態が含まれます。

外部表示アプリは`docs/architecture/ULTRASOUND_TRACE_DATA_CONTRACT.md`に従って`events.jsonl`を順次適用してください。

## Scenario benchmark (v0.12.1)

```powershell
.\bridge.exe benchmark validate .\scenarios\operational.yaml
.\bridge.exe benchmark list .\scenarios\operational.yaml
.\bridge.exe benchmark run .\scenarios\operational.yaml
.\bridge.exe benchmark run .\scenarios\operational.yaml --format json --output result.json
.\bridge.exe benchmark run .\scenarios\operational.yaml --format csv > result.csv
```

Supported output formats are `console`, `json`, `jsonl`, and `csv`. Existing output files are rejected unless `--overwrite` is specified. Acceptance failures return exit code `5` while still emitting the benchmark result.

## ObservationとTrace（v0.13.1）

Route Requestの`observation.mode`には`off`、`summary`、`metrics`、`trace`、`debug`を指定できます。Traceは明示的な出力先を指定した場合だけ保存されます。

```powershell
.\bridge.exe route --request .\route-request.json --trace-output .\trace.jsonl
```

既存Traceを上書きする場合は`--trace-overwrite`を指定します。

```powershell
.\bridge.exe benchmark run .\scenarios\operational.yaml --trace-dir .\traces
```

Benchmark Traceは`<trace-dir>/<scenario-id>/seed-<seed>-rep-<rep>.jsonl`へ分離されます。Traceは標準出力へ混入しません。


# 11. SDK利用（v0.14.0）

## Python

```bash
cd src/sdk/python
python -m pip install .
```

```python
from bridge_sdk import BridgeClient
client = BridgeClient.local()
response = client.route(request)
print(response.result)
```

## TypeScript

```bash
cd src/sdk/typescript
npm run build
```

```typescript
import { BridgeClient } from "@bridge/route-sdk";
const client = await BridgeClient.local();
const response = await client.route(request);
console.log(response.result);
```

両SDKは実行環境に対応した同梱バイナリを使用します。解決順は明示パス、`BRIDGE_BINARY`、同梱版、PATHです。ネットワークからの自動ダウンロード、暗黙のTrace保存、APIサーバー提供は行いません。APIサーバーはSDKをFastAPI、Flask、Express、Fastify等へ組み込んで利用者が構築します。
