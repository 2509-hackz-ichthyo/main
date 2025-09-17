package app

import (
	"context"
	"errors"
	"testing"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

func TestDecoderUsecaseExecuteWhitespaceToDecimal(t *testing.T) {
	t.Parallel()

	usecase := NewDecoderUsecase(domain.NewWhitespaceDecoder())

	command := DecodeCommand{
		CommandType: string(domain.CommandTypeWhitespaceToDecimal),
		Payload:     " \t\n",
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{32, 9, 10}
	if len(result.ResultDecimals) != len(want) {
		t.Fatalf("unexpected decimals length: got %d, want %d", len(result.ResultDecimals), len(want))
	}

	for i, value := range want {
		if result.ResultDecimals[i] != value {
			t.Fatalf("unexpected decimal at %d: got %d, want %d", i, result.ResultDecimals[i], value)
		}
	}

	if result.CommandType != domain.CommandTypeWhitespaceToDecimal {
		t.Fatalf("unexpected command type: %v", result.CommandType)
	}
}

func TestDecoderUsecaseExecuteWhitespaceToBinary(t *testing.T) {
	t.Parallel()

	usecase := NewDecoderUsecase(domain.NewWhitespaceDecoder())

	payload := "       \n   \t\t\t\t\n"
	command := DecodeCommand{
		CommandType: string(domain.CommandTypeWhitespaceToBinary),
		Payload:     payload,
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"0000", "1111"}
	if len(result.ResultBinaries) != len(want) {
		t.Fatalf("unexpected binaries length: got %d, want %d", len(result.ResultBinaries), len(want))
	}

	for i, value := range want {
		if result.ResultBinaries[i] != value {
			t.Fatalf("unexpected binary at %d: got %s, want %s", i, result.ResultBinaries[i], value)
		}
	}

	if result.CommandType != domain.CommandTypeWhitespaceToBinary {
		t.Fatalf("unexpected command type: %v", result.CommandType)
	}
}

func TestDecoderUsecaseValidation(t *testing.T) {
	t.Parallel()

	usecase := NewDecoderUsecase(domain.NewWhitespaceDecoder())

	_, err := usecase.Execute(context.Background(), DecodeCommand{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}
