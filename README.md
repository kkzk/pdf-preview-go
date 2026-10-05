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

## 動作環境

- Windows 10 64-bit 以降
- Microsoft Edge WebView2 ランタイム（Windows 11 には標準搭載。未導入の場合はインストーラーが導入します）
- Microsoft Office（Excel / Word。Office ファイルを PDF に変換するために使用）

## 使用方法

### 実行
```bash
# フォルダを開く
pdf-preview-go.exe .\test

# ファイルを指定すると、そのフォルダを開いてファイルを選択した状態にする
pdf-preview-go.exe .\test\testdata\testdata1.xlsx
```

### エクスプローラーの右クリックから起動

エクスプローラーの右クリックメニューに「PDF Previewで開く」を追加できます。

- アプリのメニュー「設定 → エクスプローラーの右クリックメニューに追加」で、ユーザーごとにオン・オフできます（初期状態はオフ。管理者権限は不要）
- オンにしたときに起動している exe が登録されます。exe を移動した場合は、一度オフにしてからオンにし直してください
- 対象: フォルダ、フォルダ内の何もない所、Excel / Word / PDF ファイル
- 複数の項目を選択しているときは表示されません
- Windows 11 では「その他のオプションを確認」（Shift+右クリック）の中に表示されます

### 実行時の注意事項

PDF作成には Office アプリケーションを起動します。
念のため、Word/Excel は終了させてから実行してください。

## 技術スタック

- **バックエンド**: Go + Wails v2.16.0
- **フロントエンド**: Svelte + Vite
- **デスクトップ**: Windows専用
