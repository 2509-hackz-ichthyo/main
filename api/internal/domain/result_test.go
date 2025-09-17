package domain

import "testing"

func TestNewWhitespaceResult(t *testing.T) {
	t.Parallel()

	result := NewWhitespaceResult(" \t\n")

	if result.Kind() != ResultKindWhitespace {
		t.Fatalf("unexpected result kind: %v", result.Kind())
	}

	text, ok := result.Text()
	if !ok {
		t.Fatalf("expected text result")
	}

	if text != " \t\n" {
		t.Fatalf("unexpected text: %q", text)
	}

	if _, ok := result.Decimals(); ok {
		t.Fatalf("did not expect decimal sequence")
	}
}

func TestNewDecimalResult(t *testing.T) {
	t.Parallel()

	result := NewDecimalResult([]int{32, 9, 10})

	if result.Kind() != ResultKindDecimalSequence {
		t.Fatalf("unexpected result kind: %v", result.Kind())
	}

	decimals, ok := result.Decimals()
	if !ok {
		t.Fatalf("expected decimal sequence")
	}

	decimals[0] = 0

	decimalsAgain, ok := result.Decimals()
	if !ok {
		t.Fatalf("expected decimal sequence on second read")
	}

	if decimalsAgain[0] != 32 {
		t.Fatalf("decimals slice should be defensive copy")
	}

	formatted, ok := result.DecimalString(",")
	if !ok {
		t.Fatalf("expected decimal string")
	}

	if formatted != "32,9,10" {
		t.Fatalf("unexpected formatted: %s", formatted)
	}

	if _, ok := result.Text(); ok {
		t.Fatalf("did not expect text result")
	}
}

func TestDecimalStringRejectsWrongKind(t *testing.T) {
	t.Parallel()

	result := NewWhitespaceResult(" ")

	if _, ok := result.DecimalString(" "); ok {
		t.Fatalf("decimal string should not be available for whitespace result")
	}
}

func TestNewBinarySequenceResult(t *testing.T) {
	t.Parallel()

	result := NewBinarySequenceResult([]string{"1010", "0001"})

	if result.Kind() != ResultKindBinarySequence {
		t.Fatalf("unexpected result kind: %v", result.Kind())
	}

	binaries, ok := result.Binaries()
	if !ok {
		t.Fatalf("expected binary sequence")
	}

	binaries[0] = "0000"

	binariesAgain, ok := result.Binaries()
	if !ok {
		t.Fatalf("expected binary sequence on second read")
	}

	if binariesAgain[0] != "1010" {
		t.Fatalf("binaries slice should be defensive copy")
	}

	formatted, ok := result.BinaryString(" ")
	if !ok {
		t.Fatalf("expected binary string")
	}

	if formatted != "1010 0001" {
		t.Fatalf("unexpected formatted: %s", formatted)
	}

	if _, ok := result.Text(); ok {
		t.Fatalf("did not expect text result")
	}

	if _, ok := result.Decimals(); ok {
		t.Fatalf("did not expect decimal sequence")
	}
}

func TestBinaryStringRejectsWrongKind(t *testing.T) {
	t.Parallel()

	result := NewDecimalResult([]int{0})

	if _, ok := result.BinaryString(" "); ok {
		t.Fatalf("binary string should not be available for decimal result")
	}
}
