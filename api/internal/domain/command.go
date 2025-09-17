package domain

import "fmt"

// CommandType はドメイン層で扱う命令種別を表す。
type CommandType string

const (
	// CommandTypeWhitespaceToDecimal は Whitespace 命令を 10 進数列に変換する種別を表す。
	CommandTypeWhitespaceToDecimal CommandType = "WhitespaceToDecimal"

	// CommandTypeWhitespaceToBinary は Whitespace 命令を 2 進数列に変換する種別を表す。
	CommandTypeWhitespaceToBinary CommandType = "WhitespaceToBinary"

	// CommandTypeDecimalToWhitespace は 10 進数列を Whitespace 文字列に変換する種別を表す。
	CommandTypeDecimalToWhitespace CommandType = "DecimalToWhitespace"
)

var supportedCommandTypes = map[CommandType]struct{}{
	CommandTypeWhitespaceToDecimal: {},
	CommandTypeWhitespaceToBinary:  {},
	CommandTypeDecimalToWhitespace: {},
}

// ParseCommandType は文字列を CommandType に変換し、未対応の値の場合はエラーを返す。
func ParseCommandType(raw string) (CommandType, error) {
	ct := CommandType(raw)
	if err := validateCommandType(ct); err != nil {
		return "", err
	}
	return ct, nil
}

// validateCommandType はサポート対象外の CommandType を検知してエラーを返す。
func validateCommandType(ct CommandType) error {
	if _, ok := supportedCommandTypes[ct]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidCommandType, string(ct))
	}
	return nil
}

// Command は命令種別とペイロードを保持するドメインオブジェクト。
type Command struct {
	commandType CommandType
	payload     string
}

// NewCommand は指定された種別が有効であることを確認したうえで Command を生成する。
func NewCommand(commandType CommandType, payload string) (Command, error) {
	if err := validateCommandType(commandType); err != nil {
		return Command{}, err
	}

	return Command{commandType: commandType, payload: payload}, nil
}

// Type は命令種別を返す。
func (c Command) Type() CommandType {
	return c.commandType
}

// Payload はペイロード文字列を返す。
func (c Command) Payload() string {
	return c.payload
}
