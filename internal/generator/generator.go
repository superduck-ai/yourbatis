package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/superduck-ai/yourbatis/internal/frontend/xmlmapper"
	"github.com/superduck-ai/yourbatis/internal/ir"
)

type Config struct {
	PackageDir    string
	Mapper        string
	SQLFile       string
	Output        string
	RuntimeImport string
	Dialect       string
}

type linkedMethod struct {
	Method         *MapperMethod
	Statement      *ir.Statement
	Mode           ir.ResultMode
	ReturnType     string
	ReturnExpr     ast.Expr
	Scanner        *scanPlan
	SQLParams      []GoValue
	Yield          *yieldPlan
	StatementName  string
	BuildName      string
	UsesDatabaseID bool
	Returning      bool
}

type yieldPlan struct {
	Parameter  GoValue
	ResultExpr ast.Expr
	ResultType string
}

type scanPlan struct {
	Name         string
	ResultType   string
	Pointer      bool
	Scalar       bool
	Destinations []string
}

func Generate(config Config) error {
	if config.PackageDir == "" {
		config.PackageDir = "."
	}
	absoluteDirectory, err := filepath.Abs(config.PackageDir)
	if err != nil {
		return err
	}
	sqlPath := config.SQLFile
	if !filepath.IsAbs(sqlPath) {
		sqlPath = filepath.Join(absoluteDirectory, sqlPath)
	}
	outputPath := config.Output
	if outputPath == "" {
		outputPath = snakeCase(config.Mapper) + ".sqlmap.gen.go"
	}
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(absoluteDirectory, outputPath)
	}
	if config.RuntimeImport == "" {
		config.RuntimeImport = "github.com/superduck-ai/yourbatis"
	}

	sqlFile, err := os.Open(sqlPath)
	if err != nil {
		return err
	}
	mapperIR, parseErr := xmlmapper.Parse(sqlFile, config.SQLFile)
	closeErr := sqlFile.Close()
	if parseErr != nil {
		return parseErr
	}
	if closeErr != nil {
		return closeErr
	}
	goPackage, err := LoadPackage(absoluteDirectory, config.Mapper, outputPath)
	if err != nil {
		return err
	}
	if mapperIR.Namespace != config.Mapper && !strings.HasSuffix(mapperIR.Namespace, "."+config.Mapper) {
		return fmt.Errorf(
			"%s: mapper namespace %q does not match interface %q",
			config.SQLFile,
			mapperIR.Namespace,
			config.Mapper,
		)
	}
	if err := validateTargetDialect(mapperIR, config.Dialect); err != nil {
		return err
	}

	linked, err := linkMapper(goPackage, mapperIR)
	if err != nil {
		return err
	}
	source, err := emitFile(goPackage, mapperIR, linked, config.RuntimeImport)
	if err != nil {
		return err
	}
	formatted, err := format.Source(source)
	if err != nil {
		return fmt.Errorf("format generated source: %w\n%s", err, source)
	}
	return os.WriteFile(outputPath, formatted, 0o644)
}

func validateTargetDialect(mapper *ir.Mapper, dialect string) error {
	dialect = strings.ToLower(strings.TrimSpace(dialect))
	if dialect == "" {
		return nil
	}
	switch dialect {
	case "postgres", "sqlite", "mysql":
	default:
		return fmt.Errorf(
			"unsupported target dialect %q; supported dialects are postgres, sqlite, and mysql",
			dialect,
		)
	}
	for _, statement := range mapper.Statements {
		switch {
		case statement.ResultMode == ir.ResultReturning && dialect == "mysql":
			return sourceError(
				statement.Source,
				"%s.%s uses result=returning, but target dialect mysql does not support DML returning",
				mapper.Namespace,
				statement.ID,
			)
		case statement.ResultMode == ir.ResultLastID && dialect == "postgres":
			return sourceError(
				statement.Source,
				"%s.%s uses result=lastid, but target dialect postgres does not support LastInsertId; use result=returning",
				mapper.Namespace,
				statement.ID,
			)
		}
	}
	return nil
}

func linkMapper(goPackage *GoPackage, mapper *ir.Mapper) ([]*linkedMethod, error) {
	if err := validateResultMaps(goPackage, mapper); err != nil {
		return nil, err
	}

	statements := make(map[string]*ir.Statement)
	for _, statement := range mapper.Statements {
		if _, duplicate := statements[statement.ID]; duplicate {
			return nil, sourceError(statement.Source, "duplicate statement %q", statement.ID)
		}
		statements[statement.ID] = statement
	}

	result := make([]*linkedMethod, 0, len(goPackage.Interface.Methods))
	for _, method := range goPackage.Interface.Methods {
		statement := statements[method.Name]
		if statement == nil {
			return nil, fmt.Errorf("mapper method %s.%s has no matching SQL statement", mapper.Namespace, method.Name)
		}
		delete(statements, method.Name)

		linked, err := linkMethod(goPackage, mapper, method, statement)
		if err != nil {
			return nil, err
		}
		result = append(result, linked)
	}
	if len(statements) > 0 {
		extra := make([]string, 0, len(statements))
		for name := range statements {
			extra = append(extra, name)
		}
		sort.Strings(extra)
		return nil, fmt.Errorf("SQL statements have no matching mapper methods: %s", strings.Join(extra, ", "))
	}
	return result, nil
}

func linkMethod(
	goPackage *GoPackage,
	mapper *ir.Mapper,
	method *MapperMethod,
	statement *ir.Statement,
) (*linkedMethod, error) {
	if len(method.Params) == 0 || !strings.HasSuffix(method.Params[0].Type, ".Context") {
		return nil, fmt.Errorf("%s.%s first parameter must be context.Context", mapper.Namespace, method.Name)
	}
	for _, parameter := range method.Params {
		if strings.HasPrefix(parameter.Type, "...") {
			return nil, fmt.Errorf("%s.%s: variadic parameters are not supported", mapper.Namespace, method.Name)
		}
		if !isUsableIdentifier(parameter.Name) {
			return nil, fmt.Errorf(
				"%s.%s: parameter name %q is not usable by generated code",
				mapper.Namespace,
				method.Name,
				parameter.Name,
			)
		}
		if parameter.Name == "yourbatis" {
			return nil, fmt.Errorf(
				"%s.%s: parameter name %q is reserved by the generated runtime import",
				mapper.Namespace,
				method.Name,
				parameter.Name,
			)
		}
		if parameter.Name == "_parameter" || parameter.Name == "_databaseId" {
			return nil, fmt.Errorf(
				"%s.%s: parameter name %q is reserved by the MyBatis compatibility context",
				mapper.Namespace,
				method.Name,
				parameter.Name,
			)
		}
	}
	if len(method.Results) == 0 || method.Results[len(method.Results)-1].Type != "error" {
		return nil, fmt.Errorf("%s.%s must return error as its final result", mapper.Namespace, method.Name)
	}
	yield, sqlParams, err := resolveYieldPlan(method, statement)
	if err != nil {
		return nil, sourceError(statement.Source, "%s.%s: %v", mapper.Namespace, method.Name, err)
	}
	if err := validateStatementExpressionsWithParams(goPackage, method, statement, sqlParams); err != nil {
		return nil, err
	}

	valueResults := method.Results[:len(method.Results)-1]
	var mode ir.ResultMode
	var returnExpr ast.Expr
	var returnType string
	if yield != nil {
		mode = ir.ResultEach
		returnExpr = yield.ResultExpr
		returnType = yield.ResultType
	} else {
		mode, returnExpr, returnType, err = resolveResultMode(method, statement, valueResults)
		if err != nil {
			return nil, sourceError(statement.Source, "%s.%s: %v", mapper.Namespace, method.Name, err)
		}
	}
	linked := &linkedMethod{
		Method:         method,
		Statement:      statement,
		Mode:           mode,
		ReturnExpr:     returnExpr,
		ReturnType:     returnType,
		SQLParams:      sqlParams,
		Yield:          yield,
		StatementName:  lowerFirst(mapper.Namespace) + method.Name + "Statement",
		BuildName:      "build" + mapper.Namespace + method.Name,
		UsesDatabaseID: nodesUseExpressionIdentifier(statement.Body, "_databaseId"),
		Returning:      statement.ResultMode == ir.ResultReturning,
	}
	linked.StatementName = sanitizeIdentifier(linked.StatementName)
	linked.BuildName = sanitizeIdentifier(linked.BuildName)

	switch mode {
	case ir.ResultOne, ir.ResultOptional, ir.ResultMany, ir.ResultEach, ir.ResultScalar:
		plan, err := buildScanPlan(goPackage, mapper, linked)
		if err != nil {
			return nil, err
		}
		linked.Scanner = plan
	}
	return linked, nil
}

func resolveYieldPlan(
	method *MapperMethod,
	statement *ir.Statement,
) (*yieldPlan, []GoValue, error) {
	sqlParams := append([]GoValue(nil), method.Params[1:]...)
	functionIndex := -1
	for index, parameter := range sqlParams {
		if _, ok := parameter.Expr.(*ast.FuncType); !ok {
			continue
		}
		if functionIndex >= 0 {
			return nil, nil, fmt.Errorf("yield queries require exactly one function parameter")
		}
		functionIndex = index
	}
	if functionIndex < 0 {
		return nil, sqlParams, nil
	}
	if functionIndex != len(sqlParams)-1 {
		return nil, nil, fmt.Errorf("yield callback must be the final mapper parameter")
	}
	if statement.Command != ir.Select {
		return nil, nil, fmt.Errorf("yield callbacks are only supported by select statements")
	}
	if len(method.Results) != 1 || method.Results[0].Type != "error" {
		return nil, nil, fmt.Errorf("yield select methods must return error only")
	}
	if statement.ResultMode != ir.ResultInfer {
		return nil, nil, fmt.Errorf("yield select methods infer their result mode; omit the result attribute")
	}

	parameter := sqlParams[functionIndex]
	function := parameter.Expr.(*ast.FuncType)
	resultExpr, err := singleYieldParameter(function.Params)
	if err != nil {
		return nil, nil, err
	}
	if err := validateYieldResults(function.Results); err != nil {
		return nil, nil, err
	}
	resultType, err := expressionString(resultExpr)
	if err != nil {
		return nil, nil, fmt.Errorf("print yield row type: %w", err)
	}
	return &yieldPlan{
		Parameter:  parameter,
		ResultExpr: resultExpr,
		ResultType: resultType,
	}, sqlParams[:functionIndex], nil
}

func singleYieldParameter(fields *ast.FieldList) (ast.Expr, error) {
	if fields == nil || len(fields.List) != 1 {
		return nil, fmt.Errorf("yield callback must have signature func(T) error")
	}
	field := fields.List[0]
	if len(field.Names) > 1 {
		return nil, fmt.Errorf("yield callback must have signature func(T) error")
	}
	if _, variadic := field.Type.(*ast.Ellipsis); variadic {
		return nil, fmt.Errorf("yield callback must have signature func(T) error")
	}
	return field.Type, nil
}

func validateYieldResults(fields *ast.FieldList) error {
	if fields == nil || len(fields.List) != 1 {
		return fmt.Errorf("yield callback must have signature func(T) error")
	}
	field := fields.List[0]
	if len(field.Names) > 1 {
		return fmt.Errorf("yield callback must have signature func(T) error")
	}
	result, ok := field.Type.(*ast.Ident)
	if !ok || result.Name != "error" {
		return fmt.Errorf("yield callback must have signature func(T) error")
	}
	return nil
}

func resolveResultMode(
	method *MapperMethod,
	statement *ir.Statement,
	results []GoValue,
) (ir.ResultMode, ast.Expr, string, error) {
	mode := statement.ResultMode
	if mode == ir.ResultInfer {
		switch statement.Command {
		case ir.Select:
			switch len(results) {
			case 1:
				if array, ok := results[0].Expr.(*ast.ArrayType); ok && array.Len == nil &&
					!isByteSlice(array) {
					mode = ir.ResultMany
				} else {
					mode = ir.ResultOne
				}
			case 2:
				if results[1].Type != "bool" {
					return "", nil, "", fmt.Errorf(
						"optional select methods must return (T, bool, error)",
					)
				}
				mode = ir.ResultOptional
			default:
				return "", nil, "", fmt.Errorf(
					"select methods must return (T, error), ([]T, error), or (T, bool, error)",
				)
			}
		case ir.Insert, ir.Update, ir.Delete:
			switch len(results) {
			case 0:
				mode = ir.ResultExec
			case 1:
				mode = ir.ResultRows
			default:
				return "", nil, "", fmt.Errorf("command methods must return error or one value and error")
			}
		}
	}

	switch mode {
	case ir.ResultExec:
		if len(results) != 0 {
			return "", nil, "", fmt.Errorf("result=exec requires an error-only method")
		}
		return mode, nil, "", nil
	case ir.ResultRows, ir.ResultLastID:
		if len(results) != 1 || results[0].Type != "int64" {
			return "", nil, "", fmt.Errorf("result=%s requires (int64, error)", mode)
		}
		return mode, results[0].Expr, results[0].Type, nil
	case ir.ResultReturning:
		if len(results) != 1 {
			return "", nil, "", fmt.Errorf("result=returning requires one value and error")
		}
		if array, ok := results[0].Expr.(*ast.ArrayType); ok && array.Len == nil &&
			!isByteSlice(array) {
			elementType, err := expressionString(array.Elt)
			return ir.ResultMany, array.Elt, elementType, err
		}
		return ir.ResultOne, results[0].Expr, results[0].Type, nil
	case ir.ResultMany:
		if len(results) != 1 {
			return "", nil, "", fmt.Errorf("result=many requires one slice result and error")
		}
		array, ok := results[0].Expr.(*ast.ArrayType)
		if !ok || array.Len != nil {
			return "", nil, "", fmt.Errorf("result=many requires a slice result")
		}
		elementType, err := expressionString(array.Elt)
		return mode, array.Elt, elementType, err
	case ir.ResultOptional:
		if len(results) != 2 || results[1].Type != "bool" {
			return "", nil, "", fmt.Errorf("result=optional requires (T, bool, error)")
		}
		return mode, results[0].Expr, results[0].Type, nil
	case ir.ResultOne, ir.ResultScalar:
		if len(results) != 1 {
			return "", nil, "", fmt.Errorf("result=%s requires one value and error", mode)
		}
		return mode, results[0].Expr, results[0].Type, nil
	default:
		return "", nil, "", fmt.Errorf("unsupported result mode %q", mode)
	}
}

func buildScanPlan(
	goPackage *GoPackage,
	mapper *ir.Mapper,
	linked *linkedMethod,
) (*scanPlan, error) {
	plan := &scanPlan{
		Name:       sanitizeIdentifier("scan" + mapper.Namespace + linked.Method.Name),
		ResultType: linked.ReturnType,
	}

	typeName, pointer := namedType(linked.ReturnExpr)
	plan.Pointer = pointer
	structInfo := structInfoForNamedType(goPackage, typeName)
	resultMap, err := lookupResultMap(mapper, linked.Statement.ResultMap)
	if err != nil {
		return nil, sourceError(linked.Statement.Source, "%v", err)
	}
	if resultMap != nil && typeName != "" && !declaredResultTypeMatches(resultMap.Type, linked.ReturnType) {
		return nil, sourceError(
			resultMap.Source,
			"resultMap %s maps %s, but %s.%s returns %s",
			resultMap.ID,
			resultMap.Type,
			mapper.Namespace,
			linked.Method.Name,
			linked.ReturnType,
		)
	}
	if linked.Statement.ResultType != "" && typeName != "" &&
		!declaredResultTypeMatches(linked.Statement.ResultType, linked.ReturnType) {
		return nil, sourceError(
			linked.Statement.Source,
			"resultType %s does not match mapper return type %s",
			linked.Statement.ResultType,
			linked.ReturnType,
		)
	}
	columns, projectionName, err := resultColumns(linked.Statement)
	if err != nil {
		return nil, err
	}
	if err := validateUniqueProjectionColumns(columns, linked.Statement, projectionName); err != nil {
		return nil, err
	}

	if isMapResult(goPackage, linked.ReturnExpr) {
		return nil, sourceError(
			linked.Statement.Source,
			"map return type %s is not supported; use a Go struct or scalar type",
			linked.ReturnType,
		)
	}
	if linked.Mode == ir.ResultScalar {
		if len(columns) != 1 {
			return nil, sourceError(
				linked.Statement.Source,
				"result=scalar requires exactly one %s column, found %d",
				projectionName,
				len(columns),
			)
		}
		plan.Scalar = true
		return plan, nil
	}
	if resultMap == nil && structInfo == nil {
		if len(columns) != 1 {
			return nil, sourceError(
				linked.Statement.Source,
				"scalar return type %s requires exactly one %s column, found %d; use a struct for multi-column rows",
				linked.ReturnType,
				projectionName,
				len(columns),
			)
		}
		plan.Scalar = true
		return plan, nil
	}
	if resultMap != nil && structInfo == nil && isKnownNonStructResult(goPackage, linked.ReturnExpr) {
		return nil, sourceError(
			resultMap.Source,
			"resultMap %s cannot map fields onto non-struct return type %s",
			resultMap.ID,
			linked.ReturnType,
		)
	}
	var properties map[string]string
	if resultMap != nil {
		properties = resultMapProperties(resultMap)
	} else {
		properties, err = structProperties(structInfo, linked.Statement.Source)
		if err != nil {
			return nil, err
		}
	}
	for _, column := range columns {
		property := properties[strings.ToLower(column)]
		if property == "" {
			return nil, sourceError(
				linked.Statement.Source,
				"column %q cannot be mapped to %s; add a db tag, alias, or resultMap",
				column,
				linked.ReturnType,
			)
		}
		destination := "&result." + property
		if _, err := goparser.ParseExpr(destination); err != nil {
			return nil, sourceError(linked.Statement.Source, "invalid result property %q: %v", property, err)
		}
		plan.Destinations = append(plan.Destinations, destination)
	}
	return plan, nil
}

func isByteSlice(array *ast.ArrayType) bool {
	if array == nil || array.Len != nil {
		return false
	}
	element, ok := array.Elt.(*ast.Ident)
	return ok && (element.Name == "byte" || element.Name == "uint8")
}

func lookupResultMap(mapper *ir.Mapper, reference string) (*ir.ResultMap, error) {
	if reference == "" {
		return nil, nil
	}
	if result := mapper.ResultMaps[reference]; result != nil {
		return result, nil
	}
	return nil, fmt.Errorf("resultMap %q not found", reference)
}

func namedType(expression ast.Expr) (string, bool) {
	pointer := false
	if star, ok := expression.(*ast.StarExpr); ok {
		pointer = true
		expression = star.X
	}
	switch current := expression.(type) {
	case *ast.Ident:
		return current.Name, pointer
	case *ast.SelectorExpr:
		return current.Sel.Name, pointer
	default:
		return "", pointer
	}
}

func emitFile(
	goPackage *GoPackage,
	mapper *ir.Mapper,
	methods []*linkedMethod,
	runtimeImport string,
) ([]byte, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "// Code generated by sqlmapgen; DO NOT EDIT.")
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "package %s\n\n", goPackage.Name)

	imports := make(map[string]GoImport)
	for _, linked := range methods {
		for alias, imported := range linked.Method.Imports {
			imports[alias] = imported
		}
	}
	imports["yourbatis"] = GoImport{Alias: "yourbatis", Path: runtimeImport}
	importNames := make([]string, 0, len(imports))
	for alias := range imports {
		importNames = append(importNames, alias)
	}
	sort.Slice(importNames, func(left, right int) bool {
		return imports[importNames[left]].Path < imports[importNames[right]].Path
	})
	fmt.Fprintln(&output, "import (")
	for _, alias := range importNames {
		imported := imports[alias]
		if imported.Alias != "" {
			fmt.Fprintf(&output, "\t%s %q\n", imported.Alias, imported.Path)
		} else {
			fmt.Fprintf(&output, "\t%q\n", imported.Path)
		}
	}
	fmt.Fprintln(&output, ")")
	fmt.Fprintln(&output)

	implementation := lowerFirst(goPackage.Interface.Name)
	fmt.Fprintf(&output, "type %s struct {\n\texecutor yourbatis.Executor\n}\n\n", implementation)
	fmt.Fprintf(&output, "var _ %s = (*%s)(nil)\n\n", goPackage.Interface.Name, implementation)
	fmt.Fprintf(
		&output,
		"func New%s(executor yourbatis.Executor) %s {\n\treturn &%s{executor: executor}\n}\n\n",
		goPackage.Interface.Name,
		goPackage.Interface.Name,
		implementation,
	)

	for _, linked := range methods {
		if err := emitStatement(&output, mapper, linked); err != nil {
			return nil, err
		}
		if linked.Scanner != nil {
			emitScanner(&output, linked.Scanner)
		}
		if err := emitBuildFunction(&output, linked); err != nil {
			return nil, err
		}
		emitMapperMethod(&output, implementation, linked)
	}
	return output.Bytes(), nil
}

func emitStatement(output *bytes.Buffer, mapper *ir.Mapper, linked *linkedMethod) error {
	kind := map[ir.Command]string{
		ir.Select: "yourbatis.StatementSelect",
		ir.Insert: "yourbatis.StatementInsert",
		ir.Update: "yourbatis.StatementUpdate",
		ir.Delete: "yourbatis.StatementDelete",
	}[linked.Statement.Command]
	fmt.Fprintf(output, "var %s = yourbatis.Statement{\n", linked.StatementName)
	fmt.Fprintf(output, "\tID: %q,\n", mapper.Namespace+"."+linked.Statement.ID)
	fmt.Fprintf(output, "\tSource: %q,\n", sourceLabel(linked.Statement.Source))
	fmt.Fprintf(output, "\tKind: %s,\n", kind)
	fmt.Fprintln(output, "}")
	fmt.Fprintln(output)
	return nil
}

func emitScanner(output *bytes.Buffer, plan *scanPlan) {
	fmt.Fprintf(output, "func %s(row yourbatis.Scanner) (%s, error) {\n", plan.Name, plan.ResultType)
	if plan.Pointer && !plan.Scalar {
		fmt.Fprintf(output, "\tresult := new(%s)\n", strings.TrimPrefix(plan.ResultType, "*"))
	} else {
		fmt.Fprintf(output, "\tvar result %s\n", plan.ResultType)
	}
	if plan.Scalar {
		fmt.Fprintln(output, "\terr := row.Scan(&result)")
	} else {
		fmt.Fprintln(output, "\terr := row.Scan(")
		for _, destination := range plan.Destinations {
			fmt.Fprintf(output, "\t\t%s,\n", destination)
		}
		fmt.Fprintln(output, "\t)")
	}
	fmt.Fprintln(output, "\treturn result, err")
	fmt.Fprintln(output, "}")
	fmt.Fprintln(output)
}

func emitBuildFunction(output *bytes.Buffer, linked *linkedMethod) error {
	reserved := collectGeneratedFunctionIdentifiers(linked.SQLParams, linked.Statement.Body)
	dialectName := reserveIdentifier(reserved, "dialect")
	builderName := reserveIdentifier(reserved, "builder")

	fmt.Fprintf(output, "func %s(%s yourbatis.Dialect", linked.BuildName, dialectName)
	if linked.UsesDatabaseID {
		fmt.Fprint(output, ", _databaseId string")
	}
	for _, parameter := range linked.SQLParams {
		fmt.Fprintf(output, ", %s %s", parameter.Name, parameter.Type)
	}
	fmt.Fprintln(output, ") yourbatis.BoundSQL {")
	fmt.Fprintf(output, "\t%s := yourbatis.NewBuilder(%s)\n", builderName, dialectName)
	state := &emitState{
		dialect:  dialectName,
		reserved: reserved,
	}
	if err := state.emitNodes(output, linked.Statement.Body, builderName, "\t"); err != nil {
		return err
	}
	fmt.Fprintf(output, "\treturn %s.Build()\n", builderName)
	fmt.Fprintln(output, "}")
	fmt.Fprintln(output)
	return nil
}

func emitMapperMethod(output *bytes.Buffer, implementation string, linked *linkedMethod) {
	method := linked.Method
	reserved := make(map[string]struct{}, len(method.Params))
	for _, parameter := range method.Params {
		reserved[parameter.Name] = struct{}{}
	}
	receiverName := reserveIdentifier(reserved, "mapper")
	boundName := reserveIdentifier(reserved, "__yourbatisBound")
	fmt.Fprintf(output, "func (%s *%s) %s%s {\n", receiverName, implementation, method.Name, method.Signature)
	fmt.Fprintf(output, "\t%s := %s(%s.executor.Dialect()", boundName, linked.BuildName, receiverName)
	if linked.UsesDatabaseID {
		fmt.Fprintf(output, ", yourbatis.ExecutorDatabaseID(%s.executor)", receiverName)
	}
	for _, parameter := range linked.SQLParams {
		fmt.Fprintf(output, ", %s", parameter.Name)
	}
	fmt.Fprintln(output, ")")
	contextName := method.Params[0].Name
	switch linked.Mode {
	case ir.ResultOne, ir.ResultScalar:
		queryFunction := "yourbatis.SelectOne"
		if linked.Returning {
			queryFunction = "yourbatis.QueryOne"
		}
		fmt.Fprintf(
			output,
			"\treturn %s(%s, %s.executor, %s, %s, %s)\n",
			queryFunction,
			contextName,
			receiverName,
			linked.StatementName,
			boundName,
			linked.Scanner.Name,
		)
	case ir.ResultOptional:
		fmt.Fprintf(
			output,
			"\treturn yourbatis.SelectOptional(%s, %s.executor, %s, %s, %s)\n",
			contextName,
			receiverName,
			linked.StatementName,
			boundName,
			linked.Scanner.Name,
		)
	case ir.ResultMany:
		queryFunction := "yourbatis.SelectMany"
		if linked.Returning {
			queryFunction = "yourbatis.QueryMany"
		}
		fmt.Fprintf(
			output,
			"\treturn %s(%s, %s.executor, %s, %s, %s)\n",
			queryFunction,
			contextName,
			receiverName,
			linked.StatementName,
			boundName,
			linked.Scanner.Name,
		)
	case ir.ResultEach:
		fmt.Fprintf(
			output,
			"\treturn yourbatis.SelectEach(%s, %s.executor, %s, %s, %s, %s)\n",
			contextName,
			receiverName,
			linked.StatementName,
			boundName,
			linked.Scanner.Name,
			linked.Yield.Parameter.Name,
		)
	case ir.ResultRows:
		fmt.Fprintf(
			output,
			"\treturn yourbatis.ExecuteRows(%s, %s.executor, %s, %s)\n",
			contextName,
			receiverName,
			linked.StatementName,
			boundName,
		)
	case ir.ResultLastID:
		fmt.Fprintf(
			output,
			"\treturn yourbatis.ExecuteLastInsertID(%s, %s.executor, %s, %s)\n",
			contextName,
			receiverName,
			linked.StatementName,
			boundName,
		)
	case ir.ResultExec:
		fmt.Fprintf(
			output,
			"\treturn yourbatis.Execute(%s, %s.executor, %s, %s)\n",
			contextName,
			receiverName,
			linked.StatementName,
			boundName,
		)
	}
	fmt.Fprintln(output, "}")
	fmt.Fprintln(output)
}

func nodesUseExpressionIdentifier(nodes []ir.Node, name string) bool {
	for _, node := range nodes {
		switch current := node.(type) {
		case *ir.TextExpression:
			if expressionUsesIdentifier(current.Expression, name) {
				return true
			}
		case *ir.Parameter:
			expression := current.GoPath
			if expression == "" {
				expression = current.Path
			}
			if expressionUsesIdentifier(expression, name) {
				return true
			}
		case *ir.If:
			if expressionUsesIdentifier(current.Test, name) ||
				nodesUseExpressionIdentifier(current.Body, name) {
				return true
			}
		case *ir.Choose:
			for _, when := range current.When {
				if expressionUsesIdentifier(when.Test, name) ||
					nodesUseExpressionIdentifier(when.Body, name) {
					return true
				}
			}
			if nodesUseExpressionIdentifier(current.Otherwise, name) {
				return true
			}
		case *ir.Foreach:
			if expressionUsesIdentifier(current.Collection, name) ||
				nodesUseExpressionIdentifier(current.Body, name) {
				return true
			}
		case *ir.Trim:
			if nodesUseExpressionIdentifier(current.Body, name) {
				return true
			}
		case *ir.Bind:
			if expressionUsesIdentifier(current.Expression, name) {
				return true
			}
		}
	}
	return false
}

func expressionUsesIdentifier(expression, name string) bool {
	normalized, err := normalizeQuotedStrings(expression)
	if err != nil {
		return false
	}
	for _, current := range tokenizeMyBatisExpression(normalized) {
		if current.kind == "ident" && current.text == name {
			return true
		}
	}
	return false
}

type emitState struct {
	counter  int
	dialect  string
	reserved map[string]struct{}
}

func (state *emitState) next(prefix string) string {
	for {
		state.counter++
		candidate := fmt.Sprintf("%s%d", prefix, state.counter)
		if _, exists := state.reserved[candidate]; exists {
			continue
		}
		state.reserved[candidate] = struct{}{}
		return candidate
	}
}

func (state *emitState) emitNodes(
	output *bytes.Buffer,
	nodes []ir.Node,
	builderName string,
	indent string,
) error {
	for _, node := range nodes {
		switch current := node.(type) {
		case *ir.SQLNodeBoundary:
			fmt.Fprintf(output, "%s%s.BeginSQLNode()\n", indent, builderName)
		case *ir.Text:
			fmt.Fprintf(output, "%s%s.Text(%s)\n", indent, builderName, strconv.Quote(current.Value))
		case *ir.TextExpression:
			switch current.Kind {
			case "signed":
				fmt.Fprintf(
					output,
					"%syourbatis.AppendSigned(%s, int64(%s))\n",
					indent,
					builderName,
					current.GoExpression,
				)
			case "unsigned":
				fmt.Fprintf(
					output,
					"%syourbatis.AppendUnsigned(%s, uint64(%s))\n",
					indent,
					builderName,
					current.GoExpression,
				)
			case "float":
				fmt.Fprintf(
					output,
					"%syourbatis.AppendFloat(%s, float64(%s))\n",
					indent,
					builderName,
					current.GoExpression,
				)
			case "bool":
				fmt.Fprintf(output, "%syourbatis.AppendBool(%s, %s)\n", indent, builderName, current.GoExpression)
			case "identifier":
				fmt.Fprintf(
					output,
					"%syourbatis.AppendIdentifier(%s, %s, %s)\n",
					indent,
					builderName,
					state.dialect,
					current.GoExpression,
				)
			case "fragment":
				fmt.Fprintf(output, "%s%s.Text(string(%s))\n", indent, builderName, current.GoExpression)
			default:
				return sourceError(
					current.Source,
					"text expression %q has unknown generated kind %q",
					current.Expression,
					current.Kind,
				)
			}
		case *ir.Parameter:
			goPath := current.GoPath
			if goPath == "" {
				goPath = current.Path
			}
			if err := validateGoExpression(goPath); err != nil {
				return sourceError(current.Source, "parameter %q is not a Go expression: %v", goPath, err)
			}
			if current.Sensitive {
				fmt.Fprintf(
					output,
					"%s%s.Arg(%q, %s, yourbatis.Sensitive())\n",
					indent,
					builderName,
					current.Path,
					goPath,
				)
			} else {
				fmt.Fprintf(output, "%s%s.Arg(%q, %s)\n", indent, builderName, current.Path, goPath)
			}
		case *ir.If:
			if current.GoTest == "" {
				return sourceError(current.Source, "if test %q was not compiled", current.Test)
			}
			fmt.Fprintf(output, "%sif %s {\n", indent, current.GoTest)
			if err := state.emitNodes(output, current.Body, builderName, indent+"\t"); err != nil {
				return err
			}
			fmt.Fprintf(output, "%s}\n", indent)
		case *ir.Choose:
			if len(current.When) == 0 {
				if current.Otherwise != nil {
					fmt.Fprintf(output, "%s{\n", indent)
					if err := state.emitNodes(output, current.Otherwise, builderName, indent+"\t"); err != nil {
						return err
					}
					fmt.Fprintf(output, "%s}\n", indent)
				}
				break
			}
			for index, when := range current.When {
				if when.GoTest == "" {
					return sourceError(when.Source, "when test %q was not compiled", when.Test)
				}
				if index == 0 {
					fmt.Fprintf(output, "%sif %s {\n", indent, when.GoTest)
				} else {
					fmt.Fprintf(output, "%s} else if %s {\n", indent, when.GoTest)
				}
				if err := state.emitNodes(output, when.Body, builderName, indent+"\t"); err != nil {
					return err
				}
			}
			if current.Otherwise != nil {
				fmt.Fprintf(output, "%s} else {\n", indent)
				if err := state.emitNodes(output, current.Otherwise, builderName, indent+"\t"); err != nil {
					return err
				}
			}
			fmt.Fprintf(output, "%s}\n", indent)
		case *ir.Foreach:
			if err := state.emitForeach(output, current, builderName, indent); err != nil {
				return err
			}
		case *ir.Trim:
			child := state.next("trim")
			fmt.Fprintf(output, "%s%s := yourbatis.NewBuilder(%s)\n", indent, child, state.dialect)
			if err := state.emitNodes(output, current.Body, child, indent); err != nil {
				return err
			}
			fmt.Fprintf(output, "%s%s.BeginSQLNode()\n", indent, builderName)
			fmt.Fprintf(
				output,
				"%s%s.AppendTrimmed(%s, %q, %#v, %q, %#v)\n",
				indent,
				builderName,
				child,
				current.Prefix,
				current.PrefixOverrides,
				current.Suffix,
				current.SuffixOverrides,
			)
		case *ir.Bind:
			if !token.IsIdentifier(current.Name) || token.Lookup(current.Name).IsKeyword() {
				return sourceError(current.Source, "bind name %q is not a Go identifier", current.Name)
			}
			if current.GoExpression == "" {
				return sourceError(current.Source, "bind expression %q was not compiled", current.Expression)
			}
			fmt.Fprintf(output, "%s%s := %s\n", indent, current.Name, current.GoExpression)
		default:
			return sourceError(node.Span(), "unsupported IR node %T", node)
		}
	}
	return nil
}

func (state *emitState) emitForeach(
	output *bytes.Buffer,
	foreach *ir.Foreach,
	parent string,
	indent string,
) error {
	if foreach.GoCollection == "" {
		return sourceError(foreach.Source, "foreach collection %q was not compiled", foreach.Collection)
	}
	if !token.IsIdentifier(foreach.Item) || token.Lookup(foreach.Item).IsKeyword() {
		return sourceError(foreach.Source, "foreach item %q is not a Go identifier", foreach.Item)
	}
	if !token.IsIdentifier(foreach.Index) || token.Lookup(foreach.Index).IsKeyword() {
		return sourceError(foreach.Source, "foreach index %q is not a Go identifier", foreach.Index)
	}
	if foreach.Index == foreach.Item {
		return sourceError(foreach.Source, "foreach index and item must differ")
	}

	collectionBuilder := state.next("foreach")
	itemBuilder := state.next("itemSQL")
	fmt.Fprintf(output, "%s%s := yourbatis.NewBuilder(%s)\n", indent, collectionBuilder, state.dialect)
	fmt.Fprintf(output, "%sfor %s, %s := range %s {\n", indent, foreach.Index, foreach.Item, foreach.GoCollection)
	fmt.Fprintf(output, "%s\t_ = %s\n", indent, foreach.Index)
	fmt.Fprintf(output, "%s\t_ = %s\n", indent, foreach.Item)
	fmt.Fprintf(output, "%s\t%s := yourbatis.NewBuilder(%s)\n", indent, itemBuilder, state.dialect)
	if err := state.emitNodes(output, foreach.Body, itemBuilder, indent+"\t"); err != nil {
		return err
	}
	fmt.Fprintf(
		output,
		"%s\t%s.AppendSeparated(%s, %q)\n",
		indent,
		collectionBuilder,
		itemBuilder,
		foreach.Separator,
	)
	fmt.Fprintf(output, "%s}\n", indent)
	fmt.Fprintf(output, "%sif %s.HasIterations() {\n", indent, collectionBuilder)
	if foreach.OpenSet {
		fmt.Fprintf(output, "%s\t%s.BeginSQLNode()\n", indent, parent)
		fmt.Fprintf(output, "%s\t%s.Text(%q)\n", indent, parent, foreach.Open)
	}
	fmt.Fprintf(output, "%s\t%s.Append(%s)\n", indent, parent, collectionBuilder)
	if foreach.CloseSet {
		fmt.Fprintf(output, "%s\t%s.BeginSQLNode()\n", indent, parent)
		fmt.Fprintf(output, "%s\t%s.Text(%q)\n", indent, parent, foreach.Close)
	}
	fmt.Fprintf(output, "%s}\n", indent)
	return nil
}

func validateGoExpression(expression string) error {
	_, err := goparser.ParseExpr(expression)
	return err
}

func expressionString(expression ast.Expr) (string, error) {
	fileSet := token.NewFileSet()
	return printNode(fileSet, expression)
}

func sourceLabel(source ir.Span) string {
	if source.Line > 0 {
		return fmt.Sprintf("%s:%d", source.File, source.Line)
	}
	return source.File
}

func sourceError(source ir.Span, format string, arguments ...any) error {
	return fmt.Errorf(
		"%s:%d:%d: %s",
		source.File,
		source.Line,
		source.Column,
		fmt.Sprintf(format, arguments...),
	)
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func sanitizeIdentifier(value string) string {
	var result strings.Builder
	for index, current := range value {
		if current == '_' || current >= 'a' && current <= 'z' || current >= 'A' && current <= 'Z' || index > 0 && current >= '0' && current <= '9' {
			result.WriteRune(current)
		}
	}
	return result.String()
}

func collectGeneratedFunctionIdentifiers(
	parameters []GoValue,
	nodes []ir.Node,
) map[string]struct{} {
	result := make(map[string]struct{})
	for _, parameter := range parameters {
		result[parameter.Name] = struct{}{}
	}
	collectNodeIdentifiers(result, nodes)
	return result
}

func collectNodeIdentifiers(result map[string]struct{}, nodes []ir.Node) {
	for _, node := range nodes {
		switch current := node.(type) {
		case *ir.If:
			collectNodeIdentifiers(result, current.Body)
		case *ir.Choose:
			for _, when := range current.When {
				collectNodeIdentifiers(result, when.Body)
			}
			collectNodeIdentifiers(result, current.Otherwise)
		case *ir.Foreach:
			result[current.Index] = struct{}{}
			result[current.Item] = struct{}{}
			collectNodeIdentifiers(result, current.Body)
		case *ir.Trim:
			collectNodeIdentifiers(result, current.Body)
		case *ir.Bind:
			result[current.Name] = struct{}{}
		}
	}
}

func reserveIdentifier(reserved map[string]struct{}, preferred string) string {
	if _, exists := reserved[preferred]; !exists {
		reserved[preferred] = struct{}{}
		return preferred
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s%d", preferred, suffix)
		if _, exists := reserved[candidate]; exists {
			continue
		}
		reserved[candidate] = struct{}{}
		return candidate
	}
}
