package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/app"
	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
	"github.com/gin-gonic/gin"
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

		command := app.WhitespaceCommand{
			CommandType: req.CommandType,
			Payload:     req.Payload,
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
	CommandType string `json:"command_type"`
	Payload     string `json:"payload"`
}

// decodeResponse はデコード結果のレスポンスボディ。
type decodeResponse struct {
	CommandType             string   `json:"command_type"`
	ResultKind              string   `json:"result_kind"`
	ResultDecimals          []int    `json:"result_decimals,omitempty"`
	ResultBinaries          []string `json:"result_binaries,omitempty"`
	DecimalString           *string  `json:"decimal_string,omitempty"`
	BinaryString            *string  `json:"binary_string,omitempty"`
	ResultWhitespace        *string  `json:"result_whitespace,omitempty"`
	ResultWhitespaceEncoded *string  `json:"result_whitespace_percent_encoded,omitempty"`
}

func newDecodeResponse(result app.WhitespaceResult) decodeResponse {
	resp := decodeResponse{
		CommandType:    string(result.CommandType),
		ResultKind:     string(result.ResultKind),
		ResultDecimals: result.ResultDecimals,
		ResultBinaries: result.ResultBinaries,
	}

	if len(result.ResultDecimals) > 0 {
		tokens := make([]string, len(result.ResultDecimals))
		for i, value := range result.ResultDecimals {
			tokens[i] = strconv.Itoa(value)
		}
		joined := strings.Join(tokens, " ")
		resp.DecimalString = &joined
	}

	if len(result.ResultBinaries) > 0 {
		joined := strings.Join(result.ResultBinaries, " ")
		resp.BinaryString = &joined
	}

	if result.ResultWhitespace != nil {
		resp.ResultWhitespace = result.ResultWhitespace
	}

	if result.ResultWhitespaceEncoded != nil {
		resp.ResultWhitespaceEncoded = result.ResultWhitespaceEncoded
	}

	return resp
}
