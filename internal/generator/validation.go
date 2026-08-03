package generator

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

type expressionScope struct {
	parent  *expressionScope
	values  map[string]ast.Expr
	aliases map[string]string
}

func newExpressionScope(parent *expressionScope) *expressionScope {
	return &expressionScope{
		parent:  parent,
		values:  make(map[string]ast.Expr),
		aliases: make(map[string]string),
	}
}

func (scope *expressionScope) expressionAliases() map[string]string {
	result := make(map[string]string)
	for current := scope; current != nil; current = current.parent {
		for name, replacement := range current.aliases {
			if _, exists := result[name]; !exists {
				result[name] = replacement
			}
		}
	}
	return result
}

func (scope *expressionScope) lookup(name string) (ast.Expr, bool) {
	for current := scope; current != nil; current = current.parent {
		value, ok := current.values[name]
		if ok {
			return value, true
		}
	}
	return nil, false
}

func (scope *expressionScope) declare(name string, valueType ast.Expr) bool {
	if _, duplicate := scope.values[name]; duplicate {
		return false
	}
	scope.values[name] = valueType
	return true
}

type statementExpressionValidator struct {
	goPackage *GoPackage
}

func validateStatementExpressions(
	goPackage *GoPackage,
	method *MapperMethod,
	statement *ir.Statement,
) error {
	return validateStatementExpressionsWithParams(
		goPackage,
		method,
		statement,
		method.Params[1:],
	)
}

func validateStatementExpressionsWithParams(
	goPackage *GoPackage,
	method *MapperMethod,
	statement *ir.Statement,
	parameters []GoValue,
) error {
	scope := newExpressionScope(nil)
	scope.declare("yourbatis", nil)
	for alias := range method.Imports {
		scope.declare(alias, nil)
	}
	scope.declare("_databaseId", ast.NewIdent("string"))
	for _, parameter := range parameters {
		scope.values[parameter.Name] = parameter.Expr
	}
	if len(parameters) == 1 {
		scope.aliases["_parameter"] = parameters[0].Name
	}
	validator := &statementExpressionValidator{goPackage: goPackage}
	return validator.validateNodes(statement.Body, scope)
}

func (validator *statementExpressionValidator) validateNodes(
	nodes []ir.Node,
	scope *expressionScope,
) error {
	for _, node := range nodes {
		switch current := node.(type) {
		case *ir.SQLNodeBoundary:
			continue
		case *ir.Text:
			continue
		case *ir.TextExpression:
			compiled, valueType, err := validator.compileAndValidateExpression(
				current.Expression,
				scope,
				expressionText,
			)
			if err != nil {
				return sourceError(
					current.Source,
					"invalid text expression %q: %v",
					current.Expression,
					err,
				)
			}
			current.Kind, err = validator.textExpressionKind(valueType)
			if err != nil {
				return sourceError(
					current.Source,
					"unsafe text expression %q: %v",
					current.Expression,
					err,
				)
			}
			current.GoExpression = compiled.Source
		case *ir.Parameter:
			parsed, err := goparser.ParseExpr(current.Path)
			if err != nil {
				return sourceError(current.Source, "invalid parameter %q: %v", current.Path, err)
			}
			rewriteExpressionIdentifiers(parsed, scope.expressionAliases())
			if _, err := validator.inferExpressionType(parsed, scope); err != nil {
				return sourceError(current.Source, "invalid parameter %q: %v", current.Path, err)
			}
			current.GoPath, err = expressionString(parsed)
			if err != nil {
				return sourceError(current.Source, "invalid parameter %q: %v", current.Path, err)
			}
		case *ir.If:
			compiled, valueType, err := validator.compileAndValidateExpression(
				current.Test,
				scope,
				expressionCondition,
			)
			if err != nil {
				return sourceError(current.Source, "invalid if test %q: %v", current.Test, err)
			}
			current.GoTest, err = validator.conditionSource(compiled, valueType)
			if err != nil {
				return sourceError(current.Source, "invalid if test %q: %v", current.Test, err)
			}
			if err := validator.validateNodes(current.Body, newExpressionScope(scope)); err != nil {
				return err
			}
		case *ir.Choose:
			for index := range current.When {
				when := &current.When[index]
				compiled, valueType, err := validator.compileAndValidateExpression(
					when.Test,
					scope,
					expressionCondition,
				)
				if err != nil {
					return sourceError(when.Source, "invalid when test %q: %v", when.Test, err)
				}
				when.GoTest, err = validator.conditionSource(compiled, valueType)
				if err != nil {
					return sourceError(when.Source, "invalid when test %q: %v", when.Test, err)
				}
				if err := validator.validateNodes(when.Body, newExpressionScope(scope)); err != nil {
					return err
				}
			}
			if err := validator.validateNodes(current.Otherwise, newExpressionScope(scope)); err != nil {
				return err
			}
		case *ir.Foreach:
			if err := validator.validateForeach(current, scope); err != nil {
				return err
			}
		case *ir.Trim:
			// Trim nodes create a SQL builder but not a Go lexical block. Bind
			// declarations inside them therefore remain in the surrounding scope.
			if err := validator.validateNodes(current.Body, scope); err != nil {
				return err
			}
		case *ir.Bind:
			if !token.IsIdentifier(current.Name) || token.Lookup(current.Name).IsKeyword() || current.Name == "_" {
				return sourceError(current.Source, "bind name %q is not a usable Go identifier", current.Name)
			}
			if current.Name == "yourbatis" || current.Name == "_parameter" || current.Name == "_databaseId" {
				return sourceError(current.Source, "bind name %q is reserved by generated code", current.Name)
			}
			compiled, valueType, err := validator.compileAndValidateExpression(
				current.Expression,
				scope,
				expressionValue,
			)
			if err != nil {
				return sourceError(current.Source, "invalid bind expression %q: %v", current.Expression, err)
			}
			current.GoExpression = compiled.Source
			if !scope.declare(current.Name, valueType) {
				return sourceError(current.Source, "bind name %q is already declared in this scope", current.Name)
			}
		default:
			return sourceError(node.Span(), "unsupported IR node %T", node)
		}
	}
	return nil
}

func (validator *statementExpressionValidator) validateForeach(
	foreach *ir.Foreach,
	scope *expressionScope,
) error {
	if !isUsableIdentifier(foreach.Item) {
		return sourceError(foreach.Source, "foreach item %q is not a usable Go identifier", foreach.Item)
	}
	if !isUsableIdentifier(foreach.Index) {
		return sourceError(foreach.Source, "foreach index %q is not a usable Go identifier", foreach.Index)
	}
	if isReservedExpressionIdentifier(foreach.Item) || isReservedExpressionIdentifier(foreach.Index) {
		return sourceError(foreach.Source, "foreach identifiers cannot use reserved compatibility names")
	}
	if foreach.Index == foreach.Item {
		return sourceError(foreach.Source, "foreach index and item must differ")
	}

	compiled, collectionType, err := validator.compileAndValidateExpression(
		foreach.Collection,
		scope,
		expressionCollection,
	)
	if err != nil {
		return sourceError(
			foreach.Source,
			"invalid foreach collection %q: %v",
			foreach.Collection,
			err,
		)
	}
	foreach.GoCollection = compiled.Source
	indexType, itemType, err := validator.rangeTypes(collectionType)
	if err != nil {
		return sourceError(foreach.Source, "%v", err)
	}

	loopScope := newExpressionScope(scope)
	loopScope.declare(foreach.Index, indexType)
	loopScope.declare(foreach.Item, itemType)
	return validator.validateNodes(foreach.Body, loopScope)
}

func isUsableIdentifier(name string) bool {
	return name != "_" && token.IsIdentifier(name) && !token.Lookup(name).IsKeyword()
}

func isReservedExpressionIdentifier(name string) bool {
	return name == "yourbatis" || name == "_parameter" || name == "_databaseId"
}

func (validator *statementExpressionValidator) validateExpression(
	expression string,
	translate bool,
	scope *expressionScope,
) (ast.Expr, error) {
	source := expression
	if translate {
		var err error
		source, err = translateExpression(expression)
		if err != nil {
			return nil, err
		}
	}
	parsed, err := goparser.ParseExpr(source)
	if err != nil {
		return nil, err
	}
	return validator.inferExpressionType(parsed, scope)
}

func (validator *statementExpressionValidator) compileAndValidateExpression(
	expression string,
	scope *expressionScope,
	mode expressionMode,
) (compiledExpression, ast.Expr, error) {
	compiled, err := compileExpressionForMode(expression, mode)
	if err != nil {
		return compiledExpression{}, nil, err
	}
	rewriteExpressionIdentifiers(compiled.AST, scope.expressionAliases())
	compiled.Source, err = expressionString(compiled.AST)
	if err != nil {
		return compiledExpression{}, nil, err
	}
	valueType, err := validator.inferExpressionType(compiled.AST, scope)
	if err != nil {
		return compiledExpression{}, nil, err
	}
	return compiled, valueType, nil
}

func (validator *statementExpressionValidator) conditionSource(
	compiled compiledExpression,
	valueType ast.Expr,
) (string, error) {
	if identifier, ok := compiled.AST.(*ast.Ident); ok && identifier.Name == "nil" {
		return "false", nil
	}
	if valueType == nil {
		return "", fmt.Errorf(
			"condition type cannot be determined statically; use an explicit boolean comparison",
		)
	}

	underlying := validator.underlyingType(valueType, make(map[string]bool))
	switch current := underlying.(type) {
	case *ast.ParenExpr:
		return validator.conditionSource(compiled, current.X)
	case *ast.Ident:
		switch current.Name {
		case "bool":
			return compiled.Source, nil
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"byte", "rune",
			"float32", "float64",
			"complex64", "complex128":
			return fmt.Sprintf("(%s) != 0", compiled.Source), nil
		case "string":
			return fmt.Sprintf("yourbatis.ValuePresent(%s)", compiled.Source), nil
		default:
			return "", fmt.Errorf(
				"condition type %s cannot be classified statically; use an explicit boolean comparison",
				current.Name,
			)
		}
	case *ast.StarExpr:
		return fmt.Sprintf("(%s) != nil", compiled.Source), nil
	case *ast.ArrayType:
		if current.Len == nil {
			return fmt.Sprintf("(%s) != nil", compiled.Source), nil
		}
		return fmt.Sprintf("yourbatis.ValuePresent(%s)", compiled.Source), nil
	case *ast.StructType:
		return fmt.Sprintf("yourbatis.ValuePresent(%s)", compiled.Source), nil
	case *ast.MapType, *ast.ChanType, *ast.FuncType:
		return fmt.Sprintf("(%s) != nil", compiled.Source), nil
	case *ast.InterfaceType:
		return "", fmt.Errorf(
			"interface condition requires an explicit boolean comparison; runtime reflection is disabled",
		)
	case *ast.SelectorExpr:
		typeName, _ := expressionString(valueType)
		return "", fmt.Errorf(
			"condition type %s is external and cannot be classified statically; use an explicit boolean comparison",
			typeName,
		)
	default:
		typeName, _ := expressionString(valueType)
		return "", fmt.Errorf(
			"condition type %s is unsupported; use an explicit boolean comparison",
			typeName,
		)
	}
}

func (validator *statementExpressionValidator) textExpressionKind(valueType ast.Expr) (string, error) {
	if valueType == nil {
		return "", fmt.Errorf(
			"type cannot be determined statically; use #{} binding or an explicit yourbatis.Identifier/yourbatis.Fragment conversion",
		)
	}
	underlying := validator.underlyingType(valueType, make(map[string]bool))
	switch current := underlying.(type) {
	case *ast.Ident:
		switch current.Name {
		case "int", "int8", "int16", "int32", "int64", "rune":
			return "signed", nil
		case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte":
			return "unsigned", nil
		case "float32", "float64":
			return "float", nil
		case "bool":
			return "bool", nil
		case "string":
			return "", fmt.Errorf(
				"plain string interpolation is disabled; use #{} binding, yourbatis.Identifier, or yourbatis.Fragment",
			)
		case "any":
			return "", fmt.Errorf("interface text interpolation would require runtime reflection")
		default:
			return "", fmt.Errorf("type %s is not an approved SQL text type", current.Name)
		}
	case *ast.SelectorExpr:
		typeName, err := expressionString(current)
		if err != nil {
			return "", err
		}
		switch typeName {
		case "yourbatis.Identifier":
			return "identifier", nil
		case "yourbatis.Fragment":
			return "fragment", nil
		default:
			return "", fmt.Errorf("type %s is not an approved SQL text type", typeName)
		}
	case *ast.InterfaceType:
		return "", fmt.Errorf("interface text interpolation would require runtime reflection")
	default:
		typeName, _ := expressionString(valueType)
		return "", fmt.Errorf(
			"type %s is not an approved SQL text type; use #{} binding",
			typeName,
		)
	}
}

func (validator *statementExpressionValidator) inferExpressionType(
	expression ast.Expr,
	scope *expressionScope,
) (ast.Expr, error) {
	switch current := expression.(type) {
	case *ast.Ident:
		switch current.Name {
		case "true", "false":
			return ast.NewIdent("bool"), nil
		case "nil":
			return nil, nil
		}
		valueType, ok := scope.lookup(current.Name)
		if !ok {
			return nil, fmt.Errorf("unknown identifier %q", current.Name)
		}
		return valueType, nil
	case *ast.BasicLit:
		return basicLiteralType(current), nil
	case *ast.ParenExpr:
		return validator.inferExpressionType(current.X, scope)
	case *ast.StarExpr:
		baseType, err := validator.inferExpressionType(current.X, scope)
		if err != nil {
			return nil, err
		}
		underlying := validator.underlyingType(baseType, make(map[string]bool))
		switch pointer := underlying.(type) {
		case nil:
			return nil, nil
		case *ast.StarExpr:
			return pointer.X, nil
		case *ast.SelectorExpr, *ast.InterfaceType:
			return nil, nil
		default:
			typeName, _ := expressionString(baseType)
			return nil, fmt.Errorf("cannot dereference non-pointer type %s", typeName)
		}
	case *ast.SelectorExpr:
		baseType, err := validator.inferExpressionType(current.X, scope)
		if err != nil {
			return nil, err
		}
		info := validator.structInfo(baseType, make(map[string]bool))
		if info == nil {
			if validator.isKnownScalarType(baseType) {
				typeName, _ := expressionString(baseType)
				return nil, fmt.Errorf(
					"cannot select field %q from scalar type %s",
					current.Sel.Name,
					typeName,
				)
			}
			return nil, nil
		}
		for _, field := range info.Fields {
			if field.Name == current.Sel.Name {
				return field.Type, nil
			}
		}
		if info.HasEmbedded {
			return nil, nil
		}
		return nil, fmt.Errorf("type %s has no exported field %q", info.Name, current.Sel.Name)
	case *ast.IndexExpr:
		baseType, err := validator.inferExpressionType(current.X, scope)
		if err != nil {
			return nil, err
		}
		if _, err := validator.inferExpressionType(current.Index, scope); err != nil {
			return nil, err
		}
		return validator.indexedType(baseType)
	case *ast.SliceExpr:
		baseType, err := validator.inferExpressionType(current.X, scope)
		if err != nil {
			return nil, err
		}
		for _, bound := range []ast.Expr{current.Low, current.High, current.Max} {
			if bound == nil {
				continue
			}
			if _, err := validator.inferExpressionType(bound, scope); err != nil {
				return nil, err
			}
		}
		return baseType, nil
	case *ast.UnaryExpr:
		return validator.inferExpressionType(current.X, scope)
	case *ast.BinaryExpr:
		leftType, err := validator.inferExpressionType(current.X, scope)
		if err != nil {
			return nil, err
		}
		rightType, err := validator.inferExpressionType(current.Y, scope)
		if err != nil {
			return nil, err
		}
		if err := validator.validateNilComparison(current, leftType, rightType); err != nil {
			return nil, err
		}
		switch current.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.LAND, token.LOR:
			return ast.NewIdent("bool"), nil
		default:
			return leftType, nil
		}
	case *ast.CallExpr:
		for _, argument := range current.Args {
			if _, err := validator.inferExpressionType(argument, scope); err != nil {
				return nil, err
			}
		}
		switch function := current.Fun.(type) {
		case *ast.FuncLit:
			if function.Type.Results == nil || len(function.Type.Results.List) != 1 {
				return nil, nil
			}
			return function.Type.Results.List[0].Type, nil
		case *ast.Ident:
			if function.Name == "len" || function.Name == "cap" {
				if len(current.Args) != 1 {
					return nil, fmt.Errorf("%s expects one argument", function.Name)
				}
				return ast.NewIdent("int"), nil
			}
			// Other identifiers may be package-local helpers or type
			// conversions. The generated Go compiler remains authoritative for
			// their signatures.
			return nil, nil
		case *ast.SelectorExpr:
			if _, err := validator.inferExpressionType(function.X, scope); err != nil {
				return nil, err
			}
			if identifier, ok := function.X.(*ast.Ident); ok && identifier.Name == "yourbatis" {
				switch function.Sel.Name {
				case "Identifier", "Fragment":
					return function, nil
				case "StringIndex", "StringIndexRune":
					return ast.NewIdent("int"), nil
				}
			}
			// A selector in call position may be a method rather than a field.
			return nil, nil
		default:
			if _, err := validator.inferExpressionType(current.Fun, scope); err != nil {
				return nil, err
			}
			return nil, nil
		}
	default:
		return nil, fmt.Errorf("unsupported Go expression %T", expression)
	}
}

func basicLiteralType(literal *ast.BasicLit) ast.Expr {
	switch literal.Kind {
	case token.STRING:
		return ast.NewIdent("string")
	case token.CHAR:
		return ast.NewIdent("rune")
	case token.INT:
		return ast.NewIdent("int")
	case token.FLOAT:
		return ast.NewIdent("float64")
	default:
		return nil
	}
}

func (validator *statementExpressionValidator) validateNilComparison(
	expression *ast.BinaryExpr,
	leftType ast.Expr,
	rightType ast.Expr,
) error {
	leftNil := isNilExpression(expression.X)
	rightNil := isNilExpression(expression.Y)
	if !leftNil && !rightNil {
		return nil
	}
	if expression.Op != token.EQL && expression.Op != token.NEQ {
		return fmt.Errorf("operator %s cannot be used with nil", expression.Op)
	}
	if leftNil && rightNil {
		return fmt.Errorf("nil cannot be compared with nil without a typed operand")
	}

	valueType := leftType
	if leftNil {
		valueType = rightType
	}
	supported, known := validator.typeSupportsNilComparison(valueType)
	if !known || supported {
		return nil
	}
	typeName, err := expressionString(valueType)
	if err != nil {
		typeName = "value"
	}
	return fmt.Errorf(
		"type %s cannot be compared with nil; use a pointer for nullable values or compare against an explicit value",
		typeName,
	)
}

func isNilExpression(expression ast.Expr) bool {
	switch current := expression.(type) {
	case *ast.Ident:
		return current.Name == "nil"
	case *ast.ParenExpr:
		return isNilExpression(current.X)
	default:
		return false
	}
}

func (validator *statementExpressionValidator) typeSupportsNilComparison(
	valueType ast.Expr,
) (supported bool, known bool) {
	underlying := validator.underlyingType(valueType, make(map[string]bool))
	switch current := underlying.(type) {
	case nil:
		return false, false
	case *ast.ParenExpr:
		return validator.typeSupportsNilComparison(current.X)
	case *ast.StarExpr, *ast.MapType, *ast.ChanType, *ast.FuncType, *ast.InterfaceType:
		return true, true
	case *ast.ArrayType:
		return current.Len == nil, true
	case *ast.StructType:
		return false, true
	case *ast.Ident:
		switch current.Name {
		case "any", "error":
			return true, true
		case "bool", "string",
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"byte", "rune",
			"float32", "float64",
			"complex64", "complex128":
			return false, true
		default:
			return false, false
		}
	default:
		// Qualified and instantiated named types may have a nil-able
		// underlying type that is not available in the local package AST.
		// Leave those expressions to the generated Go compiler.
		return false, false
	}
}

func (validator *statementExpressionValidator) structInfo(
	valueType ast.Expr,
	seen map[string]bool,
) *StructInfo {
	switch current := valueType.(type) {
	case *ast.ParenExpr:
		return validator.structInfo(current.X, seen)
	case *ast.StarExpr:
		return validator.structInfo(current.X, seen)
	case *ast.IndexExpr:
		return validator.structInfo(current.X, seen)
	case *ast.IndexListExpr:
		return validator.structInfo(current.X, seen)
	case *ast.Ident:
		if info := validator.goPackage.Structs[current.Name]; info != nil {
			return info
		}
		if seen[current.Name] {
			return nil
		}
		seen[current.Name] = true
		return validator.structInfo(validator.goPackage.Types[current.Name], seen)
	default:
		return nil
	}
}

func (validator *statementExpressionValidator) underlyingType(
	valueType ast.Expr,
	seen map[string]bool,
) ast.Expr {
	switch current := valueType.(type) {
	case *ast.ParenExpr:
		return validator.underlyingType(current.X, seen)
	case *ast.Ident:
		if seen[current.Name] {
			return current
		}
		underlying := validator.goPackage.Types[current.Name]
		if underlying == nil {
			return current
		}
		seen[current.Name] = true
		return validator.underlyingType(underlying, seen)
	default:
		return valueType
	}
}

func (validator *statementExpressionValidator) isKnownScalarType(valueType ast.Expr) bool {
	underlying := validator.underlyingType(valueType, make(map[string]bool))
	identifier, ok := underlying.(*ast.Ident)
	if !ok || identifier.Name == "any" {
		return false
	}
	return isBuiltinScalar(identifier.Name)
}

func (validator *statementExpressionValidator) indexedType(valueType ast.Expr) (ast.Expr, error) {
	underlying := validator.underlyingType(valueType, make(map[string]bool))
	switch current := underlying.(type) {
	case nil:
		return nil, nil
	case *ast.ArrayType:
		return current.Elt, nil
	case *ast.MapType:
		return current.Value, nil
	case *ast.Ident:
		if current.Name == "string" {
			return ast.NewIdent("byte"), nil
		}
	case *ast.SelectorExpr, *ast.InterfaceType:
		// The local AST does not contain enough information to validate external
		// or interface-backed values. Their root identifier is still checked.
		return nil, nil
	}
	typeName, _ := expressionString(valueType)
	return nil, fmt.Errorf("type %s does not support indexing", typeName)
}

func (validator *statementExpressionValidator) rangeTypes(
	valueType ast.Expr,
) (ast.Expr, ast.Expr, error) {
	underlying := validator.underlyingType(valueType, make(map[string]bool))
	switch current := underlying.(type) {
	case nil:
		return nil, nil, nil
	case *ast.ArrayType:
		return ast.NewIdent("int"), current.Elt, nil
	case *ast.MapType:
		return current.Key, current.Value, nil
	case *ast.Ident:
		if current.Name == "string" {
			return ast.NewIdent("int"), ast.NewIdent("rune"), nil
		}
		if current.Name == "any" {
			return nil, nil, fmt.Errorf(
				"interface-backed foreach requires runtime reflection and is not supported; use a concrete slice, array, map, or string",
			)
		}
	case *ast.InterfaceType:
		return nil, nil, fmt.Errorf(
			"interface-backed foreach requires runtime reflection and is not supported; use a concrete slice, array, map, or string",
		)
	case *ast.SelectorExpr:
		return nil, nil, nil
	}
	typeName, _ := expressionString(valueType)
	return nil, nil, fmt.Errorf(
		"foreach collection must be a slice, array, map, string, or externally defined iterable; got %s",
		typeName,
	)
}
