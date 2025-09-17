package app

import (
	"context"
	"errors"
	"testing"
)

func TestWhitespaceUsecaseWhitespaceToBinary(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase()

	sentence := "   \t \t\t\n    \t\t \n   \t\t \t  \t \n"
	command := WhitespaceCommand{
		CommandType: "WhitespaceToBinary",
		Payload:     []string{sentence},
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ResultBinaries) != 1 {
		t.Fatalf("expected 1 binary result, got %d", len(result.ResultBinaries))
	}

	if result.ResultBinaries[0] != "1011 0110 11010010" {
		t.Fatalf("unexpected binary string: %s", result.ResultBinaries[0])
	}
}

func TestWhitespaceUsecaseWhitespaceToDecimal(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase()

	sentences := []string{
		"   \t \t\t\n    \t\t \n   \t\t \t  \t \n",
		"       \n       \n           \n",
	}

	command := WhitespaceCommand{
		CommandType: "WhitespaceToDecimal",
		Payload:     sentences,
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := result.ResultDecimals, []string{"46802", "0"}; len(got) != len(want) {
		t.Fatalf("unexpected decimals length: got %d want %d", len(got), len(want))
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("unexpected decimal at %d: got %s want %s", i, got[i], want[i])
			}
		}
	}
}

func TestWhitespaceUsecaseDecimalToWhitespace(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase()

	command := WhitespaceCommand{
		CommandType: "DecimalToWhitespace",
		Payload:     []string{"46802"},
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "   \t \t\t\n    \t\t \n   \t\t \t  \t \n"
	if len(result.ResultWhitespace) != 1 {
		t.Fatalf("unexpected whitespace length: %d", len(result.ResultWhitespace))
	}
	if result.ResultWhitespace[0] != expected {
		t.Fatalf("unexpected whitespace output: %q", result.ResultWhitespace[0])
	}
	if len(result.ResultWhitespaceEncoded) != 1 {
		t.Fatalf("unexpected encoded length: %d", len(result.ResultWhitespaceEncoded))
	}
	if result.ResultWhitespaceEncoded[0] != "%20%20%20%09%20%09%09%0A%20%20%20%20%09%09%20%0A%20%20%20%09%09%20%09%20%20%09%20%0A" {
		t.Fatalf("unexpected encoded whitespace: %s", result.ResultWhitespaceEncoded[0])
	}
}

func TestWhitespaceUsecaseBinaryToWhitespace(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase()

	command := WhitespaceCommand{
		CommandType: "BinariesToWhitespace",
		Payload:     []string{"1011 0110 11010010"},
	}

	result, err := usecase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "   \t \t\t\n    \t\t \n   \t\t \t  \t \n"
	if len(result.ResultWhitespace) != 1 {
		t.Fatalf("unexpected whitespace length: %d", len(result.ResultWhitespace))
	}
	if result.ResultWhitespace[0] != expected {
		t.Fatalf("unexpected whitespace output: %q", result.ResultWhitespace[0])
	}
}

func TestWhitespaceUsecaseValidation(t *testing.T) {
	t.Parallel()

	usecase := NewWhitespaceUsecase()

	_, err := usecase.Execute(context.Background(), WhitespaceCommand{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}
