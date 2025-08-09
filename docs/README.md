# PDF Preview Go - ドキュメント目次

## 📋 仕様書・ガイド一覧

### 🎯 アプリケーション仕様
- **[APPLICATION_SPECIFICATIONS.md](./APPLICATION_SPECIFICATIONS.md)** - アプリケーションの動作仕様・機能詳細
  - 基本機能（ファイル管理、PDF変換、保存）
  - UI仕様（レイアウト、操作方法）
  - **終了時動作の詳細仕様**
  - 自動機能・エラー処理

### 🎨 UI/UX設計
- **[UI_GUIDELINES.md](./UI_GUIDELINES.md)** - UI/UXデザインガイドライン
  - 設計原則・コンポーネント設計
  - 実装パターン・カラーパレット
  - レイアウト仕様・アニメーション

### 🛠️ 開発者向け
- **[DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md)** - 技術仕様・開発環境ガイド
  - アーキテクチャ・セットアップ方法
  - 開発ワークフロー・技術制約
  - ファイル構成・参照プロジェクト

## 🔄 使用フロー

### 新機能開発時
1. `APPLICATION_SPECIFICATIONS.md` でアプリケーション要件確認
2. `UI_GUIDELINES.md` で UI設計方針確認
3. `DEVELOPER_GUIDE.md` で技術仕様確認
4. 実装・テスト

### UI修正時
1. `UI_GUIDELINES.md` でデザイン原則確認
2. `APPLICATION_SPECIFICATIONS.md` で動作要件確認
3. 実装・テスト

### バグ修正時
1. `APPLICATION_SPECIFICATIONS.md` で期待動作確認
2. `DEVELOPER_GUIDE.md` で技術制約確認
3. 修正・テスト

## 📁 プロジェクト構造

```
pdf-preview-go/
├── docs/                           # 📚 ドキュメント
│   ├── README.md                   # 📋 このファイル
│   ├── APPLICATION_SPECIFICATIONS.md  # 🎯 アプリ仕様
│   ├── UI_GUIDELINES.md            # 🎨 UI設計ガイド
│   └── DEVELOPER_GUIDE.md          # 🛠️ 開発者ガイド
├── .github/
│   └── copilot-instructions.md     # 🤖 Copilot動作指示
├── frontend/                       # 🖼️ フロントエンド
├── main.go, app.go                 # ⚙️ バックエンド
└── README.md                       # 📖 プロジェクト概要
```

---

## 💡 ドキュメント更新方針

### 更新タイミング
- **機能追加時**: 該当する仕様書を同時更新
- **UI変更時**: UI_GUIDELINES.md を同時更新
- **動作仕様変更時**: APPLICATION_SPECIFICATIONS.md を同時更新

### 更新責任
- 各ドキュメントは実装と同期して更新
- 古い情報や矛盾する記述の定期的な整理

このドキュメント構造により、プロジェクトの理解・保守・拡張が効率的に行えます。
