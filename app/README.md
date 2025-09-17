# app
ebitengine

## build
```sh
pwd  # /app/game

# main
env GOOS=js GOARCH=wasm go build -o ../main.wasm .

# reversiPlayer
env GOOS=js GOARCH=wasm go build -o ../reversiPlayer.wasm .
```