package domain

import (
	"fmt"
)

type Command struct {
	Type    Type    // 命令の種類
	Payload Payload // 変換の対象となるリテラル
}

// Type の定義
type Type int

const (
	CommandUnknown Type = iota
	StringToASCII
	ASCIIToString
)

// Payload 定義
type (
	Payload       interface{ isPayload() } // isPayload() メソッドを持つ型という条件付与
	StringPayload struct{ Text string }
	ASCIIPayload  struct{ Bytes []byte } // ASCII 専用の生バイト列
)

// インターフェースの実装
func (StringPayload) isPayload() {}
func (ASCIIPayload) isPayload()  {}

// 専用のコンストラクタ関数を作り、どちらのフィールドを用いるかをわかりやすくする
// 関連: result.go の実装

func NewStringCommand(t string) (Command, error) {
	return Command{
		Type:    StringToASCII,
		Payload: StringPayload{Text: t},
	}, nil
}

func NewASCIICommand(bytes []byte) (Command, error) {
	// ビジネスロジックとしてASCIIしか持たないのでここでバリデーション
	for i, b := range bytes {
		if b > 0x7F {
			return Command{},
				fmt.Errorf("validate ascii: %w", &ErrNonASCIIError{Index: i, Byte: b})
		}
	}

	return Command{
		Type:    ASCIIToString,
		Payload: ASCIIPayload{Bytes: bytes},
	}, nil
}
