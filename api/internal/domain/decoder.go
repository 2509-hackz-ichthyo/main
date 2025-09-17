package domain

type Decoder interface {
	Decode(Command) (Result, error)
}

type DefaultDecoder struct{}

func (d DefaultDecoder) Decode(c Command) (Result, error) {
	// ペイロードが nil の場合はエラー
	if c.Payload == nil {
		return Result{}, ErrInvalidPayload
	}

	switch c.Type {
	case StringToASCII:
		// 各文字列を ASCII バイト列へ

		// Payload の型アサーション
		p, ok := c.Payload.(StringPayload)
		if !ok {
			return Result{}, ErrInvalidPayload
		}

		bytes := make([]byte, 0, len(p.Text))
		for _, r := range p.Text {
			if r > 0x7F { // ASCII外
				return Result{}, ErrNonASCII
			}
			bytes = append(bytes, byte(r))
		}
		return NewBytesResult(bytes), nil

	case ASCIIToString:
		// ASCII バイト列を文字列へ

		// Payload の型アサーション
		p, ok := c.Payload.(ASCIIPayload)
		if !ok {
			return Result{}, ErrInvalidPayload
		}

		// NewASCIICommand 側で ASCII 検証（バリデーション）済み
		return NewStringResult(string(p.Bytes)), nil

	default:
		return Result{}, ErrUnknownCommandType
	}
}
