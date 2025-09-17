package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNonASCII           = errors.New("non-ascii character detected")
	ErrUnknownCommandType = errors.New("unknown command type")
	ErrInvalidPayload     = errors.New("invalid payload for command type")
)

// 非ASCIIバイトを検出した場合に使用するカスタムエラー型
type ErrNonASCIIError struct {
	Index int
	Byte  byte
}

// エラーメッセージのフォーマットを定義
func (e *ErrNonASCIIError) Error() string {
	return fmt.Sprintf("out of range error at index %d: byte value %d", e.Index, e.Byte)
}

// errors.Is(err, ) を可能に
func (e *ErrNonASCIIError) Unwrap() error { return ErrNonASCII }
