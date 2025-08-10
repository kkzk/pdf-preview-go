# テスト実行ガイドライン

## PDF Preview Go アプリケーションのテスト実行方法

### 基本的なテスト実行
アプリケーションをテストする際は、必ずテストディレクトリを指定してください。

#### コマンドライン実行
```powershell
# ビルド済みのexeファイルを使用
.\build\bin\pdf-preview-go.exe .\test\testdata

# または開発モード（wails dev）
wails dev
# その後アプリ内でフォルダを変更してtestdataディレクトリを選択
```

#### VS Code デバッグ実行
1. `Debug PDF Preview Go (Source)` 設定を使用
2. 自動的に `./test` ディレクトリが指定される
3. カスタムディレクトリをテストする場合は `Debug PDF Preview Go (Custom Dir)` を使用

### テストデータ
- **テストディレクトリ**: `c:\dev\kkzk\pdf-preview-go\test\testdata`
- **含まれるファイル**:
  - `testdata1.xlsx` - Excel テストファイル1
  - `testdata2.xlsx` - Excel テストファイル2  
  - `testdata3.docx` - Word テストファイル

### 新機能（シート選択永続化）のテスト手順
1. アプリを起動して testdata ディレクトリを開く
2. Excel ファイルを選択してシート選択を変更
3. アプリを終了
4. アプリを再起動して同じディレクトリを開く
5. 前回のシート選択状態が復元されることを確認

### 重要な注意点
- **必ずテストディレクトリを指定**してアプリケーションをテスト
- 本番データのディレクトリでテストしない
- シート選択のキャッシュは `%TEMP%\pdf-preview-go-cache\` に保存される
