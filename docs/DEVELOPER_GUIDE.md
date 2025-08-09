# PDF Preview Go - 開発者ガイド

## 概要
このリポジトリは、Go（Wails）とSvelte/ViteによるデスクトップPDFプレビューアプリです。GoバックエンドとSvelteフロントエンドが密接に連携し、クロスプラットフォームで動作します。

## アーキテクチャ
- **Go (Wails) バックエンド**: `main.go`, `app.go` でアプリのエントリーポイントとロジックを管理。`wails.Run` でSvelteビルド成果物を埋め込み、Goメソッドをフロントエンドに公開。
- **Svelte + Vite フロントエンド**: `frontend/` ディレクトリ配下。`src/App.svelte` でUI、`wailsjs/go/main/App.js` でGoメソッド呼び出し。`vite`で開発・ビルド。
- **ビルド設定**: `wails.json` でビルド・開発コマンドや出力先を管理。`build/` ディレクトリにOSごとのビルド成果物やインストーラ設定。

## 開発環境セットアップ

### 必要なツール
- Go 1.19以降
- Node.js 16以降
- Wails CLI v2.10.2

### 主要ワークフロー
- **開発サーバ起動**: `wails dev`（Go+Svelteのホットリロード）
- **フロントエンドのみ**: `Set-Location frontend; npm run dev`（Viteサーバ）
- **ビルド**: `wails build`（配布用バイナリ作成）
- **依存インストール**: `Set-Location frontend; npm install`
- **JSバインディング生成**: `wails generate module`（GoメソッドをJS化）
- **テスト時のアプリケーション起動**: `.\build\bin\pdf-preview-go.exe .\test`

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

### 参考ファイル
- `main.go`, `app.go`: Goアプリのエントリーポイントとロジック
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

## 注意事項

### バージョン依存
- WailsのバージョンやSvelte/Viteのバージョンに依存する部分があるため、`package.json`や`wails.json`の内容を参照すること

### 互換性
- クロスプラットフォーム対応のため、OS固有の機能使用時は適切な分岐処理を実装

---

このガイドは開発者がプロジェクトを理解し、効率的に開発を進めるための技術的な情報をまとめています。
