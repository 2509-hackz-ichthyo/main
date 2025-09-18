package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/app"
	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
	"github.com/gin-gonic/gin"
)

var (
	parseCommandTypeFn = domain.ParseCommandType
	pathUnescapeFn     = url.PathUnescape
)

// WhitespaceUsecase はハンドラが依存する最小限のインタフェースを表す。
// ユースケース層の実装は Execute メソッドのみを公開すれば良い。
type WhitespaceUsecase interface {
	Execute(ctx context.Context, command app.WhitespaceCommand) (app.WhitespaceResult, error)
}

// NewRouter は Gin の Engine を生成し、エンドポイントを束ねる。
// ここでミドルウェアやルーティングを一元的に設定する。
func NewRouter(uc WhitespaceUsecase) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().UTC()})
	})

	v1 := r.Group("/v1")
	{
		v1.POST("/decode", decodeHandler(uc))
	}

	return r
}

func decodeHandler(uc WhitespaceUsecase) gin.HandlerFunc {
	// decodeHandler は POST /v1/decode に届いたリクエストをユースケースへ委譲する。
	return func(c *gin.Context) {
		var req decodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "リクエストボディの形式が不正です", err)
			return
		}

		payloadSlice := make([]string, len(req.Payload))
		copy(payloadSlice, req.Payload)

		payload, err := normalizePayload(req.CommandType, payloadSlice)
		if err != nil {
			writeError(c, http.StatusBadRequest, "ペイロードが不正です", err)
			return
		}

		command := app.WhitespaceCommand{
			CommandType: req.CommandType,
			Payload:     payload,
		}

		result, err := uc.Execute(c.Request.Context(), command)
		if err != nil {
			handleUsecaseError(c, err)
			return
		}

		c.JSON(http.StatusOK, newDecodeResponse(result))
	}
}

func handleUsecaseError(c *gin.Context, err error) {
	// handleUsecaseError はユースケース層から返却されたエラーを HTTP ステータスへ写像する。
	switch {
	case errors.Is(err, app.ErrValidationFailed):
		writeError(c, http.StatusBadRequest, "入力値が不正です", err)
	case errors.Is(err, domain.ErrInvalidPayload):
		writeError(c, http.StatusBadRequest, "ペイロードが不正です", err)
	case errors.Is(err, domain.ErrInvalidCommandType):
		writeError(c, http.StatusBadRequest, "サポートされていない命令種別です", err)
	case errors.Is(err, domain.ErrTypeMismatch):
		writeError(c, http.StatusBadRequest, "命令と処理が一致しません", err)
	default:
		writeError(c, http.StatusInternalServerError, "内部エラーが発生しました", err)
	}
}

func writeError(c *gin.Context, status int, message string, err error) {
	// writeError は共通のエラーレスポンス JSON を構築して返す。
	c.JSON(status, gin.H{
		"error":   message,
		"details": err.Error(),
	})
}

// decodeRequest は POST /v1/decode のリクエストボディ。
type decodeRequest struct {
	CommandType string     `json:"command_type"`
	Payload     stringList `json:"payload"`
}

type stringList []string

func (s *stringList) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		*s = nil
		return nil
	}

	if trimmed[0] == '[' {
		var values []string
		if err := json.Unmarshal(trimmed, &values); err != nil {
			return err
		}
		*s = values
		return nil
	}

	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		*s = []string{value}
		return nil
	}

	return fmt.Errorf("payload must be a string or array of strings")
}

// decodeResponse はデコード結果のレスポンスボディ。
type decodeResponse struct {
	CommandType             string   `json:"command_type"`
	ResultKind              string   `json:"result_kind"`
	ResultDecimals          []string `json:"result_decimals,omitempty"`
	ResultBinaries          []string `json:"result_binaries,omitempty"`
	DecimalString           *string  `json:"decimal_string,omitempty"`
	BinaryString            *string  `json:"binary_string,omitempty"`
	ResultWhitespace        []string `json:"result_whitespace,omitempty"`
	ResultWhitespaceEncoded []string `json:"result_whitespace_percent_encoded,omitempty"`
}

func newDecodeResponse(result app.WhitespaceResult) decodeResponse {
	resp := decodeResponse{
		CommandType:             string(result.CommandType),
		ResultKind:              string(result.ResultKind),
		ResultDecimals:          result.ResultDecimals,
		ResultBinaries:          result.ResultBinaries,
		ResultWhitespace:        result.ResultWhitespace,
		ResultWhitespaceEncoded: result.ResultWhitespaceEncoded,
	}

	if len(result.ResultDecimals) > 0 {
		joined := strings.Join(result.ResultDecimals, " ")
		resp.DecimalString = &joined
	}

	if len(result.ResultBinaries) > 0 {
		joined := strings.Join(result.ResultBinaries, " ")
		resp.BinaryString = &joined
	}

	return resp
}

func normalizePayload(commandType string, values []string) ([]string, error) {
	ct, err := parseCommandTypeFn(commandType)
	if err != nil {
		return nil, err
	}

	normalized := make([]string, len(values))
	for i, value := range values {
		switch ct {
		case domain.CommandTypeWhitespaceToBinary, domain.CommandTypeWhitespaceToDecimal:
			decoded, err := pathUnescapeFn(value)
			if err != nil {
				return nil, fmt.Errorf("%w: failed to decode percent-encoded payload", domain.ErrInvalidPayload)
			}
			normalized[i] = decoded
		case domain.CommandTypeDecimalToWhitespace, domain.CommandTypeBinariesToWhitespace:
			normalized[i] = strings.TrimSpace(value)
		default:
			normalized[i] = value
		}
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("%w: payload must not be empty", app.ErrValidationFailed)
	}

	return normalized, nil
}
