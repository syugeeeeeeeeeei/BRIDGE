# 互換性確認

SDKはアプリケーションversionの完全一致ではなく、Capabilitiesに必要なRoute Response Schemaが含まれるかを確認します。

正式オプション:

- Python: `verify_compatibility`
- TypeScript: `verifyCompatibility`

検証を無効化すると、非対応Schemaを実行時まで検出できないため、通常は無効化しないでください。
