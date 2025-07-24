# nmock

JSONファイルを読み込んでAPIエンドポイントを動的に追加できるモックサーバーです。プラグイン機能により、複数のJSONファイルでエンドポイントを管理できます。

## 機能

- JSONファイルからAPIエンドポイントの設定を読み込み
- プラグインシステムによる動的なエンドポイント管理
- 設定ファイルとプラグインファイルの変更を監視して自動リロード
- カスタムヘッダーとステータスコードをサポート
- レスポンス遅延の設定が可能
- パス変数（例：`/api/users/{id}`）をサポート
- 管理API（プラグインの有効化/無効化、一覧表示）

## 使い方

### サーバーの起動とビルド

```bash
make start
```

### サーバーの停止と削除

```bash
make stop
```

### 開発モード

```bash
make dev
```

### 直接実行

```bash
cd app
go run main.go [config_file]
```

デフォルトでは `config.json` ファイルを使用します。別の設定ファイルを指定することも可能です：

```bash
go run main.go my-config.json
```

## 設定ファイル形式

設定ファイルはJSON形式で、以下の構造を持ちます：

```json
{
  "port": "9000",
  "plugins_dir": "plugins",
  "endpoints": [
    {
      "path": "/api/users",
      "method": "GET",
      "status_code": 200,
      "headers": {
        "Content-Type": "application/json"
      },
      "response": [
        {
          "id": 1,
          "name": "John Doe",
          "email": "john@example.com"
        }
      ],
      "delay": 100
    }
  ]
}
```

### 設定項目

- `port` (オプション): サーバーのポート番号（デフォルト: 9000）
- `plugins_dir` (オプション): プラグインディレクトリのパス（デフォルト: plugins）
- `endpoints`: エンドポイントの配列

## プラグインシステム

プラグインは `plugins` ディレクトリ内のJSONファイルとして管理されます。各プラグインファイルは以下の構造を持ちます：

```json
{
  "name": "example-plugin",
  "description": "Example plugin demonstrating various API endpoints",
  "enabled": true,
  "endpoints": [
    {
      "path": "/api/products",
      "method": "GET",
      "status_code": 200,
      "response": [
        {
          "id": 1,
          "name": "Product A",
          "price": 99.99
        }
      ]
    }
  ]
}
```

### プラグイン設定項目

- `name` (必須): プラグインの名前
- `description` (オプション): プラグインの説明
- `enabled` (必須): プラグインの有効/無効状態
- `endpoints` (必須): エンドポイントの配列

#### エンドポイント設定

- `path` (必須): APIのパス（パス変数をサポート: `/api/users/{id}`）
- `method` (必須): HTTPメソッド（GET, POST, PUT, DELETE など）
- `status_code` (オプション): HTTPステータスコード（デフォルト: 200）
- `headers` (オプション): カスタムヘッダー
- `response` (必須): レスポンスボディ（JSON オブジェクト、配列、または文字列）
- `delay` (オプション): レスポンス遅延（ミリ秒）

## 管理API

サーバーには管理API機能が組み込まれており、プラグインの管理ができます：

### プラグイン一覧

```bash
curl http://localhost:9000/_admin/plugins
```

### 特定プラグインの詳細

```bash
curl http://localhost:9000/_admin/plugins/example-plugin
```

### プラグインの有効化/無効化

```bash
curl -X POST http://localhost:9000/_admin/plugins/example-plugin/toggle
```

### プラグインのリロード

```bash
curl -X POST http://localhost:9000/_admin/reload
```

## 組み込みエンドポイント

- `GET /health`: ヘルスチェックエンドポイント
- `GET /_admin/plugins`: 全プラグインの一覧
- `GET /_admin/plugins/{name}`: 特定プラグインの詳細
- `POST /_admin/plugins/{name}/toggle`: プラグインの有効化/無効化
- `POST /_admin/reload`: プラグインのリロード

## 例

### 基本的な使用例

1. サーバーを起動:
```bash
cd app && go run main.go
```

2. APIをテスト:
```bash
# ユーザー一覧を取得（メイン設定）
curl http://localhost:9000/api/users

# 商品一覧を取得（プラグイン）
curl http://localhost:9000/api/products

# 認証エンドポイント（プラグイン）
curl -X POST http://localhost:9000/api/auth/login

# プラグイン管理
curl http://localhost:9000/_admin/plugins
```

### 新しいプラグインの追加

1. `plugins` ディレクトリに新しいJSONファイルを作成:

```json
{
  "name": "my-plugin",
  "description": "My custom plugin",
  "enabled": true,
  "endpoints": [
    {
      "path": "/api/custom",
      "method": "GET",
      "status_code": 200,
      "response": {
        "message": "Hello from my plugin!"
      }
    }
  ]
}
```

2. ファイルを保存すると自動的にサーバーがリロードされ、新しいエンドポイントが利用可能になります。

### プラグインのホットリロード

- 設定ファイルやプラグインファイルを変更すると、サーバーを再起動することなく自動的に新しい設定が適用されます
- プラグインの有効化/無効化は管理APIを使用して動的に行えます

## 開発

```bash
# 依存関係のインストール
go mod tidy

# アプリケーションの実行
go run main.go

# ビルド
go build -o nmock main.go
```
