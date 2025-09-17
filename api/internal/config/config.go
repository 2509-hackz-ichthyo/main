package config

import "os"

// Config はアプリケーション全体で共有する設定値を保持する。
type Config struct {
	// ServerPort は HTTP サーバがバインドするポート番号。
	ServerPort string
}

const (
	envServerPort = "SERVER_PORT"
)

// Load は環境変数から設定値を読み込む。指定が無い場合はデフォルト値を用いる。
func Load() (Config, error) {
	port := os.Getenv(envServerPort)
	if port == "" {
		port = "3000"
	}

	return Config{ServerPort: port}, nil
}
