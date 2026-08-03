package xmlmapper

import (
	"fmt"
	"go/parser"
	"strings"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func parseSQLText(text string, source ir.Span) ([]ir.Node, error) {
	result := make([]ir.Node, 0)
	offset := 0
	for offset < len(text) {
		start, marker := findNextSQLMarker(text, offset)
		if start < 0 {
			appendText(&result, text[offset:], source)
			break
		}
		if start > 0 && text[start-1] == '\\' {
			appendText(&result, text[offset:start-1]+marker, source)
			offset = start + 2
			continue
		}
		appendText(&result, text[offset:start], source)
		end, hasComment := findParameterEnd(text, start+2)
		if hasComment {
			if marker == "#{" {
				return nil, parameterError(source, "comments are not allowed in #{} parameter expressions")
			}
			return nil, parameterError(source, "comments are not allowed in ${} text expressions")
		}
		if end < 0 {
			return nil, fmt.Errorf(
				"%s:%d:%d: unclosed parameter starting at %q",
				source.File,
				source.Line,
				source.Column,
				text[start:],
			)
		}
		if marker == "#{" {
			parameter, err := parseParameter(text[start+2:end], source)
			if err != nil {
				return nil, err
			}
			result = append(result, parameter)
		} else {
			expression := strings.TrimSpace(text[start+2 : end])
			if expression == "" {
				return nil, parameterError(source, "empty ${} text expression")
			}
			result = append(result, &ir.TextExpression{
				Expression: expression,
				Source:     source,
			})
		}
		offset = end + 1
	}
	return result, nil
}

func findNextSQLMarker(text string, offset int) (int, string) {
	for index := offset; index+1 < len(text); index++ {
		if (text[index] == '#' || text[index] == '$') && text[index+1] == '{' {
			return index, text[index : index+2]
		}
	}
	return -1, ""
}

func appendText(nodes *[]ir.Node, text string, source ir.Span) {
	if text == "" {
		return
	}
	if len(*nodes) > 0 {
		if previous, ok := (*nodes)[len(*nodes)-1].(*ir.Text); ok && previous.Source == source {
			previous.Value += text
			return
		}
	}
	*nodes = append(*nodes, &ir.Text{Value: text, Source: source})
}

func findParameterEnd(text string, offset int) (int, bool) {
	state := parameterScanState{}
	for index := offset; index < len(text); index++ {
		if state.startsComment(text, index) {
			return -1, true
		}
		if state.consume(text, &index) {
			continue
		}
		switch current := text[index]; current {
		case '(':
			state.parentheses++
		case ')':
			if state.parentheses > 0 {
				state.parentheses--
			}
		case '[':
			state.brackets++
		case ']':
			if state.brackets > 0 {
				state.brackets--
			}
		case '{':
			state.braces++
		case '}':
			if state.braces > 0 {
				state.braces--
			} else if state.topLevel() {
				return index, false
			}
		}
	}
	return -1, false
}

func parseParameter(content string, source ir.Span) (*ir.Parameter, error) {
	if _, hasComment := findParameterEnd(content, 0); hasComment {
		return nil, parameterError(source, "comments are not allowed in #{} parameter expressions")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, parameterError(source, "empty #{} parameter")
	}

	pathEnd, separator := findTopLevelParameterSeparator(content)
	path := strings.TrimSpace(content[:pathEnd])
	if path == "" {
		return nil, parameterError(source, "parameter property is empty")
	}
	if _, err := parser.ParseExpr(path); err != nil {
		return nil, parameterError(source, "invalid parameter %q: %v", path, err)
	}

	options := make(map[string]string)
	remaining := strings.TrimSpace(content[pathEnd:])
	if separator == ':' {
		remaining = strings.TrimSpace(remaining[1:])
		end, _ := findTopLevelParameterSeparator(remaining)
		if end < 0 {
			if err := addParameterOption(options, "jdbcType", strings.TrimSpace(remaining), source); err != nil {
				return nil, err
			}
			remaining = ""
		} else {
			if err := addParameterOption(options, "jdbcType", strings.TrimSpace(remaining[:end]), source); err != nil {
				return nil, err
			}
			remaining = remaining[end:]
		}
	}
	if separator == ',' && strings.HasPrefix(remaining, ",") {
		remaining = remaining[1:]
		if strings.TrimSpace(remaining) == "" {
			return nil, parameterError(source, "parameter option is empty")
		}
	}
	for _, option := range splitTopLevelParameterOptions(remaining) {
		option = strings.TrimSpace(option)
		if option == "" {
			return nil, parameterError(source, "parameter option is empty")
		}
		name, value, ok := strings.Cut(option, "=")
		if !ok {
			return nil, parameterError(source, "invalid parameter option %q", option)
		}
		if err := addParameterOption(options, strings.TrimSpace(name), strings.TrimSpace(value), source); err != nil {
			return nil, err
		}
	}

	sensitive := strings.EqualFold(options["sensitive"], "true")
	return &ir.Parameter{
		Path:      path,
		Options:   options,
		Sensitive: sensitive,
		Source:    source,
	}, nil
}

type parameterScanState struct {
	quote       byte
	escaped     bool
	parentheses int
	brackets    int
	braces      int
}

func (state parameterScanState) startsComment(value string, index int) bool {
	return state.quote == 0 &&
		index+1 < len(value) &&
		value[index] == '/' &&
		(value[index+1] == '/' || value[index+1] == '*')
}

// consume advances lexical constructs whose contents must not terminate or
// split a Go expression. The caller still handles structural delimiters.
func (state *parameterScanState) consume(value string, index *int) bool {
	current := value[*index]
	if state.quote != 0 {
		if state.quote == '`' {
			if current == '`' {
				state.quote = 0
			}
			return true
		}
		if state.escaped {
			state.escaped = false
			return true
		}
		if current == '\\' {
			state.escaped = true
			return true
		}
		if current == state.quote {
			state.quote = 0
		}
		return true
	}
	switch current {
	case '\'', '"', '`':
		state.quote = current
		return true
	default:
		return false
	}
}

func (state parameterScanState) topLevel() bool {
	return state.parentheses == 0 && state.brackets == 0 && state.braces == 0
}

func findTopLevelParameterSeparator(value string) (int, byte) {
	state := parameterScanState{}
	for index := 0; index < len(value); index++ {
		if state.consume(value, &index) {
			continue
		}
		switch current := value[index]; current {
		case '(':
			state.parentheses++
		case ')':
			if state.parentheses > 0 {
				state.parentheses--
			}
		case '[':
			state.brackets++
		case ']':
			if state.brackets > 0 {
				state.brackets--
			}
		case '{':
			state.braces++
		case '}':
			if state.braces > 0 {
				state.braces--
			}
		case ',', ':':
			if state.topLevel() {
				return index, current
			}
		}
	}
	return len(value), 0
}

func splitTopLevelParameterOptions(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	result := make([]string, 0, 2)
	offset := 0
	for offset < len(value) {
		end, separator := findTopLevelParameterSeparator(value[offset:])
		result = append(result, value[offset:offset+end])
		if separator != ',' {
			if separator == ':' {
				result[len(result)-1] += value[offset+end:]
			}
			break
		}
		offset += end + 1
		if offset == len(value) {
			result = append(result, "")
			break
		}
	}
	return result
}

func addParameterOption(options map[string]string, name, value string, source ir.Span) error {
	if name == "" || value == "" {
		return parameterError(source, "parameter options require non-empty names and values")
	}
	if _, duplicate := options[name]; duplicate {
		return parameterError(source, "duplicate parameter option %q", name)
	}
	if name != "sensitive" {
		return parameterError(
			source,
			`parameter option %q is not supported in the Go-first XML dialect; only sensitive=true|false is supported`,
			name,
		)
	}
	if !strings.EqualFold(value, "true") && !strings.EqualFold(value, "false") {
		return parameterError(source, `parameter option "sensitive" must be true or false`)
	}
	options[name] = value
	return nil
}

func parameterError(source ir.Span, format string, arguments ...any) error {
	return fmt.Errorf(
		"%s:%d:%d: %s",
		source.File,
		source.Line,
		source.Column,
		fmt.Sprintf(format, arguments...),
	)
}
