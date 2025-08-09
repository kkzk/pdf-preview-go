# GitHub Actions自動ビルド・リリース設定 (Windows専用)

このディレクトリには、PDF Preview Go プロジェクトのWindows用自動ビルドとリリース用のGitHub Actionsワークフローが含まれています。

## ワークフローファイル

### 1. `ci.yml` - 継続的インテグレーション (Windows)
**トリガー条件:**
- プルリクエスト作成・更新時
- `master`/`main`ブランチへのプッシュ時

**実行内容:**
- Go言語コードの品質チェック（`go fmt`, `go vet`, `go mod tidy`）
- フロントエンド（Svelte/Vite）のビルドテスト
- Windows環境での開発モードビルド確認（`-devtools`フラグ付き）

### 2. `release.yml` - 自動リリース作成 (Windows)
**トリガー条件:**
- `master`/`main`ブランチへのプッシュ時
- `v*`形式のタグプッシュ時

**実行内容:**
- Windows 64-bit用のリリースビルド
- NSISインストーラーの自動作成
- GitHub Releasesへの自動アップロード
- 実行可能ファイル（.exe）とインストーラーの両方を提供

## 設定要件

### 必須設定
特別な設定は不要です。標準のGitHub Actionsの権限で動作します。

### 依存関係の自動インストール
- **NSIS**: Windows用インストーラー作成ツールが自動インストールされます
- **Chocolatey**: NSISインストール用（GitHub Actions環境に標準搭載）

## ビルド成果物

### 自動生成されるファイル
Windows用リリース成果物：
- **`pdf-preview-go.exe`**: 単体実行可能ファイル
- **`pdf-preview-go-*-installer.exe`**: NSISインストーラー（推奨）

### リリース作成タイミング
1. **手動タグでのリリース**: `git tag v1.0.0 && git push origin v1.0.0`
2. **自動タグでのリリース**: `master`ブランチプッシュ時（タイムスタンプベース）

## インストール・使用方法

### エンドユーザー向け
1. **GitHub Releases**ページから最新リリースをダウンロード
2. **推奨**: `*-installer.exe`をダウンロードしてインストール
3. **簡単**: `pdf-preview-go.exe`を直接ダウンロードして実行

## トラブルシューティング

### よくある問題

#### 1. ビルドが失敗する
```powershell
# ローカルでテスト
wails build -platform windows/amd64
wails build -platform windows/amd64 -nsis
```

#### 2. フロントエンド依存関係エラー
```powershell
Set-Location "frontend"
npm install
npm run build
```

#### 3. NSISインストーラーが作成されない
- `build/windows/installer/` ディレクトリの確認
- NSISの設定ファイル（`.nsi`）の存在確認
- Wailsの設定ファイル（`wails.json`）の確認

### ログの確認方法
1. GitHubリポジトリ → Actions タブ
2. 失敗したワークフロー実行をクリック
3. 各ジョブの詳細ログを確認

## カスタマイズ

### ビルドオプションの変更
```yaml
# より詳細なビルドオプション
build_cmd: 'wails build -platform windows/amd64 -nsis -clean -ldflags "-s -w"'
```

### Windows以外のプラットフォーム対応
現在はWindows専用ですが、必要に応じて他プラットフォームを追加可能：
```yaml
# Linux追加例
- os: 'ubuntu-latest'
  name: 'linux'
  build_cmd: 'wails build -platform linux/amd64'
```

## Windows固有の設定

### システム要件
- **対象OS**: Windows 10 64-bit 以降
- **依存関係**: .NET Framework（通常は既にインストール済み）
- **WebView2**: Microsoft Edge WebView2（Windows 10/11では標準）

### インストーラーの特徴
- **NSIS使用**: 業界標準のWindows用インストーラー
- **自動パス設定**: プログラム用フォルダに適切にインストール
- **アンインストール対応**: コントロールパネルから削除可能
- **ショートカット作成**: デスクトップとスタートメニュー

## セキュリティ考慮事項

1. **依存関係の更新**: GitHub Dependabotが有効
2. **Windows Defender**: 署名なし実行可能ファイルの警告が表示される場合があります
3. **SmartScreen**: 初回実行時に警告が表示される場合があります

## パフォーマンス最適化

- **ビルド時間**: Windows専用により約3分の1に短縮
- **アーティファクトサイズ**: プラットフォーム固有により効率化
- **デプロイ時間**: 単一プラットフォームによる高速化

## サポート環境

- **Go**: 1.23以降
- **Node.js**: 18以降  
- **Wails**: v2.10.2
- **対象OS**: Windows 10/11 (64-bit)
