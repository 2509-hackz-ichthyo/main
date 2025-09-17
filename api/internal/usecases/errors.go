package usecases

import "errors"

var (
	// ErrExecutionNotFound は指定した識別子のコマンド履歴が存在しない場合に返す。
	ErrExecutionNotFound = errors.New("usecases: command execution not found")

	// ErrValidationFailed は入力値の検証に失敗した場合に返す。
	ErrValidationFailed = errors.New("usecases: validation failed")
)
