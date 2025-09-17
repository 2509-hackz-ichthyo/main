package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

// DecodeCommand はデコーダユースケースが受け取る命令を表す。
// フロントの入力をそのまま保持し、検証はユースケース層で行う。
type DecodeCommand struct {
	CommandType string // 実行したい命令の種類（文字列表現）
	Payload     string // Whitespace ソースコード文字列
}

// DecodingResult はデコード結果の表現を提供する。
// ドメイン層の Result を API 応答に変換しやすい形で保持する。
type DecodingResult struct {
	CommandType    domain.CommandType
	ResultKind     domain.ResultKind
	ResultDecimals []int
	ResultBinaries []string
}

// DecoderUsecase はドメイン層の Decoder を用いて命令を評価する役割を担う。
// 受け取った入力の検証と結果の整形を担当する。
type DecoderUsecase struct {
	decoder domain.Decoder
}

// NewDecoderUsecase は DecoderUsecase を生成する。
// デコーダ実装を差し替えたい場合に備えて依存を注入する。
func NewDecoderUsecase(decoder domain.Decoder) *DecoderUsecase {
	return &DecoderUsecase{decoder: decoder}
}

// Execute は入力を検証し、Whitespace のデコード結果を返す。
func (u *DecoderUsecase) Execute(_ context.Context, command DecodeCommand) (DecodingResult, error) {
	if strings.TrimSpace(command.CommandType) == "" {
		return DecodingResult{}, fmt.Errorf("%w: commandType must not be blank", ErrValidationFailed)
	}
	if command.Payload == "" {
		return DecodingResult{}, fmt.Errorf("%w: payload must not be blank", ErrValidationFailed)
	}

	commandType, err := domain.ParseCommandType(command.CommandType)
	if err != nil {
		return DecodingResult{}, err
	}

	domainCommand, err := domain.NewCommand(commandType, command.Payload)
	if err != nil {
		return DecodingResult{}, err
	}

	result, err := u.decoder.Execute(domainCommand)
	if err != nil {
		return DecodingResult{}, err
	}

	output := DecodingResult{
		CommandType: commandType,
		ResultKind:  result.Kind(),
	}

	if decimals, ok := result.Decimals(); ok {
		output.ResultDecimals = decimals
	}

	if binaries, ok := result.Binaries(); ok {
		output.ResultBinaries = binaries
	}

	return output, nil
}
