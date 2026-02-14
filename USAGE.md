
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
   - `--global` でインストール後に `mise use -g` を実行
   - `--continue` でエラーがあっても継続

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
