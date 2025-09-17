package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
	"github.com/google/uuid"
)

// CommandExecutor は CommandRepository とドメインサービスを仲介し、アプリケーションフローを担う。
type CommandExecutor struct {
	repo       CommandRepository
	decoder    domain.Decoder
	encoder    domain.Encoder
	ids        IDGenerator
	clock      Clock
	defaultLim int
}

// CommandExecutorOption は CommandExecutor のオプション設定を表す。
type CommandExecutorOption func(*CommandExecutor)

// WithIDGenerator は識別子生成器を差し替えるオプション。
func WithIDGenerator(generator IDGenerator) CommandExecutorOption {
	return func(executor *CommandExecutor) {
		if generator != nil {
			executor.ids = generator
		}
	}
}

// WithClock は時刻取得を差し替えるオプション。
func WithClock(clock Clock) CommandExecutorOption {
	return func(executor *CommandExecutor) {
		if clock != nil {
			executor.clock = clock
		}
	}
}

// WithDefaultLimit は履歴取得時のデフォルト件数を設定するオプション。
func WithDefaultLimit(limit int) CommandExecutorOption {
	return func(executor *CommandExecutor) {
		if limit > 0 {
			executor.defaultLim = limit
		}
	}
}

// NewCommandExecutor はユースケース実装を生成する。
func NewCommandExecutor(repo CommandRepository, decoder domain.Decoder, encoder domain.Encoder, opts ...CommandExecutorOption) *CommandExecutor {
	executor := &CommandExecutor{
		repo:       repo,
		decoder:    decoder,
		encoder:    encoder,
		ids:        defaultIDGenerator{},
		clock:      systemClock{},
		defaultLim: 20,
	}

	for _, opt := range opts {
		opt(executor)
	}

	return executor
}

// ExecuteCommand は入力を検証し、適切なドメインコンポーネントを呼び出して結果を永続化する。
func (c *CommandExecutor) ExecuteCommand(ctx context.Context, input ExecuteCommandInput) (ExecuteCommandOutput, error) {
	if err := validateExecuteInput(input); err != nil {
		return ExecuteCommandOutput{}, err
	}

	commandType, err := domain.ParseCommandType(input.CommandType)
	if err != nil {
		return ExecuteCommandOutput{}, fmt.Errorf("parse command type: %w", err)
	}

	command, err := domain.NewCommand(commandType, input.Payload)
	if err != nil {
		return ExecuteCommandOutput{}, fmt.Errorf("build command: %w", err)
	}

	result, err := c.dispatch(command)
	if err != nil {
		return ExecuteCommandOutput{}, err
	}

	execution := convertResultToExecution(c.ids.NewID(), command, result, c.clock.Now())

	if err := c.repo.Save(ctx, execution); err != nil {
		return ExecuteCommandOutput{}, fmt.Errorf("save execution: %w", err)
	}

	return execution.ToOutput(), nil
}

// GetExecution は識別子で履歴を取得する。
func (c *CommandExecutor) GetExecution(ctx context.Context, id string) (CommandExecution, error) {
	if strings.TrimSpace(id) == "" {
		return CommandExecution{}, fmt.Errorf("%w: id must not be blank", ErrValidationFailed)
	}
	execution, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return CommandExecution{}, err
	}
	return execution, nil
}

// ListExecutions は最新順で履歴を返す。
func (c *CommandExecutor) ListExecutions(ctx context.Context, limit int) ([]CommandExecution, error) {
	if limit <= 0 {
		limit = c.defaultLim
	}
	executions, err := c.repo.ListRecent(ctx, limit)
	if err != nil {
		return nil, err
	}
	return executions, nil
}

func (c *CommandExecutor) dispatch(cmd domain.Command) (domain.Result, error) {
	switch cmd.Type() {
	case domain.CommandTypeDecimalToWhitespace:
		return c.encoder.Execute(cmd)
	case domain.CommandTypeWhitespaceToDecimal, domain.CommandTypeWhitespaceToBinary:
		return c.decoder.Execute(cmd)
	default:
		return domain.Result{}, domain.ErrTypeMismatch
	}
}

func validateExecuteInput(input ExecuteCommandInput) error {
	if strings.TrimSpace(input.CommandType) == "" {
		return fmt.Errorf("%w: commandType must not be blank", ErrValidationFailed)
	}
	if input.Payload == "" {
		return fmt.Errorf("%w: payload must not be blank", ErrValidationFailed)
	}
	return nil
}

func convertResultToExecution(id string, cmd domain.Command, result domain.Result, now time.Time) CommandExecution {
	execution := CommandExecution{
		ID:          id,
		CommandType: cmd.Type(),
		Payload:     cmd.Payload(),
		ResultKind:  result.Kind(),
		CreatedAt:   now.UTC(),
	}

	if text, ok := result.Text(); ok {
		execution.ResultText = &text
		return execution
	}

	if decimals, ok := result.Decimals(); ok {
		execution.ResultDecimals = decimals
		return execution
	}

	if binaries, ok := result.Binaries(); ok {
		execution.ResultBinaries = binaries
		return execution
	}

	return execution
}

// Clock は現在時刻を取得するインタフェース。
type Clock interface {
	Now() time.Time
}

// IDGenerator は新しい識別子を発行するインタフェース。
type IDGenerator interface {
	NewID() string
}

type systemClock struct{}

type defaultIDGenerator struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func (defaultIDGenerator) NewID() string {
	return uuid.NewString()
}
