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

	expected := " \t\n"
	if text != expected {
		t.Fatalf("unexpected whitespace output: %q", text)
	}
}

func TestWhitespaceEncoderRejectsInvalidDecimal(t *testing.T) {
	t.Parallel()

	encoder := NewWhitespaceEncoder()
	cmd := mustCommand(t, CommandTypeDecimalToWhitespace, "65 66 67")

	if _, err := encoder.Execute(cmd); err == nil {
		t.Fatalf("expected validation error but got nil")
	}
}

func TestWhitespaceEncoderRejectsTypeMismatch(t *testing.T) {
	t.Parallel()

	encoder := NewWhitespaceEncoder()
	cmd := mustCommand(t, CommandTypeWhitespaceToDecimal, " \t\n")

	if _, err := encoder.Execute(cmd); err != ErrTypeMismatch {
		t.Fatalf("expected ErrTypeMismatch but got %v", err)
	}
}

// mustCommand はテスト内で Command を生成し、失敗時にはテストを中断する補助関数。
func mustCommand(t *testing.T, commandType CommandType, payload string) Command {
	t.Helper()

	cmd, err := NewCommand(commandType, payload)
	if err != nil {
		t.Fatalf("failed to create command: %v", err)
	}

	return cmd
}
