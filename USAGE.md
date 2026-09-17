
## 旧PCのbrewから新PCのbrewへの移行
Homebrew 環境をファイルとして書き出し、新PCで再構築します。`goodbye` はデフォルトで dry-run なので、まずは確認だけ行ってから `--apply` を付けて実行します。

手順:
1. 旧PCで Homebrew の内容をエクスポートする。
   ```bash
   # まずは確認（dry-run）
   goodbye export brew --dir ~/goodbye-export

   # 実行
   goodbye export brew --dir ~/goodbye-export --apply
   ```
   出力物: `formula.txt` / `cask.txt` / `tap.txt`
2. 出力ディレクトリを新PCへコピーする。
3. 新PCで Homebrew をインポートする。
   ```bash
   # まずは確認（dry-run）
   goodbye import brew --dir ~/goodbye-export

   # 実行
   goodbye import brew --dir ~/goodbye-export --apply
   ```
4. 必要に応じてオプションを使い分ける。
   - `--only formula|cask|tap` で対象を限定
   - `--skip-taps` で tap を無視
   - `--continue` でエラーがあっても継続

補足:
- export の取得内容は `~/.goodbye.toml` の `brew.export.*_cmd` で変更できます。

## 旧PCのmiseから新PCのmiseへの移行
mise で管理しているツールをファイルに書き出し、新PCで復元します

手順:
1. 旧PCで mise のインストール済みツールをエクスポートする。
   ```bash
   # まずは確認（dry-run）
   goodbye export mise --dir ~/goodbye-export

   # 実行（.mise.toml を作成）
   goodbye export mise --dir ~/goodbye-export --apply

   # .tool-versions 形式で出力したい場合
   goodbye export mise --dir ~/goodbye-export --format tool-versions --apply
   ```
2. 出力ディレクトリを新PCへコピーする。
3. 新PCで mise をインポートする。
   ```bash
   # まずは確認（dry-run）
   goodbye import mise --dir ~/goodbye-export

   # 実行
   goodbye import mise --dir ~/goodbye-export --apply

   # 特定ファイルから読み込む場合
   goodbye import mise --dir ~/goodbye-export --file .tool-versions --apply
   ```
4. 必要に応じてオプションを使い分ける。
   - `--global` で `mise use -g --pin` を実行し、インストールとグローバル登録を行う
   - `--continue` でエラーがあっても継続

## pnpm / Go / Cargo / uv の移行

旧PCで一覧を export し、出力ディレクトリを新PCにコピーして import します。
Node.js / Go / Rust / Python 自体やパッケージマネージャーの移行は対象外です。

```bash
# 旧PC: 一覧を確認し、保存
goodbye export pnpm --dir ~/goodbye-export
goodbye export pnpm --dir ~/goodbye-export --apply
goodbye export go --dir ~/goodbye-export --apply
goodbye export cargo --dir ~/goodbye-export --apply
goodbye export uv --dir ~/goodbye-export --apply

# 新PC: 内容を確認してから復元（pnpm を go / cargo / uv に置換可能）
goodbye import pnpm --dir ~/goodbye-export
goodbye import pnpm --dir ~/goodbye-export --apply
goodbye import pnpm --dir ~/goodbye-export --apply --continue --verbose
```

| 対象 | 一覧の取得 | 復元コマンド |
| --- | --- | --- |
| pnpm | `pnpm list -g --depth=0 --json` | `pnpm add -g -- <name>@<version>` |
| Go | `go env -json GOBIN GOPATH` とバイナリのビルド情報 | `go install <package>@<version>` |
| Cargo | `cargo install --list --color never` | `cargo install <crate> --version =<version>` |
| uv | `uv tool list --show-version-specifiers --show-extras --show-with --color never --offline` | `uv tool install -- <name>==<version>` |

出力はそれぞれ `pnpm-global.txt` / `go-tools.txt` / `cargo-tools.txt` / `uv-tools.txt`。
一覧はソート・重複排除され、空一覧でも `--apply` なら空ファイルを保存します。
取得に失敗した場合は既存ファイルを書き換えずエラーになります。
import はファイルがない場合・不正な形式の場合にエラーになり、インストールを開始しません。
`--continue` はインストール失敗後の継続指定であり、不正なファイルを許容する指定ではありません。

復元範囲:

- pnpm: registry パッケージの直接依存（optional/dev を含む）を対象に、スコープ名・固定バージョンを保持します。エイリアスや、ローカルリンクなど固定バージョンとして取得できない項目は警告して除外します。
- Go: `GOBIN`、未指定なら `GOPATH` の先頭エントリの `bin` を走査します。配置済みバイナリ自体は実行しません。非Goバイナリ、バージョン不明・`(devel)`、ローカル変更あり（`+dirty` / `vcs.modified=true`）、replace を含むビルドは警告して除外します。別の配置先や過去の Go 環境までは探索しません。
- Cargo: crates.io の crate 名・バージョンを対象にします。Git／ローカル／別ソースの表示がある項目は警告して除外します。features、ビルドフラグ、選択したバイナリの情報は復元しません。
- uv: registry の通常のツールを対象にします。Git／ローカルソース、extras、`--with` などの追加要件は警告して除外します。詳細一覧の各フラグに対応する uv が必要です。Python のバージョンや index の設定は復元しません。
- 保存するのは直接導入したツールのバージョンです。依存全体のロック、認証情報、レジストリ設定、ビルド環境は含みません。Go は `go install` 由来かどうかの厳密な判別はできません。

### mise 形式でまとめて管理する

npm / pnpm / Go / Cargo / uv の export に `--format mise` を指定すると、通常の `.txt` の代わりに `<出力先>/.mise.toml` へ集約します。
`--format text`（既定）は従来の形式です。`goodbye export mise --format toml` とは別の機能です。

```bash
# まずは集約結果を確認（ファイル変更なし）
goodbye export npm --format mise --dir ~/goodbye-export

# 同じディレクトリに順番に書き出す
goodbye export npm --format mise --dir ~/goodbye-export --apply
goodbye export pnpm --format mise --dir ~/goodbye-export --apply
goodbye export go --format mise --dir ~/goodbye-export --apply
goodbye export cargo --format mise --dir ~/goodbye-export --apply
goodbye export uv --format mise --dir ~/goodbye-export --apply

# 移行先で mise のグローバル設定へ登録
goodbye import mise --dir ~/goodbye-export --global
goodbye import mise --dir ~/goodbye-export --global --apply --verbose
```

| 元の管理方法 | mise のキー例 |
| --- | --- |
| npm / pnpm | `"npm:@scope/cli" = "1.2.3"` |
| Go | `"go:example.com/tools/cmd/tool" = "v1.2.3"` |
| Cargo | `"cargo:ripgrep" = "14.1.1"` |
| uv | `"pypi:black" = "25.1.0"` |

- 復元先には mise と、各 CLI のビルド・実行に必要な Node.js / Go / Rust / Python / uv 等を用意してください。ランタイムのバージョンは自動で追加しません。`pypi:` など各バックエンドに対応する mise が必要です。
- npm の mise 形式は `npm list -g --depth=0 --json --long` で実際のバージョンを取得します。`npm.export.global_cmd` は通常形式だけに適用されます。ローカルリンク・エイリアス・判別できた非registryソース・バージョン不明の項目は理由付きで除外します。他の管理方法の除外条件は通常形式と同じです。
- 同じ完全なキー・同じバージョンは変更しません。異なるバージョンや、同じキーの配列・オプション付き設定がある場合は競合として停止します。短縮名とバックエンド名の同一性は推測しません。
- npm と pnpm の同じパッケージは同じ `npm:` キーになります。両方に異なるバージョンがあれば、どちらを採用するか決めて export 元または出力設定を調整してください。
- 競合・不正な TOML の場合は既存ファイルを変更しません。追加時は既存の設定値を保持して TOML を再出力するため、コメント・書式・記述順序は保持しません。
- 出力は一時ファイルから置き換えます。同時書き込みは `.mise.toml.lock` で拒否します。異常終了でロックだけ残った場合は、他の export が動いていないことを確認してからロックを取り除いてください。
- `goodbye import mise` はバージョン文字列・文字列配列を読み込みます。追加オプション付きの既存設定もまとめた場合は、mise 自体で設定を適用してください。
- export 自体は mise のグローバル設定を変更せず、元の npm / Go 等のインストールも削除しません。移行後は `which <コマンド名>` で古い配置先が優先されていないか確認してください。

### 古い npm export ファイルについて

以前の npm export は、一覧ルートを `lib` として含め、`@scope/name` を `name` に短縮していました。
修正版はルートを除外し、スコープ付きの名前を保持します。
古い `npm-global.txt` から失われたスコープは復元できないため、元の環境で修正版の `goodbye export npm --dir <保存先> --apply` を実行し直してください。

## brewからmiseへの移行
Homebrew で入れているもので、mise が管理できるツールを候補として抽出し、段階的に移行します。
デフォルトは dry-run で、候補と実行内容だけ表示されます。

手順:
1. 現在の Homebrew formula を取得し、mise registry と照合する。
2. 移行候補一覧を確認する（dry-run）。
   ```bash
   goodbye brew --mise
   ```
3. 問題なければ実行する。
   ```bash
   goodbye brew --mise --apply
   ```
4. 実行時の流れ（ツールごと）:
   1) `mise install <tool>@latest`  
   2) `mise use -g <tool>@latest`  
   3) `mise current <tool>` で疎通確認  
   4) 成功したものだけ `brew uninstall <tool>`

注意:
- すべてを一気に置き換える前提ではありません。候補を見てから段階的に進めてください。

## dotfilesの同期・インポート
dotfiles リポジトリをクローンし、設定ファイルやディレクトリをホームディレクトリに配置します。

手順:
1. dotfiles リポジトリを同期してインポートする。
   ```bash
   # まずは確認（dry-run）
   goodbye import dotfiles --url https://github.com/username/dotfiles

   # 実行
   goodbye import dotfiles --url https://github.com/username/dotfiles --apply

   # カスタムパスを指定
   goodbye import dotfiles --url https://github.com/username/dotfiles --path ~/my-dotfiles --apply
   ```
2. 既に同期済みなら直接インポートする。
   ```bash
   # まずは確認（dry-run）
   goodbye import dotfiles

   # 実行
   goodbye import dotfiles --apply
   ```
3. 必要に応じてオプションを使い分ける。
   - `--copy` でシンボリックリンクの代わりにコピー
   - `--no-backup` で既存ファイルのバックアップを無効化
   - `--continue` でエラーがあっても継続

設定例 (`~/.goodbye.toml`):
```toml
[dotfiles]
repository = "https://github.com/username/dotfiles"
local_path = "~/.dotfiles"
source_dir = "macOS"
files = [".zshrc", ".vimrc", ".gitconfig"]
symlink = true
backup = true

# ディレクトリ単位でのインポート
[[dotfiles.directories]]
source = "macOS/claude"  # リポジトリ内のディレクトリ
target = ".claude"       # ~/.claude に配置
```

補足:
- `files` はホームディレクトリ直下に配置されます（source_dir からの相対パス）
- `directories` はリポジトリルートからの相対パスで指定し、ホームディレクトリ配下に配置されます

## 環境のドリフトチェック
現在の環境が設定ファイルや推奨状態と乖離していないか確認します。

手順:
1. 環境の状態を確認する。
   ```bash
   # 確認のみ
   goodbye status

   # 詳細表示
   goodbye status -v
   ```
2. 問題があれば修正を適用する。
   ```bash
   goodbye status --apply
   ```

チェック内容:
- dotfiles の PATH 設定（ハードコードされた Homebrew パスなど）
- 推奨ツールのインストール状態（mise, fzf, starship など）

設定例 (`~/.goodbye.toml`):
```toml
[status]
# PATH のハードコード検出ルール
[[status.path_rules]]
pattern = "/usr/local/bin/"
replacement = "$HOMEBREW_PREFIX/bin/"
description = "Intel Homebrew パスを汎用変数に置換"

# ツールのインストールチェック
[[status.tool_checks]]
name = "mise"
command = "mise --version"
```
