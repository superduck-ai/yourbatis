package parsing

import "strings"

type GenericTokenParser struct {
	openToken  string
	closeToken string
	handler    TokenHandler
}

func NewGenericTokenParser(openToken, closeToken string, handler TokenHandler) *GenericTokenParser {
	return &GenericTokenParser{
		openToken:  openToken,
		closeToken: closeToken,
		handler:    handler,
	}
}

func (parser *GenericTokenParser) Parse(text string) string {
	if text == "" {
		return ""
	}

	start := strings.Index(text, parser.openToken)
	if start == -1 {
		return text
	}

	src := []byte(text)
	offset := 0
	var builder strings.Builder
	var expression *strings.Builder

	for {
		if start > 0 && src[start-1] == '\\' {
			builder.Write(src[offset : start-1])
			builder.WriteString(parser.openToken)
			offset = start + len(parser.openToken)
		} else {
			if expression == nil {
				expression = &strings.Builder{}
			} else {
				expression.Reset()
			}
			builder.Write(src[offset:start])
			offset = start + len(parser.openToken)
			end := indexFrom(text, parser.closeToken, offset)
			for end > -1 {
				if end <= offset || src[end-1] != '\\' {
					expression.Write(src[offset:end])
					break
				}
				expression.Write(src[offset : end-1])
				expression.WriteString(parser.closeToken)
				offset = end + len(parser.closeToken)
				end = indexFrom(text, parser.closeToken, offset)
			}

			if end == -1 {
				builder.Write(src[start:])
				offset = len(src)
			} else {
				tokenValue := parser.handler.HandleToken(expression.String())
				if tokenValue == nil {
					builder.WriteString("null")
				} else {
					builder.WriteString(*tokenValue)
				}
				offset = end + len(parser.closeToken)
			}
		}

		start = indexFrom(text, parser.openToken, offset)
		if start == -1 {
			break
		}
	}

	if offset < len(src) {
		builder.Write(src[offset:])
	}
	return builder.String()
}

func indexFrom(text, token string, offset int) int {
	if offset <= 0 {
		return strings.Index(text, token)
	}
	index := strings.Index(text[offset:], token)
	if index < 0 {
		return -1
	}
	return offset + index
}
