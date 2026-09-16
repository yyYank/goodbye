# goodbye

<p align="center">
<img align="center" src="https://github.com/yyYank/goodbye/blob/main/logo.png" width=400 height="auto" />
</p>

---


`goodbye` は、macOS を中心とした **開発環境のパッケージ管理を段階的に整理・移行するための CLI ツール**です。  
「一気に置き換えない」「dry-run 前提」「依存の所在を明確にする」を基本思想としています。

## 特徴

- Homebrew 環境の **export / import**
- npm / pnpm / Go / Cargo / uv の **export / import**
- Homebrew → **mise / asdf** への段階的移行（brew --mise, brew --asf）
- **dotfiles リポジトリの同期・インポート**
- ユーザー定義コマンドによる柔軟な取得

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
│   ├── brew
│   ├── mise
│   ├── npm
│   ├── pnpm
│   ├── go
│   ├── cargo
│   └── uv
├── import
│   ├── brew
│   ├── mise
│   ├── npm
│   ├── pnpm
│   ├── go
│   ├── cargo
│   ├── uv
│   └── dotfiles [--url <repository-url>]
├── status
├── edit
└── brew
    ├── --mise
    └── --asdf
```

すべてのコマンドは **デフォルトで dry-run** です。
実際に変更を行う場合は `--apply` を明示的に指定します。

---

## pnpm / Go / Cargo / uv の移行

ツール名とインストール済みバージョンを保存し、別環境で復元します。
対象のパッケージマネージャーは事前にインストールしてください。

| コマンドの対象 | 出力ファイル | 保存形式 |
| --- | --- | --- |
| `pnpm` | `pnpm-global.txt` | `@scope/name@1.2.3` または `name@1.2.3` |
| `go` | `go-tools.txt` | `example.com/tools/cmd/tool@v1.2.3` |
| `cargo` | `cargo-tools.txt` | `crate-name@1.2.3` |
| `uv` | `uv-tools.txt` | `tool-name==1.2.3` |

```bash
# pnpm を go / cargo / uv に置き換えて利用できます
goodbye export pnpm --dir ~/goodbye-export
goodbye export pnpm --dir ~/goodbye-export --apply
goodbye import pnpm --dir ~/goodbye-export
goodbye import pnpm --dir ~/goodbye-export --apply --continue
```

- export の dry-run は一覧を取得・表示しますが、ファイルは作りません。
- import の dry-run はファイルを検証して予定コマンドを表示し、インストールしません。
- import は空行・`#` コメント・重複を無視し、全行を検証してから実行します。
- `--continue` は失敗後も続行しますが、1件でも失敗するとエラー終了します。
- `--verbose` はインストール出力を表示します。失敗時の出力は指定なしでも表示します。

対応範囲・除外条件は [USAGE.md](USAGE.md#pnpm--go--cargo--uv-の移行) を参照してください。

## `goodbye export brew`

現在の Homebrew 環境を **最小かつ意図的な粒度**で書き出します。

### デフォルトの取得内容

* formula: `brew list --installed-on-request`
* cask: `brew list --cask`
* tap: `brew tap`

### 実行例

```bash
goodbye export brew --dir ~/goodbye-export
```

### 出力例

```text
goodbye-export/
├── formula.txt   # formula 一覧
├── cask.txt      # cask 一覧
└── tap.txt       # tap 一覧
```

---

## `goodbye import brew`

`goodbye export brew` で作成したファイル群を使い、
**新しい PC に Homebrew 環境を再構築**します。

### 実行例

```bash
# dry-run（デフォルト）
goodbye import brew --dir ~/goodbye-export

# 実行
goodbye import brew --dir ~/goodbye-export --apply
```

### オプション

* `--only formula|cask|tap`
* `--skip-taps`
* `--continue`（エラーがあっても継続）

---

## `goodbye brew --mise`

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

## `goodbye edit`

`~/.goodbye.toml` 設定ファイルを**お好みのエディタで開きます**。

### 実行例

```bash
# デフォルトエディタで開く（$EDITOR 環境変数、なければ vim）
goodbye edit

# エディタを指定して開く
goodbye edit --editor vim
goodbye edit --editor emacs
goodbye edit --editor nano
goodbye edit --editor code
```

### オプション

* `--editor, -e` : 使用するエディタを指定（デフォルト: `$EDITOR` または `vim`）

### 動作

1. `~/.goodbye.toml` が存在しない場合は自動的に作成
2. 指定されたエディタ（またはデフォルト）で設定ファイルを開く

---

## `goodbye import dotfiles`

dotfiles リポジトリから、**設定ファイルやディレクトリをホームディレクトリに配置**します。
`--url` オプションを指定すると、リポジトリのクローン・同期も同時に行います。

### 実行例

```bash
# dry-run（デフォルト）
goodbye import dotfiles

# 実行
goodbye import dotfiles --apply

# リポジトリをクローン/同期してからインポート
goodbye import dotfiles --url https://github.com/username/dotfiles --apply

# カスタムパスを指定
goodbye import dotfiles --url https://github.com/username/dotfiles --path ~/my-dotfiles --apply

# コピーモード（シンボリックリンクの代わり）
goodbye import dotfiles --apply --copy

# バックアップなし
goodbye import dotfiles --apply --no-backup

# エラーがあっても継続
goodbye import dotfiles --apply --continue
```

### 動作

1. `--url` が指定された場合、リポジトリをクローンまたは pull
2. `~/.goodbye.toml` の `[dotfiles]` 設定を読み込み
3. 指定されたファイル（`.zshrc`, `.vimrc` など）をホームディレクトリに配置
4. 指定されたディレクトリ（`directories` 設定）をホームディレクトリに配置
5. デフォルトでシンボリックリンクを作成（`--copy` でコピーモード）
6. 既存ファイル/ディレクトリがある場合はバックアップを作成（`--no-backup` で無効化）

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

### dotfiles 設定例

```toml
[dotfiles]
repository = "https://github.com/username/dotfiles"
local_path = "~/.dotfiles"
source_dir = "macOS"  # リポジトリ内のサブディレクトリ（オプション）
files = [".zshrc", ".vimrc", ".tmux.conf", ".gitconfig"]
symlink = true   # false にするとコピーモード
backup = true    # 既存ファイルのバックアップ

# ディレクトリ単位でのインポート
[[dotfiles.directories]]
source = "macOS/claude"  # リポジトリ内のディレクトリ
target = ".claude"       # ホームディレクトリからの相対パス
```

### 設定項目

| キー | 説明 | デフォルト |
|------|------|-----------|
| `repository` | dotfiles リポジトリの URL | （なし） |
| `local_path` | ローカルのクローン先 | `~/.dotfiles` |
| `source_dir` | リポジトリ内のソースディレクトリ（files用） | （ルート） |
| `files` | インポートするファイル一覧 | `.zshrc`, `.bashrc` など |
| `directories` | インポートするディレクトリのマッピング | （なし） |
| `symlink` | シンボリックリンクを使用 | `true` |
| `backup` | 既存ファイル/ディレクトリをバックアップ | `true` |

### 適用範囲

* `goodbye export brew` が `brew.export.*_cmd` を参照します
* `goodbye import brew` は export 済みのファイルを入力として使用します
* `goodbye import dotfiles` が `dotfiles.*` を参照します

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
goodbye export brew --dir ~/goodbye-export --apply

# 新 PC
goodbye import brew --dir ~/goodbye-export --apply
```

### brew 依存を削減しmiseへ移行(tap以外)

```bash
goodbye brew --mise
goodbye brew --mise --apply
```

### dotfiles の同期・インポート

```bash
# dotfiles リポジトリを同期してインポート
goodbye import dotfiles --url https://github.com/username/dotfiles --apply

# 既に同期済みなら直接インポート
goodbye import dotfiles --apply
```

---

## 将来拡張（予定）

* `goodbye brew --uv`
* toml manifest 対応

---

## 注意事項

* GUI アプリ（cask）に完全な移行先はないためこのコマンドの対象外です。nixなどを選択肢としてください
* DBなどの常駐サービスは goodbye brew 対象外を推奨します
* 実行は自己責任で行ってください

---

## ライセンス

Apache 2
