# PDF Preview Go

## 概要

Excel ファイルから PDF への変換・プレビュー・保存を行うデスクトップアプリケーションです。
Go（Wails v2.10.2）+ Svelte/Vite で構築されています。

## 主な機能

- 📁 ディレクトリツリー表示とファイル選択
- 📊 Excel シート選択とPDF変換
- 👀 リアルタイムPDFプレビュー  
- 💾 PDF保存・自動更新機能
- 🖱️ 直感的なUI操作（行全体クリック対応）

## 開発・使用方法

### 開発環境での実行
```bash
wails dev
```

### 本番用ビルド
```bash
wails build
```

### テスト実行
```bash
.\build\bin\pdf-preview-go.exe .\test
```

## 📚 ドキュメント

詳細な仕様・開発ガイドは [`docs/`](./docs/) フォルダを参照してください：

- 📋 **[ドキュメント目次](./docs/README.md)** - 全ドキュメントの概要
- 🎯 **[アプリケーション仕様](./docs/APPLICATION_SPECIFICATIONS.md)** - 機能・動作仕様
- 🎨 **[UI設計ガイド](./docs/UI_GUIDELINES.md)** - デザイン・実装指針
- 🛠️ **[開発者ガイド](./docs/DEVELOPER_GUIDE.md)** - 技術仕様・セットアップ

## 技術スタック

- **バックエンド**: Go + Wails v2.10.2
- **フロントエンド**: Svelte + Vite
- **デスクトップ**: クロスプラットフォーム対応

---

**開発者向け**: `.github/copilot-instructions.md` に Copilot 用の開発指示が記載されています。
