package domain

import (
	"bufio"
	"fmt"
	"net/url"
	"strings"
)

// Decoder は Whitespace 文字列を 10 進数列へ変換する純粋なコンポーネントを表す。
type Decoder interface {
	Execute(Command) (Result, error)
}

// NewWhitespaceDecoder は Whitespace 変換専用の Decoder を生成する。
func NewWhitespaceDecoder() Decoder {
	return whitespaceDecoder{}
}

type whitespaceDecoder struct{}

var whitespaceToDecimals = map[rune]int{
	'\t': 9,
	'\n': 10,
	' ':  32,
}

// Execute は Whitespace に関するコマンドを評価し、対応する結果を返す。
func (whitespaceDecoder) Execute(cmd Command) (Result, error) {
	switch cmd.Type() {
	case CommandTypeWhitespaceToDecimal:
		return decodeWhitespaceToDecimal(cmd.Payload())
	case CommandTypeWhitespaceToBinary:
		return decodeWhitespaceToBinary(cmd.Payload())
	default:
		return Result{}, ErrTypeMismatch
	}
}

// decodeWhitespaceToDecimal は Whitespace 文字列を 10 進数列へ変換する。
func decodeWhitespaceToDecimal(payload string) (Result, error) {
	if payload == "" {
		return Result{}, fmt.Errorf("%w: payload must not be empty", ErrInvalidPayload)
	}

	decimals := make([]int, 0, len(payload))
	for _, r := range payload {
		value, ok := whitespaceToDecimals[r]
		if !ok {
			return Result{}, fmt.Errorf("%w: unsupported rune %#U", ErrInvalidPayload, r)
		}
		decimals = append(decimals, value)
	}

	return NewDecimalResult(decimals), nil
}

// decodeWhitespaceToBinary は Whitespace 文字列を 2 進数列へ変換する。
func decodeWhitespaceToBinary(payload string) (Result, error) {
	if payload == "" {
		return Result{}, fmt.Errorf("%w: payload must not be empty", ErrInvalidPayload)
	}

	decoded, err := url.PathUnescape(payload)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(decoded))
	lineCount := 0
	binaries := make([]string, 0, 16)

	for scanner.Scan() {
		lineCount++
		if lineCount > 64 {
			return Result{}, fmt.Errorf("%w: too many sentences", ErrInvalidPayload)
		}

		binary, err := convertSentence(scanner.Text())
		if err != nil {
			return Result{}, err
		}

		binaries = append(binaries, binary)
	}

	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}

	if lineCount == 0 {
		return Result{},
			fmt.Errorf("%w: payload must contain at least one sentence",
				ErrInvalidPayload)
	}

	return NewBinarySequenceResult(binaries), nil
}

// convertSentence は Whitespace 文字列の一文を 2 進数文字列へ変換する。
func convertSentence(line string) (string, error) {
	const prefixSpaces = 3

	trimmed := strings.TrimSuffix(line, "\r")
	if len(trimmed) < prefixSpaces+4 {
		return "", fmt.Errorf("%w: sentence is too short", ErrInvalidPayload)
	}

	for i := range prefixSpaces {
		if trimmed[i] != ' ' {
			return "",
				fmt.Errorf("%w: sentence must start with three spaces", ErrInvalidPayload)
		}
	}

	body := trimmed[prefixSpaces:]
	if len(body) != 4 && len(body) != 8 {
		return "",
			fmt.Errorf("%w: sentence must contain 4 or 8 whitespace characters",
				ErrInvalidPayload)
	}

	var builder strings.Builder
	builder.Grow(len(body))

	for _, r := range body {
		switch r {
		case ' ':
			builder.WriteByte('0')
		case '\t':
			builder.WriteByte('1')
		default:
			return "", fmt.Errorf("%w: unsupported rune %#U", ErrInvalidPayload, r)
		}
	}

	return builder.String(), nil
}
