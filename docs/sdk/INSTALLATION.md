# SDKインストール

## Python

対応: Python 3.10以上。

リポジトリから開発用にインストール:

````bash
python -m pip install -e ./src/sdk/python
````

## TypeScript

対応: Node.js 18以上。

````bash
cd src/sdk/typescript
npm install
npm run build
````

SDKには対応プラットフォーム用BRIDGEバイナリを同梱できます。探索順序と検証規則は`BUNDLED_BINARY.md`を参照してください。
