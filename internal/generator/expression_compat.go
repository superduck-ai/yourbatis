package generator

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
)

type myBatisToken struct {
	kind string
	text string
}

type myBatisLexer struct {
	scanner scanner.Scanner
}

func newMyBatisLexer(source string) *myBatisLexer {
	lexer := &myBatisLexer{}
	lexer.scanner.Init(strings.NewReader(source))
	lexer.scanner.Mode = scanner.ScanIdents |
		scanner.ScanInts |
		scanner.ScanFloats |
		scanner.ScanChars |
		scanner.ScanStrings |
		scanner.ScanRawStrings |
		scanner.SkipComments
	return lexer
}

func (lexer *myBatisLexer) next() myBatisToken {
	current := lexer.scanner.Scan()
	switch current {
	case scanner.EOF:
		return myBatisToken{kind: "eof"}
	case scanner.Ident:
		return myBatisToken{kind: "ident", text: lexer.scanner.TokenText()}
	case scanner.Int:
		return myBatisToken{kind: "int", text: lexer.scanner.TokenText()}
	case scanner.Float:
		return myBatisToken{kind: "float", text: lexer.scanner.TokenText()}
	case scanner.Char:
		return myBatisToken{kind: "char", text: lexer.scanner.TokenText()}
	case scanner.String, scanner.RawString:
		return myBatisToken{kind: "string", text: lexer.scanner.TokenText()}
	}

	text := lexer.scanner.TokenText()
	if strings.ContainsRune("=!<>|&", current) {
		next := lexer.scanner.Peek()
		if next == '=' || next == current && (current == '|' || current == '&') {
			lexer.scanner.Next()
			text += string(next)
		}
	}
	return myBatisToken{kind: text, text: text}
}

func tokenizeMyBatisExpression(source string) []myBatisToken {
	lexer := newMyBatisLexer(source)
	result := make([]myBatisToken, 0, 16)
	for {
		current := lexer.next()
		result = append(result, current)
		if current.kind == "eof" {
			return result
		}
	}
}

func needsMyBatisCompatibility(source string) bool {
	tokens := tokenizeMyBatisExpression(source)
	for index, current := range tokens {
		switch current.kind {
		case "?", "{":
			return true
		case "ident":
			switch current.text {
			case "null", "and", "or", "not", "in",
				"eq", "ne", "neq", "lt", "le", "lte", "gt", "ge", "gte":
				return true
			case "size", "length", "isEmpty", "indexOf", "toString":
				if index > 0 && tokens[index-1].kind == "." {
					return true
				}
			}
		}
	}
	return false
}

type myBatisExpr interface {
	myBatisExpression()
}

type myBatisIdentifier struct{ name string }
type myBatisLiteral struct {
	kind  string
	value string
}
type myBatisUnary struct {
	operator string
	value    myBatisExpr
}
type myBatisBinary struct {
	operator    string
	left, right myBatisExpr
}
type myBatisSelector struct {
	value myBatisExpr
	name  string
}
type myBatisCall struct {
	function  myBatisExpr
	arguments []myBatisExpr
}
type myBatisIndex struct {
	value, index myBatisExpr
}
type myBatisCollection struct{ values []myBatisExpr }
type myBatisMembership struct {
	value      myBatisExpr
	collection *myBatisCollection
	negated    bool
}
type myBatisTernary struct {
	condition, whenTrue, whenFalse myBatisExpr
}

func (*myBatisIdentifier) myBatisExpression() {}
func (*myBatisLiteral) myBatisExpression()    {}
func (*myBatisUnary) myBatisExpression()      {}
func (*myBatisBinary) myBatisExpression()     {}
func (*myBatisSelector) myBatisExpression()   {}
func (*myBatisCall) myBatisExpression()       {}
func (*myBatisIndex) myBatisExpression()      {}
func (*myBatisCollection) myBatisExpression() {}
func (*myBatisMembership) myBatisExpression() {}
func (*myBatisTernary) myBatisExpression()    {}

type myBatisExpressionParser struct {
	tokens  []myBatisToken
	current int
}

func parseMyBatisExpression(source string) (myBatisExpr, error) {
	parser := &myBatisExpressionParser{tokens: tokenizeMyBatisExpression(source)}
	expression, err := parser.parseTernary()
	if err != nil {
		return nil, err
	}
	if current := parser.peek(); current.kind != "eof" {
		return nil, fmt.Errorf("unexpected token %q", current.text)
	}
	return expression, nil
}

func (parser *myBatisExpressionParser) peek() myBatisToken {
	if parser.current >= len(parser.tokens) {
		return myBatisToken{kind: "eof"}
	}
	return parser.tokens[parser.current]
}

func (parser *myBatisExpressionParser) advance() myBatisToken {
	current := parser.peek()
	if parser.current < len(parser.tokens) {
		parser.current++
	}
	return current
}

func (parser *myBatisExpressionParser) accept(kind string) bool {
	if parser.peek().kind != kind {
		return false
	}
	parser.advance()
	return true
}

func (parser *myBatisExpressionParser) acceptWord(words ...string) (string, bool) {
	current := parser.peek()
	if current.kind != "ident" {
		return "", false
	}
	for _, word := range words {
		if current.text == word {
			parser.advance()
			return word, true
		}
	}
	return "", false
}

func (parser *myBatisExpressionParser) expect(kind string) error {
	if parser.accept(kind) {
		return nil
	}
	return fmt.Errorf("expected %q, found %q", kind, parser.peek().text)
}

func (parser *myBatisExpressionParser) parseTernary() (myBatisExpr, error) {
	condition, err := parser.parseOr()
	if err != nil || !parser.accept("?") {
		return condition, err
	}
	whenTrue, err := parser.parseTernary()
	if err != nil {
		return nil, err
	}
	if err := parser.expect(":"); err != nil {
		return nil, err
	}
	whenFalse, err := parser.parseTernary()
	if err != nil {
		return nil, err
	}
	return &myBatisTernary{
		condition: condition,
		whenTrue:  whenTrue,
		whenFalse: whenFalse,
	}, nil
}

func (parser *myBatisExpressionParser) parseOr() (myBatisExpr, error) {
	left, err := parser.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		operator := ""
		if parser.accept("||") {
			operator = "||"
		} else if word, ok := parser.acceptWord("or"); ok {
			operator = word
		}
		if operator == "" {
			return left, nil
		}
		right, err := parser.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &myBatisBinary{operator: operator, left: left, right: right}
	}
}

func (parser *myBatisExpressionParser) parseAnd() (myBatisExpr, error) {
	left, err := parser.parseComparison()
	if err != nil {
		return nil, err
	}
	for {
		operator := ""
		if parser.accept("&&") {
			operator = "&&"
		} else if word, ok := parser.acceptWord("and"); ok {
			operator = word
		}
		if operator == "" {
			return left, nil
		}
		right, err := parser.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &myBatisBinary{operator: operator, left: left, right: right}
	}
}

func (parser *myBatisExpressionParser) parseComparison() (myBatisExpr, error) {
	left, err := parser.parseAdditive()
	if err != nil {
		return nil, err
	}
	for {
		current := parser.peek()
		operator := ""
		switch current.kind {
		case "==", "!=", "<", "<=", ">", ">=":
			operator = parser.advance().kind
		case "ident":
			switch current.text {
			case "eq", "ne", "neq", "lt", "le", "lte", "gt", "ge", "gte":
				operator = parser.advance().text
			case "in":
				parser.advance()
				collection, err := parser.parseMembershipCollection()
				if err != nil {
					return nil, err
				}
				left = &myBatisMembership{value: left, collection: collection}
				continue
			case "not":
				saved := parser.current
				parser.advance()
				if _, ok := parser.acceptWord("in"); !ok {
					parser.current = saved
					return left, nil
				}
				collection, err := parser.parseMembershipCollection()
				if err != nil {
					return nil, err
				}
				left = &myBatisMembership{value: left, collection: collection, negated: true}
				continue
			}
		}
		if operator == "" {
			return left, nil
		}
		right, err := parser.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &myBatisBinary{operator: operator, left: left, right: right}
	}
}

func (parser *myBatisExpressionParser) parseMembershipCollection() (*myBatisCollection, error) {
	expression, err := parser.parsePrimary()
	if err != nil {
		return nil, err
	}
	collection, ok := expression.(*myBatisCollection)
	if !ok {
		return nil, fmt.Errorf("in/not in currently requires a collection literal")
	}
	return collection, nil
}

func (parser *myBatisExpressionParser) parseAdditive() (myBatisExpr, error) {
	left, err := parser.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for parser.peek().kind == "+" || parser.peek().kind == "-" {
		operator := parser.advance().kind
		right, err := parser.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &myBatisBinary{operator: operator, left: left, right: right}
	}
	return left, nil
}

func (parser *myBatisExpressionParser) parseMultiplicative() (myBatisExpr, error) {
	left, err := parser.parseUnary()
	if err != nil {
		return nil, err
	}
	for parser.peek().kind == "*" || parser.peek().kind == "/" || parser.peek().kind == "%" {
		operator := parser.advance().kind
		right, err := parser.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &myBatisBinary{operator: operator, left: left, right: right}
	}
	return left, nil
}

func (parser *myBatisExpressionParser) parseUnary() (myBatisExpr, error) {
	if parser.peek().kind == "!" ||
		parser.peek().kind == "+" ||
		parser.peek().kind == "-" ||
		parser.peek().kind == "*" {
		operator := parser.advance().kind
		value, err := parser.parseUnary()
		if err != nil {
			return nil, err
		}
		return &myBatisUnary{operator: operator, value: value}, nil
	}
	if word, ok := parser.acceptWord("not"); ok {
		value, err := parser.parseUnary()
		if err != nil {
			return nil, err
		}
		return &myBatisUnary{operator: word, value: value}, nil
	}
	return parser.parsePostfix()
}

func (parser *myBatisExpressionParser) parsePostfix() (myBatisExpr, error) {
	value, err := parser.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch {
		case parser.accept("."):
			name := parser.advance()
			if name.kind != "ident" {
				return nil, fmt.Errorf("expected selector after '.', found %q", name.text)
			}
			value = &myBatisSelector{value: value, name: name.text}
		case parser.accept("("):
			arguments, err := parser.parseArguments(")")
			if err != nil {
				return nil, err
			}
			value = &myBatisCall{function: value, arguments: arguments}
		case parser.accept("["):
			index, err := parser.parseTernary()
			if err != nil {
				return nil, err
			}
			if err := parser.expect("]"); err != nil {
				return nil, err
			}
			value = &myBatisIndex{value: value, index: index}
		default:
			return value, nil
		}
	}
}

func (parser *myBatisExpressionParser) parseArguments(close string) ([]myBatisExpr, error) {
	if parser.accept(close) {
		return nil, nil
	}
	result := make([]myBatisExpr, 0, 2)
	for {
		value, err := parser.parseTernary()
		if err != nil {
			return nil, err
		}
		result = append(result, value)
		if parser.accept(close) {
			return result, nil
		}
		if err := parser.expect(","); err != nil {
			return nil, err
		}
	}
}

func (parser *myBatisExpressionParser) parsePrimary() (myBatisExpr, error) {
	current := parser.advance()
	switch current.kind {
	case "ident":
		switch current.text {
		case "true", "false":
			return &myBatisLiteral{kind: "bool", value: current.text}, nil
		case "null", "nil":
			return &myBatisLiteral{kind: "nil", value: "nil"}, nil
		default:
			return &myBatisIdentifier{name: current.text}, nil
		}
	case "int", "float", "char", "string":
		return &myBatisLiteral{kind: current.kind, value: current.text}, nil
	case "(":
		value, err := parser.parseTernary()
		if err != nil {
			return nil, err
		}
		if err := parser.expect(")"); err != nil {
			return nil, err
		}
		return value, nil
	case "{":
		values, err := parser.parseArguments("}")
		if err != nil {
			return nil, err
		}
		return &myBatisCollection{values: values}, nil
	default:
		return nil, fmt.Errorf("unexpected token %q", current.text)
	}
}

func lowerMyBatisExpression(expression myBatisExpr) (ast.Expr, error) {
	switch current := expression.(type) {
	case *myBatisIdentifier:
		return ast.NewIdent(current.name), nil
	case *myBatisLiteral:
		switch current.kind {
		case "bool", "nil":
			return ast.NewIdent(current.value), nil
		case "int":
			return &ast.BasicLit{Kind: token.INT, Value: current.value}, nil
		case "float":
			return &ast.BasicLit{Kind: token.FLOAT, Value: current.value}, nil
		case "char":
			return &ast.BasicLit{Kind: token.CHAR, Value: current.value}, nil
		case "string":
			return &ast.BasicLit{Kind: token.STRING, Value: current.value}, nil
		default:
			return nil, fmt.Errorf("unsupported literal %q", current.value)
		}
	case *myBatisUnary:
		value, err := lowerMyBatisExpression(current.value)
		if err != nil {
			return nil, err
		}
		operator := map[string]token.Token{
			"!": token.NOT, "not": token.NOT,
			"+": token.ADD, "-": token.SUB, "*": token.MUL,
		}[current.operator]
		if operator == token.ILLEGAL {
			return nil, fmt.Errorf("unsupported unary operator %q", current.operator)
		}
		return &ast.UnaryExpr{Op: operator, X: value}, nil
	case *myBatisBinary:
		left, err := lowerMyBatisExpression(current.left)
		if err != nil {
			return nil, err
		}
		right, err := lowerMyBatisExpression(current.right)
		if err != nil {
			return nil, err
		}
		operator, ok := tokenForBinaryOperator(current.operator)
		if !ok {
			return nil, fmt.Errorf("unsupported binary operator %q", current.operator)
		}
		return &ast.BinaryExpr{X: left, Op: operator, Y: right}, nil
	case *myBatisSelector:
		value, err := lowerMyBatisExpression(current.value)
		if err != nil {
			return nil, err
		}
		switch current.name {
		case "size", "length":
			return &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{value}}, nil
		default:
			return &ast.SelectorExpr{X: value, Sel: ast.NewIdent(current.name)}, nil
		}
	case *myBatisCall:
		return lowerMyBatisCall(current)
	case *myBatisIndex:
		value, err := lowerMyBatisExpression(current.value)
		if err != nil {
			return nil, err
		}
		index, err := lowerMyBatisExpression(current.index)
		if err != nil {
			return nil, err
		}
		return &ast.IndexExpr{X: value, Index: index}, nil
	case *myBatisCollection:
		return lowerMyBatisCollection(current)
	case *myBatisMembership:
		return lowerMyBatisMembership(current)
	case *myBatisTernary:
		return lowerMyBatisTernary(current)
	default:
		return nil, fmt.Errorf("unsupported MyBatis expression %T", expression)
	}
}

func lowerMyBatisCall(call *myBatisCall) (ast.Expr, error) {
	if selector, ok := call.function.(*myBatisSelector); ok {
		receiver, err := lowerMyBatisExpression(selector.value)
		if err != nil {
			return nil, err
		}
		switch selector.name {
		case "isEmpty":
			if len(call.arguments) != 0 {
				return nil, fmt.Errorf("isEmpty expects no arguments")
			}
			return &ast.BinaryExpr{
				X:  &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{receiver}},
				Op: token.EQL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "0"},
			}, nil
		case "size", "length":
			if len(call.arguments) != 0 {
				return nil, fmt.Errorf("%s expects no arguments", selector.name)
			}
			return &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{receiver}}, nil
		case "indexOf":
			if len(call.arguments) != 1 {
				return nil, fmt.Errorf("indexOf expects one argument")
			}
			argument, err := lowerMyBatisExpression(call.arguments[0])
			if err != nil {
				return nil, err
			}
			helper := "StringIndex"
			if literal, ok := call.arguments[0].(*myBatisLiteral); ok && literal.kind == "char" {
				helper = "StringIndexRune"
			}
			return &ast.CallExpr{
				Fun:  &ast.SelectorExpr{X: ast.NewIdent("yourbatis"), Sel: ast.NewIdent(helper)},
				Args: []ast.Expr{receiver, argument},
			}, nil
		case "toString":
			if len(call.arguments) != 0 {
				return nil, fmt.Errorf("toString expects no arguments")
			}
			return &ast.CallExpr{
				Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent("String")},
			}, nil
		default:
			if selector.name == "" || unicode.IsLower(rune(selector.name[0])) {
				return nil, fmt.Errorf(
					"method %q is not an exported Go method or a supported MyBatis adapter",
					selector.name,
				)
			}
		}
	}

	function, err := lowerMyBatisExpression(call.function)
	if err != nil {
		return nil, err
	}
	arguments := make([]ast.Expr, 0, len(call.arguments))
	for _, argument := range call.arguments {
		lowered, err := lowerMyBatisExpression(argument)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, lowered)
	}
	return &ast.CallExpr{Fun: function, Args: arguments}, nil
}

func lowerMyBatisCollection(collection *myBatisCollection) (ast.Expr, error) {
	elementType, err := myBatisCollectionElementType(collection)
	if err != nil {
		return nil, err
	}
	values := make([]ast.Expr, 0, len(collection.values))
	for _, value := range collection.values {
		lowered, err := lowerMyBatisExpression(value)
		if err != nil {
			return nil, err
		}
		values = append(values, lowered)
	}
	return &ast.CompositeLit{
		Type: &ast.ArrayType{Elt: elementType},
		Elts: values,
	}, nil
}

func myBatisCollectionElementType(collection *myBatisCollection) (ast.Expr, error) {
	if len(collection.values) == 0 {
		return nil, fmt.Errorf("empty collection literal needs an explicit native Go slice type")
	}
	var result ast.Expr
	var resultName string
	for _, value := range collection.values {
		current, err := myBatisExpressionType(value)
		if err != nil {
			return nil, err
		}
		if current == nil {
			return nil, fmt.Errorf(
				"collection literal element type is not statically known; use a native Go slice literal",
			)
		}
		currentName, err := expressionString(current)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = current
			resultName = currentName
			continue
		}
		if currentName != resultName {
			return nil, fmt.Errorf(
				"collection literal mixes %s and %s; use a native Go slice literal",
				resultName,
				currentName,
			)
		}
	}
	return result, nil
}

func lowerMyBatisMembership(membership *myBatisMembership) (ast.Expr, error) {
	value, err := lowerMyBatisExpression(membership.value)
	if err != nil {
		return nil, err
	}
	if len(membership.collection.values) == 0 {
		return ast.NewIdent(strconv.FormatBool(membership.negated)), nil
	}

	var result ast.Expr
	for _, member := range membership.collection.values {
		right, err := lowerMyBatisExpression(member)
		if err != nil {
			return nil, err
		}
		operator := token.EQL
		join := token.LOR
		if membership.negated {
			operator = token.NEQ
			join = token.LAND
		}
		comparison := &ast.BinaryExpr{X: cloneExpression(value), Op: operator, Y: right}
		if result == nil {
			result = comparison
		} else {
			result = &ast.BinaryExpr{X: result, Op: join, Y: comparison}
		}
	}
	return result, nil
}

func lowerMyBatisTernary(ternary *myBatisTernary) (ast.Expr, error) {
	condition, err := lowerMyBatisExpression(ternary.condition)
	if err != nil {
		return nil, err
	}
	whenTrue, err := lowerMyBatisExpression(ternary.whenTrue)
	if err != nil {
		return nil, err
	}
	whenFalse, err := lowerMyBatisExpression(ternary.whenFalse)
	if err != nil {
		return nil, err
	}
	trueType, err := myBatisExpressionType(ternary.whenTrue)
	if err != nil {
		return nil, err
	}
	falseType, err := myBatisExpressionType(ternary.whenFalse)
	if err != nil {
		return nil, err
	}
	if trueType == nil || falseType == nil {
		return nil, fmt.Errorf(
			"ternary result type is not statically known; cast both branches to the same Go type",
		)
	}
	trueName, _ := expressionString(trueType)
	falseName, _ := expressionString(falseType)
	if trueName != falseName {
		return nil, fmt.Errorf("ternary branches have different types %s and %s", trueName, falseName)
	}

	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params:  &ast.FieldList{},
				Results: &ast.FieldList{List: []*ast.Field{{Type: trueType}}},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.IfStmt{
					Cond: condition,
					Body: &ast.BlockStmt{List: []ast.Stmt{
						&ast.ReturnStmt{Results: []ast.Expr{whenTrue}},
					}},
				},
				&ast.ReturnStmt{Results: []ast.Expr{whenFalse}},
			}},
		},
	}, nil
}

func myBatisExpressionType(expression myBatisExpr) (ast.Expr, error) {
	switch current := expression.(type) {
	case *myBatisLiteral:
		switch current.kind {
		case "bool":
			return ast.NewIdent("bool"), nil
		case "int":
			return ast.NewIdent("int"), nil
		case "float":
			return ast.NewIdent("float64"), nil
		case "char":
			return ast.NewIdent("rune"), nil
		case "string":
			return ast.NewIdent("string"), nil
		case "nil":
			return nil, nil
		}
	case *myBatisUnary:
		if current.operator == "!" || current.operator == "not" {
			return ast.NewIdent("bool"), nil
		}
	case *myBatisBinary:
		switch current.operator {
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||", "and", "or",
			"eq", "ne", "neq", "lt", "le", "lte", "gt", "ge", "gte":
			return ast.NewIdent("bool"), nil
		}
	case *myBatisMembership:
		return ast.NewIdent("bool"), nil
	case *myBatisCall:
		if selector, ok := current.function.(*myBatisSelector); ok {
			if identifier, ok := selector.value.(*myBatisIdentifier); ok &&
				identifier.name == "yourbatis" &&
				selector.name != "" &&
				unicode.IsUpper(rune(selector.name[0])) {
				return &ast.SelectorExpr{
					X:   ast.NewIdent(identifier.name),
					Sel: ast.NewIdent(selector.name),
				}, nil
			}
		}
	}
	return nil, nil
}

// cloneExpression reparses a printer-stable Go expression. It prevents one AST
// node from being shared by multiple generated comparison parents.
func cloneExpression(expression ast.Expr) ast.Expr {
	source, err := expressionString(expression)
	if err != nil {
		return expression
	}
	cloned, err := goparser.ParseExpr(source)
	if err != nil {
		return expression
	}
	return cloned
}
