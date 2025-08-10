# Copilot Instructions for pdf-preview-go

## プロジェクト概要
Go（Wails v2.10.2）+ Svelte/Viteによるデスクトップ PDF プレビューアプリケーション。
ExcelファイルからPDF変換・プレビュー・保存機能を提供します。

## 開発方針

### 参照すべき仕様書
作業前に以下のドキュメントを必ず参照してください：
- `docs/APPLICATION_SPECIFICATIONS.md`: アプリケーション動作仕様
- `docs/UI_GUIDELINES.md`: UI/UX設計ガイドライン  
- `docs/DEVELOPER_GUIDE.md`: 技術仕様・開発環境ガイド

### PowerShell環境対応
Windowsターミナルコマンド生成時：
- `Set-Location` を使用（`cd` ではなく）
- パス指定は引用符で囲む
- 複数コマンドはセミコロン（`;`）で連結

### コード品質管理（必須）
すべての作業完了時に以下を自動実行：

1. **エラーチェック**: `get_errors` ツールで全ファイルのエラー確認
2. **自動修正**: 検出された警告・エラーを自動修正
3. **Go言語**: `go fmt` と `go vet` を実行
4. **フロントエンド**: TypeScript/Svelte の型エラー・構文エラー修正  
5. **最終確認**: 再度 `get_errors` でエラー解消を確認
6. **ビルドテスト**: コンパイルエラーがないことを確認

### 自動エラー修正プロセス
```
機能実装 → get_errors → 自動修正 → go fmt/vet → 再チェック → ビルド確認
```

## 重要な技術制約

### アーキテクチャ固有
- Go メソッドは `Bind` でフロントエンドに公開
- データ受け渡しはシンプルな型（string, number, object）推奨
- Pythonサブプロジェクト（`pdf_preview`）は参照のみ、変更禁止

### Wails固有
- JSバインディングは `wails generate module` で生成
- フロントエンドビルドは自動でGoバイナリに埋め込み
- 型定義は `jsconfig.json` で管理

## 継続的品質保証
`.vscode/settings.json` により保存時自動フォーマット・lint修正が実行されます。
新機能追加や修正の最後に必ずエラーチェックを実行し、クリーンな状態を維持してください。

## テスト実行ルール

### 必須事項
**アプリケーションのテスト実行時は必ずテストディレクトリを指定してください**

### テスト実行方法
```powershell
# ビルド後のテスト実行
.\build\bin\pdf-preview-go.exe .\test\testdata

# 開発モードテスト
wails dev  # その後アプリ内でtestdataフォルダを選択
```

### VS Codeデバッグ
- `Debug PDF Preview Go (Source)` を使用（自動的に./testが指定される）
- カスタムディレクトリテスト用に `Debug PDF Preview Go (Custom Dir)` も利用可能

### テストデータ場所
- **メインテストディレクトリ**: `./test/testdata/`
- **含まれるファイル**: testdata1.xlsx, testdata2.xlsx, testdata3.docx

---

**重要**: アプリケーションの動作仕様や UI 設計に関する詳細は `docs/` フォルダ内の仕様書を参照し、
それらの仕様に従って開発してください。

---

この内容に不明点や追加したいプロジェクト固有の知見があれば、フィードバックをお願いします。
