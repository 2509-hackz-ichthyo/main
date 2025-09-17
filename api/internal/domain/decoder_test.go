package domain

import (
	"net/url"
	"strings"
	"testing"
)

func TestWhitespaceDecoderExecute(t *testing.T) {
	t.Parallel()

	decoder := NewWhitespaceDecoder()
	cmd := mustCommand(t, CommandTypeWhitespaceToDecimal, " \t\n")

	result, err := decoder.Execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decimals, ok := result.Decimals()
	if !ok {
		t.Fatalf("expected decimal sequence, got %v", result.Kind())
	}

	expected := []int{32, 9, 10}
	if len(decimals) != len(expected) {
		t.Fatalf("unexpected decimal length: got %d want %d", len(decimals), len(expected))
	}

	for i, value := range decimals {
		if value != expected[i] {
			t.Fatalf("unexpected decimal at index %d: got %d want %d", i, value, expected[i])
		}
	}
}

func TestWhitespaceDecoderExecuteBinary(t *testing.T) {
	t.Parallel()

	decoder := NewWhitespaceDecoder()
	line1 := strings.Repeat(" ", 3) + " \t\t " + "\n"
	line2 := strings.Repeat(" ", 3) + string([]rune{'	', ' ', '	', ' ', ' ', '	', '	', ' '}) + "\n"
	cmd := mustCommand(t, CommandTypeWhitespaceToBinary, line1+line2)

	result, err := decoder.Execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binaries, ok := result.Binaries()
	if !ok {
		t.Fatalf("expected binary sequence, got %v", result.Kind())
	}

	if len(binaries) != 2 {
		t.Fatalf("unexpected binary length: got %d want %d", len(binaries), 2)
	}

	if binaries[0] != "0110" {
		t.Fatalf("unexpected binary at index 0: got %s want %s", binaries[0], "0110")
	}

	if binaries[1] != "10100110" {
		t.Fatalf("unexpected binary at index 1: got %s want %s", binaries[1], "10100110")
	}

	joined, ok := result.BinaryString(" ")
	if !ok {
		t.Fatalf("expected binary string")
	}

	if joined != "0110 10100110" {
		t.Fatalf("unexpected binary string: %s", joined)
	}

	if _, ok := result.Decimals(); ok {
		t.Fatalf("did not expect decimal sequence")
	}
}

func TestWhitespaceDecoderBinaryRejectsTooManySentences(t *testing.T) {
	t.Parallel()

	decoder := NewWhitespaceDecoder()

	var builder strings.Builder
	for i := 0; i < 65; i++ {
		builder.WriteString(strings.Repeat(" ", 3))
		builder.WriteString(" \t\t ")
		builder.WriteByte('\n')
	}

	cmd := mustCommand(t, CommandTypeWhitespaceToBinary, builder.String())

	if _, err := decoder.Execute(cmd); err == nil {
		t.Fatalf("expected validation error but got nil")
	}
}

func TestWhitespaceDecoderBinaryRejectsInvalidStructure(t *testing.T) {
	t.Parallel()

	decoder := NewWhitespaceDecoder()
	cmd := mustCommand(t, CommandTypeWhitespaceToBinary, "  \t\n")

	if _, err := decoder.Execute(cmd); err == nil {
		t.Fatalf("expected validation error but got nil")
	}
}

func TestWhitespaceDecoderBinaryRejectsUnsupportedRune(t *testing.T) {
	t.Parallel()

	decoder := NewWhitespaceDecoder()
	line := strings.Repeat(" ", 3) + "a bc" + "\n"
	cmd := mustCommand(t, CommandTypeWhitespaceToBinary, line)

	if _, err := decoder.Execute(cmd); err == nil {
		t.Fatalf("expected validation error but got nil")
	}
}

func TestWhitespaceDecoderBinarySupportsPercentEncoding(t *testing.T) {
	t.Parallel()

	decoder := NewWhitespaceDecoder()
	line := strings.Repeat(" ", 3) + "\t \t " + "\n"
	encoded := url.PathEscape(line)
	cmd := mustCommand(t, CommandTypeWhitespaceToBinary, encoded)

	result, err := decoder.Execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if joined, ok := result.BinaryString(" "); !ok || joined != "1010" {
		t.Fatalf("unexpected binary string: %s (ok=%v)", joined, ok)
	}
}

func TestWhitespaceDecoderRejectsInvalidCharacter(t *testing.T) {
	t.Parallel()

	decoder := NewWhitespaceDecoder()
	cmd := mustCommand(t, CommandTypeWhitespaceToDecimal, "ABC")

	if _, err := decoder.Execute(cmd); err == nil {
		t.Fatalf("expected validation error but got nil")
	}
}

func mustCommand(t *testing.T, commandType CommandType, payload string) Command {
	t.Helper()

	cmd, err := NewCommand(commandType, payload)
	if err != nil {
		t.Fatalf("unexpected error while building command: %v", err)
	}

	return cmd
}
