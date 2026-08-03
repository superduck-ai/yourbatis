package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

type GoPackage struct {
	Name        string
	Dir         string
	Interface   *MapperInterface
	Structs     map[string]*StructInfo
	Types       map[string]ast.Expr
	SourceFiles []string
	typeInfo    packageTypeInfo
}

type MapperInterface struct {
	Name    string
	Methods []*MapperMethod
}

type MapperMethod struct {
	Name       string
	Signature  string
	Params     []GoValue
	Results    []GoValue
	Imports    map[string]GoImport
	ResultExpr ast.Expr
}

type GoValue struct {
	Name string
	Type string
	Expr ast.Expr
}

type GoImport struct {
	Alias string
	Path  string
}

type StructInfo struct {
	Name        string
	Fields      []StructField
	HasEmbedded bool
}

type StructField struct {
	Name   string
	Column string
	Type   ast.Expr
}

func LoadPackage(directory, interfaceName, outputPath string) (*GoPackage, error) {
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, directory, func(info os.FileInfo) bool {
		name := info.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return false
		}
		if outputPath != "" && filepath.Base(outputPath) == name {
			return false
		}
		return !strings.HasSuffix(name, ".sqlmap.gen.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if len(packages) != 1 {
		return nil, fmt.Errorf("%s: expected one Go package, found %d", directory, len(packages))
	}

	var parsedPackage *ast.Package
	for _, candidate := range packages {
		parsedPackage = candidate
	}
	result := &GoPackage{
		Name:    parsedPackage.Name,
		Dir:     directory,
		Structs: make(map[string]*StructInfo),
		Types:   make(map[string]ast.Expr),
	}

	fileNames := make([]string, 0, len(parsedPackage.Files))
	for name := range parsedPackage.Files {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)
	result.SourceFiles = append(result.SourceFiles, fileNames...)

	for _, name := range fileNames {
		file := parsedPackage.Files[name]
		result.collectStructs(file)
		if result.Interface == nil {
			mapper, err := findMapperInterface(fileSet, file, interfaceName, directory)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if mapper != nil {
				result.Interface = mapper
			}
		}
	}
	if result.Interface == nil {
		return nil, fmt.Errorf("%s: interface %s not found", directory, interfaceName)
	}
	return result, nil
}

func findMapperInterface(
	fileSet *token.FileSet,
	file *ast.File,
	interfaceName string,
	directory string,
) (*MapperInterface, error) {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != interfaceName {
				continue
			}
			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				return nil, fmt.Errorf("%s is not an interface", interfaceName)
			}
			imports, err := importsByAlias(directory, file)
			if err != nil {
				return nil, fmt.Errorf("resolve imports for %s: %w", interfaceName, err)
			}
			mapper := &MapperInterface{Name: interfaceName}
			for _, field := range interfaceType.Methods.List {
				if len(field.Names) != 1 {
					return nil, fmt.Errorf("%s embeds an interface; embedded interfaces are not supported yet", interfaceName)
				}
				function, ok := field.Type.(*ast.FuncType)
				if !ok {
					return nil, fmt.Errorf("%s.%s is not a method", interfaceName, field.Names[0].Name)
				}
				method, err := newMapperMethod(fileSet, field.Names[0].Name, function, imports)
				if err != nil {
					return nil, err
				}
				mapper.Methods = append(mapper.Methods, method)
			}
			return mapper, nil
		}
	}
	return nil, nil
}

func newMapperMethod(
	fileSet *token.FileSet,
	name string,
	function *ast.FuncType,
	imports map[string]GoImport,
) (*MapperMethod, error) {
	signature, err := printNode(fileSet, function)
	if err != nil {
		return nil, err
	}
	signature = strings.TrimPrefix(signature, "func")
	method := &MapperMethod{
		Name:      name,
		Signature: signature,
		Imports:   make(map[string]GoImport),
	}
	method.Params, err = flattenFields(fileSet, function.Params, true)
	if err != nil {
		return nil, fmt.Errorf("%s params: %w", name, err)
	}
	method.Results, err = flattenFields(fileSet, function.Results, false)
	if err != nil {
		return nil, fmt.Errorf("%s results: %w", name, err)
	}
	if len(method.Params) == 0 {
		return nil, fmt.Errorf("%s must accept context.Context as its first parameter", name)
	}
	for alias := range referencedQualifiers(function) {
		importSpec, ok := imports[alias]
		if !ok {
			return nil, fmt.Errorf("%s references unknown import qualifier %s", name, alias)
		}
		method.Imports[alias] = importSpec
	}
	return method, nil
}

func flattenFields(fileSet *token.FileSet, fields *ast.FieldList, requireNames bool) ([]GoValue, error) {
	if fields == nil {
		return nil, nil
	}
	result := make([]GoValue, 0)
	for _, field := range fields.List {
		typeName, err := printNode(fileSet, field.Type)
		if err != nil {
			return nil, err
		}
		if len(field.Names) == 0 {
			if requireNames {
				return nil, fmt.Errorf("all mapper parameters must be named")
			}
			result = append(result, GoValue{Type: typeName, Expr: field.Type})
			continue
		}
		for _, name := range field.Names {
			result = append(result, GoValue{
				Name: name.Name,
				Type: typeName,
				Expr: field.Type,
			})
		}
	}
	return result, nil
}

func printNode(fileSet *token.FileSet, node any) (string, error) {
	var result strings.Builder
	if err := printer.Fprint(&result, fileSet, node); err != nil {
		return "", err
	}
	return result.String(), nil
}

func importsByAlias(directory string, file *ast.File) (map[string]GoImport, error) {
	result := make(map[string]GoImport)
	defaultImportPaths := make([]string, 0)
	for _, specification := range file.Imports {
		pathValue, err := strconv.Unquote(specification.Path.Value)
		if err != nil {
			continue
		}
		if specification.Name == nil {
			defaultImportPaths = append(defaultImportPaths, pathValue)
			continue
		}
		alias := specification.Name.Name
		if alias == "_" || alias == "." {
			continue
		}
		result[alias] = GoImport{Alias: alias, Path: pathValue}
	}
	if len(defaultImportPaths) == 0 {
		return result, nil
	}

	sort.Strings(defaultImportPaths)
	loaded, err := packages.Load(&packages.Config{
		Dir:  directory,
		Mode: packages.NeedName,
	}, defaultImportPaths...)
	if err != nil {
		return nil, fmt.Errorf("load imported packages: %w", err)
	}
	packageNames := make(map[string]string, len(loaded))
	for _, importedPackage := range loaded {
		if importedPackage == nil {
			continue
		}
		if len(importedPackage.Errors) > 0 {
			return nil, fmt.Errorf("load import %q: %s", importedPackage.PkgPath, importedPackage.Errors[0])
		}
		if importedPackage.PkgPath != "" && importedPackage.Name != "" {
			packageNames[importedPackage.PkgPath] = importedPackage.Name
		}
	}
	for _, pathValue := range defaultImportPaths {
		alias := packageNames[pathValue]
		if alias == "" {
			return nil, fmt.Errorf("cannot determine package name for import %q", pathValue)
		}
		result[alias] = GoImport{Path: pathValue}
	}
	return result, nil
}

func referencedQualifiers(node ast.Node) map[string]struct{} {
	result := make(map[string]struct{})
	ast.Inspect(node, func(current ast.Node) bool {
		selector, ok := current.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if identifier, ok := selector.X.(*ast.Ident); ok {
			result[identifier.Name] = struct{}{}
		}
		return true
	})
	return result
}

func (goPackage *GoPackage) collectStructs(file *ast.File) {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			goPackage.Types[typeSpec.Name.Name] = typeSpec.Type
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			info := &StructInfo{Name: typeSpec.Name.Name}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					info.HasEmbedded = true
					continue
				}
				for _, fieldIdentifier := range field.Names {
					if !fieldIdentifier.IsExported() {
						continue
					}
					fieldName := fieldIdentifier.Name
					column := snakeCase(fieldName)
					if field.Tag != nil {
						if tagValue, err := strconv.Unquote(field.Tag.Value); err == nil {
							tag := reflect.StructTag(tagValue)
							if value := firstTagValue(tag, "sqlmap", "db"); value != "" {
								if value == "-" {
									continue
								}
								column = value
							}
						}
					}
					info.Fields = append(info.Fields, StructField{
						Name:   fieldName,
						Column: column,
						Type:   field.Type,
					})
				}
			}
			goPackage.Structs[info.Name] = info
		}
	}
}

func firstTagValue(tag reflect.StructTag, names ...string) string {
	for _, name := range names {
		if value, ok := tag.Lookup(name); ok {
			return strings.Split(value, ",")[0]
		}
	}
	return ""
}

func snakeCase(value string) string {
	var result strings.Builder
	runes := []rune(value)
	for index, current := range runes {
		if unicode.IsUpper(current) {
			if index > 0 {
				previous := runes[index-1]
				nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
				if unicode.IsLower(previous) || unicode.IsDigit(previous) || nextIsLower {
					result.WriteByte('_')
				}
			}
			result.WriteRune(unicode.ToLower(current))
			continue
		}
		result.WriteRune(current)
	}
	return result.String()
}
