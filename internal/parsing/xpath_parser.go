package parsing

import (
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type EntityResolver interface {
	ResolveEntity(publicID, systemID string) ([]byte, error)
}

type XPathParserValidator func(*XPathParser) error

type XPathParserConfig struct {
	Validation     bool
	EntityResolver EntityResolver
	Validator      XPathParserValidator
	Variables      Properties
	Resource       string
}

type XPathParser struct {
	document       *Document
	validation     bool
	entityResolver EntityResolver
	variables      Properties
	resource       string
}

type xpathResult struct {
	node           *Node
	stringValue    string
	hasStringValue bool
}

type xpathAxis int

const (
	xpathAxisChild xpathAxis = iota
	xpathAxisDescendant
)

type xpathStepKind int

const (
	xpathStepName xpathStepKind = iota
	xpathStepWildcard
	xpathStepAttributeName
	xpathStepAttributeWildcard
	xpathStepSelf
	xpathStepParent
	xpathStepText
	xpathStepNode
)

type xpathStep struct {
	axis       xpathAxis
	kind       xpathStepKind
	value      string
	predicates []xpathPredicate
}

type xpathPredicateKind int

const (
	xpathPredicateNone xpathPredicateKind = iota
	xpathPredicatePositionIndex
	xpathPredicatePositionLast
	xpathPredicatePositionComparison
	xpathPredicateAttributeExists
	xpathPredicateAttributeEqualsString
	xpathPredicateStringValueEqualsString
	xpathPredicateTextEqualsString
	xpathPredicateBooleanExpression
)

type xpathPredicate struct {
	kind               xpathPredicateKind
	position           int
	attributeName      string
	stringValue        string
	expression         string
	comparisonOperator xpathComparisonOperator
	compareToLast      bool
}

type xpathValueKind int

const (
	xpathValueNodeSet xpathValueKind = iota
	xpathValueString
	xpathValueNumber
	xpathValueBoolean
)

type xpathValue struct {
	kind         xpathValueKind
	results      []xpathResult
	stringValue  string
	numberValue  float64
	booleanValue bool
}

type xpathComparisonOperator int

const (
	xpathComparisonEqual xpathComparisonOperator = iota
	xpathComparisonNotEqual
	xpathComparisonLessThan
	xpathComparisonLessThanOrEqual
	xpathComparisonGreaterThan
	xpathComparisonGreaterThanOrEqual
)

type xpathLogicalOperator int

const (
	xpathLogicalOr xpathLogicalOperator = iota
	xpathLogicalAnd
)

type xpathArithmeticOperator int

const (
	xpathArithmeticAdd xpathArithmeticOperator = iota
	xpathArithmeticSubtract
	xpathArithmeticMultiply
	xpathArithmeticDivide
	xpathArithmeticMod
)

type xpathExpressionContext struct {
	root     *Node
	position int
	size     int
}

func newXPathExpressionContext(root *Node) xpathExpressionContext {
	if root == nil {
		return xpathExpressionContext{}
	}
	return xpathExpressionContext{
		root:     root,
		position: 1,
		size:     1,
	}
}

func NewXPathParser(xmlText string, configs ...XPathParserConfig) (*XPathParser, error) {
	return newXPathParser(strings.NewReader(xmlText), nil, true, configs...)
}

func NewXPathParserFromReader(reader io.Reader, configs ...XPathParserConfig) (*XPathParser, error) {
	return newXPathParser(reader, nil, true, configs...)
}

func NewXPathParserFromInputStream(inputStream io.Reader, configs ...XPathParserConfig) (*XPathParser, error) {
	return newXPathParser(inputStream, nil, false, configs...)
}

func NewXPathParserFromDocument(document *Document, configs ...XPathParserConfig) (*XPathParser, error) {
	return newXPathParser(nil, document, true, configs...)
}

func newXPathParser(
	reader io.Reader,
	document *Document,
	inputAlreadyDecoded bool,
	configs ...XPathParserConfig,
) (*XPathParser, error) {
	config := firstConfig(configs)
	parser := &XPathParser{}
	parser.commonConstructor(config)

	if document != nil {
		parser.document = document
		return parser, nil
	}

	createdDocument, err := parser.createDocument(reader, inputAlreadyDecoded)
	if err != nil {
		return nil, err
	}
	parser.document = createdDocument
	if config.Validator != nil {
		if err := config.Validator(parser); err != nil {
			return nil, err
		}
	}
	return parser, nil
}

func firstConfig(configs []XPathParserConfig) XPathParserConfig {
	if len(configs) == 0 {
		return XPathParserConfig{}
	}
	return configs[0]
}

func (parser *XPathParser) commonConstructor(config XPathParserConfig) {
	parser.validation = config.Validation
	parser.entityResolver = config.EntityResolver
	parser.variables = config.Variables
	parser.resource = config.Resource
}

func (parser *XPathParser) createDocument(reader io.Reader, inputAlreadyDecoded bool) (*Document, error) {
	return ParseDocument(reader, parser.resource, ParseDocumentConfig{
		Validation:          parser.validation,
		EntityResolver:      parser.entityResolver,
		InputAlreadyDecoded: inputAlreadyDecoded,
	})
}

func (parser *XPathParser) SetVariables(variables Properties) {
	if parser == nil {
		return
	}
	parser.variables = variables
}

func (parser *XPathParser) IsValidationEnabled() bool {
	if parser == nil {
		return false
	}
	return parser.validation
}

func (parser *XPathParser) GetDocument() *Document {
	if parser == nil {
		return nil
	}
	return parser.document
}

func (parser *XPathParser) GetDocumentType() DocumentType {
	if parser == nil || parser.document == nil {
		return DocumentType{}
	}
	return parser.document.GetDocumentType()
}

func (parser *XPathParser) EvalString(expression string) string {
	return parser.EvalStringWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalStringWithRoot(root *Node, expression string) string {
	return PropertyParser.Parse(parser.evalXPathValue(root, expression).stringValueForXPath(), parser.variables)
}

func (parser *XPathParser) EvalBoolean(expression string) bool {
	return parser.EvalBooleanWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalBooleanWithRoot(root *Node, expression string) bool {
	return parser.evalXPathValue(root, expression).booleanValueForXPath()
}

func (parser *XPathParser) EvalShort(expression string) int16 {
	return parser.EvalShortWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalShortWithRoot(root *Node, expression string) int16 {
	return mustParseShort(parser.EvalStringWithRoot(root, expression))
}

func (parser *XPathParser) EvalInteger(expression string) int {
	return parser.EvalIntegerWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalIntegerWithRoot(root *Node, expression string) int {
	return mustParseInt(parser.EvalStringWithRoot(root, expression))
}

func (parser *XPathParser) EvalLong(expression string) int64 {
	return parser.EvalLongWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalLongWithRoot(root *Node, expression string) int64 {
	return mustParseInt64(parser.EvalStringWithRoot(root, expression))
}

func (parser *XPathParser) EvalFloat(expression string) float32 {
	return parser.EvalFloatWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalFloatWithRoot(root *Node, expression string) float32 {
	return mustParseFloat32(parser.EvalStringWithRoot(root, expression))
}

func (parser *XPathParser) EvalDouble(expression string) float64 {
	return parser.EvalDoubleWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalDoubleWithRoot(root *Node, expression string) float64 {
	return parser.evalXPathValue(root, expression).numberValueForXPath()
}

func (parser *XPathParser) EvalNodes(expression string) []*XNode {
	return parser.EvalNodesWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalNodesWithRoot(root *Node, expression string) []*XNode {
	value := parser.evalXPathValue(root, expression)
	if value.kind != xpathValueNodeSet {
		panic(newXPathNodeSetConversionException(value.kind))
	}
	xnodes := make([]*XNode, 0, len(value.results))
	for _, result := range value.results {
		if result.node == nil {
			continue
		}
		xnodes = append(xnodes, NewXNode(parser, result.node, parser.variables))
	}
	return xnodes
}

func (parser *XPathParser) EvalNode(expression string) *XNode {
	return parser.EvalNodeWithRoot(parser.rootNode(), expression)
}

func (parser *XPathParser) EvalNodeWithRoot(root *Node, expression string) *XNode {
	value := parser.evalXPathValue(root, expression)
	if value.kind != xpathValueNodeSet {
		panic(newXPathNodeSetConversionException(value.kind))
	}
	if len(value.results) == 0 || value.results[0].node == nil {
		return nil
	}
	return NewXNode(parser, value.results[0].node, parser.variables)
}

func (parser *XPathParser) rootNode() *Node {
	if parser == nil || parser.document == nil {
		return nil
	}
	return parser.document.GetNode()
}

func (parser *XPathParser) evaluateNodes(expression string, root *Node) []*Node {
	results := parser.evaluateResults(expression, root)
	nodes := make([]*Node, 0, len(results))
	for _, result := range results {
		if result.node != nil {
			nodes = append(nodes, result.node)
		}
	}
	return nodes
}

func (parser *XPathParser) evaluateResults(expression string, root *Node) []xpathResult {
	if parser == nil || root == nil || expression == "" {
		return nil
	}

	seen := make(map[*Node]struct{})
	results := make([]xpathResult, 0)
	for _, currentExpression := range parser.splitUnion(expression) {
		for _, result := range parser.evaluateSinglePath(currentExpression, root) {
			if result.node == nil {
				continue
			}
			if _, ok := seen[result.node]; ok {
				continue
			}
			seen[result.node] = struct{}{}
			results = append(results, result)
		}
	}

	documentOrderIndex := buildDocumentOrderIndex(parser.documentOrderRoot(root))
	sort.SliceStable(results, func(left, right int) bool {
		leftIndex, leftIndexed := documentOrderIndex[results[left].node]
		rightIndex, rightIndexed := documentOrderIndex[results[right].node]
		switch {
		case leftIndexed && rightIndexed:
			return leftIndex < rightIndex
		case leftIndexed:
			return true
		case rightIndexed:
			return false
		default:
			return results[left].node.order < results[right].node.order
		}
	})
	return results
}

func (parser *XPathParser) evalXPathValue(root *Node, expression string) xpathValue {
	return parser.evalXPathValueInContext(newXPathExpressionContext(root), expression)
}

func (parser *XPathParser) evalXPathValueInContext(context xpathExpressionContext, expression string) xpathValue {
	root := context.root
	expression = trimXPathWhitespace(expression)
	if expression == "" {
		return xpathValue{kind: xpathValueNodeSet}
	}

	if stringValue, ok := parseXPathStringLiteral(expression); ok {
		return xpathValue{
			kind:        xpathValueString,
			stringValue: stringValue,
		}
	}
	if numberValue, ok := parseXPathNumericLiteral(expression); ok {
		return xpathValue{
			kind:        xpathValueNumber,
			numberValue: numberValue,
		}
	}
	if groupedExpression, ok := parseXPathGroupedExpression(expression); ok {
		return parser.evalXPathValueInContext(context, groupedExpression)
	}
	if leftExpression, operator, rightExpression, ok := parseXPathLogicalExpression(expression); ok {
		return xpathValue{
			kind: xpathValueBoolean,
			booleanValue: parser.evalXPathLogicalInContext(
				context,
				leftExpression,
				operator,
				rightExpression,
			),
		}
	}
	if leftExpression, operator, rightExpression, ok := parseXPathComparisonExpression(expression); ok {
		return xpathValue{
			kind: xpathValueBoolean,
			booleanValue: parser.evalXPathComparisonInContext(
				context,
				leftExpression,
				operator,
				rightExpression,
			),
		}
	}
	if leftExpression, operator, rightExpression, ok := parseXPathAdditiveExpression(expression); ok {
		return xpathValue{
			kind: xpathValueNumber,
			numberValue: parser.evalXPathArithmeticInContext(
				context,
				leftExpression,
				operator,
				rightExpression,
			),
		}
	}
	if leftExpression, operator, rightExpression, ok := parseXPathMultiplicativeExpression(expression); ok {
		return xpathValue{
			kind: xpathValueNumber,
			numberValue: parser.evalXPathArithmeticInContext(
				context,
				leftExpression,
				operator,
				rightExpression,
			),
		}
	}
	if operator, operandExpression, ok := parseXPathUnaryExpression(expression); ok {
		return xpathValue{
			kind:        xpathValueNumber,
			numberValue: parser.evalXPathUnaryInContext(context, operator, operandExpression),
		}
	}

	if functionName, argument, ok := parseXPathFunctionCall(expression); ok {
		switch functionName {
		case "count":
			return xpathValue{
				kind:        xpathValueNumber,
				numberValue: float64(len(parser.evaluateResults(argument, root))),
			}
		case "name":
			return xpathValue{
				kind:        xpathValueString,
				stringValue: parser.evalXPathName(root, argument),
			}
		case "string":
			return xpathValue{
				kind:        xpathValueString,
				stringValue: parser.evalXPathStringInContext(context, argument),
			}
		case "number":
			return xpathValue{
				kind:        xpathValueNumber,
				numberValue: parser.evalXPathNumberInContext(context, argument),
			}
		case "boolean":
			return xpathValue{
				kind:         xpathValueBoolean,
				booleanValue: parser.evalXPathBooleanInContext(context, argument),
			}
		case "not":
			return xpathValue{
				kind:         xpathValueBoolean,
				booleanValue: !parser.evalXPathBooleanInContext(context, argument),
			}
		case "position":
			if argument == "" {
				return xpathValue{
					kind:        xpathValueNumber,
					numberValue: float64(context.position),
				}
			}
		case "last":
			if argument == "" {
				return xpathValue{
					kind:        xpathValueNumber,
					numberValue: float64(context.size),
				}
			}
		case "true":
			if argument == "" {
				return xpathValue{
					kind:         xpathValueBoolean,
					booleanValue: true,
				}
			}
		case "false":
			if argument == "" {
				return xpathValue{
					kind:         xpathValueBoolean,
					booleanValue: false,
				}
			}
		}
	}

	if invalidToken := invalidXPathLocationPathToken(expression); invalidToken != "" {
		panic(NewParsingException(
			"Error evaluating XPath. Cause: A location path was expected, but the following token was encountered:  "+invalidToken,
			nil,
		))
	}
	if invalidNumber := invalidXPathNumberLiteral(expression); invalidNumber != "" {
		panic(NewParsingException(
			"Error evaluating XPath. Cause: "+invalidNumber+" could not be formatted to a number!",
			nil,
		))
	}

	return xpathValue{
		kind:    xpathValueNodeSet,
		results: parser.evaluateResults(expression, root),
	}
}

func invalidXPathLocationPathToken(expression string) string {
	if expression == "" {
		return ""
	}
	if expression[0] == '+' {
		return "+"
	}
	return ""
}

func invalidXPathNumberLiteral(expression string) string {
	if expression == "" {
		return ""
	}
	if (expression[0] < '0' || expression[0] > '9') && expression[0] != '.' {
		return ""
	}

	hasDigit := false
	hasExponent := false
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case current >= '0' && current <= '9':
			hasDigit = true
		case current == '.':
		case current == 'e' || current == 'E':
			hasExponent = true
		case (current == '+' || current == '-') && index > 0 && (expression[index-1] == 'e' || expression[index-1] == 'E'):
			hasExponent = true
		default:
			return ""
		}
	}
	if !hasDigit || !hasExponent {
		return ""
	}
	return expression
}

func (parser *XPathParser) documentOrderRoot(root *Node) *Node {
	if parserRoot := parser.rootNode(); parserRoot != nil {
		return parserRoot
	}
	if root == nil {
		return nil
	}
	if ownerDocument := root.GetOwnerDocument(); ownerDocument != nil {
		if documentRoot := ownerDocument.GetNode(); documentRoot != nil {
			return documentRoot
		}
	}
	return root
}

func buildDocumentOrderIndex(root *Node) map[*Node]int {
	documentOrderIndex := make(map[*Node]int)
	position := 0
	var visit func(node *Node)
	visit = func(node *Node) {
		if node == nil {
			return
		}

		position++
		documentOrderIndex[node] = position
		for _, attribute := range node.GetAttributes() {
			visit(attribute)
		}
		for _, child := range node.GetChildNodes() {
			visit(child)
		}
	}
	visit(root)
	return documentOrderIndex
}

func (parser *XPathParser) splitUnion(expression string) []string {
	result := make([]string, 0)
	start := 0
	bracketDepth := 0
	parenthesisDepth := 0
	inString := byte(0)
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case inString != 0:
			if current == inString {
				inString = 0
			}
		case current == '\'' || current == '"':
			inString = current
		case current == '[':
			bracketDepth++
		case current == ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case current == '(':
			parenthesisDepth++
		case current == ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
		case current == '|' && bracketDepth == 0 && parenthesisDepth == 0:
			part := expression[start:index]
			if invalidToken := invalidXPathEdgeWhitespaceToken(part); invalidToken != "" {
				panic(NewParsingException(
					"Error evaluating XPath. Cause: A location path was expected, but the following token was encountered: "+invalidToken,
					nil,
				))
			}
			part = trimXPathWhitespace(part)
			if part != "" {
				result = append(result, part)
			}
			start = index + 1
		}
	}

	part := expression[start:]
	if invalidToken := invalidXPathEdgeWhitespaceToken(part); invalidToken != "" {
		panic(NewParsingException(
			"Error evaluating XPath. Cause: A location path was expected, but the following token was encountered: "+invalidToken,
			nil,
		))
	}
	part = trimXPathWhitespace(part)
	if part != "" {
		result = append(result, part)
	}
	return result
}

func trimXPathWhitespace(value string) string {
	start := 0
	for start < len(value) && isXPathWhitespaceByte(value[start]) {
		start++
	}

	end := len(value)
	for start < end && isXPathWhitespaceByte(value[end-1]) {
		end--
	}

	return value[start:end]
}

func invalidXPathEdgeWhitespaceToken(value string) string {
	for len(value) > 0 {
		current, size := utf8.DecodeRuneInString(value)
		if !unicode.IsSpace(current) {
			break
		}
		if !isXPathWhitespaceRune(current) {
			return string(current)
		}
		value = value[size:]
	}

	for len(value) > 0 {
		current, size := utf8.DecodeLastRuneInString(value)
		if !unicode.IsSpace(current) {
			break
		}
		if !isXPathWhitespaceRune(current) {
			return string(current)
		}
		value = value[:len(value)-size]
	}

	return ""
}

func isXPathWhitespaceByte(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func isXPathWhitespaceRune(value rune) bool {
	switch value {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func (parser *XPathParser) evaluateSinglePath(expression string, root *Node) []xpathResult {
	expression = trimXPathWhitespace(expression)
	if expression == "" {
		return nil
	}
	if groupedExpression, predicates, continuation, ok := parseXPathGroupedResultExpression(expression); ok {
		groupedValue := parser.evalXPathValue(root, groupedExpression)
		if groupedValue.kind != xpathValueNodeSet {
			return nil
		}
		currentResults := groupedValue.results
		groupedResultSize := len(currentResults)
		for _, predicate := range predicates {
			currentResults = parser.applyXPathResultPredicateWithContextSize(currentResults, predicate, groupedResultSize)
			if len(currentResults) == 0 {
				return nil
			}
		}
		if continuation == "" {
			return currentResults
		}
		_, continuationSteps := parseXPathSteps(continuation)
		return parser.applyXPathStepsToResults(currentResults, continuationSteps)
	}

	absolute, steps := parseXPathSteps(expression)
	if absolute {
		root = parser.rootNode()
	}
	if root == nil {
		return nil
	}

	currentResults := []xpathResult{newXPathResult(root)}
	return parser.applyXPathStepsToResults(currentResults, steps)
}

func (parser *XPathParser) applyXPathStepsToResults(currentResults []xpathResult, steps []xpathStep) []xpathResult {
	if len(currentResults) == 0 || len(steps) == 0 {
		return currentResults
	}

	for _, step := range steps {
		nextResults := make([]xpathResult, 0)
		for _, currentResult := range currentResults {
			matched := parser.applyStep(currentResult.node, step)
			for _, predicate := range step.predicates {
				if predicate.kind == xpathPredicateNone {
					continue
				}
				matched = parser.applyXPathResultPredicate(matched, predicate)
				if len(matched) == 0 {
					break
				}
			}
			nextResults = append(nextResults, matched...)
		}
		currentResults = nextResults
		if len(currentResults) == 0 {
			break
		}
	}

	return currentResults
}

func (parser *XPathParser) applyStep(node *Node, step xpathStep) []xpathResult {
	if node == nil {
		return nil
	}

	if step.axis == xpathAxisDescendant {
		return parser.applyDescendantStep(node, step)
	}

	switch step.kind {
	case xpathStepSelf:
		return []xpathResult{newXPathResult(node)}
	case xpathStepParent:
		parent := xpathParentNode(node)
		if parent == nil {
			return nil
		}
		return []xpathResult{newXPathResult(parent)}
	case xpathStepWildcard:
		return newXPathResults(node.elementChildren())
	case xpathStepAttributeName:
		attribute := parser.attributeForXPathStep(node, step.value)
		if attribute == nil {
			return nil
		}
		return []xpathResult{newXPathResult(attribute)}
	case xpathStepAttributeWildcard:
		return newXPathResults(node.GetAttributes())
	case xpathStepText:
		return xpathTextResults(node)
	case xpathStepNode:
		return xpathNodeResults(node)
	default:
		return newXPathResults(parser.childElementsForXPathStep(node, step.value))
	}
}

func (parser *XPathParser) applyDescendantStep(node *Node, step xpathStep) []xpathResult {
	candidates := xpathDescendantOrSelfNodes(node)
	results := make([]xpathResult, 0)
	childStep := step
	childStep.axis = xpathAxisChild
	for _, candidate := range candidates {
		results = append(results, parser.applyStep(candidate, childStep)...)
	}
	return results
}

func (parser *XPathParser) attributeForXPathStep(node *Node, step string) *Node {
	if node == nil || step == "" || strings.Contains(step, ":") {
		return nil
	}
	for _, attribute := range node.GetAttributes() {
		if xpathStepMatchesNodeName(step, attribute.GetNodeName()) {
			return attribute
		}
	}
	return nil
}

func (parser *XPathParser) childElementsForXPathStep(node *Node, step string) []*Node {
	if node == nil || step == "" || strings.Contains(step, ":") {
		return nil
	}
	children := make([]*Node, 0)
	for _, child := range node.GetChildNodes() {
		if child.GetNodeType() != ELEMENT_NODE {
			continue
		}
		if xpathStepMatchesNodeName(step, child.GetNodeName()) {
			children = append(children, child)
		}
	}
	return children
}

func newXPathResult(node *Node) xpathResult {
	return xpathResult{node: node}
}

func newXPathResults(nodes []*Node) []xpathResult {
	results := make([]xpathResult, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		results = append(results, newXPathResult(node))
	}
	return results
}

func (result xpathResult) resolvedStringValue() string {
	if result.hasStringValue {
		return result.stringValue
	}
	if result.node == nil {
		return ""
	}
	return result.node.stringValue()
}

func (value xpathValue) stringValueForXPath() string {
	switch value.kind {
	case xpathValueString:
		return value.stringValue
	case xpathValueNumber:
		return strconv.FormatFloat(value.numberValue, 'f', -1, 64)
	case xpathValueBoolean:
		if value.booleanValue {
			return "true"
		}
		return "false"
	case xpathValueNodeSet:
		if len(value.results) == 0 {
			return ""
		}
		return value.results[0].resolvedStringValue()
	default:
		return ""
	}
}

func (value xpathValue) numberValueForXPath() float64 {
	switch value.kind {
	case xpathValueNumber:
		return value.numberValue
	case xpathValueBoolean:
		if value.booleanValue {
			return 1
		}
		return 0
	case xpathValueString:
		parsed, err := strconv.ParseFloat(trimXPathWhitespace(value.stringValue), 64)
		if err != nil {
			return math.NaN()
		}
		return parsed
	case xpathValueNodeSet:
		if len(value.results) == 0 {
			return math.NaN()
		}
		parsed, err := strconv.ParseFloat(trimXPathWhitespace(value.results[0].resolvedStringValue()), 64)
		if err != nil {
			return math.NaN()
		}
		return parsed
	default:
		return math.NaN()
	}
}

func (value xpathValue) booleanValueForXPath() bool {
	switch value.kind {
	case xpathValueBoolean:
		return value.booleanValue
	case xpathValueNumber:
		return value.numberValue != 0 && !math.IsNaN(value.numberValue)
	case xpathValueString:
		return value.stringValue != ""
	case xpathValueNodeSet:
		return len(value.results) > 0
	default:
		return false
	}
}

func xpathPredicateValueMatchesContextPosition(value xpathValue, position int) bool {
	if value.kind == xpathValueNumber {
		return xpathCompareNumbers(value.numberValue, float64(position), xpathComparisonEqual)
	}
	return value.booleanValueForXPath()
}

func newXPathGroupedUnionBooleanComparisonException() error {
	return NewParsingException(
		"Error evaluating XPath. Cause: class com.sun.org.apache.xpath.internal.objects.XBooleanStatic cannot be cast to class com.sun.org.apache.xpath.internal.objects.XNodeSet (com.sun.org.apache.xpath.internal.objects.XBooleanStatic and com.sun.org.apache.xpath.internal.objects.XNodeSet are in module java.xml of loader 'bootstrap')",
		nil,
	)
}

func newXPathNodeSetConversionException(kind xpathValueKind) error {
	return NewParsingException(
		"Error evaluating XPath. Cause: Can not convert "+xpathValueTypeName(kind)+" to a NodeList!",
		nil,
	)
}

func xpathValueTypeName(kind xpathValueKind) string {
	switch kind {
	case xpathValueString:
		return "#STRING"
	case xpathValueNumber:
		return "#NUMBER"
	case xpathValueBoolean:
		return "#BOOLEAN"
	default:
		return "#NODESET"
	}
}

func xpathParentNode(node *Node) *Node {
	if node == nil {
		return nil
	}
	return node.parent
}

func xpathTextResults(node *Node) []xpathResult {
	if !xpathSupportsChildAxis(node) {
		return nil
	}

	children := node.GetChildNodes()
	results := make([]xpathResult, 0)
	for index := 0; index < len(children); {
		if !isXPathTextLikeNode(children[index]) {
			index++
			continue
		}

		anchor, stringValue, nextIndex := xpathTextRun(children, index)
		if anchor != nil {
			results = append(results, xpathResult{
				node:           anchor,
				stringValue:    stringValue,
				hasStringValue: true,
			})
		}
		index = nextIndex
	}
	return results
}

func xpathNodeResults(node *Node) []xpathResult {
	if !xpathSupportsChildAxis(node) {
		return nil
	}

	children := node.GetChildNodes()
	results := make([]xpathResult, 0, len(children))
	for index := 0; index < len(children); {
		child := children[index]
		if isXPathTextLikeNode(child) {
			anchor, stringValue, nextIndex := xpathTextRun(children, index)
			if anchor != nil {
				results = append(results, xpathResult{
					node:           anchor,
					stringValue:    stringValue,
					hasStringValue: true,
				})
			}
			index = nextIndex
			continue
		}
		if child.GetNodeType() == ELEMENT_NODE {
			results = append(results, newXPathResult(child))
		}
		index++
	}
	return results
}

func xpathTextRun(children []*Node, start int) (anchor *Node, stringValue string, nextIndex int) {
	nextIndex = start
	for nextIndex < len(children) && isXPathTextLikeNode(children[nextIndex]) {
		current := children[nextIndex]
		switch current.GetNodeType() {
		case TEXT_NODE, CDATA_SECTION_NODE:
			if anchor == nil {
				anchor = current
			}
			stringValue += current.GetNodeValue()
		}
		nextIndex++
	}
	return anchor, stringValue, nextIndex
}

func isXPathTextLikeNode(node *Node) bool {
	if node == nil {
		return false
	}
	switch node.GetNodeType() {
	case TEXT_NODE, CDATA_SECTION_NODE, ENTITY_REFERENCE_NODE:
		return true
	default:
		return false
	}
}

func xpathSupportsChildAxis(node *Node) bool {
	if node == nil {
		return false
	}
	switch node.GetNodeType() {
	case DOCUMENT_NODE, ELEMENT_NODE:
		return true
	default:
		return false
	}
}

func xpathDescendantOrSelfNodes(node *Node) []*Node {
	if node == nil {
		return nil
	}

	nodes := []*Node{node}
	for _, child := range xpathNodeResults(node) {
		nodes = append(nodes, xpathDescendantOrSelfNodes(child.node)...)
	}
	return nodes
}

func xpathStepMatchesNodeName(step, nodeName string) bool {
	if step == "" || nodeName == "" {
		return false
	}
	return step == xpathLocalName(nodeName)
}

func xpathLocalName(nodeName string) string {
	if separatorIndex := strings.LastIndex(nodeName, ":"); separatorIndex >= 0 && separatorIndex < len(nodeName)-1 {
		return nodeName[separatorIndex+1:]
	}
	return nodeName
}

func parseXPathFunctionCall(expression string) (functionName string, argument string, ok bool) {
	if expression == "" || !strings.HasSuffix(expression, ")") {
		return "", "", false
	}

	openIndex := strings.Index(expression, "(")
	if openIndex <= 0 {
		return "", "", false
	}

	functionName = trimXPathWhitespace(expression[:openIndex])
	argument = trimXPathWhitespace(expression[openIndex+1 : len(expression)-1])
	if functionName == "" {
		return "", "", false
	}
	return functionName, argument, true
}

func parseXPathLogicalExpression(expression string) (left string, operator xpathLogicalOperator, right string, ok bool) {
	if index := findTopLevelXPathKeyword(expression, "or"); index >= 0 {
		left = trimXPathWhitespace(expression[:index])
		right = trimXPathWhitespace(expression[index+len("or"):])
		if left == "" || right == "" {
			return "", 0, "", false
		}
		return left, xpathLogicalOr, right, true
	}
	if index := findTopLevelXPathKeyword(expression, "and"); index >= 0 {
		left = trimXPathWhitespace(expression[:index])
		right = trimXPathWhitespace(expression[index+len("and"):])
		if left == "" || right == "" {
			return "", 0, "", false
		}
		return left, xpathLogicalAnd, right, true
	}
	return "", 0, "", false
}

func parseXPathGroupedExpression(expression string) (groupedExpression string, ok bool) {
	groupedExpression, remainder, ok := splitLeadingXPathGroupedExpression(expression)
	if !ok || remainder != "" {
		return "", false
	}
	return groupedExpression, true
}

func splitLeadingXPathGroupedExpression(expression string) (groupedExpression, remainder string, ok bool) {
	expression = trimXPathWhitespace(expression)
	if len(expression) < 2 || expression[0] != '(' {
		return "", "", false
	}

	parenthesisDepth := 0
	inString := byte(0)
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case inString != 0:
			if current == inString {
				inString = 0
			}
		case current == '\'' || current == '"':
			inString = current
		case current == '(':
			parenthesisDepth++
		case current == ')':
			parenthesisDepth--
			if parenthesisDepth == 0 {
				groupedExpression = trimXPathWhitespace(expression[1:index])
				if groupedExpression == "" {
					return "", "", false
				}
				return groupedExpression, trimXPathWhitespace(expression[index+1:]), true
			}
		}
	}
	return "", "", false
}

func parseXPathGroupedResultExpression(expression string) (groupedExpression string, predicates []xpathPredicate, continuation string, ok bool) {
	groupedExpression, remainder, ok := splitLeadingXPathGroupedExpression(expression)
	if !ok {
		return "", nil, "", false
	}

	predicates = make([]xpathPredicate, 0)
	for strings.HasPrefix(remainder, "[") {
		predicateText, nextRemainder, ok := cutLeadingXPathPredicate(remainder)
		if !ok {
			return "", nil, "", false
		}
		predicate, ok := parseXPathPredicateTextValue(predicateText)
		if !ok {
			return "", nil, "", false
		}
		predicates = append(predicates, predicate)
		remainder = nextRemainder
	}
	if remainder != "" && !strings.HasPrefix(remainder, "/") {
		return "", nil, "", false
	}
	return groupedExpression, predicates, remainder, true
}

func cutLeadingXPathPredicate(expression string) (predicateText, remainder string, ok bool) {
	expression = trimXPathWhitespace(expression)
	if expression == "" || expression[0] != '[' {
		return "", "", false
	}

	bracketDepth := 0
	inString := byte(0)
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case inString != 0:
			if current == inString {
				inString = 0
			}
		case current == '\'' || current == '"':
			inString = current
		case current == '[':
			bracketDepth++
		case current == ']':
			bracketDepth--
			if bracketDepth == 0 {
				predicateText = trimXPathWhitespace(expression[1:index])
				if predicateText == "" {
					return "", "", false
				}
				return predicateText, trimXPathWhitespace(expression[index+1:]), true
			}
		}
	}
	return "", "", false
}

func parseXPathUnaryExpression(expression string) (operator byte, operandExpression string, ok bool) {
	if len(expression) == 0 {
		return 0, "", false
	}
	switch expression[0] {
	case '-':
		operandExpression = trimXPathWhitespace(expression[1:])
		if operandExpression == "" {
			return 0, "", false
		}
		return expression[0], operandExpression, true
	default:
		return 0, "", false
	}
}

func findTopLevelXPathKeyword(expression, keyword string) int {
	if expression == "" || keyword == "" || len(keyword) > len(expression) {
		return -1
	}

	bracketDepth := 0
	parenthesisDepth := 0
	inString := byte(0)
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case inString != 0:
			if current == inString {
				inString = 0
			}
			continue
		case current == '\'' || current == '"':
			inString = current
			continue
		case current == '[':
			bracketDepth++
			continue
		case current == ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case current == '(':
			parenthesisDepth++
			continue
		case current == ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
			continue
		}

		if bracketDepth != 0 || parenthesisDepth != 0 {
			continue
		}
		if !strings.HasPrefix(expression[index:], keyword) {
			continue
		}
		if !isXPathKeywordBoundary(expression, index-1) || !isXPathKeywordBoundary(expression, index+len(keyword)) {
			continue
		}
		return index
	}
	return -1
}

func isXPathKeywordBoundary(expression string, index int) bool {
	if index < 0 || index >= len(expression) {
		return true
	}
	return !isXPathNameCharByte(expression[index])
}

func isXPathNameCharByte(current byte) bool {
	switch {
	case current >= 'a' && current <= 'z':
		return true
	case current >= 'A' && current <= 'Z':
		return true
	case current >= '0' && current <= '9':
		return true
	}

	switch current {
	case '_', '-', '.', ':':
		return true
	default:
		return false
	}
}

func parseXPathComparisonExpression(expression string) (left string, operator xpathComparisonOperator, right string, ok bool) {
	for index := 0; index < len(expression); index++ {
		operator, width, matches := xpathComparisonOperatorAt(expression, index)
		if !matches {
			continue
		}
		if !isTopLevelXPathExpressionIndex(expression, index) {
			continue
		}

		left = trimXPathWhitespace(expression[:index])
		right = trimXPathWhitespace(expression[index+width:])
		if left == "" || right == "" {
			return "", 0, "", false
		}
		return left, operator, right, true
	}
	return "", 0, "", false
}

func parseXPathAdditiveExpression(expression string) (left string, operator xpathArithmeticOperator, right string, ok bool) {
	operatorIndex, operator, ok := findTopLevelXPathAdditiveOperator(expression)
	if !ok {
		return "", 0, "", false
	}

	left = trimXPathWhitespace(expression[:operatorIndex])
	right = trimXPathWhitespace(expression[operatorIndex+1:])
	if left == "" || right == "" {
		return "", 0, "", false
	}
	return left, operator, right, true
}

func parseXPathMultiplicativeExpression(expression string) (left string, operator xpathArithmeticOperator, right string, ok bool) {
	operatorIndex, operatorWidth, operator, ok := findTopLevelXPathMultiplicativeOperator(expression)
	if !ok {
		return "", 0, "", false
	}

	left = trimXPathWhitespace(expression[:operatorIndex])
	right = trimXPathWhitespace(expression[operatorIndex+operatorWidth:])
	if left == "" || right == "" {
		return "", 0, "", false
	}
	return left, operator, right, true
}

func findTopLevelXPathAdditiveOperator(expression string) (operatorIndex int, operator xpathArithmeticOperator, ok bool) {
	bracketDepth := 0
	parenthesisDepth := 0
	inString := byte(0)
	operatorIndex = -1
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case inString != 0:
			if current == inString {
				inString = 0
			}
			continue
		case current == '\'' || current == '"':
			inString = current
			continue
		case current == '[':
			bracketDepth++
			continue
		case current == ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case current == '(':
			parenthesisDepth++
			continue
		case current == ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
			continue
		}

		if bracketDepth != 0 || parenthesisDepth != 0 {
			continue
		}
		if current != '+' && current != '-' {
			continue
		}
		if !isBinaryXPathAdditiveOperator(expression, index) {
			continue
		}

		operatorIndex = index
		if current == '+' {
			operator = xpathArithmeticAdd
		} else {
			operator = xpathArithmeticSubtract
		}
	}
	if operatorIndex < 0 {
		return -1, 0, false
	}
	return operatorIndex, operator, true
}

func isBinaryXPathAdditiveOperator(expression string, index int) bool {
	previousIndex := previousNonWhitespaceXPathIndex(expression, index-1)
	nextIndex := nextNonWhitespaceXPathIndex(expression, index+1)
	if previousIndex < 0 || nextIndex < 0 {
		return false
	}

	previous := expression[previousIndex]
	next := expression[nextIndex]
	if expression[index] == '-' && isXPathNameCharByte(previous) && isXPathNameCharByte(next) {
		leftTokenStart := previousIndex
		for leftTokenStart > 0 && isXPathNameCharByte(expression[leftTokenStart-1]) {
			leftTokenStart--
		}
		if _, ok := parseXPathNumericLiteral(expression[leftTokenStart : previousIndex+1]); !ok {
			return false
		}
	}
	if previous == 'e' || previous == 'E' {
		exponentBaseIndex := previousNonWhitespaceXPathIndex(expression, previousIndex-1)
		if exponentBaseIndex >= 0 {
			exponentBase := expression[exponentBaseIndex]
			if (exponentBase >= '0' && exponentBase <= '9') || exponentBase == '.' {
				return false
			}
		}
	}
	if isXPathUnaryOperatorBoundary(previous) {
		return false
	}
	return true
}

func findTopLevelXPathMultiplicativeOperator(expression string) (operatorIndex int, operatorWidth int, operator xpathArithmeticOperator, ok bool) {
	bracketDepth := 0
	parenthesisDepth := 0
	inString := byte(0)
	operatorIndex = -1
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case inString != 0:
			if current == inString {
				inString = 0
			}
			continue
		case current == '\'' || current == '"':
			inString = current
			continue
		case current == '[':
			bracketDepth++
			continue
		case current == ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case current == '(':
			parenthesisDepth++
			continue
		case current == ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
			continue
		}

		if bracketDepth != 0 || parenthesisDepth != 0 {
			continue
		}
		switch {
		case current == '*' && isBinaryXPathMultiplicativeOperator(expression, index):
			operatorIndex = index
			operatorWidth = 1
			operator = xpathArithmeticMultiply
		case strings.HasPrefix(expression[index:], "div") &&
			isXPathKeywordBoundary(expression, index-1) &&
			isXPathKeywordBoundary(expression, index+len("div")):
			operatorIndex = index
			operatorWidth = len("div")
			operator = xpathArithmeticDivide
		case strings.HasPrefix(expression[index:], "mod") &&
			isXPathKeywordBoundary(expression, index-1) &&
			isXPathKeywordBoundary(expression, index+len("mod")):
			operatorIndex = index
			operatorWidth = len("mod")
			operator = xpathArithmeticMod
		}
	}
	if operatorIndex < 0 {
		return -1, 0, 0, false
	}
	return operatorIndex, operatorWidth, operator, true
}

func isBinaryXPathMultiplicativeOperator(expression string, index int) bool {
	previousIndex := previousNonWhitespaceXPathIndex(expression, index-1)
	nextIndex := nextNonWhitespaceXPathIndex(expression, index+1)
	if previousIndex < 0 || nextIndex < 0 {
		return false
	}
	return !isXPathUnaryOperatorBoundary(expression[previousIndex])
}

func previousNonWhitespaceXPathIndex(expression string, index int) int {
	for index >= 0 {
		if !isXPathWhitespaceByte(expression[index]) {
			return index
		}
		index--
	}
	return -1
}

func nextNonWhitespaceXPathIndex(expression string, index int) int {
	for index < len(expression) {
		if !isXPathWhitespaceByte(expression[index]) {
			return index
		}
		index++
	}
	return -1
}

func isXPathUnaryOperatorBoundary(current byte) bool {
	switch current {
	case '+', '-', '*', '/', '=', '<', '>', '!', '(', '[', '|', ',', '@':
		return true
	default:
		return false
	}
}

func xpathComparisonOperatorAt(expression string, index int) (operator xpathComparisonOperator, width int, ok bool) {
	if index < 0 || index >= len(expression) {
		return 0, 0, false
	}

	switch expression[index] {
	case '!':
		if index+1 < len(expression) && expression[index+1] == '=' {
			return xpathComparisonNotEqual, 2, true
		}
	case '<':
		if index+1 < len(expression) && expression[index+1] == '=' {
			return xpathComparisonLessThanOrEqual, 2, true
		}
		return xpathComparisonLessThan, 1, true
	case '>':
		if index+1 < len(expression) && expression[index+1] == '=' {
			return xpathComparisonGreaterThanOrEqual, 2, true
		}
		return xpathComparisonGreaterThan, 1, true
	case '=':
		if index > 0 {
			switch expression[index-1] {
			case '!', '<', '>':
				return 0, 0, false
			}
		}
		return xpathComparisonEqual, 1, true
	}

	return 0, 0, false
}

func isTopLevelXPathExpressionIndex(expression string, target int) bool {
	bracketDepth := 0
	parenthesisDepth := 0
	inString := byte(0)
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case inString != 0:
			if current == inString {
				inString = 0
			}
		case current == '\'' || current == '"':
			inString = current
		case current == '[':
			bracketDepth++
		case current == ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case current == '(':
			parenthesisDepth++
		case current == ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
		}

		if index == target {
			return inString == 0 && bracketDepth == 0 && parenthesisDepth == 0
		}
	}
	return false
}

func parseXPathStringLiteral(expression string) (value string, ok bool) {
	if len(expression) < 2 {
		return "", false
	}
	quote := expression[0]
	if (quote != '\'' && quote != '"') || expression[len(expression)-1] != quote {
		return "", false
	}
	if strings.IndexByte(expression[1:len(expression)-1], quote) >= 0 {
		return "", false
	}
	return expression[1 : len(expression)-1], true
}

func parseXPathNumericLiteral(expression string) (value float64, ok bool) {
	hasDigit := false
	hasDecimalPoint := false
	for index := 0; index < len(expression); index++ {
		current := expression[index]
		switch {
		case current >= '0' && current <= '9':
			hasDigit = true
		case current == '.':
			if hasDecimalPoint {
				return 0, false
			}
			hasDecimalPoint = true
		default:
			return 0, false
		}
	}
	if !hasDigit {
		return 0, false
	}

	parsed, err := strconv.ParseFloat(expression, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (parser *XPathParser) evalXPathName(root *Node, argument string) string {
	if argument == "" {
		return xpathName(root)
	}

	value := parser.evalXPathValue(root, argument)
	if value.kind != xpathValueNodeSet || len(value.results) == 0 {
		return ""
	}
	return xpathName(value.results[0].node)
}

func (parser *XPathParser) evalXPathString(root *Node, argument string) string {
	return parser.evalXPathStringInContext(newXPathExpressionContext(root), argument)
}

func (parser *XPathParser) evalXPathStringInContext(context xpathExpressionContext, argument string) string {
	root := context.root
	if argument == "" {
		if root == nil {
			return ""
		}
		return root.stringValue()
	}

	return parser.evalXPathValueInContext(context, argument).stringValueForXPath()
}

func (parser *XPathParser) evalXPathNumber(root *Node, argument string) float64 {
	return parser.evalXPathNumberInContext(newXPathExpressionContext(root), argument)
}

func (parser *XPathParser) evalXPathNumberInContext(context xpathExpressionContext, argument string) float64 {
	root := context.root
	if argument == "" {
		if root == nil {
			return math.NaN()
		}
		return xpathValue{
			kind:        xpathValueString,
			stringValue: root.stringValue(),
		}.numberValueForXPath()
	}

	return parser.evalXPathValueInContext(context, argument).numberValueForXPath()
}

func (parser *XPathParser) evalXPathBoolean(root *Node, argument string) bool {
	return parser.evalXPathBooleanInContext(newXPathExpressionContext(root), argument)
}

func (parser *XPathParser) evalXPathBooleanInContext(context xpathExpressionContext, argument string) bool {
	if argument == "" {
		return false
	}
	return parser.evalXPathValueInContext(context, argument).booleanValueForXPath()
}

func (parser *XPathParser) evalXPathComparison(
	root *Node,
	leftExpression string,
	operator xpathComparisonOperator,
	rightExpression string,
) bool {
	return parser.evalXPathComparisonInContext(
		newXPathExpressionContext(root),
		leftExpression,
		operator,
		rightExpression,
	)
}

func (parser *XPathParser) evalXPathComparisonInContext(
	context xpathExpressionContext,
	leftExpression string,
	operator xpathComparisonOperator,
	rightExpression string,
) bool {
	leftValue := parser.evalXPathValueInContext(context, leftExpression)
	rightValue := parser.evalXPathValueInContext(context, rightExpression)
	if parser.isBareGroupedUnionExpression(leftExpression) {
		if rightValue.kind == xpathValueBoolean {
			panic(newXPathGroupedUnionBooleanComparisonException())
		}
		if operator == xpathComparisonEqual && rightValue.kind == xpathValueNodeSet {
			return rightValue.booleanValueForXPath()
		}
	}
	switch operator {
	case xpathComparisonEqual:
		return xpathValuesEqual(leftValue, rightValue)
	case xpathComparisonNotEqual:
		return xpathValuesNotEqual(leftValue, rightValue)
	case xpathComparisonLessThan, xpathComparisonLessThanOrEqual, xpathComparisonGreaterThan, xpathComparisonGreaterThanOrEqual:
		return xpathValuesRelationalCompare(leftValue, rightValue, operator)
	default:
		return false
	}
}

func (parser *XPathParser) isBareGroupedUnionExpression(expression string) bool {
	groupedExpression, ok := parseXPathGroupedExpression(expression)
	if !ok {
		return false
	}
	return len(parser.splitUnion(groupedExpression)) > 1
}

func (parser *XPathParser) evalXPathLogical(
	root *Node,
	leftExpression string,
	operator xpathLogicalOperator,
	rightExpression string,
) bool {
	return parser.evalXPathLogicalInContext(
		newXPathExpressionContext(root),
		leftExpression,
		operator,
		rightExpression,
	)
}

func (parser *XPathParser) evalXPathLogicalInContext(
	context xpathExpressionContext,
	leftExpression string,
	operator xpathLogicalOperator,
	rightExpression string,
) bool {
	switch operator {
	case xpathLogicalOr:
		return parser.evalXPathBooleanInContext(context, leftExpression) || parser.evalXPathBooleanInContext(context, rightExpression)
	case xpathLogicalAnd:
		return parser.evalXPathBooleanInContext(context, leftExpression) && parser.evalXPathBooleanInContext(context, rightExpression)
	default:
		return false
	}
}

func (parser *XPathParser) evalXPathArithmeticInContext(
	context xpathExpressionContext,
	leftExpression string,
	operator xpathArithmeticOperator,
	rightExpression string,
) float64 {
	leftValue := parser.evalXPathValueInContext(context, leftExpression)
	rightValue := parser.evalXPathValueInContext(context, rightExpression)
	switch operator {
	case xpathArithmeticAdd:
		return leftValue.numberValueForXPath() + rightValue.numberValueForXPath()
	case xpathArithmeticSubtract:
		return leftValue.numberValueForXPath() - rightValue.numberValueForXPath()
	case xpathArithmeticMultiply:
		return leftValue.numberValueForXPath() * rightValue.numberValueForXPath()
	case xpathArithmeticDivide:
		return leftValue.numberValueForXPath() / rightValue.numberValueForXPath()
	case xpathArithmeticMod:
		return math.Mod(leftValue.numberValueForXPath(), rightValue.numberValueForXPath())
	default:
		return math.NaN()
	}
}

func (parser *XPathParser) evalXPathUnaryInContext(context xpathExpressionContext, operator byte, operandExpression string) float64 {
	operandValue := parser.evalXPathValueInContext(context, operandExpression).numberValueForXPath()
	switch operator {
	case '+':
		return operandValue
	case '-':
		return -operandValue
	default:
		return math.NaN()
	}
}

func xpathValuesEqual(leftValue, rightValue xpathValue) bool {
	if leftValue.kind == xpathValueNodeSet {
		return xpathNodeSetEqualsValue(leftValue.results, rightValue)
	}
	if rightValue.kind == xpathValueNodeSet {
		return xpathNodeSetEqualsValue(rightValue.results, leftValue)
	}

	switch {
	case leftValue.kind == xpathValueBoolean || rightValue.kind == xpathValueBoolean:
		return leftValue.booleanValueForXPath() == rightValue.booleanValueForXPath()
	case leftValue.kind == xpathValueNumber || rightValue.kind == xpathValueNumber:
		return leftValue.numberValueForXPath() == rightValue.numberValueForXPath()
	default:
		return leftValue.stringValueForXPath() == rightValue.stringValueForXPath()
	}
}

func xpathValuesNotEqual(leftValue, rightValue xpathValue) bool {
	if leftValue.kind == xpathValueNodeSet {
		return xpathNodeSetNotEqualValue(leftValue.results, rightValue)
	}
	if rightValue.kind == xpathValueNodeSet {
		return xpathNodeSetNotEqualValue(rightValue.results, leftValue)
	}

	switch {
	case leftValue.kind == xpathValueBoolean || rightValue.kind == xpathValueBoolean:
		return leftValue.booleanValueForXPath() != rightValue.booleanValueForXPath()
	case leftValue.kind == xpathValueNumber || rightValue.kind == xpathValueNumber:
		return leftValue.numberValueForXPath() != rightValue.numberValueForXPath()
	default:
		return leftValue.stringValueForXPath() != rightValue.stringValueForXPath()
	}
}

func xpathNodeSetEqualsValue(results []xpathResult, otherValue xpathValue) bool {
	switch otherValue.kind {
	case xpathValueNodeSet:
		for _, leftResult := range results {
			for _, rightResult := range otherValue.results {
				if leftResult.resolvedStringValue() == rightResult.resolvedStringValue() {
					return true
				}
			}
		}
		return false
	case xpathValueBoolean:
		return (xpathValue{kind: xpathValueNodeSet, results: results}).booleanValueForXPath() == otherValue.booleanValueForXPath()
	case xpathValueNumber:
		otherNumber := otherValue.numberValueForXPath()
		for _, result := range results {
			if (xpathValue{
				kind:        xpathValueString,
				stringValue: result.resolvedStringValue(),
			}).numberValueForXPath() == otherNumber {
				return true
			}
		}
		return false
	default:
		otherString := otherValue.stringValueForXPath()
		for _, result := range results {
			if result.resolvedStringValue() == otherString {
				return true
			}
		}
		return false
	}
}

func xpathNodeSetNotEqualValue(results []xpathResult, otherValue xpathValue) bool {
	switch otherValue.kind {
	case xpathValueNodeSet:
		for _, leftResult := range results {
			for _, rightResult := range otherValue.results {
				if leftResult.resolvedStringValue() != rightResult.resolvedStringValue() {
					return true
				}
			}
		}
		return false
	case xpathValueBoolean:
		return (xpathValue{kind: xpathValueNodeSet, results: results}).booleanValueForXPath() != otherValue.booleanValueForXPath()
	case xpathValueNumber:
		otherNumber := otherValue.numberValueForXPath()
		for _, result := range results {
			if xpathStringToNumber(result.resolvedStringValue()) != otherNumber {
				return true
			}
		}
		return false
	default:
		otherString := otherValue.stringValueForXPath()
		for _, result := range results {
			if result.resolvedStringValue() != otherString {
				return true
			}
		}
		return false
	}
}

func xpathValuesRelationalCompare(leftValue, rightValue xpathValue, operator xpathComparisonOperator) bool {
	if leftValue.kind == xpathValueNodeSet {
		return xpathNodeSetRelationalCompare(leftValue.results, rightValue, operator)
	}
	if rightValue.kind == xpathValueNodeSet {
		return xpathValueRelationalCompareNodeSet(leftValue, rightValue.results, operator)
	}
	return xpathCompareNumbers(leftValue.numberValueForXPath(), rightValue.numberValueForXPath(), operator)
}

func xpathNodeSetRelationalCompare(results []xpathResult, otherValue xpathValue, operator xpathComparisonOperator) bool {
	switch otherValue.kind {
	case xpathValueNodeSet:
		for _, leftResult := range results {
			leftNumber := xpathStringToNumber(leftResult.resolvedStringValue())
			for _, rightResult := range otherValue.results {
				if xpathCompareNumbers(leftNumber, xpathStringToNumber(rightResult.resolvedStringValue()), operator) {
					return true
				}
			}
		}
		return false
	case xpathValueBoolean:
		return xpathCompareNumbers(
			xpathBooleanValueToNumber((xpathValue{kind: xpathValueNodeSet, results: results}).booleanValueForXPath()),
			otherValue.numberValueForXPath(),
			operator,
		)
	default:
		otherNumber := otherValue.numberValueForXPath()
		for _, result := range results {
			if xpathCompareNumbers(xpathStringToNumber(result.resolvedStringValue()), otherNumber, operator) {
				return true
			}
		}
		return false
	}
}

func xpathValueRelationalCompareNodeSet(leftValue xpathValue, results []xpathResult, operator xpathComparisonOperator) bool {
	if leftValue.kind == xpathValueBoolean {
		return xpathCompareNumbers(
			leftValue.numberValueForXPath(),
			xpathBooleanValueToNumber((xpathValue{kind: xpathValueNodeSet, results: results}).booleanValueForXPath()),
			operator,
		)
	}
	leftNumber := leftValue.numberValueForXPath()
	for _, result := range results {
		if xpathCompareNumbers(leftNumber, xpathStringToNumber(result.resolvedStringValue()), operator) {
			return true
		}
	}
	return false
}

func xpathBooleanValueToNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func xpathStringToNumber(value string) float64 {
	return (xpathValue{
		kind:        xpathValueString,
		stringValue: value,
	}).numberValueForXPath()
}

func xpathCompareNumbers(left, right float64, operator xpathComparisonOperator) bool {
	switch operator {
	case xpathComparisonEqual:
		return left == right
	case xpathComparisonNotEqual:
		return left != right
	case xpathComparisonLessThan:
		return left < right
	case xpathComparisonLessThanOrEqual:
		return left <= right
	case xpathComparisonGreaterThan:
		return left > right
	case xpathComparisonGreaterThanOrEqual:
		return left >= right
	default:
		return false
	}
}

func xpathName(node *Node) string {
	if node == nil {
		return ""
	}
	switch node.GetNodeType() {
	case ELEMENT_NODE, ATTRIBUTE_NODE:
		return node.GetNodeName()
	default:
		return ""
	}
}

func parseXPathSteps(expression string) (absolute bool, steps []xpathStep) {
	expression = trimXPathWhitespace(expression)
	if expression == "" {
		return false, nil
	}

	absolute = strings.HasPrefix(expression, "/")
	axis := xpathAxisChild
	index := 0
	if absolute {
		index = 1
		if index < len(expression) && expression[index] == '/' {
			axis = xpathAxisDescendant
			index++
		}
	}

	for index < len(expression) {
		if expression[index] == '/' {
			if index+1 < len(expression) && expression[index+1] == '/' {
				axis = xpathAxisDescendant
				index += 2
			} else {
				axis = xpathAxisChild
				index++
			}
			continue
		}

		start := index
		bracketDepth := 0
		parenthesisDepth := 0
		for index < len(expression) {
			current := expression[index]
			switch current {
			case '[':
				bracketDepth++
			case ']':
				if bracketDepth > 0 {
					bracketDepth--
				}
			case '(':
				parenthesisDepth++
			case ')':
				if parenthesisDepth > 0 {
					parenthesisDepth--
				}
			case '/':
				if bracketDepth == 0 && parenthesisDepth == 0 {
					goto tokenComplete
				}
			}
			index++
		}

	tokenComplete:
		token := trimXPathWhitespace(expression[start:index])
		if token == "" {
			continue
		}

		step := parseXPathStep(token)
		step.axis = axis
		steps = append(steps, step)
		axis = xpathAxisChild
	}

	return absolute, steps
}

func parseXPathStep(token string) xpathStep {
	base, predicates := parseXPathPredicates(token)
	step := xpathStep{
		value:      trimXPathWhitespace(base),
		predicates: predicates,
	}

	switch {
	case step.value == "*":
		step.kind = xpathStepWildcard
	case step.value == "@*":
		step.kind = xpathStepAttributeWildcard
	case step.value == ".":
		step.kind = xpathStepSelf
	case step.value == "..":
		step.kind = xpathStepParent
	case step.value == "text()":
		step.kind = xpathStepText
	case step.value == "node()":
		step.kind = xpathStepNode
	case strings.HasPrefix(step.value, "@"):
		step.kind = xpathStepAttributeName
		step.value = strings.TrimPrefix(step.value, "@")
	default:
		step.kind = xpathStepName
	}

	return step
}

func parseXPathPredicates(step string) (base string, predicates []xpathPredicate) {
	base = step
	for {
		nextBase, predicate, ok := parseXPathPredicate(base)
		if !ok {
			break
		}
		predicates = append(predicates, predicate)
		base = nextBase
	}
	reverseXPathPredicates(predicates)
	return base, predicates
}

func reverseXPathPredicates(predicates []xpathPredicate) {
	for left, right := 0, len(predicates)-1; left < right; left, right = left+1, right-1 {
		predicates[left], predicates[right] = predicates[right], predicates[left]
	}
}

func parseXPathPredicate(step string) (base string, predicate xpathPredicate, ok bool) {
	step = trimXPathWhitespace(step)
	if step == "" || !strings.HasSuffix(step, "]") {
		return step, xpathPredicate{}, false
	}

	openIndex := strings.LastIndex(step, "[")
	if openIndex <= 0 || openIndex >= len(step)-1 {
		return step, xpathPredicate{}, false
	}

	base = trimXPathWhitespace(step[:openIndex])
	if base == "" {
		return step, xpathPredicate{}, false
	}

	predicateText := trimXPathWhitespace(step[openIndex+1 : len(step)-1])
	if predicateText == "" {
		return step, xpathPredicate{}, false
	}

	if predicate, ok := parseXPathPredicateTextValue(predicateText); ok {
		return base, predicate, true
	}

	return step, xpathPredicate{}, false
}

func parseXPathPredicateTextValue(predicateText string) (xpathPredicate, bool) {
	predicateText = trimXPathWhitespace(predicateText)
	if predicateText == "" {
		return xpathPredicate{}, false
	}
	if predicate, ok := parseXPathPositionPredicateText(predicateText); ok {
		return predicate, true
	}
	if predicate, ok := parseXPathAttributePredicateText(predicateText); ok {
		return predicate, true
	}
	if predicate, ok := parseXPathStringValuePredicateText(predicateText); ok {
		return predicate, true
	}
	if predicate, ok := parseXPathTextPredicateText(predicateText); ok {
		return predicate, true
	}
	return xpathPredicate{
		kind:       xpathPredicateBooleanExpression,
		expression: predicateText,
	}, true
}

func parseXPathPositionPredicateText(predicateText string) (xpathPredicate, bool) {
	if parsed, ok := parseXPathPositionLiteral(predicateText); ok {
		return xpathPredicate{
			kind:     xpathPredicatePositionIndex,
			position: parsed,
		}, true
	}

	if predicateText == "last()" {
		return xpathPredicate{kind: xpathPredicatePositionLast}, true
	}

	if left, operator, right, ok := parseXPathComparisonExpression(predicateText); ok {
		switch {
		case left == "position()":
			return parseXPathPositionPredicateComparison(operator, right)
		case right == "position()":
			reversedOperator, ok := reverseXPathComparisonOperator(operator)
			if !ok {
				return xpathPredicate{}, false
			}
			return parseXPathPositionPredicateComparison(reversedOperator, left)
		}
	}

	return xpathPredicate{}, false
}

func reverseXPathComparisonOperator(operator xpathComparisonOperator) (xpathComparisonOperator, bool) {
	switch operator {
	case xpathComparisonEqual, xpathComparisonNotEqual:
		return operator, true
	case xpathComparisonLessThan:
		return xpathComparisonGreaterThan, true
	case xpathComparisonLessThanOrEqual:
		return xpathComparisonGreaterThanOrEqual, true
	case xpathComparisonGreaterThan:
		return xpathComparisonLessThan, true
	case xpathComparisonGreaterThanOrEqual:
		return xpathComparisonLessThanOrEqual, true
	default:
		return 0, false
	}
}

func parseXPathPositionPredicateComparison(operator xpathComparisonOperator, value string) (xpathPredicate, bool) {
	if value == "last()" {
		if operator == xpathComparisonEqual {
			return xpathPredicate{kind: xpathPredicatePositionLast}, true
		}
		return xpathPredicate{
			kind:               xpathPredicatePositionComparison,
			comparisonOperator: operator,
			compareToLast:      true,
		}, true
	}

	parsed, ok := parseXPathPositionLiteral(value)
	if !ok {
		return xpathPredicate{}, false
	}
	if operator == xpathComparisonEqual {
		return xpathPredicate{
			kind:     xpathPredicatePositionIndex,
			position: parsed,
		}, true
	}
	return xpathPredicate{
		kind:               xpathPredicatePositionComparison,
		position:           parsed,
		comparisonOperator: operator,
	}, true
}

func parseXPathPositionLiteral(value string) (int, bool) {
	value = trimXPathWhitespace(value)
	if value == "" || strings.HasPrefix(value, "+") {
		return 0, false
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func parseXPathAttributePredicateText(predicateText string) (xpathPredicate, bool) {
	if predicateText == "@*" {
		return xpathPredicate{
			kind:          xpathPredicateAttributeExists,
			attributeName: "*",
		}, true
	}

	if strings.HasPrefix(predicateText, "@") {
		attributeName := trimXPathWhitespace(strings.TrimPrefix(predicateText, "@"))
		if attributeName != "" && !strings.ContainsAny(attributeName, " \t\r\n=!<>[]()'\"|,+-*/") {
			return xpathPredicate{
				kind:          xpathPredicateAttributeExists,
				attributeName: attributeName,
			}, true
		}
	}

	if left, operator, right, ok := parseXPathComparisonExpression(predicateText); ok && operator == xpathComparisonEqual {
		left = trimXPathWhitespace(left)
		right = trimXPathWhitespace(right)
		switch {
		case strings.HasPrefix(left, "@"):
			return parseXPathAttributePredicateComparison(strings.TrimPrefix(left, "@"), right)
		case strings.HasPrefix(right, "@"):
			return parseXPathAttributePredicateComparison(strings.TrimPrefix(right, "@"), left)
		}
	}

	return xpathPredicate{}, false
}

func parseXPathAttributePredicateComparison(attributeName, value string) (xpathPredicate, bool) {
	attributeName = trimXPathWhitespace(attributeName)
	stringValue, ok := parseXPathStringLiteral(trimXPathWhitespace(value))
	if attributeName == "" || !ok {
		return xpathPredicate{}, false
	}
	return xpathPredicate{
		kind:          xpathPredicateAttributeEqualsString,
		attributeName: attributeName,
		stringValue:   stringValue,
	}, true
}

func parseXPathStringValuePredicateText(predicateText string) (xpathPredicate, bool) {
	left, right, hasEquals := strings.Cut(predicateText, "=")
	if !hasEquals {
		return xpathPredicate{}, false
	}

	left = trimXPathWhitespace(left)
	right = trimXPathWhitespace(right)
	switch {
	case left == ".":
		return parseXPathStringValuePredicateComparison(right)
	case right == ".":
		return parseXPathStringValuePredicateComparison(left)
	default:
		return xpathPredicate{}, false
	}
}

func parseXPathStringValuePredicateComparison(value string) (xpathPredicate, bool) {
	stringValue, ok := parseXPathStringLiteral(trimXPathWhitespace(value))
	if !ok {
		return xpathPredicate{}, false
	}
	return xpathPredicate{
		kind:        xpathPredicateStringValueEqualsString,
		stringValue: stringValue,
	}, true
}

func parseXPathTextPredicateText(predicateText string) (xpathPredicate, bool) {
	left, right, hasEquals := strings.Cut(predicateText, "=")
	if !hasEquals {
		return xpathPredicate{}, false
	}

	left = trimXPathWhitespace(left)
	right = trimXPathWhitespace(right)
	switch {
	case left == "text()":
		return parseXPathTextPredicateComparison(right)
	case right == "text()":
		return parseXPathTextPredicateComparison(left)
	default:
		return xpathPredicate{}, false
	}
}

func parseXPathTextPredicateComparison(value string) (xpathPredicate, bool) {
	stringValue, ok := parseXPathStringLiteral(trimXPathWhitespace(value))
	if !ok {
		return xpathPredicate{}, false
	}
	return xpathPredicate{
		kind:        xpathPredicateTextEqualsString,
		stringValue: stringValue,
	}, true
}

func applyXPathPredicate(nodes []*Node, predicate xpathPredicate) []*Node {
	switch predicate.kind {
	case xpathPredicatePositionIndex:
		if predicate.position <= 0 || predicate.position > len(nodes) {
			return nil
		}
		return nodes[predicate.position-1 : predicate.position]
	case xpathPredicatePositionLast:
		if len(nodes) == 0 {
			return nil
		}
		return nodes[len(nodes)-1:]
	case xpathPredicatePositionComparison:
		filtered := make([]*Node, 0, len(nodes))
		targetPosition := predicate.position
		if predicate.compareToLast {
			targetPosition = len(nodes)
		}
		for index, node := range nodes {
			if xpathCompareNumbers(float64(index+1), float64(targetPosition), predicate.comparisonOperator) {
				filtered = append(filtered, node)
			}
		}
		return filtered
	case xpathPredicateAttributeExists, xpathPredicateAttributeEqualsString:
		filtered := make([]*Node, 0, len(nodes))
		for _, node := range nodes {
			if xpathPredicateMatchesNode(node, predicate) {
				filtered = append(filtered, node)
			}
		}
		return filtered
	case xpathPredicateStringValueEqualsString:
		filtered := make([]*Node, 0, len(nodes))
		for _, node := range nodes {
			if node != nil && node.stringValue() == predicate.stringValue {
				filtered = append(filtered, node)
			}
		}
		return filtered
	case xpathPredicateTextEqualsString:
		filtered := make([]*Node, 0, len(nodes))
		for _, node := range nodes {
			if xpathNodeHasPredicateText(node, predicate.stringValue) {
				filtered = append(filtered, node)
			}
		}
		return filtered
	default:
		return nodes
	}
}

func (parser *XPathParser) applyXPathResultPredicate(results []xpathResult, predicate xpathPredicate) []xpathResult {
	return parser.applyXPathResultPredicateWithContextSize(results, predicate, len(results))
}

func (parser *XPathParser) applyXPathResultPredicateWithContextSize(results []xpathResult, predicate xpathPredicate, contextSize int) []xpathResult {
	switch predicate.kind {
	case xpathPredicatePositionIndex:
		if predicate.position <= 0 || predicate.position > len(results) {
			return nil
		}
		return results[predicate.position-1 : predicate.position]
	case xpathPredicatePositionLast:
		if contextSize <= 0 || contextSize > len(results) {
			return nil
		}
		return results[contextSize-1 : contextSize]
	case xpathPredicatePositionComparison:
		filtered := make([]xpathResult, 0, len(results))
		targetPosition := predicate.position
		if predicate.compareToLast {
			targetPosition = contextSize
		}
		for index, result := range results {
			if xpathCompareNumbers(float64(index+1), float64(targetPosition), predicate.comparisonOperator) {
				filtered = append(filtered, result)
			}
		}
		return filtered
	case xpathPredicateAttributeExists, xpathPredicateAttributeEqualsString, xpathPredicateStringValueEqualsString, xpathPredicateTextEqualsString:
		filtered := make([]xpathResult, 0, len(results))
		for _, result := range results {
			if xpathPredicateMatchesResult(result, predicate) {
				filtered = append(filtered, result)
			}
		}
		return filtered
	case xpathPredicateBooleanExpression:
		filtered := make([]xpathResult, 0, len(results))
		for index, result := range results {
			value := parser.evalXPathValueInContext(xpathExpressionContext{
				root:     result.node,
				position: index + 1,
				size:     contextSize,
			}, predicate.expression)
			if xpathPredicateValueMatchesContextPosition(value, index+1) {
				filtered = append(filtered, result)
			}
		}
		return filtered
	default:
		return results
	}
}

func xpathPredicateMatchesResult(result xpathResult, predicate xpathPredicate) bool {
	switch predicate.kind {
	case xpathPredicateStringValueEqualsString:
		return result.resolvedStringValue() == predicate.stringValue
	case xpathPredicateTextEqualsString:
		return xpathNodeHasPredicateText(result.node, predicate.stringValue)
	default:
		return xpathPredicateMatchesNode(result.node, predicate)
	}
}

func xpathPredicateMatchesNode(node *Node, predicate xpathPredicate) bool {
	switch predicate.kind {
	case xpathPredicateAttributeExists:
		return xpathNodeHasPredicateAttribute(node, predicate.attributeName)
	case xpathPredicateAttributeEqualsString:
		return xpathNodePredicateAttributeEquals(node, predicate.attributeName, predicate.stringValue)
	default:
		return false
	}
}

func xpathNodeHasPredicateText(node *Node, expected string) bool {
	for _, textResult := range xpathTextResults(node) {
		if textResult.resolvedStringValue() == expected {
			return true
		}
	}
	return false
}

func xpathNodeHasPredicateAttribute(node *Node, attributeName string) bool {
	if node == nil {
		return false
	}
	if attributeName == "*" {
		return len(node.GetAttributes()) > 0
	}
	for _, attribute := range node.GetAttributes() {
		if xpathPredicateMatchesAttributeName(attributeName, attribute.GetNodeName()) {
			return true
		}
	}
	return false
}

func xpathNodePredicateAttributeEquals(node *Node, attributeName, expected string) bool {
	if node == nil {
		return false
	}
	for _, attribute := range node.GetAttributes() {
		if !xpathPredicateMatchesAttributeName(attributeName, attribute.GetNodeName()) {
			continue
		}
		if attribute.stringValue() == expected {
			return true
		}
	}
	return false
}

func xpathPredicateMatchesAttributeName(attributeName, nodeName string) bool {
	if attributeName == "*" {
		return true
	}
	if attributeName == "" || strings.Contains(attributeName, ":") {
		return false
	}
	return xpathStepMatchesNodeName(attributeName, nodeName)
}
