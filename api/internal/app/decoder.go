package app

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strings"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

// WhitespaceCommand はユースケースが受け取る命令を表す。
// Payload はクライアントから渡された順番を保持する。
type WhitespaceCommand struct {
	CommandType string   // 命令の種類（文字列表現）
	Payload     []string // 変換対象の配列（Whitespace / 2 進数 / 10 進数）
}

// WhitespaceResult は変換結果を API 層へ渡すための DTO。
type WhitespaceResult struct {
	CommandType             domain.CommandType
	ResultKind              domain.ResultKind
	ResultDecimals          []string
	ResultBinaries          []string
	ResultWhitespace        []string
	ResultWhitespaceEncoded []string
}

// WhitespaceUsecase は入力を検証し、各種フォーマット間の変換を担う。
type WhitespaceUsecase struct{}

// NewWhitespaceUsecase は WhitespaceUsecase を生成する。
func NewWhitespaceUsecase() *WhitespaceUsecase {
	return &WhitespaceUsecase{}
}

// Execute は入力を検証し、Whitespace の変換結果を返す。
func (u *WhitespaceUsecase) Execute(_ context.Context, command WhitespaceCommand) (WhitespaceResult, error) {
	if strings.TrimSpace(command.CommandType) == "" {
		return WhitespaceResult{}, fmt.Errorf("%w: commandType must not be blank", ErrValidationFailed)
	}
	if len(command.Payload) == 0 {
		return WhitespaceResult{}, fmt.Errorf("%w: payload must not be blank", ErrValidationFailed)
	}

	commandType, err := domain.ParseCommandType(command.CommandType)
	if err != nil {
		return WhitespaceResult{}, err
	}

	switch commandType {
	case domain.CommandTypeWhitespaceToBinary:
		return u.whitespaceToBinary(command.Payload)
	case domain.CommandTypeWhitespaceToDecimal:
		return u.whitespaceToDecimal(command.Payload)
	case domain.CommandTypeDecimalToWhitespace:
		return u.decimalToWhitespace(command.Payload)
	case domain.CommandTypeBinariesToWhitespace:
		return u.binaryToWhitespace(command.Payload)
	default:
		return WhitespaceResult{}, domain.ErrTypeMismatch
	}
}

func (WhitespaceUsecase) whitespaceToBinary(payload []string) (WhitespaceResult, error) {
	binaries := make([]string, len(payload))
	for i, sentence := range payload {
		binary, err := parseWhitespaceSentence(sentence)
		if err != nil {
			return WhitespaceResult{}, err
		}
		binaries[i] = binary
	}

	return WhitespaceResult{
		CommandType:    domain.CommandTypeWhitespaceToBinary,
		ResultKind:     domain.ResultKindBinarySequence,
		ResultBinaries: binaries,
	}, nil
}

func (WhitespaceUsecase) whitespaceToDecimal(payload []string) (WhitespaceResult, error) {
	decimals := make([]string, len(payload))
	for i, sentence := range payload {
		binary, err := parseWhitespaceSentence(sentence)
		if err != nil {
			return WhitespaceResult{}, err
		}

		value, ok := new(big.Int).SetString(binary, 2)
		if !ok {
			return WhitespaceResult{}, fmt.Errorf("%w: invalid binary sequence", domain.ErrInvalidPayload)
		}
		decimals[i] = value.String()
	}

	return WhitespaceResult{
		CommandType:    domain.CommandTypeWhitespaceToDecimal,
		ResultKind:     domain.ResultKindDecimalSequence,
		ResultDecimals: decimals,
	}, nil
}

func (WhitespaceUsecase) decimalToWhitespace(payload []string) (WhitespaceResult, error) {
	whitespaces := make([]string, len(payload))
	encoded := make([]string, len(payload))
	for i, decimal := range payload {
		binary, err := decimalStringToBinary(decimal)
		if err != nil {
			return WhitespaceResult{}, err
		}

		whitespace, err := binaryToWhitespace(binary)
		if err != nil {
			return WhitespaceResult{}, err
		}

		whitespaces[i] = whitespace
		encoded[i] = url.PathEscape(whitespace)
	}

	return WhitespaceResult{
		CommandType:             domain.CommandTypeDecimalToWhitespace,
		ResultKind:              domain.ResultKindWhitespace,
		ResultWhitespace:        whitespaces,
		ResultWhitespaceEncoded: encoded,
	}, nil
}

func (WhitespaceUsecase) binaryToWhitespace(payload []string) (WhitespaceResult, error) {
	whitespaces := make([]string, len(payload))
	encoded := make([]string, len(payload))
	for i, binary := range payload {
		whitespace, err := binaryToWhitespace(binary)
		if err != nil {
			return WhitespaceResult{}, err
		}
		whitespaces[i] = whitespace
		encoded[i] = url.PathEscape(whitespace)
	}

	return WhitespaceResult{
		CommandType:             domain.CommandTypeBinariesToWhitespace,
		ResultKind:              domain.ResultKindWhitespace,
		ResultWhitespace:        whitespaces,
		ResultWhitespaceEncoded: encoded,
	}, nil
}

func parseWhitespaceSentence(sentence string) (string, error) {
	segments, err := extractSegments(sentence)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, segment := range segments {
		for _, r := range segment {
			switch r {
			case ' ':
				builder.WriteByte('0')
			case '\t':
				builder.WriteByte('1')
			default:
				return "", fmt.Errorf("%w: unsupported rune %#U", domain.ErrInvalidPayload, r)
			}
		}
	}

	return builder.String(), nil
}

func decimalStringToBinary(decimal string) (string, error) {
	value, ok := new(big.Int).SetString(strings.TrimSpace(decimal), 10)
	if !ok {
		return "", fmt.Errorf("%w: decimal value %q is not a number", domain.ErrInvalidPayload, decimal)
	}
	if value.Sign() < 0 {
		return "", fmt.Errorf("%w: negative decimal %q", domain.ErrInvalidPayload, decimal)
	}
	if value.BitLen() > 16 {
		return "", fmt.Errorf("%w: decimal %q exceeds 16 bit", domain.ErrInvalidPayload, decimal)
	}

	return fmt.Sprintf("%016b", value.Uint64()), nil
}

func binaryToWhitespace(binary string) (string, error) {
	trimmed := strings.TrimSpace(binary)
	if trimmed == "" {
		return "", fmt.Errorf("%w: binary must not be blank", domain.ErrInvalidPayload)
	}
	if len(trimmed) != 16 {
		return "", fmt.Errorf("%w: binary must be 16 bits", domain.ErrInvalidPayload)
	}
	for _, r := range trimmed {
		if r != '0' && r != '1' {
			return "", fmt.Errorf("%w: binary contains invalid rune %#U", domain.ErrInvalidPayload, r)
		}
	}

	segments := []string{trimmed[:4], trimmed[4:8], trimmed[8:]}
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString("   ")
		for _, bit := range segment {
			if bit == '0' {
				builder.WriteByte(' ')
			} else {
				builder.WriteByte('\t')
			}
		}
		builder.WriteByte('\n')
	}

	return builder.String(), nil
}

func extractSegments(sentence string) ([]string, error) {
	if sentence == "" {
		return nil, fmt.Errorf("%w: sentence must not be blank", domain.ErrInvalidPayload)
	}

	normalized := strings.ReplaceAll(sentence, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	segments := make([]string, 0, 3)
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(line) < 3 || !strings.HasPrefix(line, "   ") {
			return nil, fmt.Errorf("%w: line must start with three spaces", domain.ErrInvalidPayload)
		}
		segments = append(segments, line[3:])
	}

	if len(segments) != 3 {
		return nil, fmt.Errorf("%w: sentence must contain three lines", domain.ErrInvalidPayload)
	}

	lengths := []int{4, 4, 8}
	for i, segment := range segments {
		if len([]rune(segment)) != lengths[i] {
			return nil, fmt.Errorf("%w: line %d must contain %d characters", domain.ErrInvalidPayload, i+1, lengths[i])
		}
	}

	return segments, nil
}
