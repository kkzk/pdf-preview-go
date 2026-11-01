# PDF Preview Go - 開発者ガイド

## 概要
このリポジトリは、Go（Wails）とSvelte/ViteによるデスクトップPDFプレビューアプリです。GoバックエンドとSvelteフロントエンドが密接に連携し、クロスプラットフォームで動作します。

## アーキテクチャ
- **Go (Wails) バックエンド**: ルートの `main` パッケージがエントリーポイントと、フロントエンドに公開する `App` を持つ。`wails.Run` でSvelteビルド成果物を埋め込み、`App` のメソッドをフロントエンドに公開。処理の本体は `internal/` 配下のパッケージに分かれている（下記「ファイル構成」参照）。
- **Svelte + Vite フロントエンド**: `frontend/` ディレクトリ配下。`src/App.svelte` でUI、`wailsjs/go/main/App.js` でGoメソッド呼び出し。`vite`で開発・ビルド。
- **ビルド設定**: `wails.json` でビルド・開発コマンドや出力先を管理。`build/` ディレクトリにOSごとのビルド成果物やインストーラ設定。

## 開発環境セットアップ

### 必要なツール
- Go 1.25以降
- Node.js 22.13 以降（推奨: 24）
- Wails CLI v2.16.0

### 主要ワークフロー
- **開発サーバ起動**: `wails dev`（Go+Svelteのホットリロード）
- **フロントエンドのみ**: `Set-Location frontend; npm run dev`（Viteサーバ）
- **ビルド**: `wails build`（配布用バイナリ作成）
- **依存インストール**: `Set-Location frontend; npm install`
- **JSバインディング生成**: `wails generate module`（GoメソッドをJS化）
- **テスト時のアプリケーション起動**: `.\build\bin\pdf-preview-go.exe .\test`

### デバッグ実行
VS Codeでのデバッグが設定済みです：

#### デバッグ設定（`.vscode/launch.json`）
1. **Debug PDF Preview Go**: `./test`ディレクトリで起動
2. **Debug PDF Preview Go (Custom Dir)**: カスタムディレクトリを入力して起動
3. **Debug Wails Dev**: Wails開発モードでデバッグ

#### デバッグ実行方法
1. VS Codeで `F5` を押すか、「実行とデバッグ」パネルから選択
2. ブレークポイントを設定してステップ実行
3. 変数の値やスタックトレースを確認

#### ログ出力
- アプリケーション起動時に詳細ログが出力されます
- コマンドライン引数、ディレクトリパス、各種初期化処理の状況を確認可能

### PowerShell環境での注意点
- **コマンド実行**: PowerShellでは `cd` の代わりに `Set-Location` を使用
- **パス指定**: PowerShellでは引用符で囲む（例：`Set-Location "c:\dev\kkzk\pdf-preview-go"`）
- **連続コマンド**: セミコロン（`;`）で連結（例：`Set-Location frontend; npm install`）

## 技術仕様

### 重要なパターン・規約
- Goのメソッドは`Bind`でフロントエンドに公開し、`frontend/wailsjs/go/main/App.js`経由で呼び出し。
- フロントエンドの型定義は`jsconfig.json`で管理。`global.d.ts`は未使用。
- アイコンやインストーラ設定は`build/`配下でOSごとに分離。
- Svelteの状態管理やルーティングは最小限。複雑な状態管理は未導入。

### データ受け渡し
- GoとSvelte間のデータ受け渡しはシンプルな型（string, number, object）を推奨
- 複雑なオブジェクトはJSONシリアライゼーションで処理

## ファイル構成

### Goバックエンド
```
main.go          # エントリーポイント（frontend/dist を埋め込むためルートに置く）
app.go           # App 構造体と起動・終了処理
bindings.go      # internal パッケージに委譲するだけの公開メソッド
conversion.go    # 変換・結合と自動再生成
pdf_manager.go   # 生成したPDFの保存と未保存状態
dialogs.go       # ダイアログ・ウィンドウタイトル
startup.go       # コマンドライン引数の解釈
internal/
├── appdata/     # ディレクトリ履歴・セッションの保存、データ/キャッシュの場所
├── contextmenu/ # エクスプローラーの右クリックメニュー登録
├── convert/     # Office→PDF変換（COM）、PDF結合、シート一覧
│   └── testdata/  # テスト用のExcel/Wordファイル（手動確認にも使う）
├── files/       # ファイル一覧・ファイルを開く
├── pdfserver/   # 変換したPDFをフロントエンドへ配信
└── watch/       # ファイル変更の検知
```
- フロントエンドに公開するメソッドは `App` に置く（`frontend/wailsjs/go/main/App.js` が生成される）。ロジックは `internal/` に置き、`App` からはそれを呼ぶだけにする

### 参考ファイル
- `main.go`, `app.go`: Goアプリのエントリーポイントと `App`
- `frontend/src/App.svelte`: UIとGoメソッド呼び出し例
- `frontend/wailsjs/go/main/App.js`: GoメソッドのJSラッパー
- `wails.json`: ビルド・開発コマンド設定
- `build/`: OSごとのビルド成果物・インストーラ設定

### ドキュメント
- `docs/APPLICATION_SPECIFICATIONS.md`: アプリケーション仕様
- `docs/UI_GUIDELINES.md`: UI/UX設計ガイドライン
- `README.md`: プロジェクト概要と使用方法

## 既存プロジェクトとの関係

### Python参照プロジェクト
- `main_window.py` および `saveAsPdf.py` の機能を理解し、Go言語で実装することが目的
- Pythonサブプロジェクト（`pdf_preview`）は参照のみで変更禁止

## リリース

バージョンは [セマンティックバージョニング](https://semver.org/lang/ja/)（`vX.Y.Z`）で管理します。

1. `wails.json` の `Info.productVersion` をリリースするバージョンに更新して main にマージする（ローカルビルドの表示用。リリースのビルドではタグのバージョンが使われる）
2. main でタグを作成して push する

   ```powershell
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. GitHub Actions（`release.yml`）がインストーラーと単体の exe をビルドし、GitHub Releases に公開する
   - `v0.3.0-beta.1` のように `-` を含むタグはプレリリースになる
   - main への push ではリリースは作成されない（CI のビルドとテストのみ）

リリースのワークフローを変更したときは、GitHub の Actions 画面から「Build and Release (Windows)」を手動実行（`gh workflow run release.yml`）すると、リリースを作成せずにビルドからアーティファクトの受け渡しまでを試せます。

## 注意事項

### バージョン依存
- WailsのバージョンやSvelte/Viteのバージョンに依存する部分があるため、`package.json`や`wails.json`の内容を参照すること

### 互換性
- クロスプラットフォーム対応のため、OS固有の機能使用時は適切な分岐処理を実装

---

このガイドは開発者がプロジェクトを理解し、効率的に開発を進めるための技術的な情報をまとめています。
