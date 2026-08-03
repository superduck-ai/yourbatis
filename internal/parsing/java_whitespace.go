package parsing

// TrimJavaWhitespace matches java.lang.String.trim(), which removes leading and
// trailing characters with code points less than or equal to U+0020.
func TrimJavaWhitespace(value string) string {
	start := 0
	for start < len(value) && value[start] <= ' ' {
		start++
	}

	end := len(value)
	for start < end && value[end-1] <= ' ' {
		end--
	}

	return value[start:end]
}
