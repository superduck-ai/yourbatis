package yourbatis

import "strings"

// ValuePresent implements the non-null branch of MyBatis condition coercion
// for statically non-nil Go values such as strings, arrays, and structs. It is
// generic and performs no reflection.
func ValuePresent[T any](value T) bool {
	_ = value
	return true
}

// StringIndex is the generated-code adapter for MyBatis String.indexOf.
func StringIndex(value, substring string) int {
	return strings.Index(value, substring)
}

// StringIndexRune preserves the Character overload used by OGNL single-quoted
// one-rune literals.
func StringIndexRune(value string, character rune) int {
	return strings.IndexRune(value, character)
}
