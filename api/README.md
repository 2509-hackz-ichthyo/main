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

## 仕様

### 入力

- 入力は 1 文～最大 64 文。
- 1 文の構造（空白などを記号化して説明）:
  ```
  SSS {TまたはSが4つ} L
  SSS {TまたはSが4つ} L
  SSS {TまたはSが8つ} L
  ```
  - `S` = スペース
  - `T` = タブ文字
  - `L` = 改行

### ルール

- `{TまたはSが4つ}` や `{TまたはSが8つ}` の部分を変換対象とする。
- 変換規則:
  - `S` → `0`
  - `T` → `1`
- これにより 4 ビット、または 8 ビットの 2 進数文字列が得られる。
- 各文を順に変換して結合する。

### 出力

- 入力文ごとに得られた 2 進数列を空白区切りで出力する。

### 例

- 入力:
  ```
  SSSSSSSL
  SSSSSSSL
  SSSSSSSSSSL
  ```
  出力:
  ```
  0000 0000 00000000
  ```
- 入力:
  ```
  SSSTSTTL
  SSSSTTSL
  SSS11S1SS1SL
  ```
  出力:
  ```
  1011 0110 11010010
  ```

### 補足

- 入力は HTTP 経由で送られてくるため、パーセントエンコーディングされている可能性がある。
- 最大 64 文まで対応する必要がある。

## ユビキタス言語

- Command: 入力命令（`Type` と `Payload` を持つ純粋オブジェクト）
  - Type: 命令の種類(`WhitespaceToDecimal` / `DecimalToWhitespace`)
  - Payload: 変換の対象となるリテラルで、命令タイプ(Type)に適合する値
    - `DecimalToWhitespace` の場合: Whitespace を表す 10 進数列（`32` = Space, `9` = Tab, `10` = LF。例: `32 9 10`）
    - `WhitespaceToDecimal` の場合: Whitespace 構文テキスト（例: `" \t\n \n\t\t\n"`）
- Encoder: Whitespace エンコーダ
  - `DecimalToWhitespace` の振る舞いを持つ純粋なオブジェクト
- Decoder: Whitespace デコーダ
  - `WhitespaceToDecimal` の振る舞いを持つ純粋なオブジェクト
- Result：文字列 or 数字列

## ドメイン層

- 概念「Command」を定義する
  - コマンドは「CommandType」と「Payload」を持つものとする
  - CommandType は「Whitespace → 10 進数数字」または「10 進数数字列 → Whitespace」である
- 概念「Encoder」を定義する
  - Encoder は「コマンドを受け取り、結果を返す」純粋な振る舞いを持つ
  - DtoW のこと
    - Whitespace を構成する 10 進数（`32` / `9` / `10` のみ）を Whitespace 文字列にする
  - 不正値（未サポートの数値 / 数字以外）は「不正である」ことを返す
- 概念「Decoder」を定義する
  - decoder は「コマンドを受け取り、結果を返す」純粋な振る舞いを持つ
  - WtoD のこと
    - Whitespace（Space / Tab / LF のみ）を 10 進数数字列 にする
  - 不正値（対象外の文字）が混じる場合は「不正である」ことを返す
- 概念「Result」を定義する
  - 結果は 文字列 または 数値列 のどちらか一方を保持するものとする
  - 数値列は配列として保持し、フォーマットはアプリケーション層で実施する
- ドメインエラー
  - 未サポートの CommandType: `ErrInvalidCommandType`
  - コンポーネントと CommandType の不整合: `ErrTypeMismatch`
  - ペイロードの構文/値エラー: `ErrInvalidPayload`
