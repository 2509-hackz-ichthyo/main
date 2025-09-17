package domain

import "testing"

func TestWhitespaceEncoderExecute(t *testing.T) {
	t.Parallel()

	encoder := NewWhitespaceEncoder()

	cmd := mustCommand(t, CommandTypeDecimalToWhitespace, "32 9 10")
	result, err := encoder.Execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text, ok := result.Text()
	if !ok {
		t.Fatalf("expected whitespace result, got %v", result.Kind())
	}

	if text != " \t\n" {
		t.Fatalf("unexpected whitespace text: %q", text)
	}
}

func TestWhitespaceEncoderRejectsInvalidDecimal(t *testing.T) {
	t.Parallel()

	encoder := NewWhitespaceEncoder()

	cmd := mustCommand(t, CommandTypeDecimalToWhitespace, "33")
	if _, err := encoder.Execute(cmd); err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestWhitespaceEncoderRejectsNonInteger(t *testing.T) {
	t.Parallel()

	encoder := NewWhitespaceEncoder()

	cmd := mustCommand(t, CommandTypeDecimalToWhitespace, "abc")
	if _, err := encoder.Execute(cmd); err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestWhitespaceEncoderRejectsEmptyPayload(t *testing.T) {
	t.Parallel()

	encoder := NewWhitespaceEncoder()

	cmd := mustCommand(t, CommandTypeDecimalToWhitespace, " ")
	if _, err := encoder.Execute(cmd); err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestWhitespaceEncoderTypeMismatch(t *testing.T) {
	t.Parallel()

	encoder := NewWhitespaceEncoder()
	cmd := mustCommand(t, CommandTypeWhitespaceToDecimal, " ")

	if _, err := encoder.Execute(cmd); err != ErrTypeMismatch {
		t.Fatalf("expected ErrTypeMismatch, got %v", err)
	}
}
