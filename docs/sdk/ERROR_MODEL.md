# SDKエラーモデル

CLI終了コード2、3、4、5、10を、それぞれValidation、I/O、Timeout、Acceptance、Internal例外へ変換する。終了コード0でstderrが存在する場合は成功結果とwarningを返す。不正なstdout JSONまたはschema versionはProtocol Errorとする。
