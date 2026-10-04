# PDF Preview Go

## 概要

Excel または Word ファイルから PDF への変換・プレビュー・保存を行うデスクトップアプリケーションです。
Go（Wails v2.16.0）+ Svelte/Vite で構築されています。

[pdf-preview](https://github.com/kkzk/pdf-preview) をコンバージョンしたものです。

## 主な機能

- 📁 ディレクトリツリー表示とファイル選択
- 📊 Excel シート選択とPDF変換
- 👀 リアルタイムPDFプレビュー  
- 💾 PDF保存・自動更新機能

## 使用方法

### 実行
```bash
pdf-preview-go.exe .\test
```

### 実行時の注意事項

PDF作成には Office アプリケーションを起動します。
念のため、Word/Excel は終了させてから実行してください。

## 技術スタック

- **バックエンド**: Go + Wails v2.16.0
- **フロントエンド**: Svelte + Vite
- **デスクトップ**: Windows専用
