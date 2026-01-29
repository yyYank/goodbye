# goodbye

<p align="center">
<img align="center" src="https://github.com/yyYank/goodbye/blob/main/logo.png" width=400 height="auto" />
</p>

---


`goodbye` は、macOS を中心とした **開発環境のパッケージ管理を段階的に整理・移行するための CLI ツール**です。  
「一気に置き換えない」「dry-run 前提」「依存の所在を明確にする」を基本思想としています。

## 特徴

- Homebrew 環境の **export / import**
- Homebrew → **mise / asdf** への段階的移行（brew --mise, brew --asf）
- ユーザー定義コマンドによる柔軟な取得
- 将来拡張（uv等）を前提とした構造

---

## 技術スタック

- 言語: **Go**
- CLI フレームワーク: **cobra**
- 設定: **TOML**（`~/.goodbye.toml`）
- 配布形態: 単一バイナリ

---

## インストール

```bash
go install github.com/yyYank/goodbye@latest
````

またはリポジトリを clone して

```bash
go build
```

---

## コマンド概要

```text
goodbye
├── export
│   └── --brew
│   └── --mise
├── import
│   └── --brew
│   └── --mise
└── brew
    ├── --mise
    └── --asdf
```

すべてのコマンドは **デフォルトで dry-run** です。
実際に変更を行う場合は `--apply` を明示的に指定します。

---

## `goodbye export --brew`

現在の Homebrew 環境を **最小かつ意図的な粒度**で書き出します。

### デフォルトの取得内容

* formula: `brew list --installed-on-request`
* cask: `brew list --cask`
* tap: `brew tap`

### 実行例

```bash
goodbye export --brew --dir ~/goodbye-export
```

### 出力例

```text
goodbye-export/
├── formula.txt   # formula 一覧
├── cask.txt      # cask 一覧
└── tap.txt       # tap 一覧
```

---

## `goodbye import --brew`

`goodbye export --brew` で作成したファイル群を使い、
**新しい PC に Homebrew 環境を再構築**します。

### 実行例

```bash
# dry-run（デフォルト）
goodbye import --brew --dir ~/goodbye-export

# 実行
goodbye import --brew --dir ~/goodbye-export --apply
```

### オプション

* `--only formula|cask|tap`
* `--skip-taps`
* `--continue`（エラーがあっても継続）

---

## `goodbye goodbyebrew --mise`

Homebrew で管理しているツールのうち、
**mise で管理可能なものを抽出し、段階的に移行**します。

### 動作フロー

1. Homebrew formula 一覧を取得
2. 正規化（例: `python@3.12 → python`）
3. `mise registry` と突合
4. 移行候補を表示
5. confirm（y/N）
6. `mise install`
7. 簡易疎通確認
8. 成功したもののみ `brew uninstall`

### 実行例

```bash
# 確認のみ
goodbye brew --mise

# 実行
goodbye brew --mise --apply
```

---

## `goodbye brew --asdf`

Homebrew → asdf 移行を補助します。

### 注意点

* asdf は **version 指定が必須**になりやすいため、
  `.tool-versions` を基準に動作します
* brew の一覧だけからの完全自動移行は行いません

### 実行例

```bash
goodbye brew --asdf
goodbye brew --asdf --apply
```

---

## 設定ファイル（`~/.goodbye.toml`）

`goodbye` の取得挙動は `~/.goodbye.toml` によってカスタマイズできます。

### 基本例（デフォルトと同等）

```toml
[brew.export]
formula_cmd = "brew list --installed-on-request"
cask_cmd    = "brew list --cask"
tap_cmd     = "brew tap"
```

### 例: formula をすべて取得する（非推奨だが可能）

```toml
[brew.export]
formula_cmd = "brew list"
cask_cmd    = "brew list --cask"
tap_cmd     = "brew tap"
```

### 例: 出力を整形して保存する（パイプ可）

```toml
[brew.export]
formula_cmd = "brew list --installed-on-request | sort -u"
cask_cmd    = "brew list --cask | sort -u"
tap_cmd     = "brew tap | sort -u"
```

> 注意
> パイプや複合コマンドを含む場合、`goodbye` はシェル経由でコマンドを実行します。
> **信頼できるコマンドのみ**を設定してください。

### 適用範囲

* `goodbye export --brew` が `brew.export.*_cmd` を参照します
* `goodbye import --brew` は export 済みのファイルを入力として使用します

---

## 設計思想

* dry-run をデフォルトにする
* 「全部を一気に置き換えない」
* 依存の所在を常に見える化する
* GUI アプリ（cask）は無理に移行しない
* ランタイム管理は専用ツールへ切り出す

---

## 想定される利用フロー

### 新 PC への移行

```bash
# 旧 PC
goodbye export --brew

# 新 PC
goodbye import --brew
```

### brew 依存を削減しmiseへ移行(tap以外)

```bash
goodbye goodbyebrew --mise
goodbye goodbyebrew --mise --apply
```

---

## 将来拡張（予定）

* `goodbye goodbyebrew --uv`
* toml manifest 対応

---

## 注意事項

* GUI アプリ（cask）に完全な移行先はないためこのコマンドの対象外です。nixなどを選択肢としてください
* DBなどの常駐サービスは goodbyebrew 対象外を推奨します
* 実行は自己責任で行ってください

---

## ライセンス

Apache 2
