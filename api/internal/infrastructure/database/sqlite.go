package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// SQLite ドライバを匿名インポートして database/sql に登録する。
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteConfig は SQLite 接続の設定値をまとめた構造体。
type SQLiteConfig struct {
	// Path は DB ファイルへのパス。":memory:" を指定するとオンメモリ DB を利用する。
	Path string
	// MaxOpenConns は最大同時接続数。SQLite はシングルライタであるため小さめを推奨。
	MaxOpenConns int
	// MaxIdleConns はアイドル接続数。
	MaxIdleConns int
	// ConnMaxLifetime はコネクションのライフタイム。
	ConnMaxLifetime time.Duration
}

// OpenSQLite は設定に基づき SQLite データベースを開く。
func OpenSQLite(cfg SQLiteConfig) (*sql.DB, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("open sqlite: path must not be empty")
	}

	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns >= 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	return db, nil
}

// EnsureSchema は必要なテーブルが存在することを保証するマイグレーション関数。
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	const createTable = `
CREATE TABLE IF NOT EXISTS command_executions (
    id TEXT PRIMARY KEY,
    command_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    result_kind TEXT NOT NULL,
    result_text TEXT,
    result_decimals TEXT,
    result_binaries TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

	if _, err := db.ExecContext(ctx, createTable); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	return nil
}
