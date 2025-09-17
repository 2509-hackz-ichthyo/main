package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Encoder は 10 進数列から Whitespace 文字列を生成する純粋なコンポーネントを表す。
type Encoder interface {
	Execute(Command) (Result, error)
}

// NewWhitespaceEncoder は Whitespace 変換専用の Encoder を生成する。
func NewWhitespaceEncoder() Encoder {
	return whitespaceEncoder{}
}

type whitespaceEncoder struct{}

var decimalsToWhitespace = map[int]rune{
	9:  '\t',
	10: '\n',
	32: ' ',
}

// Execute は DecimalToWhitespace コマンドを評価し、対応する Whitespace 文字列を返す。
func (whitespaceEncoder) Execute(cmd Command) (Result, error) {
	if cmd.Type() != CommandTypeDecimalToWhitespace {
		return Result{}, ErrTypeMismatch
	}

	payload := strings.TrimSpace(cmd.Payload())
	if payload == "" {
		return Result{}, fmt.Errorf("%w: payload must contain at least one decimal", ErrInvalidPayload)
	}

	tokens := strings.Fields(payload)
	var builder strings.Builder
	builder.Grow(len(tokens))

	for _, token := range tokens {
		value, err := strconv.Atoi(token)
		if err != nil {
			return Result{}, fmt.Errorf("%w: token %q is not an integer", ErrInvalidPayload, token)
		}

		whitespace, ok := decimalsToWhitespace[value]
		if !ok {
			return Result{}, fmt.Errorf("%w: unsupported decimal %d", ErrInvalidPayload, value)
		}

		builder.WriteRune(whitespace)
	}

	return NewWhitespaceResult(builder.String()), nil
}
