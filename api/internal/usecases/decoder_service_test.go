package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

func TestDecoderServiceDecodeWhitespaceToDecimal(t *testing.T) {
	t.Parallel()

	service := NewDecoderService(domain.NewWhitespaceDecoder())

	input := DecodeInput{
		CommandType: string(domain.CommandTypeWhitespaceToDecimal),
		Payload:     " \t\n",
	}

	output, err := service.Decode(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{32, 9, 10}
	if len(output.ResultDecimals) != len(want) {
		t.Fatalf("unexpected decimals length: got %d, want %d", len(output.ResultDecimals), len(want))
	}

	for i, value := range want {
		if output.ResultDecimals[i] != value {
			t.Fatalf("unexpected decimal at %d: got %d, want %d", i, output.ResultDecimals[i], value)
		}
	}

	if output.CommandType != domain.CommandTypeWhitespaceToDecimal {
		t.Fatalf("unexpected command type: %v", output.CommandType)
	}
}

func TestDecoderServiceDecodeWhitespaceToBinary(t *testing.T) {
	t.Parallel()

	service := NewDecoderService(domain.NewWhitespaceDecoder())

	payload := "       \n   \t\t\t\t\n"
	input := DecodeInput{
		CommandType: string(domain.CommandTypeWhitespaceToBinary),
		Payload:     payload,
	}

	output, err := service.Decode(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"0000", "1111"}
	if len(output.ResultBinaries) != len(want) {
		t.Fatalf("unexpected binaries length: got %d, want %d", len(output.ResultBinaries), len(want))
	}

	for i, value := range want {
		if output.ResultBinaries[i] != value {
			t.Fatalf("unexpected binary at %d: got %s, want %s", i, output.ResultBinaries[i], value)
		}
	}

	if output.CommandType != domain.CommandTypeWhitespaceToBinary {
		t.Fatalf("unexpected command type: %v", output.CommandType)
	}
}

func TestDecoderServiceValidation(t *testing.T) {
	t.Parallel()

	service := NewDecoderService(domain.NewWhitespaceDecoder())

	_, err := service.Decode(context.Background(), DecodeInput{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}
