# Copilot Instructions for pdf-preview-go

## 概要
このリポジトリは、Go（Wails）とSvelte/ViteによるデスクトップPDFプレビューアプリです。GoバックエンドとSvelteフロントエンドが密接に連携し、クロスプラットフォームで動作します。

## アーキテクチャ
- **Go (Wails) バックエンド**: `main.go`, `app.go` でアプリのエントリーポイントとロジックを管理。`wails.Run` でSvelteビルド成果物を埋め込み、Goメソッドをフロントエンドに公開。
- **Svelte + Vite フロントエンド**: `frontend/` ディレクトリ配下。`src/App.svelte` でUI、`wailsjs/go/main/App.js` でGoメソッド呼び出し。`vite`で開発・ビルド。
- **ビルド設定**: `wails.json` でビルド・開発コマンドや出力先を管理。`build/` ディレクトリにOSごとのビルド成果物やインストーラ設定。

## 主要ワークフロー
- **開発サーバ起動**: `wails dev`（Go+Svelteのホットリロード）
- **フロントエンドのみ**: `Set-Location frontend; npm run dev`（Viteサーバ）
- **ビルド**: `wails build`（配布用バイナリ作成）
- **依存インストール**: `Set-Location frontend; npm install`
- **JSバインディング生成**: `wails generate module`（GoメソッドをJS化）
- **テスト時のアプリケーション起動**: `.\build\bin\pdf-preview-go.exe .\test` （引数に test ディレクトリを指定して起動）

## PowerShell環境での注意点
- **コマンド実行**: PowerShellでは `cd` の代わりに `Set-Location` を使用
- **パス指定**: PowerShellでは引用符で囲む（例：`Set-Location "c:\dev\kkzk\pdf-preview-go"`）
- **連続コマンド**: セミコロン（`;`）で連結（例：`Set-Location frontend; npm install`）

## 重要なパターン・規約
- Goのメソッドは`Bind`でフロントエンドに公開し、`frontend/wailsjs/go/main/App.js`経由で呼び出し。
- フロントエンドの型定義は`jsconfig.json`で管理。`global.d.ts`は未使用。
- アイコンやインストーラ設定は`build/`配下でOSごとに分離。
- Svelteの状態管理やルーティングは最小限。複雑な状態管理は未導入。

## 参考ファイル
- `main.go`, `app.go`: Goアプリのエントリーポイントとロジック
- `frontend/src/App.svelte`: UIとGoメソッド呼び出し例
- `frontend/wailsjs/go/main/App.js`: GoメソッドのJSラッパー
- `wails.json`: ビルド・開発コマンド設定
- `build/`: OSごとのビルド成果物・インストーラ設定

## 注意点
- WailsのバージョンやSvelte/Viteのバージョンに依存する部分があるため、`package.json`や`wails.json`の内容を参照すること。
- GoとSvelte間のデータ受け渡しはシンプルな型（string, number, object）を推奨。
- Pythonサブプロジェクト（`pdf_preview`）は本体とは独立しているため、参照のみで変更してはいけない。

- `main_window.py` および `saveAsPdf.py` の機能を理解すること。この機能をgo言語で実装するのが目的です。

---

この内容に不明点や追加したいプロジェクト固有の知見があれば、フィードバックをお願いします。
