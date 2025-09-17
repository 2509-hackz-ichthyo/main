package app

import (
	"context"
	"errors"
	"testing"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

func TestWhitespaceUsecaseExecuteDecimalToWhitespace(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase(domain.NewWhitespaceDecoder(), domain.NewWhitespaceEncoder())

	command := WhitespaceCommand{
		CommandType: string(domain.CommandTypeDecimalToWhitespace),
		Payload:     "32 9 10",
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ResultWhitespace == nil {
		t.Fatalf("expected whitespace result")
	}

	if *result.ResultWhitespace != " \t\n" {
		t.Fatalf("unexpected whitespace result: %q", *result.ResultWhitespace)
	}

	if result.ResultWhitespaceEncoded == nil || *result.ResultWhitespaceEncoded != "%20%09%0A" {
		t.Fatalf("unexpected encoded whitespace: %v", result.ResultWhitespaceEncoded)
	}
}

func TestWhitespaceUsecaseExecuteWhitespaceToDecimal(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase(domain.NewWhitespaceDecoder(), domain.NewWhitespaceEncoder())

	command := WhitespaceCommand{
		CommandType: string(domain.CommandTypeWhitespaceToDecimal),
		Payload:     " \t\n",
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{32, 9, 10}
	if len(result.ResultDecimals) != len(want) {
		t.Fatalf("unexpected decimals length: got %d want %d", len(result.ResultDecimals), len(want))
	}

	for i, value := range want {
		if result.ResultDecimals[i] != value {
			t.Fatalf("unexpected decimal at %d: got %d want %d", i, result.ResultDecimals[i], value)
		}
	}
}

func TestWhitespaceUsecaseExecuteWhitespaceToBinary(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase(domain.NewWhitespaceDecoder(), domain.NewWhitespaceEncoder())

	command := WhitespaceCommand{
		CommandType: string(domain.CommandTypeWhitespaceToBinary),
		Payload:     "       \n   \t\t\t\t\n",
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"0000", "1111"}
	if len(result.ResultBinaries) != len(want) {
		t.Fatalf("unexpected binaries length: got %d want %d", len(result.ResultBinaries), len(want))
	}

	for i, value := range want {
		if result.ResultBinaries[i] != value {
			t.Fatalf("unexpected binary at %d: got %s want %s", i, result.ResultBinaries[i], value)
		}
	}
}

func TestWhitespaceUsecaseValidation(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase(domain.NewWhitespaceDecoder(), domain.NewWhitespaceEncoder())

	_, err := usecase.Execute(context.Background(), WhitespaceCommand{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}
