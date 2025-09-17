package domain

import (
	"strconv"
	"strings"
)

// ResultKind は計算結果の形態を区別するラベルである。
type ResultKind string

const (
	// ResultKindWhitespace は Whitespace 文字列を保持する結果を示す。
	ResultKindWhitespace ResultKind = "Whitespace"

	// ResultKindDecimalSequence は 10 進数列を保持する結果を示す。
	ResultKindDecimalSequence ResultKind = "DecimalSequence"

	// ResultKindBinarySequence は 2 進数列を保持する結果を示す。
	ResultKindBinarySequence ResultKind = "BinarySequence"
)

// Result はコマンド実行の結果を表し、文字列または数列を保持する。
type Result struct {
	kind     ResultKind
	text     string
	decimals []int
	binaries []string
}

// NewWhitespaceResult は Whitespace 文字列を保持する Result を生成する。
func NewWhitespaceResult(text string) Result {
	return Result{kind: ResultKindWhitespace, text: text}
}

// NewDecimalResult は 10 進数列を保持する Result を生成する。
func NewDecimalResult(decimals []int) Result {
	clone := make([]int, len(decimals))
	copy(clone, decimals)
	return Result{kind: ResultKindDecimalSequence, decimals: clone}
}

// NewBinarySequenceResult は 2 進数列を保持する Result を生成する。
func NewBinarySequenceResult(binaries []string) Result {
	clone := make([]string, len(binaries))
	copy(clone, binaries)
	return Result{kind: ResultKindBinarySequence, binaries: clone}
}

// Kind は結果の種類を返す。
func (r Result) Kind() ResultKind {
	return r.kind
}

// Text は Whitespace 文字列を保持している場合に値と真偽値を返す。
func (r Result) Text() (string, bool) {
	if r.kind != ResultKindWhitespace {
		return "", false
	}
	return r.text, true
}

// Decimals は 10 進数列を保持している場合に防御的コピーと真偽値を返す。
func (r Result) Decimals() ([]int, bool) {
	if r.kind != ResultKindDecimalSequence {
		return nil, false
	}

	clone := make([]int, len(r.decimals))
	copy(clone, r.decimals)
	return clone, true
}

// Binaries は 2 進数列を保持している場合に防御的コピーと真偽値を返す。
func (r Result) Binaries() ([]string, bool) {
	if r.kind != ResultKindBinarySequence {
		return nil, false
	}

	clone := make([]string, len(r.binaries))
	copy(clone, r.binaries)
	return clone, true
}

// DecimalString は 10 進数列を指定された区切り文字で連結した文字列を返す。
// 10 進数列を保持していない場合は空文字列と false を返す。
func (r Result) DecimalString(separator string) (string, bool) {
	decimals, ok := r.Decimals()
	if !ok {
		return "", false
	}

	if len(decimals) == 0 {
		return "", true
	}

	tokens := make([]string, len(decimals))
	for i, value := range decimals {
		tokens[i] = strconv.Itoa(value)
	}

	return strings.Join(tokens, separator), true
}

// BinaryString は 2 進数列を指定された区切り文字で連結した文字列を返す。
// 2 進数列を保持していない場合は空文字列と false を返す。
func (r Result) BinaryString(separator string) (string, bool) {
	binaries, ok := r.Binaries()
	if !ok {
		return "", false
	}

	if len(binaries) == 0 {
		return "", true
	}

	return strings.Join(binaries, separator), true
}
