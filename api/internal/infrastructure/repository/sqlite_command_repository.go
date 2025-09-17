package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/2509-hackz-ichthyo/main/api/internal/usecases"
)

// SQLiteCommandRepository は SQLite をバックエンドとする CommandRepository の実装。
type SQLiteCommandRepository struct {
	db *sql.DB
}

// NewSQLiteCommandRepository は SQLiteCommandRepository を生成する。
func NewSQLiteCommandRepository(db *sql.DB) *SQLiteCommandRepository {
	return &SQLiteCommandRepository{db: db}
}

// Save はコマンド実行履歴を挿入する。
func (r *SQLiteCommandRepository) Save(ctx context.Context, execution usecases.CommandExecution) error {
	if r.db == nil {
		return errors.New("sqlite command repository: db is nil")
	}

	const query = `
INSERT INTO command_executions (
    id, command_type, payload, result_kind, result_text, result_decimals, result_binaries, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

	resultText := sql.NullString{}
	if execution.ResultText != nil {
		resultText = sql.NullString{String: *execution.ResultText, Valid: true}
	}

	decimalsJSON, err := encodeIntSlice(execution.ResultDecimals)
	if err != nil {
		return fmt.Errorf("encode decimals: %w", err)
	}
	binariesJSON, err := encodeStringSlice(execution.ResultBinaries)
	if err != nil {
		return fmt.Errorf("encode binaries: %w", err)
	}

	decimals := sql.NullString{}
	if decimalsJSON != "" {
		decimals = sql.NullString{String: decimalsJSON, Valid: true}
	}

	binaries := sql.NullString{}
	if binariesJSON != "" {
		binaries = sql.NullString{String: binariesJSON, Valid: true}
	}

	if _, err := r.db.ExecContext(
		ctx,
		query,
		execution.ID,
		execution.CommandType,
		execution.Payload,
		execution.ResultKind,
		resultText,
		decimals,
		binaries,
		execution.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert command execution: %w", err)
	}

	return nil
}

// FindByID は指定された ID の履歴を取得する。
func (r *SQLiteCommandRepository) FindByID(ctx context.Context, id string) (usecases.CommandExecution, error) {
	const query = `
SELECT id, command_type, payload, result_kind, result_text, result_decimals, result_binaries, created_at
FROM command_executions
WHERE id = ?
`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanExecution(row)
}

// ListRecent は作成日時の降順で履歴を取得する。
func (r *SQLiteCommandRepository) ListRecent(ctx context.Context, limit int) ([]usecases.CommandExecution, error) {
	const query = `
SELECT id, command_type, payload, result_kind, result_text, result_decimals, result_binaries, created_at
FROM command_executions
ORDER BY created_at DESC
LIMIT ?
`

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer rows.Close()

	executions := make([]usecases.CommandExecution, 0, limit)
	for rows.Next() {
		execution, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate executions: %w", err)
	}

	return executions, nil
}

func scanExecution(scanner interface {
	Scan(dest ...any) error
}) (usecases.CommandExecution, error) {
	var (
		execution          usecases.CommandExecution
		resultText         sql.NullString
		resultDecimalsJSON sql.NullString
		resultBinariesJSON sql.NullString
	)

	if err := scanner.Scan(
		&execution.ID,
		&execution.CommandType,
		&execution.Payload,
		&execution.ResultKind,
		&resultText,
		&resultDecimalsJSON,
		&resultBinariesJSON,
		&execution.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return usecases.CommandExecution{}, usecases.ErrExecutionNotFound
		}
		return usecases.CommandExecution{}, fmt.Errorf("scan execution: %w", err)
	}

	if resultText.Valid {
		txt := resultText.String
		execution.ResultText = &txt
	}

	if resultDecimalsJSON.Valid {
		values, err := decodeIntSlice(resultDecimalsJSON.String)
		if err != nil {
			return usecases.CommandExecution{}, fmt.Errorf("decode decimals: %w", err)
		}
		execution.ResultDecimals = values
	}

	if resultBinariesJSON.Valid {
		values, err := decodeStringSlice(resultBinariesJSON.String)
		if err != nil {
			return usecases.CommandExecution{}, fmt.Errorf("decode binaries: %w", err)
		}
		execution.ResultBinaries = values
	}

	execution.CreatedAt = execution.CreatedAt.UTC()

	return execution, nil
}

func encodeIntSlice(values []int) (string, error) {
	if values == nil {
		return "", nil
	}
	bytes, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func encodeStringSlice(values []string) (string, error) {
	if values == nil {
		return "", nil
	}
	bytes, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func decodeIntSlice(raw string) ([]int, error) {
	var values []int
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func decodeStringSlice(raw string) ([]string, error) {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	return values, nil
}
