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
      "payload": "SSSTSTTLSSSSTTSLSSSTTSTSSTSL" // 実際には空白・タブ・改行からなる文字列
    }
    ```
    - `command_type`: `WhitespaceToDecimal` / `WhitespaceToBinary` / `DecimalToWhitespace`
    - `payload`: 対象となる Whitespace 文字列（URL エンコード可）または 10 進数列
  - Response
    ```json
    {
      "command_type": "WhitespaceToBinary",
      "result_kind": "BinarySequence",
      "result_binaries": ["1011011011010010"],
      "binary_string": "1011011011010010"
    }
    ```
    - 10 進数への変換時は `result_decimals` / `decimal_string` がセットされます
    - Whitespace への変換時は生の文字列を `result_whitespace` に、パーセントエンコードされた文字列を `result_whitespace_percent_encoded` に格納します

## 仕様

- 入力は 1 文～最大 64 文。
- 1 文の構造（空白などを記号化して説明）:
  ```
  SSS {TまたはSが4つ} LSSS {TまたはSが4つ} LSSS {TまたはSが8つ} L
  ```
  - `S` = スペース
  - `T` = タブ文字
  - `L` = 改行
- `{TまたはSが4つ}` や `{TまたはSが8つ}` の部分を変換対象とし、
  `S` を `0`、`T` を `1` に写像する。
  - つまり、**変換対象の部分が 4 つとは 4 bit 2 進数、8 つとは 8 bit 2 進数 と言える**。
  - `L`が区切り文字となり、先頭の`SSS`はここでは特別な解釈をしない。
- 各文を順に変換して結合する。
  - `result_binaries` は 1 文ごとの 2 進数列、`binary_string` は空白区切りで連結したもの。
  - `result_decimals` は 1 文ごとの 10 進数、`decimal_string` は空白区切りで連結したもの
- `WhitespaceToBinary` の場合、Whitespace の各行がもつ変換対象の部分を、
  写像を使って 2 進数列に変換し `result_binaries` に格納する。
  また、空白区切りで連結したものを `binary_string` に格納する。
- `WhitespaceToDecimal` の場合、Whitespace の各行がもつ変換対象の部分を、
  写像を使って 2 進数列に変換し、
  それを 10 進数に変換したものを `result_decimals` に格納する。
  また、空白区切りで連結したものを `decimal_string` に格納する。
- `BinariesToWhitespace` の場合、2 進数列のそれぞれ `0` を `S`、`1` を `T` に写像させ、
  １文の形式 `SSS {4 bit} LSSS {4 bit} LSSS {8 bit} L` に従って、
  スペースとタブ文字と改行で表現したものを `result_whitespace` に格納する。
  また、パーセントエンコードしたものを `result_whitespace_percent_encoded` に格納する。
- `DecimalToWhitespace` の場合、10 進数列を 2 進数表記に変換し、
  以降は`BinariesToWhitespace` と同様に変換する。

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
curl -s -X POST http://localhost:3000/v1/decode -H 'Content-Type: application/json' -d '{"command_type":"WhitespaceToDecimal","payload":["   \t \t\t\n    \t\t \n   \t\t \t  \t \n","       \n       \n           \n"]}'
```

- レスポンス例: 成功
  ```
  {"command_type":"WhitespaceToDecimal","result_kind":"DecimalSequence","result_decimals":["11 6 210", "0 0 0"],"decimal_string":"11 6 210 0 0 0"}
  ```

## Whitespace（パーセントエンコード） → 2 進数

- 次の例では、改行は LF 改行(%0A)でエンコードだが、
  CR 改行(%0D) や、 CRLF 改行(%0D%0A) でも動作します。

```
curl -s -X POST http://localhost:3000/v1/decode -H 'Content-Type: application/json' -d '{"command_type":"WhitespaceToBinary","payload":["%20%20%20%09%20%09%09%0A%20%20%20%20%09%09%20%0A%20%20%20%09%09%20%09%20%20%09%20%0A","%20%20%20%20%20%20%20%0A%20%20%20%20%20%20%20%0A%20%20%20%20%20%20%20%20%20%20%20%0A"]}'
```

- レスポンス例: 成功
  ```
  {"command_type":"WhitespaceToBinary","result_kind":"BinarySequence","result_binaries":["1011 0110 11010010","0000 0000 00000000"],"binary_string":"1011 0110 11010010 0000 0000 00000000"}
  ```

## 10 進数 → Whitespace

```
curl -s -X POST http://localhost:3000/v1/decode -H 'Content-Type: application/json' -d '{"command_type":"DecimalToWhitespace","payload":["11 3 125","2 0 0", "8 15 228"]}'
```

- レスポンス例: 成功
  ```
  {"command_type":"DecimalToWhitespace","result_kind":"Whitespace","result_binaries":["1011 0011 01111101","0010 0000 00000000","1000 1111 11100100"],"binary_string":"1011 0011 01111101 0010 0000 00000000 1000 1111 11100100","result_whitespace":["   \t \t\t\n     \t\t\n    \t\t\t\t\t \t\n","     \t \n       \n           \n","   \t   \n   \t\t\t\t\n   \t\t\t  \t  \n"],"result_whitespace_percent_encoded":["%20%20%20%09%20%09%09%0A%20%20%20%20%20%09%09%0A%20%20%20%20%09%09%09%09%09%20%09%0A","%20%20%20%20%20%09%20%0A%20%20%20%20%20%20%20%0A%20%20%20%20%20%20%20%20%20%20%20%0A","%20%20%20%09%20%20%20%0A%20%20%20%09%09%09%09%0A%20%20%20%09%09%09%20%20%09%20%20%0A"]}
  ```

## 2 進数 → Whitespace

```
curl -s -X POST http://localhost:3000/v1/decode -H 'Content-Type: application/json' -d '{"command_type":"BinariesToWhitespace","payload":["1011 0110 11010010","0000 0000 00000000"]}'
```

- レスポンス例: 成功
  ```
  {"command_type":"BinariesToWhitespace","result_kind":"Whitespace","result_whitespace":["   \t \t\t\n    \t\t \n   \t\t \t  \t \n","       \n       \n           \n"],"result_whitespace_percent_encoded":["%20%20%20%09%20%09%09%0A%20%20%20%20%09%09%20%0A%20%20%20%09%09%20%09%20%20%09%20%0A","%20%20%20%20%20%20%20%0A%20%20%20%20%20%20%20%0A%20%20%20%20%20%20%20%20%20%20%20%0A"]}
  ```
