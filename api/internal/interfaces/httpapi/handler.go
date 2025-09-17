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

// CommandUsecase はハンドラが利用するユースケースの最小インタフェース。
type CommandUsecase interface {
	ExecuteCommand(ctx context.Context, input usecases.ExecuteCommandInput) (usecases.ExecuteCommandOutput, error)
	GetExecution(ctx context.Context, id string) (usecases.CommandExecution, error)
	ListExecutions(ctx context.Context, limit int) ([]usecases.CommandExecution, error)
}

// NewRouter は Gin の Engine を生成しルーティングを設定する。
func NewRouter(commandUC CommandUsecase) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().UTC()})
	})

	v1 := r.Group("/v1")
	{
		v1.POST("/commands", executeCommandHandler(commandUC))
		v1.POST("/decode", executeCommandHandler(commandUC)) // 互換のためのエイリアス
		v1.GET("/commands/:id", getExecutionHandler(commandUC))
		v1.GET("/commands", listExecutionsHandler(commandUC))
	}

	return r
}

func executeCommandHandler(commandUC CommandUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req executeCommandRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "リクエストボディの形式が不正です", err)
			return
		}

		input := usecases.ExecuteCommandInput{
			CommandType: req.CommandType,
			Payload:     req.Payload,
		}

		output, err := commandUC.ExecuteCommand(c.Request.Context(), input)
		if err != nil {
			handleUsecaseError(c, err)
			return
		}

		c.JSON(http.StatusCreated, newExecuteCommandResponse(output))
	}
}

func getExecutionHandler(commandUC CommandUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := commandUC.GetExecution(c.Request.Context(), id)
		if err != nil {
			handleUsecaseError(c, err)
			return
		}

		c.JSON(http.StatusOK, newCommandExecutionResponse(result))
	}
}

func listExecutionsHandler(commandUC CommandUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 0
		if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
			value, err := strconv.Atoi(rawLimit)
			if err != nil {
				writeError(c, http.StatusBadRequest, "limit は数値で指定してください", err)
				return
			}
			limit = value
		}

		results, err := commandUC.ListExecutions(c.Request.Context(), limit)
		if err != nil {
			handleUsecaseError(c, err)
			return
		}

		response := make([]commandExecutionResponse, len(results))
		for i, res := range results {
			response[i] = newCommandExecutionResponse(res)
		}

		c.JSON(http.StatusOK, response)
	}
}

func handleUsecaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecases.ErrValidationFailed):
		writeError(c, http.StatusBadRequest, "入力値が不正です", err)
	case errors.Is(err, usecases.ErrExecutionNotFound):
		writeError(c, http.StatusNotFound, "指定された履歴が見つかりません", err)
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

// executeCommandRequest は POST /v1/commands のリクエストボディ。
type executeCommandRequest struct {
	CommandType string `json:"command_type"`
	Payload     string `json:"payload"`
}

// executeCommandResponse はコマンド実行直後のレスポンスボディ。
type executeCommandResponse struct {
	ID             string    `json:"id"`
	CommandType    string    `json:"command_type"`
	ResultKind     string    `json:"result_kind"`
	ResultText     *string   `json:"result_text,omitempty"`
	ResultDecimals []int     `json:"result_decimals,omitempty"`
	ResultBinaries []string  `json:"result_binaries,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func newExecuteCommandResponse(output usecases.ExecuteCommandOutput) executeCommandResponse {
	return executeCommandResponse{
		ID:             output.ID,
		CommandType:    string(output.CommandType),
		ResultKind:     string(output.ResultKind),
		ResultText:     output.ResultText,
		ResultDecimals: output.ResultDecimals,
		ResultBinaries: output.ResultBinaries,
		CreatedAt:      output.CreatedAt,
	}
}

// commandExecutionResponse は履歴取得時のレスポンス。
type commandExecutionResponse struct {
	ID             string    `json:"id"`
	CommandType    string    `json:"command_type"`
	Payload        string    `json:"payload"`
	ResultKind     string    `json:"result_kind"`
	ResultText     *string   `json:"result_text,omitempty"`
	ResultDecimals []int     `json:"result_decimals,omitempty"`
	ResultBinaries []string  `json:"result_binaries,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func newCommandExecutionResponse(execution usecases.CommandExecution) commandExecutionResponse {
	return commandExecutionResponse{
		ID:             execution.ID,
		CommandType:    string(execution.CommandType),
		Payload:        execution.Payload,
		ResultKind:     string(execution.ResultKind),
		ResultText:     execution.ResultText,
		ResultDecimals: execution.ResultDecimals,
		ResultBinaries: execution.ResultBinaries,
		CreatedAt:      execution.CreatedAt,
	}
}
