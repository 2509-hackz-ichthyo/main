package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

func TestWhitespaceUsecaseInvalidCommandType(t *testing.T) {
	usecase := NewWhitespaceUsecase()

	_, err := usecase.Execute(context.Background(), WhitespaceCommand{
		CommandType: "Unknown",
		Payload:     []string{"dummy"},
	})

	if err == nil || !errors.Is(err, domain.ErrInvalidCommandType) {
		t.Fatalf("expected domain.ErrInvalidCommandType, got %v", err)
	}
}

func TestWhitespaceUsecaseTypeMismatch(t *testing.T) {
	original := parseCommandTypeFunc
	parseCommandTypeFunc = func(string) (domain.CommandType, error) {
		return domain.CommandType("Unsupported"), nil
	}
	defer func() { parseCommandTypeFunc = original }()

	usecase := NewWhitespaceUsecase()

	_, err := usecase.Execute(context.Background(), WhitespaceCommand{
		CommandType: string(domain.CommandTypeWhitespaceToBinary),
		Payload:     []string{"   \t \t\t\n    \t\t \n   \t\t \t  \t \n"},
	})

	if err == nil || !errors.Is(err, domain.ErrTypeMismatch) {
		t.Fatalf("expected domain.ErrTypeMismatch, got %v", err)
	}
}

func TestWhitespaceUsecaseEmptyPayload(t *testing.T) {
	usecase := NewWhitespaceUsecase()

	_, err := usecase.Execute(context.Background(), WhitespaceCommand{
		CommandType: string(domain.CommandTypeWhitespaceToBinary),
	})

	if err == nil || !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestWhitespaceUsecaseWhitespaceToBinaryInvalidPayload(t *testing.T) {
	usecase := NewWhitespaceUsecase()

	sentence := "   abcd\n   abcd\n   abcdefgh"
	command := WhitespaceCommand{CommandType: "WhitespaceToBinary", Payload: []string{sentence}}

	_, err := usecase.Execute(context.Background(), command)
	if err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
	}
}

func TestWhitespaceUsecaseWhitespaceToDecimalInvalidPayload(t *testing.T) {
	usecase := NewWhitespaceUsecase()

	sentence := ""
	command := WhitespaceCommand{CommandType: "WhitespaceToDecimal", Payload: []string{sentence}}

	_, err := usecase.Execute(context.Background(), command)
	if err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
	}
}

func TestWhitespaceUsecaseDecimalToWhitespaceInvalidDecimal(t *testing.T) {
	usecase := NewWhitespaceUsecase()

	command := WhitespaceCommand{CommandType: "DecimalToWhitespace", Payload: []string{"1 2"}}

	_, err := usecase.Execute(context.Background(), command)
	if err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
	}
}

func TestWhitespaceUsecaseBinariesToWhitespaceInvalidBinary(t *testing.T) {
	usecase := NewWhitespaceUsecase()

	command := WhitespaceCommand{CommandType: "BinariesToWhitespace", Payload: []string{"1010"}}

	_, err := usecase.Execute(context.Background(), command)
	if err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
	}
}

func TestDecimalToWhitespaceBitsToWhitespaceError(t *testing.T) {
	original := bitsToWhitespaceFunc
	bitsToWhitespaceFunc = func(string) (string, error) {
		return "", fmt.Errorf("forced error")
	}
	defer func() { bitsToWhitespaceFunc = original }()

	_, err := (WhitespaceUsecase{}).decimalToWhitespace([]string{"0 0 0"})
	if err == nil || err.Error() != "forced error" {
		t.Fatalf("expected forced error, got %v", err)
	}
}

func TestBinaryToWhitespaceBitsToWhitespaceError(t *testing.T) {
	originalBits := bitsToWhitespaceFunc
	bitsToWhitespaceFunc = func(string) (string, error) {
		return "", fmt.Errorf("bits error")
	}
	originalNormalize := normalizeBinaryStringFunc
	normalizeBinaryStringFunc = func(input string) (string, error) {
		return "0000000000000000", nil
	}
	defer func() {
		bitsToWhitespaceFunc = originalBits
		normalizeBinaryStringFunc = originalNormalize
	}()

	_, err := (WhitespaceUsecase{}).binaryToWhitespace([]string{"0000000000000000"})
	if err == nil || err.Error() != "bits error" {
		t.Fatalf("expected bits error, got %v", err)
	}
}

func TestDecimalStringToBinaryErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "too few tokens", input: "1 2"},
		{name: "non integer", input: "a b c"},
		{name: "negative", input: "-1 0 0"},
		{name: "out of range", input: "16 0 0"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decimalStringToBinary(tc.input); err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
				t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
			}
		})
	}
}

func TestNormalizeBinaryStringErrors(t *testing.T) {
	cases := map[string]string{
		"blank":          "",
		"invalid length": "1010",
		"invalid rune":   "0000000000000002",
	}

	for name, input := range cases {
		input := input
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeBinaryString(input); err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
				t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
			}
		})
	}
}

func TestBitsToWhitespaceErrors(t *testing.T) {
	cases := map[string]string{
		"invalid length": "1010",
		"invalid rune":   "0000000000000002",
	}
	for name, input := range cases {
		input := input
		t.Run(name, func(t *testing.T) {
			if _, err := bitsToWhitespace(input); err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
				t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
			}
		})
	}
}

func TestExtractSegmentsErrors(t *testing.T) {
	cases := map[string]string{
		"blank":              "",
		"missing prefix":     "abc",
		"insufficient lines": "   abcd\n   abcd",
		"invalid length":     "   abcd\n   abcd\n   abcdefg",
	}

	for name, sentence := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := extractSegments(sentence); err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
				t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
			}
		})
	}
}

func TestParseWhitespaceSentenceUnsupportedRune(t *testing.T) {
	sentence := "   abcd\n   abcd\n   abcdefgh"
	if _, err := parseWhitespaceSentence(sentence); err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
	}
}

func TestParseWhitespaceSentenceInvalidBinary(t *testing.T) {
	original := extractSegmentsFunc
	extractSegmentsFunc = func(string) ([]string, error) {
		return []string{"", "", ""}, nil
	}
	defer func() { extractSegmentsFunc = original }()

	if _, err := parseWhitespaceSentence("dummy"); err == nil || !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected domain.ErrInvalidPayload, got %v", err)
	}
}
