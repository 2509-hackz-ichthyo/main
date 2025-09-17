# api

Whitespace の構文を解析して結果を返すだけの軽量な HTTP API です。永続化は行わず、入力をそのまま評価してレスポンスに変換します。

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

## API

- `POST /v1/decode`
  - Request
    ```json
    {
      "command_type": "WhitespaceToBinary",
      "payload": "SSS..." // 実際には空白・タブ・改行からなる文字列
    }
    ```
    - `command_type`: `WhitespaceToDecimal` または `WhitespaceToBinary`
    - `payload`: 対象となる Whitespace コマンド文字列（URL エンコードされた文字列も利用可）
  - Response
    ```json
    {
      "command_type": "WhitespaceToBinary",
      "result_kind": "BinarySequence",
      "result_binaries": ["0000", "1111"],
      "binary_string": "0000 1111"
    }
    ```
    - 10 進数への変換時は `result_decimals` / `decimal_string` がセットされます

## 仕様

- 入力は 1 文～最大 64 文。
- 1 文の構造（空白などを記号化して説明）:
  ```
  SSS {TまたはSが4つ} L
  ```
  または
  ```
  SSS {TまたはSが8つ} L
  ```
  - `S` = スペース
  - `T` = タブ文字
  - `L` = 改行
- `{TまたはSが4つ}` や `{TまたはSが8つ}` の部分を変換対象とし、`S` を `0`、`T` を `1` に写像する。
- 各文を順に変換して結合する。`result_binaries` は 1 文ごとの 2 進数列、`binary_string` は空白区切りで連結したもの。
- `WhitespaceToDecimal` の場合は各文字を ASCII コードに変換して 10 進数列として返す。

## 開発

- フレームワーク: Gin
- ポート: `3000`
- メインエントリ: `cmd/ws-decode-api`
- ディレクトリ構成: `internal/domain` (ドメイン), `internal/app` (ユースケース), `internal/server/httpserver` (HTTP サーバー)
- 依存する外部ミドルウェアはありません

---

# リクエスト・レスポンス

## ヘルスチェック

```
curl -s http://localhost:3000/healthz
```

- レスポンス例: 成功
  ```
  {"status":"ok","timestamp":"2025-09-17T18:25:52.259651519Z"}
  ```

## Whitespace → 10 進数

```
curl -s -X POST http://localhost:3000/v1/decode -H 'Content-Type: application/json' \
-d '{"command_type":"WhitespaceToDecimal","payload":" \t\n"}'
```

- レスポンス例: 成功
  ```
  {"command_type":"WhitespaceToDecimal","result_kind":"DecimalSequence","result_decimals":[32,9,10],"decimal_string":"32 9 10"}
  ```

## Whitespace（パーセントエンコード） → 2 進数

```
curl -s -X POST http://localhost:3000/v1/decode -H 'Content-Type: application/json' \
-d '{"command_type":"WhitespaceToBinary","payload":"%20%20%20%20%09%20%09%0A"}'
```

- レスポンス例: 成功
  ```
  {"command_type":"WhitespaceToBinary","result_kind":"BinarySequence","result_binaries":["0101"],"binary_string":"0101"}
  ```
