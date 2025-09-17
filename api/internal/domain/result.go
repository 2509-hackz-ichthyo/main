package domain

type Result struct {
	String *string
	Bytes  *[]byte
}

// Result のフィールドの一方のみを使う想定なので、
// 専用のコンストラクタ関数を作り、どちらのフィールドを用いるかをわかりやすくする
// 使わない場合は次のように定義することになり、見分けづらい
//   r := Result{Text: &s}     // 文字列の結果
//   r := Result{Bytes: &nums} // 数値列の結果

func NewStringResult(s string) Result {
	return Result{String: &s}
}

func NewBytesResult(b []byte) Result {
	return Result{Bytes: &b}
}
