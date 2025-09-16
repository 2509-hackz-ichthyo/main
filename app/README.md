# app
ebitengine

## build
```sh
pwd  # /app/game
env GOOS=js GOARCH=wasm go build -o ../main.wasm main.go
```