package app

import "errors"

var (
	// ErrValidationFailed は入力値の検証に失敗した場合に返される共通エラー。
	ErrValidationFailed = errors.New("app: validation failed")
)
