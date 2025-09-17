package usecases

import (
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

// ExecuteCommandInput はユースケース層が受け取るコマンド実行要求を表す。
type ExecuteCommandInput struct {
	// CommandType はドメインで定義された命令種別の文字列表現。
	CommandType string
	// Payload は命令に付随するペイロード文字列。
	Payload string
}

// ExecuteCommandOutput はコマンド実行後に返却するレスポンスを表す。
type ExecuteCommandOutput struct {
	// ID は永続化したコマンド履歴の識別子。
	ID string
	// CommandType は実行された命令の種類。
	CommandType domain.CommandType
	// ResultKind は成果物の形態。
	ResultKind domain.ResultKind
	// ResultText は ResultKind が Whitespace の場合のみ値を持つ。
	ResultText *string
	// ResultDecimals は ResultKind が DecimalSequence の場合のみ値を持つ。
	ResultDecimals []int
	// ResultBinaries は ResultKind が BinarySequence の場合のみ値を持つ。
	ResultBinaries []string
	// CreatedAt は履歴を記録した日時。
	CreatedAt time.Time
}

// CommandExecution は永続化されたコマンド履歴をユースケース層で扱うための構造体。
type CommandExecution struct {
	ID             string
	CommandType    domain.CommandType
	Payload        string
	ResultKind     domain.ResultKind
	ResultText     *string
	ResultDecimals []int
	ResultBinaries []string
	CreatedAt      time.Time
}

// ToOutput は履歴情報を ExecuteCommandOutput へ変換する補助関数。
func (c CommandExecution) ToOutput() ExecuteCommandOutput {
	return ExecuteCommandOutput{
		ID:             c.ID,
		CommandType:    c.CommandType,
		ResultKind:     c.ResultKind,
		ResultText:     cloneStringPointer(c.ResultText),
		ResultDecimals: cloneIntSlice(c.ResultDecimals),
		ResultBinaries: cloneStringSlice(c.ResultBinaries),
		CreatedAt:      c.CreatedAt,
	}
}

func cloneStringPointer(src *string) *string {
	if src == nil {
		return nil
	}
	clone := *src
	return &clone
}

func cloneIntSlice(src []int) []int {
	if len(src) == 0 {
		return nil
	}
	clone := make([]int, len(src))
	copy(clone, src)
	return clone
}

func cloneStringSlice(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	clone := make([]string, len(src))
	copy(clone, src)
	return clone
}
