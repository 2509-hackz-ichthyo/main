package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2509-hackz-ichthyo/main/api/internal/app"
	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
	"github.com/gin-gonic/gin"
)

type stubUsecase struct {
	result          app.WhitespaceResult
	err             error
	receivedCommand app.WhitespaceCommand
}

func (s *stubUsecase) Execute(_ context.Context, cmd app.WhitespaceCommand) (app.WhitespaceResult, error) {
	s.receivedCommand = cmd
	return s.result, s.err
}

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	return ctx, recorder
}

func TestNewRouter_Healthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &stubUsecase{}
	r := NewRouter(uc)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("status field = %v, want ok", body["status"])
	}
}

func TestDecodeHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &stubUsecase{
		result: app.WhitespaceResult{
			CommandType:    domain.CommandTypeWhitespaceToBinary,
			ResultKind:     domain.ResultKindBinarySequence,
			ResultBinaries: []string{"0101", "1010"},
		},
	}
	r := NewRouter(usecase)

	payload := `{"command_type":"WhitespaceToBinary","payload":["%20%20%20%20%0A"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/decode", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if usecase.receivedCommand.CommandType != "WhitespaceToBinary" {
		t.Fatalf("command type = %q, want %q", usecase.receivedCommand.CommandType, "WhitespaceToBinary")
	}

	if got := usecase.receivedCommand.Payload[0]; got != "    \n" {
		t.Fatalf("normalized payload = %q, want %q", got, "    \n")
	}

	var resp decodeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.BinaryString == nil || *resp.BinaryString != "0101 1010" {
		t.Fatalf("BinaryString = %v, want 0101 1010", resp.BinaryString)
	}
}

func TestDecodeHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &stubUsecase{}
	r := NewRouter(uc)

	req := httptest.NewRequest(http.MethodPost, "/v1/decode", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecodeHandler_NormalizePayloadError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &stubUsecase{}
	r := NewRouter(uc)

	payload := `{"command_type":"WhitespaceToBinary","payload":["%ZZ"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/decode", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecodeHandler_UsecaseErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	errs := []error{
		app.ErrValidationFailed,
		domain.ErrInvalidPayload,
		domain.ErrInvalidCommandType,
		domain.ErrTypeMismatch,
		errors.New("boom"),
	}

	statuses := []int{
		http.StatusBadRequest,
		http.StatusBadRequest,
		http.StatusBadRequest,
		http.StatusBadRequest,
		http.StatusInternalServerError,
	}

	for i, err := range errs {
		usecase := &stubUsecase{err: err}
		r := NewRouter(usecase)

		payload := `{"command_type":"WhitespaceToBinary","payload":["%20%20%20%20%0A"]}`
		req := httptest.NewRequest(http.MethodPost, "/v1/decode", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != statuses[i] {
			t.Fatalf("case %d status = %d, want %d", i, rec.Code, statuses[i])
		}
	}
}

func TestWriteError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := newTestContext()

	writeError(ctx, http.StatusTeapot, "message", errors.New("detail"))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body["error"] != "message" || body["details"] != "detail" {
		t.Fatalf("unexpected response body: %+v", body)
	}
}

func TestStringListUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect []string
		error  bool
	}{
		{name: "array", input: `["a","b"]`, expect: []string{"a", "b"}},
		{name: "string", input: `"abc"`, expect: []string{"abc"}},
		{name: "empty", input: ` `, expect: nil},
		{name: "arrayInvalid", input: `[123]`, error: true},
		{name: "stringInvalid", input: "\"", error: true},
		{name: "invalid", input: `123`, error: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var list stringList
			err := list.UnmarshalJSON([]byte(tc.input))
			if tc.error {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(list) != len(tc.expect) {
				t.Fatalf("len = %d, want %d", len(list), len(tc.expect))
			}

			for i := range list {
				if list[i] != tc.expect[i] {
					t.Fatalf("element %d = %q, want %q", i, list[i], tc.expect[i])
				}
			}
		})
	}
}

func TestNewDecodeResponse(t *testing.T) {
	result := app.WhitespaceResult{
		CommandType:             domain.CommandTypeDecimalToWhitespace,
		ResultKind:              domain.ResultKindWhitespace,
		ResultDecimals:          []string{"1", "2"},
		ResultBinaries:          []string{"0101"},
		ResultWhitespace:        []string{" ", "\t"},
		ResultWhitespaceEncoded: []string{"%20"},
	}

	resp := newDecodeResponse(result)

	if resp.DecimalString == nil || *resp.DecimalString != "1 2" {
		t.Fatalf("DecimalString = %v, want 1 2", resp.DecimalString)
	}

	if resp.BinaryString == nil || *resp.BinaryString != "0101" {
		t.Fatalf("BinaryString = %v, want 0101", resp.BinaryString)
	}

	empty := app.WhitespaceResult{}

	emptyResp := newDecodeResponse(empty)

	if emptyResp.DecimalString != nil || emptyResp.BinaryString != nil {
		t.Fatalf("expected nil strings for empty result")
	}
}

func TestNormalizePayload(t *testing.T) {
	t.Run("whitespace unescape", func(t *testing.T) {
		values, err := normalizePayload("WhitespaceToDecimal", []string{"%20%09"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if values[0] != " \t" {
			t.Fatalf("value = %q, want \" \\t\"", values[0])
		}
	})

	t.Run("decimal trim", func(t *testing.T) {
		values, err := normalizePayload("DecimalToWhitespace", []string{"  1 2 3  "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if values[0] != "1 2 3" {
			t.Fatalf("value = %q, want %q", values[0], "1 2 3")
		}
	})

	t.Run("binary trim", func(t *testing.T) {
		values, err := normalizePayload("BinariesToWhitespace", []string{" 0101 "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if values[0] != "0101" {
			t.Fatalf("value = %q, want %q", values[0], "0101")
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		if _, err := normalizePayload("WhitespaceToBinary", nil); err == nil {
			t.Fatalf("expected error but got nil")
		}
	})

	t.Run("invalid command", func(t *testing.T) {
		if _, err := normalizePayload("Unknown", []string{""}); err == nil {
			t.Fatalf("expected error but got nil")
		}
	})

	t.Run("invalid unescape", func(t *testing.T) {
		if _, err := normalizePayload("WhitespaceToBinary", []string{"%ZZ"}); err == nil {
			t.Fatalf("expected error but got nil")
		}
	})

	t.Run("default branch", func(t *testing.T) {
		original := parseCommandTypeFn
		parseCommandTypeFn = func(string) (domain.CommandType, error) {
			return domain.CommandType("Custom"), nil
		}
		defer func() { parseCommandTypeFn = original }()

		values, err := normalizePayload("Custom", []string{" keep "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if values[0] != " keep " {
			t.Fatalf("value = %q, want %q", values[0], " keep ")
		}
	})
}
