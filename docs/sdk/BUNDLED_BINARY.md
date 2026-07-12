# 同梱バイナリ方針

対応対象はLinux amd64/arm64、Windows amd64、macOS amd64/arm64である。各SDKに全バイナリを静的ファイルとして含め、実行時にOSとCPUで選択する。自動ダウンロードと自動更新は行わない。`binary-manifest.json`にSHA-256とBRIDGE versionを記録する。
