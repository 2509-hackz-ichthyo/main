package domain

import "testing"

func TestParseCommandType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    CommandType
		wantErr bool
	}{
		{name: "WhitespaceToDecimal", input: string(CommandTypeWhitespaceToDecimal), want: CommandTypeWhitespaceToDecimal},
		{name: "WhitespaceToBinary", input: string(CommandTypeWhitespaceToBinary), want: CommandTypeWhitespaceToBinary},
		{name: "DecimalToWhitespace", input: string(CommandTypeDecimalToWhitespace), want: CommandTypeDecimalToWhitespace},
		{name: "BinariesToWhitespace", input: string(CommandTypeBinariesToWhitespace), want: CommandTypeBinariesToWhitespace},
		{name: "Invalid", input: "Unknown", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCommandType(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("unexpected command type: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(CommandTypeWhitespaceToDecimal, " \t\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Type() != CommandTypeWhitespaceToDecimal {
		t.Fatalf("unexpected command type: %v", cmd.Type())
	}

	if cmd.Payload() != " \t\n" {
		t.Fatalf("unexpected payload: %s", cmd.Payload())
	}

	if _, err := NewCommand(CommandType("invalid"), ""); err == nil {
		t.Fatalf("expected error for invalid command type")
	}
}
