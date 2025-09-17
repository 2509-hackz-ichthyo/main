package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

// DecodeInput はデコーダユースケースが受け取る入力値を表す。
type DecodeInput struct {
	CommandType string
	Payload     string
}

// DecodeOutput はデコーダユースケースの出力値を表す。
type DecodeOutput struct {
	CommandType    domain.CommandType
	ResultKind     domain.ResultKind
	ResultDecimals []int
	ResultBinaries []string
}

// DecoderService はドメインの Decoder を利用してコマンドを評価する。
type DecoderService struct {
	decoder domain.Decoder
}

// NewDecoderService は DecoderService を生成する。
func NewDecoderService(decoder domain.Decoder) *DecoderService {
	return &DecoderService{decoder: decoder}
}

// Decode は入力値を検証し、対応する結果を返す。
func (s *DecoderService) Decode(_ context.Context, input DecodeInput) (DecodeOutput, error) {
	if strings.TrimSpace(input.CommandType) == "" {
		return DecodeOutput{}, fmt.Errorf("%w: commandType must not be blank", ErrValidationFailed)
	}
	if input.Payload == "" {
		return DecodeOutput{}, fmt.Errorf("%w: payload must not be blank", ErrValidationFailed)
	}

	commandType, err := domain.ParseCommandType(input.CommandType)
	if err != nil {
		return DecodeOutput{}, err
	}

	command, err := domain.NewCommand(commandType, input.Payload)
	if err != nil {
		return DecodeOutput{}, err
	}

	result, err := s.decoder.Execute(command)
	if err != nil {
		return DecodeOutput{}, err
	}

	output := DecodeOutput{
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
