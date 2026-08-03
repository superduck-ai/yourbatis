package xmlmapper

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/superduck-ai/yourbatis/internal/ir"
	"github.com/superduck-ai/yourbatis/internal/parsing"
)

type Parser struct {
	resource  string
	namespace string
	variables parsing.Properties
	fragments strictMap[*parsing.XNode]
	stack     map[string]bool
}

func ParseFile(path string) (*ir.Mapper, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return Parse(file, path)
}

func Parse(reader io.Reader, resource string) (*ir.Mapper, error) {
	xpath, err := parsing.NewXPathParserFromReader(reader, parsing.XPathParserConfig{
		EntityResolver: &parsing.XMLMapperEntityResolver{},
		Resource:       resource,
	})
	if err != nil {
		return nil, err
	}
	root := xpath.EvalNode("/mapper")
	if root == nil {
		return nil, fmt.Errorf("%s: mapper root element not found", resource)
	}
	if err := validateAttributes(root, "namespace"); err != nil {
		return nil, err
	}
	if err := validateElementOnlyContent(root); err != nil {
		return nil, err
	}
	namespace, ok := root.LookupStringAttribute("namespace")
	namespace = strings.TrimSpace(namespace)
	if !ok || namespace == "" {
		return nil, diagnostic(root, "mapper requires a non-empty namespace")
	}

	parser := &Parser{
		resource:  resource,
		namespace: namespace,
		variables: parsing.NewProperties(),
		fragments: newStrictMap[*parsing.XNode](),
		stack:     make(map[string]bool),
	}
	mapper := &ir.Mapper{
		Namespace:  namespace,
		ResultMaps: make(map[string]*ir.ResultMap),
	}

	for _, child := range root.GetChildren() {
		switch child.GetName() {
		case "sql":
			if err := validateAttributes(child, "id"); err != nil {
				return nil, err
			}
			id := strings.TrimSpace(child.GetStringAttribute("id"))
			if id == "" {
				return nil, diagnostic(child, "sql fragment requires id")
			}
			canonicalID, err := parser.applyCurrentNamespace(id, false, "SQL fragment")
			if err != nil {
				return nil, diagnostic(child, "%v", err)
			}
			if !parser.fragments.add(canonicalID, child) {
				return nil, diagnostic(child, "duplicate sql fragment %q", canonicalID)
			}
		case "resultMap":
			resultMap, err := parser.parseResultMap(child)
			if err != nil {
				return nil, err
			}
			if _, exists := mapper.ResultMaps[resultMap.ID]; exists {
				return nil, diagnostic(child, "duplicate resultMap %q", resultMap.ID)
			}
			mapper.ResultMaps[resultMap.ID] = resultMap
		case "select", "insert", "update", "delete":
			// Statements are parsed after all local fragments and result maps have
			// been collected.
		default:
			return nil, diagnostic(child, "unsupported mapper element <%s>", child.GetName())
		}
	}

	statementIDs := make(map[string]struct{})
	for _, child := range root.GetChildren() {
		command := ir.Command(child.GetName())
		switch command {
		case ir.Select, ir.Insert, ir.Update, ir.Delete:
			statement, err := parser.parseStatement(child, command)
			if err != nil {
				return nil, err
			}
			if _, duplicate := statementIDs[statement.ID]; duplicate {
				return nil, diagnostic(child, "duplicate statement %q", statement.ID)
			}
			statementIDs[statement.ID] = struct{}{}
			mapper.Statements = append(mapper.Statements, statement)
		}
	}
	return mapper, nil
}

func (parser *Parser) parseResultMap(node *parsing.XNode) (*ir.ResultMap, error) {
	if err := validateAttributes(node, "id", "type"); err != nil {
		return nil, err
	}
	if err := validateElementOnlyContent(node); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(node.GetStringAttribute("id"))
	typeName := strings.TrimSpace(node.GetStringAttribute("type"))
	if id == "" || typeName == "" {
		return nil, diagnostic(node, "resultMap requires id and type")
	}
	canonicalID, err := parser.applyCurrentNamespace(id, false, "resultMap")
	if err != nil {
		return nil, diagnostic(node, "%v", err)
	}
	result := &ir.ResultMap{
		ID:     canonicalID,
		Type:   typeName,
		Source: span(node),
	}
	for _, child := range node.GetChildren() {
		switch child.GetName() {
		case "id", "result":
			if err := validateAttributes(child, "column", "property"); err != nil {
				return nil, err
			}
			if err := validateEmptyElement(child); err != nil {
				return nil, err
			}
			column := strings.TrimSpace(child.GetStringAttribute("column"))
			property := strings.TrimSpace(child.GetStringAttribute("property"))
			if column == "" || property == "" {
				return nil, diagnostic(child, "%s requires column and property", child.GetName())
			}
			result.Fields = append(result.Fields, ir.ResultField{
				Column:   column,
				Property: property,
				ID:       child.GetName() == "id",
				Source:   span(child),
			})
		case "constructor", "association", "collection", "discriminator":
			return nil, diagnostic(child, "%s is not supported by the Go-first result mapper", child.GetName())
		default:
			return nil, diagnostic(child, "unsupported resultMap element <%s>", child.GetName())
		}
	}
	return result, nil
}

func (parser *Parser) parseStatement(node *parsing.XNode, command ir.Command) (*ir.Statement, error) {
	if err := validateAttributes(node, "id", "result", "resultMap", "resultType"); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(node.GetStringAttribute("id"))
	if id == "" {
		return nil, diagnostic(node, "%s requires id", command)
	}
	mode := ir.ResultMode(strings.ToLower(node.GetStringAttribute("result")))
	switch mode {
	case ir.ResultInfer,
		ir.ResultOne,
		ir.ResultOptional,
		ir.ResultMany,
		ir.ResultScalar,
		ir.ResultRows,
		ir.ResultExec,
		ir.ResultLastID,
		ir.ResultReturning:
	default:
		return nil, diagnostic(node, "unsupported result mode %q", mode)
	}
	resultMap := strings.TrimSpace(node.GetStringAttribute("resultMap"))
	resultType := strings.TrimSpace(node.GetStringAttribute("resultType"))
	if resultMap != "" && resultType != "" {
		return nil, diagnostic(node, "resultMap and resultType are mutually exclusive")
	}
	switch command {
	case ir.Select:
		switch mode {
		case ir.ResultInfer, ir.ResultOne, ir.ResultOptional, ir.ResultMany, ir.ResultScalar:
		default:
			return nil, diagnostic(node, "select does not support result mode %q", mode)
		}
		if mode == ir.ResultScalar && resultMap != "" {
			return nil, diagnostic(node, "result=scalar cannot be combined with resultMap")
		}
	default:
		if mode != ir.ResultReturning && (resultMap != "" || resultType != "") {
			return nil, diagnostic(node, "%s does not support resultMap or resultType", command)
		}
		switch mode {
		case ir.ResultInfer, ir.ResultRows, ir.ResultExec, ir.ResultLastID, ir.ResultReturning:
		default:
			return nil, diagnostic(node, "%s does not support result mode %q", command, mode)
		}
		if mode == ir.ResultLastID && command != ir.Insert {
			return nil, diagnostic(node, "result=lastid is only supported by insert")
		}
	}
	if resultMap != "" {
		canonicalResultMap, err := parser.applyCurrentNamespace(resultMap, true, "resultMap")
		if err != nil {
			return nil, diagnostic(node, "%v", err)
		}
		resultMap = canonicalResultMap
	}
	body, err := parser.parseChildren(node, parser.variables)
	if err != nil {
		return nil, err
	}
	return &ir.Statement{
		ID:         id,
		Command:    command,
		ResultMode: mode,
		ResultMap:  resultMap,
		ResultType: resultType,
		Body:       body,
		Source:     span(node),
	}, nil
}

func (parser *Parser) parseChildren(node *parsing.XNode, variables parsing.Properties) ([]ir.Node, error) {
	result := make([]ir.Node, 0)
	for _, raw := range node.GetNode().GetChildNodes() {
		child := node.NewXNode(raw)
		switch raw.GetNodeType() {
		case parsing.TEXT_NODE, parsing.CDATA_SECTION_NODE:
			text := substituteSQLText(raw.GetNodeValue(), variables)
			nodes, err := parseSQLText(text, span(child))
			if err != nil {
				return nil, err
			}
			// MyBatis creates one StaticTextSqlNode/TextSqlNode (or cached
			// EmptySqlNode) for each XML text/CDATA child. Preserve that
			// appendSql call instead of manufacturing trailing text.
			result = append(result, &ir.SQLNodeBoundary{Source: span(child)})
			result = append(result, nodes...)
		case parsing.ELEMENT_NODE:
			nodes, err := parser.parseElement(child, variables)
			if err != nil {
				return nil, err
			}
			result = append(result, nodes...)
		}
	}
	return result, nil
}

func (parser *Parser) parseElement(node *parsing.XNode, variables parsing.Properties) ([]ir.Node, error) {
	switch node.GetName() {
	case "if":
		if err := validateAttributes(node, "test"); err != nil {
			return nil, err
		}
		test, ok := node.LookupStringAttribute("test")
		if !ok || strings.TrimSpace(test) == "" {
			return nil, diagnostic(node, "if requires a non-empty test")
		}
		body, err := parser.parseChildren(node, variables)
		if err != nil {
			return nil, err
		}
		return []ir.Node{&ir.If{
			Test:   substitute(test, variables),
			Body:   body,
			Source: span(node),
		}}, nil
	case "choose":
		if err := validateAttributes(node); err != nil {
			return nil, err
		}
		return parser.parseChoose(node, variables)
	case "foreach":
		return parser.parseForeach(node, variables)
	case "where":
		if err := validateAttributes(node); err != nil {
			return nil, err
		}
		body, err := parser.parseChildren(node, variables)
		if err != nil {
			return nil, err
		}
		return []ir.Node{&ir.Trim{
			Prefix: "WHERE ",
			PrefixOverrides: []string{
				"AND ", "OR ",
				"AND\n", "OR\n",
				"AND\r", "OR\r",
				"AND\t", "OR\t",
			},
			Body:   body,
			Source: span(node),
		}}, nil
	case "set":
		if err := validateAttributes(node); err != nil {
			return nil, err
		}
		body, err := parser.parseChildren(node, variables)
		if err != nil {
			return nil, err
		}
		return []ir.Node{&ir.Trim{
			Prefix:          "SET ",
			PrefixOverrides: []string{","},
			SuffixOverrides: []string{","},
			Body:            body,
			Source:          span(node),
		}}, nil
	case "trim":
		if err := validateAttributes(node, "prefix", "prefixOverrides", "suffix", "suffixOverrides"); err != nil {
			return nil, err
		}
		body, err := parser.parseChildren(node, variables)
		if err != nil {
			return nil, err
		}
		return []ir.Node{&ir.Trim{
			Prefix:          trimPrefixAffix(substitute(node.GetStringAttribute("prefix"), variables)),
			PrefixOverrides: splitOverrides(substitute(node.GetStringAttribute("prefixOverrides"), variables)),
			Suffix:          trimSuffixAffix(substitute(node.GetStringAttribute("suffix"), variables)),
			SuffixOverrides: splitOverrides(substitute(node.GetStringAttribute("suffixOverrides"), variables)),
			Body:            body,
			Source:          span(node),
		}}, nil
	case "bind":
		if err := validateAttributes(node, "name", "value"); err != nil {
			return nil, err
		}
		if err := validateEmptyElement(node); err != nil {
			return nil, err
		}
		name := node.GetStringAttribute("name")
		value := node.GetStringAttribute("value")
		if name == "" || value == "" {
			return nil, diagnostic(node, "bind requires name and value")
		}
		return []ir.Node{&ir.Bind{
			Name:       name,
			Expression: substitute(value, variables),
			Source:     span(node),
		}}, nil
	case "include":
		if err := validateAttributes(node, "refid"); err != nil {
			return nil, err
		}
		return parser.parseInclude(node, variables)
	default:
		return nil, diagnostic(node, "unsupported SQL element <%s>", node.GetName())
	}
}

func (parser *Parser) parseChoose(node *parsing.XNode, variables parsing.Properties) ([]ir.Node, error) {
	choose := &ir.Choose{Source: span(node)}
	seenOtherwise := false
	for _, child := range node.GetChildren() {
		switch child.GetName() {
		case "when":
			if seenOtherwise {
				return nil, diagnostic(child, "choose requires when elements to appear before otherwise")
			}
			if err := validateAttributes(child, "test"); err != nil {
				return nil, err
			}
			test, ok := child.LookupStringAttribute("test")
			if !ok || strings.TrimSpace(test) == "" {
				return nil, diagnostic(child, "when requires a non-empty test")
			}
			body, err := parser.parseChildren(child, variables)
			if err != nil {
				return nil, err
			}
			choose.When = append(choose.When, ir.When{
				Test:   substitute(test, variables),
				Body:   body,
				Source: span(child),
			})
		case "otherwise":
			if err := validateAttributes(child); err != nil {
				return nil, err
			}
			if seenOtherwise {
				return nil, diagnostic(child, "choose allows only one otherwise")
			}
			seenOtherwise = true
			body, err := parser.parseChildren(child, variables)
			if err != nil {
				return nil, err
			}
			choose.Otherwise = body
		default:
			return nil, diagnostic(child, "choose only accepts when and otherwise")
		}
	}
	return []ir.Node{choose}, nil
}

func (parser *Parser) parseForeach(node *parsing.XNode, variables parsing.Properties) ([]ir.Node, error) {
	if err := validateAttributes(node, "collection", "item", "index", "open", "separator", "close"); err != nil {
		return nil, err
	}
	collection := substitute(node.GetStringAttribute("collection"), variables)
	if collection == "" {
		return nil, diagnostic(node, "foreach requires collection")
	}
	item := node.GetStringAttribute("item", "item")
	index := node.GetStringAttribute("index", "index")
	open, openSet := node.LookupStringAttribute("open")
	close, closeSet := node.LookupStringAttribute("close")
	body, err := parser.parseChildren(node, variables)
	if err != nil {
		return nil, err
	}
	return []ir.Node{&ir.Foreach{
		Collection: collection,
		Item:       item,
		Index:      index,
		Open:       substitute(open, variables),
		OpenSet:    openSet,
		Separator:  substitute(node.GetStringAttribute("separator"), variables),
		Close:      substitute(close, variables),
		CloseSet:   closeSet,
		Body:       body,
		Source:     span(node),
	}}, nil
}

func (parser *Parser) parseInclude(node *parsing.XNode, variables parsing.Properties) ([]ir.Node, error) {
	if err := validateElementOnlyContent(node); err != nil {
		return nil, err
	}
	refID := substitute(node.GetStringAttribute("refid"), variables)
	if refID == "" {
		return nil, diagnostic(node, "include requires refid")
	}
	canonicalRefID, err := parser.applyCurrentNamespace(refID, true, "SQL fragment")
	if err != nil {
		return nil, diagnostic(node, "%v", err)
	}
	fragment, exists := parser.fragments.get(canonicalRefID)
	if !exists {
		return nil, diagnostic(node, "SQL fragment %q not found", canonicalRefID)
	}
	if parser.stack[canonicalRefID] {
		return nil, diagnostic(node, "circular SQL include %q", canonicalRefID)
	}

	nextVariables := variables.Clone()
	includeProperties := make(map[string]struct{})
	for _, property := range node.GetChildren() {
		if property.GetName() != "property" {
			return nil, diagnostic(property, "include only accepts property children")
		}
		if err := validateAttributes(property, "name", "value"); err != nil {
			return nil, err
		}
		if err := validateEmptyElement(property); err != nil {
			return nil, err
		}
		name := property.GetStringAttribute("name")
		rawValue, hasValue := property.LookupStringAttribute("value")
		if name == "" {
			return nil, diagnostic(property, "include property requires name")
		}
		if !hasValue {
			return nil, diagnostic(property, "include property requires value")
		}
		if _, duplicate := includeProperties[name]; duplicate {
			return nil, diagnostic(property, "duplicate include property %q", name)
		}
		value := substitute(rawValue, variables)
		if placeholder, unresolved := findUnresolvedIncludePlaceholder(value); unresolved {
			return nil, diagnostic(
				property,
				"include property %q contains unresolved placeholder %q; property values can only reference variables from the enclosing scope (sibling include properties are not visible)",
				name,
				placeholder,
			)
		}
		includeProperties[name] = struct{}{}
		nextVariables[name] = value
	}

	parser.stack[canonicalRefID] = true
	defer delete(parser.stack, canonicalRefID)
	return parser.parseChildren(fragment, nextVariables)
}

// applyCurrentNamespace follows MyBatis' definition/reference rules. Local
// definitions and bare references are normalized to one canonical key. A
// reference that already contains a namespace remains unchanged.
func (parser *Parser) applyCurrentNamespace(base string, isReference bool, element string) (string, error) {
	if isReference {
		if strings.Contains(base, ".") {
			return base, nil
		}
	} else {
		if strings.HasPrefix(base, parser.namespace+".") {
			return base, nil
		}
		if strings.Contains(base, ".") {
			return "", fmt.Errorf("dots are not allowed in %s ids; remove them from %q", element, base)
		}
	}
	return parser.namespace + "." + base, nil
}

func substitute(value string, variables parsing.Properties) string {
	return parsing.PropertyParser.Parse(value, variables)
}

func substituteSQLText(value string, variables parsing.Properties) string {
	// PropertyParser and parseSQLText both consume the backslash that escapes a
	// ${ opener. Add one for the property pass so parseSQLText still receives
	// the original escape and emits the placeholder as literal SQL text.
	value = strings.ReplaceAll(value, `\${`, `\\${`)
	return substitute(value, variables)
}

func findUnresolvedIncludePlaceholder(value string) (string, bool) {
	for offset := 0; offset < len(value); {
		relativeStart := strings.Index(value[offset:], "${")
		if relativeStart < 0 {
			return "", false
		}
		start := offset + relativeStart
		if start > 0 && value[start-1] == '\\' {
			offset = start + 2
			continue
		}
		end, _ := findParameterEnd(value, start+2)
		if end < 0 {
			return value[start:], true
		}
		return value[start : end+1], true
	}
	return "", false
}

func splitOverrides(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "|")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		// MyBatis treats whitespace inside each override as part of the
		// matching rule. In particular, a trailing space distinguishes
		// "AND " from "AND(". Only empty tokens produced by adjacent
		// delimiters are ignored.
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func trimPrefixAffix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return value + " "
}

func trimSuffixAffix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return " " + value
}

func validateAttributes(node *parsing.XNode, allowed ...string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}

	attributes := node.GetAttributes()
	unsupported := make([]string, 0)
	for name := range attributes {
		if _, ok := allowedSet[name]; !ok {
			unsupported = append(unsupported, name)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	sort.Strings(unsupported)
	return diagnostic(
		node,
		`<%s> does not support attribute %q in the Go-first XML dialect`,
		node.GetName(),
		unsupported[0],
	)
}

func validateElementOnlyContent(node *parsing.XNode) error {
	for _, raw := range node.GetNode().GetChildNodes() {
		switch raw.GetNodeType() {
		case parsing.TEXT_NODE, parsing.CDATA_SECTION_NODE:
			child := node.NewXNode(raw)
			if strings.TrimSpace(child.GetStringBody("")) != "" {
				return diagnostic(node, "<%s> only accepts child elements and whitespace", node.GetName())
			}
		}
	}
	return nil
}

func validateEmptyElement(node *parsing.XNode) error {
	for _, raw := range node.GetNode().GetChildNodes() {
		switch raw.GetNodeType() {
		case parsing.TEXT_NODE, parsing.CDATA_SECTION_NODE:
			child := node.NewXNode(raw)
			if strings.TrimSpace(child.GetStringBody("")) == "" {
				continue
			}
		}
		return diagnostic(node, "<%s> must be empty", node.GetName())
	}
	return nil
}

func span(node *parsing.XNode) ir.Span {
	position := node.GetPosition()
	return ir.Span{
		File:   position.Source,
		Line:   position.Line,
		Column: position.Column,
	}
}

func diagnostic(node *parsing.XNode, format string, arguments ...any) error {
	position := node.GetPosition()
	return fmt.Errorf(
		"%s:%d:%d: %s",
		position.Source,
		position.Line,
		position.Column,
		fmt.Sprintf(format, arguments...),
	)
}
