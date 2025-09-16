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

---

## 現在の構成（バックエンド /api）

- フレームワーク: Gin
- エントリポイント: `main.go`
  - `/` に 200 で "Hello, World!" を返す
- ポート: `3000`
- コンテナ: `Dockerfile`
  - `golang:1.24.1-alpine` ベース、`/app/main` 実行
- デプロイ: ECR に push → ECS サービスで `--force-new-deployment`
- ディレクトリ構成（抜粋）:
  - `api/main.go` … HTTP サーバの起動
  - `api/ws/` … 将来のリアルタイム処理（現在プレースホルダ）
