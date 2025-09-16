# api

## build + start

```sh
docker build -t 2509-hackz-ichthyo .
docker run -p 3000:3000 2509-hackz-ichthyo
```

## deploy

```sh
cd api
aws ecr get-login-password --region ap-northeast-1 | docker login --username AWS --password-stdin 471112951833.dkr.ecr.ap-northeast-1.amazonaws.com
docker build -t 2509-hackz-ichthyo .
docker tag 2509-hackz-ichthyo:latest 471112951833.dkr.ecr.ap-northeast-1.amazonaws.com/2509-hackz-ichthyo:latest
docker push 471112951833.dkr.ecr.ap-northeast-1.amazonaws.com/2509-hackz-ichthyo:latest
aws ecs update-service --cluster hackz-ichthyo-ecs-cluster --service hackz-ichthyo-ecs-service --force-new-deployment --region ap-northeast-1
```

## バックエンドの構成

- 機能: Whitespace の構文１命令を解釈し、実行結果を返すデコーダ（インタプリタ）
  - １セッションにつき、１命令
    - 入力: Whitespace 構文テキスト（１命令）
    - 出力: 通常の文字列、または数字列
  - 命令の種類
    - StoA: 文字列 → 数字列（各文字の ASCII 10 進）
    - AtoS: 数字列 → 文字列
- 実装: クリーンアーキテクチャ
  - エンドポイント: `POST /v1/decode`
  - フレームワーク: Gin
  - ポート: `3000`
  - デプロイ: ECR に push → ECS サービスで `--force-new-deployment`
  - コンテナ: `Dockerfile`
    - `golang:1.24.1-alpine` ベース、`/app/main` 実行

## ユビキタス言語

- Command: 解析済みの 1 命令（`CommandType` と `Payload` を持つ純粋オブジェクト）
  - CommandType: `StringToAscii` / `AsciiToString`
  - Payload: 命令タイプに適合する値
    - `StringToAscii` の場合: 通常文字列（例: `"ABC"`）
    - `AsciiToString` の場合: 10 進数列（例: `65 66 67`）
- Decoder: `Command -> Result` を定義（純粋関数的）、ドメインサービス
- Result：文字列 or 数字列（Whitespace の実行結果）

## ユースケース（アプリケーション層）設計

- デコード処理のみ
  1. Parse: 構文 → Command
  2. Validate: 命令 1 個であること・値の範囲・表記整合
  3. Execute: Decoder を呼んで Result を得る
  4. Assemble Response: ドメイン Result → ユースケース出力
