package app

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

// WhitespaceCommand はユースケースが受け取る命令を表す。
// フロントの入力をそのまま保持し、検証はユースケース層で行う。
type WhitespaceCommand struct {
	CommandType string // 実行したい命令の種類（文字列表現）
	Payload     string // Whitespace または 10 進数列の文字列表現
}

// WhitespaceResult は変換結果を API 層へ渡すための DTO。
// デコードとエンコードの双方に対応するため、必要なフィールドのみが値を持つ。
type WhitespaceResult struct {
	CommandType             domain.CommandType
	ResultKind              domain.ResultKind
	ResultDecimals          []int
	ResultBinaries          []string
	ResultWhitespace        *string
	ResultWhitespaceEncoded *string
}

// WhitespaceUsecase はドメイン層の Decoder / Encoder を用いて命令を評価する。
// 受け取った入力を検証し、結果を適切な DTO に詰め替えて返却する。
type WhitespaceUsecase struct {
	decoder domain.Decoder
	encoder domain.Encoder
}

// NewWhitespaceUsecase は WhitespaceUsecase を生成する。
// デコーダとエンコーダの実装を引数で受け取り、テスト容易性を高める。
func NewWhitespaceUsecase(decoder domain.Decoder, encoder domain.Encoder) *WhitespaceUsecase {
	return &WhitespaceUsecase{decoder: decoder, encoder: encoder}
}

// Execute は入力を検証し、Whitespace の変換結果を返す。
func (u *WhitespaceUsecase) Execute(_ context.Context, command WhitespaceCommand) (WhitespaceResult, error) {
	if strings.TrimSpace(command.CommandType) == "" {
		return WhitespaceResult{}, fmt.Errorf("%w: commandType must not be blank", ErrValidationFailed)
	}
	if command.Payload == "" {
		return WhitespaceResult{}, fmt.Errorf("%w: payload must not be blank", ErrValidationFailed)
	}

	commandType, err := domain.ParseCommandType(command.CommandType)
	if err != nil {
		return WhitespaceResult{}, err
	}

	domainCommand, err := domain.NewCommand(commandType, command.Payload)
	if err != nil {
		return WhitespaceResult{}, err
	}

	var result domain.Result
	switch commandType {
	case domain.CommandTypeDecimalToWhitespace:
		result, err = u.encoder.Execute(domainCommand)
	default:
		result, err = u.decoder.Execute(domainCommand)
	}
	if err != nil {
		return WhitespaceResult{}, err
	}

	output := WhitespaceResult{
		CommandType: commandType,
		ResultKind:  result.Kind(),
	}

	if decimals, ok := result.Decimals(); ok {
		output.ResultDecimals = decimals
	}

	if binaries, ok := result.Binaries(); ok {
		output.ResultBinaries = binaries
	}

	if text, ok := result.Text(); ok {
		output.ResultWhitespace = &text
		encoded := url.PathEscape(text)
		output.ResultWhitespaceEncoded = &encoded
	}

	return output, nil
}
