package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config はアプリケーション全体で共有する設定値を保持する。
type Config struct {
	// ServerPort は HTTP サーバがバインドするポート番号。
	ServerPort string
	// DatabasePath は SQLite ファイルの配置パス。
	DatabasePath string
}

const (
	envServerPort  = "SERVER_PORT"
	envDatabaseDir = "DATABASE_DIR"
	envDatabaseURI = "DATABASE_PATH"
)

// Load は環境変数から設定値を読み込む。指定が無い場合はデフォルト値を用いる。
func Load() (Config, error) {
	port := os.Getenv(envServerPort)
	if port == "" {
		port = "3000"
	}

	databasePath := os.Getenv(envDatabaseURI)
	if databasePath == "" {
		dir := os.Getenv(envDatabaseDir)
		if dir == "" {
			dir = "./data"
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Config{}, fmt.Errorf("create database dir: %w", err)
		}
		databasePath = filepath.Join(dir, "commands.sqlite3")
	}

	return Config{ServerPort: port, DatabasePath: databasePath}, nil
}
