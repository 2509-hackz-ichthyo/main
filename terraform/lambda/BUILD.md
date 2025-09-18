## ビルド
GLIBC のバージョン不整合に注意すること。

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
```