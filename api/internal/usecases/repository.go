package usecases

import "context"

// CommandRepository はコマンド実行履歴を永続化・取得するインタフェースである。
type CommandRepository interface {
	// Save は新しいコマンド実行履歴を永続化する。
	Save(ctx context.Context, execution CommandExecution) error
	// FindByID は識別子に一致する履歴を返す。
	FindByID(ctx context.Context, id string) (CommandExecution, error)
	// ListRecent は新しい順に履歴を返す。limit が 0 以下の場合は実装側のデフォルト値を利用する。
	ListRecent(ctx context.Context, limit int) ([]CommandExecution, error)
}
