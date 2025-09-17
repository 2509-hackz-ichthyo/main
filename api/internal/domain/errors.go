package domain

import "errors"

var (
	// ErrInvalidCommandType は未サポートの命令種別が与えられた場合に返される。
	ErrInvalidCommandType = errors.New("domain: invalid command type")

	// ErrTypeMismatch はコンポーネントと命令種別が整合しない場合に返される。
	ErrTypeMismatch = errors.New("domain: command type mismatch for component")

	// ErrInvalidPayload はペイロードの構文または値が不正な場合に返される。
	ErrInvalidPayload = errors.New("domain: invalid command payload")
)
