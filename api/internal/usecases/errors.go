package usecases

import "errors"

var (
	// ErrValidationFailed は入力値の検証に失敗した場合に返す。
	ErrValidationFailed = errors.New("usecases: validation failed")
)
