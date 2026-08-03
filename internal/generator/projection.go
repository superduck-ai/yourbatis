package generator

import (
	"fmt"
	"strings"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

type dynamicSQLRange struct {
	start   int
	end     int
	element string
	source  ir.Span
}

type projectionFlattener struct {
	sql     strings.Builder
	dynamic []dynamicSQLRange
}

func selectColumns(statement *ir.Statement) ([]string, error) {
	columns, _, err := projectionColumns(statement, "SELECT", selectProjectionRange, true)
	return columns, err
}

func returningColumns(statement *ir.Statement) ([]string, error) {
	columns, _, err := projectionColumns(statement, "RETURNING", returningProjectionRange, false)
	return columns, err
}

func resultColumns(statement *ir.Statement) ([]string, string, error) {
	if statement.ResultMode == ir.ResultReturning {
		columns, err := returningColumns(statement)
		return columns, "RETURNING", err
	}
	columns, err := selectColumns(statement)
	return columns, "SELECT", err
}

func projectionColumns(
	statement *ir.Statement,
	projectionName string,
	projectionRange func(string) (int, int, error),
	stripModifier bool,
) ([]string, string, error) {
	var flattener projectionFlattener
	flattener.appendNodes(statement.Body)
	sqlText := flattener.sql.String()

	projectionStart, projectionEnd, err := projectionRange(sqlText)
	if err != nil {
		return nil, projectionName, sourceError(statement.Source, "%v", err)
	}
	for _, dynamic := range flattener.dynamic {
		if dynamic.start < projectionEnd && dynamic.end > projectionStart {
			return nil, projectionName, sourceError(
				dynamic.source,
				"dynamic <%s> cannot change the %s projection; generated scanners require a fixed column list",
				dynamic.element,
				projectionName,
			)
		}
	}

	projection := sqlText[projectionStart:projectionEnd]
	items, err := splitProjection(projection, projectionName)
	if err != nil {
		return nil, projectionName, sourceError(statement.Source, "%v", err)
	}
	if stripModifier {
		if err := stripProjectionModifier(items); err != nil {
			return nil, projectionName, sourceError(statement.Source, "%v", err)
		}
	}

	columns := make([]string, 0, len(items))
	for _, item := range items {
		column, err := projectionColumn(item, projectionName)
		if err != nil {
			return nil, projectionName, sourceError(statement.Source, "%v", err)
		}
		columns = append(columns, column)
	}
	return columns, projectionName, nil
}

func (flattener *projectionFlattener) appendNodes(nodes []ir.Node) {
	for _, node := range nodes {
		switch current := node.(type) {
		case *ir.SQLNodeBoundary:
			flattener.sql.WriteByte(' ')
		case *ir.Text:
			flattener.sql.WriteString(current.Value)
		case *ir.Parameter:
			flattener.sql.WriteByte('?')
		case *ir.If:
			flattener.appendDynamic("if", current.Source, func() {
				flattener.appendNodes(current.Body)
			})
		case *ir.Choose:
			flattener.appendDynamic("choose", current.Source, func() {
				for _, when := range current.When {
					flattener.appendNodes(when.Body)
				}
				flattener.appendNodes(current.Otherwise)
			})
		case *ir.Foreach:
			flattener.appendDynamic("foreach", current.Source, func() {
				flattener.sql.WriteString(current.Open)
				flattener.appendNodes(current.Body)
				flattener.sql.WriteString(current.Close)
			})
		case *ir.Trim:
			flattener.sql.WriteString(current.Prefix)
			flattener.appendNodes(current.Body)
			flattener.sql.WriteString(current.Suffix)
		case *ir.Bind:
			// Bind declares a Go value but emits no SQL.
		}
	}
}

func (flattener *projectionFlattener) appendDynamic(
	element string,
	source ir.Span,
	appendBody func(),
) {
	start := flattener.sql.Len()
	appendBody()
	end := flattener.sql.Len()
	sqlText := flattener.sql.String()
	for start < end && isSQLSpace(sqlText[start]) {
		start++
	}
	for end > start && isSQLSpace(sqlText[end-1]) {
		end--
	}
	if start == end {
		return
	}
	flattener.dynamic = append(flattener.dynamic, dynamicSQLRange{
		start:   start,
		end:     end,
		element: element,
		source:  source,
	})
}

func selectProjectionRange(sqlText string) (int, int, error) {
	_, selectEnd, ok := findTopLevelKeyword(sqlText, 0, "select")
	if !ok {
		return 0, 0, fmt.Errorf("top-level SELECT keyword not found")
	}

	projectionEnd := len(sqlText)
	for _, keyword := range []string{
		"from",
		"into",
		"where",
		"group",
		"having",
		"window",
		"order",
		"limit",
		"offset",
		"fetch",
		"for",
		"union",
		"intersect",
		"except",
	} {
		start, _, found := findTopLevelKeyword(sqlText, selectEnd, keyword)
		if found && start < projectionEnd {
			projectionEnd = start
		}
	}
	for projectionEnd > selectEnd {
		current := sqlText[projectionEnd-1]
		if isSQLSpace(current) || current == ';' {
			projectionEnd--
			continue
		}
		break
	}
	if projectionEnd <= selectEnd {
		return 0, 0, fmt.Errorf("empty SELECT projection")
	}
	return selectEnd, projectionEnd, nil
}

func returningProjectionRange(sqlText string) (int, int, error) {
	_, returningEnd, ok := findTopLevelKeyword(sqlText, 0, "returning")
	if !ok {
		return 0, 0, fmt.Errorf("top-level RETURNING keyword not found")
	}

	projectionEnd := len(sqlText)
	for projectionEnd > returningEnd {
		current := sqlText[projectionEnd-1]
		if isSQLSpace(current) || current == ';' {
			projectionEnd--
			continue
		}
		break
	}
	if projectionEnd <= returningEnd {
		return 0, 0, fmt.Errorf("empty RETURNING projection")
	}
	return returningEnd, projectionEnd, nil
}

func findTopLevelKeyword(sqlText string, start int, keyword string) (int, int, bool) {
	depth := 0
	for index := start; index < len(sqlText); {
		switch sqlText[index] {
		case '\'':
			index = skipQuoted(sqlText, index, '\'')
			continue
		case '"':
			index = skipQuoted(sqlText, index, '"')
			continue
		case '`':
			index = skipQuoted(sqlText, index, '`')
			continue
		case '[':
			index = skipBracketIdentifier(sqlText, index)
			continue
		case '-':
			if index+1 < len(sqlText) && sqlText[index+1] == '-' {
				index = skipLineComment(sqlText, index+2)
				continue
			}
		case '/':
			if index+1 < len(sqlText) && sqlText[index+1] == '*' {
				index = skipBlockComment(sqlText, index+2)
				continue
			}
		case '(':
			depth++
			index++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			index++
			continue
		}
		if depth == 0 &&
			len(sqlText)-index >= len(keyword) &&
			strings.EqualFold(sqlText[index:index+len(keyword)], keyword) &&
			hasSQLWordBoundaryBefore(sqlText, index) &&
			hasSQLWordBoundaryAfter(sqlText, index+len(keyword)) {
			return index, index + len(keyword), true
		}
		index++
	}
	return 0, 0, false
}

func skipQuoted(sqlText string, start int, quote byte) int {
	for index := start + 1; index < len(sqlText); index++ {
		if sqlText[index] != quote {
			continue
		}
		if index+1 < len(sqlText) && sqlText[index+1] == quote {
			index++
			continue
		}
		return index + 1
	}
	return len(sqlText)
}

func skipBracketIdentifier(sqlText string, start int) int {
	for index := start + 1; index < len(sqlText); index++ {
		if sqlText[index] == ']' {
			return index + 1
		}
	}
	return len(sqlText)
}

func skipLineComment(sqlText string, start int) int {
	for index := start; index < len(sqlText); index++ {
		if sqlText[index] == '\n' || sqlText[index] == '\r' {
			return index + 1
		}
	}
	return len(sqlText)
}

func skipBlockComment(sqlText string, start int) int {
	for index := start; index+1 < len(sqlText); index++ {
		if sqlText[index] == '*' && sqlText[index+1] == '/' {
			return index + 2
		}
	}
	return len(sqlText)
}

func hasSQLWordBoundaryBefore(value string, index int) bool {
	return index == 0 || !isSQLWordByte(value[index-1])
}

func hasSQLWordBoundaryAfter(value string, index int) bool {
	return index == len(value) || !isSQLWordByte(value[index])
}

func isSQLWordByte(value byte) bool {
	return value == '_' || value == '$' ||
		value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9'
}

func isSQLSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func splitProjection(projection, projectionName string) ([]string, error) {
	result := make([]string, 0)
	start := 0
	depth := 0
	for index := 0; index < len(projection); {
		switch projection[index] {
		case '\'', '"', '`':
			index = skipQuoted(projection, index, projection[index])
			continue
		case '[':
			index = skipBracketIdentifier(projection, index)
			continue
		case '-':
			if index+1 < len(projection) && projection[index+1] == '-' {
				index = skipLineComment(projection, index+2)
				continue
			}
		case '/':
			if index+1 < len(projection) && projection[index+1] == '*' {
				index = skipBlockComment(projection, index+2)
				continue
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced projection parentheses")
			}
		case ',':
			if depth == 0 {
				item := strings.TrimSpace(projection[start:index])
				if item == "" {
					return nil, fmt.Errorf("empty %s projection item", projectionName)
				}
				result = append(result, item)
				start = index + 1
			}
		}
		index++
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced %s projection", projectionName)
	}
	last := strings.TrimSpace(projection[start:])
	if last == "" {
		return nil, fmt.Errorf("empty %s projection item", projectionName)
	}
	result = append(result, last)
	return result, nil
}

func stripProjectionModifier(items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("empty SELECT projection")
	}
	first := strings.TrimSpace(items[0])
	for _, modifier := range []string{"all", "distinct"} {
		if !hasLeadingSQLKeyword(first, modifier) {
			continue
		}
		first = strings.TrimSpace(first[len(modifier):])
		if modifier == "distinct" && hasLeadingSQLKeyword(first, "on") {
			return fmt.Errorf("SELECT DISTINCT ON is not supported by static projection analysis")
		}
		if first == "" {
			return fmt.Errorf("empty SELECT projection after %s", strings.ToUpper(modifier))
		}
		items[0] = first
		return nil
	}
	return nil
}

func hasLeadingSQLKeyword(value, keyword string) bool {
	return len(value) >= len(keyword) &&
		strings.EqualFold(value[:len(keyword)], keyword) &&
		hasSQLWordBoundaryAfter(value, len(keyword))
}

func projectionColumn(expression, projectionName string) (string, error) {
	expression = stripSQLComments(expression)
	expression = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(expression), ";"))
	if expression == "" {
		return "", fmt.Errorf("empty %s projection item", projectionName)
	}
	if expression == "*" || strings.HasSuffix(expression, ".*") {
		return "", fmt.Errorf(
			"%s * cannot be statically mapped; list columns explicitly",
			projectionName,
		)
	}

	if _, end, found := findLastTopLevelKeyword(expression, "as"); found {
		alias := strings.TrimSpace(expression[end:])
		if column := cleanIdentifier(alias); column != "" && isSingleSQLIdentifier(alias) {
			return column, nil
		}
		return "", fmt.Errorf("projection %q has an invalid AS alias", expression)
	}

	if !isSimpleColumnReference(expression) {
		return "", fmt.Errorf("projection %q requires an explicit AS alias", expression)
	}
	if dot := strings.LastIndex(expression, "."); dot >= 0 {
		expression = expression[dot+1:]
	}
	column := cleanIdentifier(expression)
	if column == "" {
		return "", fmt.Errorf("projection %q has no output column name", expression)
	}
	return column, nil
}

func stripSQLComments(value string) string {
	var result strings.Builder
	for index := 0; index < len(value); {
		switch value[index] {
		case '\'', '"', '`':
			end := skipQuoted(value, index, value[index])
			result.WriteString(value[index:end])
			index = end
			continue
		case '[':
			end := skipBracketIdentifier(value, index)
			result.WriteString(value[index:end])
			index = end
			continue
		case '-':
			if index+1 < len(value) && value[index+1] == '-' {
				index = skipLineComment(value, index+2)
				result.WriteByte(' ')
				continue
			}
		case '/':
			if index+1 < len(value) && value[index+1] == '*' {
				index = skipBlockComment(value, index+2)
				result.WriteByte(' ')
				continue
			}
		}
		result.WriteByte(value[index])
		index++
	}
	return result.String()
}

func findLastTopLevelKeyword(value, keyword string) (int, int, bool) {
	search := 0
	var foundStart, foundEnd int
	found := false
	for search < len(value) {
		start, end, ok := findTopLevelKeyword(value, search, keyword)
		if !ok {
			break
		}
		foundStart, foundEnd, found = start, end, true
		search = end
	}
	return foundStart, foundEnd, found
}

func isSimpleColumnReference(expression string) bool {
	parts := strings.Split(expression, ".")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !isSingleSQLIdentifier(strings.TrimSpace(part)) {
			return false
		}
	}
	return true
}

func isSingleSQLIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		switch {
		case value[0] == '"' && value[len(value)-1] == '"':
			return !strings.Contains(value[1:len(value)-1], `"`)
		case value[0] == '`' && value[len(value)-1] == '`':
			return !strings.Contains(value[1:len(value)-1], "`")
		case value[0] == '[' && value[len(value)-1] == ']':
			return !strings.Contains(value[1:len(value)-1], "]")
		}
	}
	if value == "" || !isSQLIdentifierStart(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !isSQLWordByte(value[index]) {
			return false
		}
	}
	return true
}

func isSQLIdentifierStart(value byte) bool {
	return value == '_' ||
		value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z'
}

func cleanIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		switch {
		case value[0] == '"' && value[len(value)-1] == '"',
			value[0] == '`' && value[len(value)-1] == '`',
			value[0] == '[' && value[len(value)-1] == ']':
			return value[1 : len(value)-1]
		}
	}
	return value
}
