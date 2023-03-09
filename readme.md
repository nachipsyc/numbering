# About

- 特定のディレクトリ内の JPEG ファイルのファイル名をまとめて変更する
  - "接辞後+ 0 からの連番".jpeg となる
  - 元のファイルを残したまま、名称だけ変更されたファイルが同フォルダ内に作成される

# Use

go run resize.go *1 *2

- \*1…ファイル取得元兼書き出し先ディレクトリの相対パス
- \*2…ファイル名の接辞語
  <br>
  （省略可能）

## Sample

- Desktop にある"hoge"フォルダの中にあるファイルを同フォルダに書き出す

  - "working1", "working2", "working3", ...

  <br>
  $ cd fuga/fuga/rename
  <br>
  $ go run rename.go ../../../Desktop/hoge working
