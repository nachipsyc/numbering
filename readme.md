# numbering

ディレクトリ内の画像・RAW ファイルを、**共通の接頭辞 + 連番**にまとめてリネームする CLI ツールです。

名前順・更新日時順に加えて、**Exif の撮影日時順**で並べてから採番できるので、複数台のカメラやコピーで順番がばらけたカットを時系列に並べ直すのに使えます。

## 仕組み

1. 対象ディレクトリ(`-dir`)内のファイルから、`-exts` で指定したグループの拡張子だけを抽出する
2. `-sort` の指定に従って並べ替える（`exif` 失敗時は更新日時にフォールバック）
3. `-start` から始まる連番を `-width` 桁・ゼロ埋め有無(`-pad`)で振り、`接頭辞 + 連番 + 元の拡張子` にリネームする
4. リネーム結果を `logs/numbering_log_<タイムスタンプ>.json` に保存する

### 対応拡張子グループ（`-exts`）

| グループ | 拡張子 |
| --- | --- |
| `jpeg` | `.jpeg` `.jpg` |
| `raw` | `.cr2` `.cr3` `.nef` `.arw` `.raf` `.rw2` `.orf` `.dng` |
| `heif` | `.heic` `.heif` |

カンマ区切りで複数指定できます（例: `-exts=jpeg,raw`）。

## 使い方

```bash
# 例: photos ディレクトリの JPEG を撮影日時順で trip_001.jpg ... にリネーム
go run numbering.go -dir=<ディレクトリ> -prefix=trip_ -sort=exif -start=1 -width=3

# 直前のリネームを取り消す（最新のログをもとに元の名前へ戻す）
go run numbering.go -undo
```

### オプション

| フラグ | デフォルト | 説明 |
| --- | --- | --- |
| `-dir` | （必須） | 対象ディレクトリのパス |
| `-prefix` | （必須） | ファイル名の共通接頭辞 |
| `-sort` | `name` | 並び順。`name`(名前順) / `time`(更新日時順) / `exif`(撮影日時順) |
| `-reverse` | `false` | 並び順を逆にする |
| `-start` | `1` | 採番の開始番号 |
| `-width` | `3` | 採番の桁数 |
| `-pad` | `true` | 桁数までゼロ埋めするか |
| `-exts` | `jpeg` | 対象拡張子グループ（`jpeg,raw,heif` をカンマ区切り） |
| `-undo` | | 最新のログをもとにリネームを取り消す |

## ログと取り消し

リネームのたびに、対象ディレクトリと「元の名前 / 新しい名前」の対応が `logs/` 配下の JSON に保存されます。`-undo` は **最新のログファイル**を読み込み、新しい名前から元の名前へ戻します。

## 出力例

```
success: DSCF0123.JPG -> trip_001.JPG
success: DSCF0119.JPG -> trip_002.JPG
リネームログを保存しました: /path/to/numbering/logs/numbering_log_20260629_073800.json
Completed: 2 files renamed
```

## 依存ライブラリ

- [`github.com/rwcarlsen/goexif`](https://github.com/rwcarlsen/goexif) — Exif の撮影日時取得（`-sort=exif`）
