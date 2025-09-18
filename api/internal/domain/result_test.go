package domain

import "testing"

func TestResultWhitespace(t *testing.T) {
	t.Parallel()

	r := NewWhitespaceResult(" \t")

	if gotKind := r.Kind(); gotKind != ResultKindWhitespace {
		t.Fatalf("Kind() = %v, want %v", gotKind, ResultKindWhitespace)
	}

	if text, ok := r.Text(); !ok || text != " \t" {
		t.Fatalf("Text() = (%q, %v), want (\" \\t\", true)", text, ok)
	}

	if _, ok := r.Decimals(); ok {
		t.Fatalf("Decimals() ok = true, want false")
	}

	if _, ok := r.Binaries(); ok {
		t.Fatalf("Binaries() ok = true, want false")
	}

	if _, ok := r.DecimalString(","); ok {
		t.Fatalf("DecimalString() ok = true, want false")
	}

	if _, ok := r.BinaryString(","); ok {
		t.Fatalf("BinaryString() ok = true, want false")
	}
}

func TestResultDecimal(t *testing.T) {
	t.Parallel()

	decimals := []int{1, 2, 3}
	r := NewDecimalResult(decimals)

	if gotKind := r.Kind(); gotKind != ResultKindDecimalSequence {
		t.Fatalf("Kind() = %v, want %v", gotKind, ResultKindDecimalSequence)
	}

	t.Run("defensive copy", func(t *testing.T) {
		decimals[0] = 99

		copyVals, ok := r.Decimals()
		if !ok {
			t.Fatalf("Decimals() ok = false, want true")
		}

		if copyVals[0] != 1 {
			t.Fatalf("Decimals()[0] = %d, want 1", copyVals[0])
		}
	})

	t.Run("string joins", func(t *testing.T) {
		joined, ok := r.DecimalString(" ")
		if !ok {
			t.Fatalf("DecimalString() ok = false, want true")
		}

		if joined != "1 2 3" {
			t.Fatalf("DecimalString() = %q, want %q", joined, "1 2 3")
		}
	})

	if _, ok := r.Text(); ok {
		t.Fatalf("Text() ok = true, want false")
	}
}

func TestResultBinary(t *testing.T) {
	t.Parallel()

	binaries := []string{"0101", "1010"}
	r := NewBinarySequenceResult(binaries)

	if gotKind := r.Kind(); gotKind != ResultKindBinarySequence {
		t.Fatalf("Kind() = %v, want %v", gotKind, ResultKindBinarySequence)
	}

	t.Run("defensive copy", func(t *testing.T) {
		binaries[0] = "0000"

		copyVals, ok := r.Binaries()
		if !ok {
			t.Fatalf("Binaries() ok = false, want true")
		}

		if copyVals[0] != "0101" {
			t.Fatalf("Binaries()[0] = %q, want %q", copyVals[0], "0101")
		}
	})

	t.Run("string joins", func(t *testing.T) {
		joined, ok := r.BinaryString(",")
		if !ok {
			t.Fatalf("BinaryString() ok = false, want true")
		}

		if joined != "0101,1010" {
			t.Fatalf("BinaryString() = %q, want %q", joined, "0101,1010")
		}
	})

	if _, ok := r.Text(); ok {
		t.Fatalf("Text() ok = true, want false")
	}
}

func TestDecimalAndBinaryEmpty(t *testing.T) {
	t.Parallel()

	rDecimal := NewDecimalResult(nil)

	joinedDecimal, ok := rDecimal.DecimalString(",")
	if !ok {
		t.Fatalf("DecimalString() ok = false, want true")
	}
	if joinedDecimal != "" {
		t.Fatalf("DecimalString() = %q, want empty string", joinedDecimal)
	}

	rBinary := NewBinarySequenceResult(nil)

	joinedBinary, ok := rBinary.BinaryString(",")
	if !ok {
		t.Fatalf("BinaryString() ok = false, want true")
	}
	if joinedBinary != "" {
		t.Fatalf("BinaryString() = %q, want empty string", joinedBinary)
	}
}
