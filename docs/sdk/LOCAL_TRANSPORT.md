# Local Transport

SDKは`bridge route`を`shell=false`で起動し、Route Request JSONをstdinへ1件渡す。stdoutはRoute Result JSON専用、stderrはwarning/error専用とする。SDKは暗黙にファイルを生成しない。
