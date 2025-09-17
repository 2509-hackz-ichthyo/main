package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
	"github.com/2509-hackz-ichthyo/main/api/internal/usecases"
	"github.com/gin-gonic/gin"
)

// DecoderUsecase はハンドラが利用するユースケースの最小インタフェース。
type DecoderUsecase interface {
	Decode(ctx context.Context, input usecases.DecodeInput) (usecases.DecodeOutput, error)
}

// NewRouter は Gin の Engine を生成しルーティングを設定する。
func NewRouter(decoderUC DecoderUsecase) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().UTC()})
	})

	v1 := r.Group("/v1")
	{
		v1.POST("/decode", decodeHandler(decoderUC))
	}

	return r
}

func decodeHandler(decoderUC DecoderUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req decodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "リクエストボディの形式が不正です", err)
			return
		}

		input := usecases.DecodeInput{
			CommandType: req.CommandType,
			Payload:     req.Payload,
		}

		output, err := decoderUC.Decode(c.Request.Context(), input)
		if err != nil {
			handleUsecaseError(c, err)
			return
		}

		c.JSON(http.StatusOK, newDecodeResponse(output))
	}
}

func handleUsecaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecases.ErrValidationFailed):
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
	CommandType    string   `json:"command_type"`
	ResultKind     string   `json:"result_kind"`
	ResultDecimals []int    `json:"result_decimals,omitempty"`
	ResultBinaries []string `json:"result_binaries,omitempty"`
	DecimalString  *string  `json:"decimal_string,omitempty"`
	BinaryString   *string  `json:"binary_string,omitempty"`
}

func newDecodeResponse(output usecases.DecodeOutput) decodeResponse {
	resp := decodeResponse{
		CommandType:    string(output.CommandType),
		ResultKind:     string(output.ResultKind),
		ResultDecimals: output.ResultDecimals,
		ResultBinaries: output.ResultBinaries,
	}

	if len(output.ResultDecimals) > 0 {
		tokens := make([]string, len(output.ResultDecimals))
		for i, value := range output.ResultDecimals {
			tokens[i] = strconv.Itoa(value)
		}
		joined := strings.Join(tokens, " ")
		resp.DecimalString = &joined
	}

	if len(output.ResultBinaries) > 0 {
		joined := strings.Join(output.ResultBinaries, " ")
		resp.BinaryString = &joined
	}

	return resp
}
