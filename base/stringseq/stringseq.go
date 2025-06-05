// Package stringseq provides functions for converting iterator sequences to strings.
package stringseq

import (
	"fmt"
	"iter"
	"strings"
)

// Append appends the elements of its second argument to the given string builder. The separator
// string sep is placed between elements in the resulting string.
func Append(b *strings.Builder, seq iter.Seq[string], sep string) {
	n := 0
	for item := range seq {
		if n > 0 {
			b.WriteString(sep)
		}
		b.WriteString(item)
		n++
	}
}

// AppendStringer appends the stringified elements of its second argument to the given string
// builder. The separator string sep is placed between elements in the resulting string.
func AppendStringer[T fmt.Stringer](b *strings.Builder, seq iter.Seq[T], sep string) {
	n := 0
	for item := range seq {
		if n > 0 {
			b.WriteString(sep)
		}
		b.WriteString(item.String())
		n++
	}
}

// Join concatenates the elements of its first argument to create a single string. The separator
// string sep is placed between elements in the resulting string.
func Join(seq iter.Seq[string], sep string) string {
	var b strings.Builder
	Append(&b, seq, sep)
	return b.String()
}

// JoinStringer concatenates the stringified elements of its first argument to create a single
// string. The separator string sep is placed between elements in the resulting string.
func JoinStringer[T fmt.Stringer](seq iter.Seq[T], sep string) string {
	var b strings.Builder
	AppendStringer(&b, seq, sep)
	return b.String()
}
