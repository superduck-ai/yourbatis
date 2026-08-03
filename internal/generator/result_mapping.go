package generator

import (
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func validateResultMaps(goPackage *GoPackage, mapper *ir.Mapper) error {
	names := make([]string, 0, len(mapper.ResultMaps))
	for name := range mapper.ResultMaps {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		resultMap := mapper.ResultMaps[name]
		if len(resultMap.Fields) == 0 {
			return sourceError(resultMap.Source, "resultMap %s must declare at least one field", resultMap.ID)
		}

		var structInfo *StructInfo
		var structTypeName string
		if !strings.Contains(resultMap.Type, ".") {
			typeName := simpleTypeName(resultMap.Type)
			structTypeName = typeName
			structInfo = structInfoForNamedType(goPackage, typeName)
			if structInfo == nil {
				if _, exists := goPackage.Types[typeName]; !exists {
					return sourceError(
						resultMap.Source,
						"resultMap %s references unknown local Go type %s",
						resultMap.ID,
						resultMap.Type,
					)
				}
			}
		}
		columns := make(map[string]ir.ResultField)
		properties := make(map[string]ir.ResultField)
		for _, field := range resultMap.Fields {
			columnKey := strings.ToLower(field.Column)
			if previous, duplicate := columns[columnKey]; duplicate {
				return sourceError(
					field.Source,
					"resultMap %s maps column %q more than once (first declared at %s)",
					resultMap.ID,
					field.Column,
					sourceLabel(previous.Source),
				)
			}
			columns[columnKey] = field

			if previous, duplicate := properties[field.Property]; duplicate {
				return sourceError(
					field.Source,
					"resultMap %s maps property %q more than once (first declared at %s)",
					resultMap.ID,
					field.Property,
					sourceLabel(previous.Source),
				)
			}
			properties[field.Property] = field

			if strings.Contains(field.Property, ".") {
				return sourceError(
					field.Source,
					"resultMap %s property %q is nested; only direct Go struct fields are supported",
					resultMap.ID,
					field.Property,
				)
			}
			if structInfo != nil {
				switch lookupStructField(goPackage, structTypeName, structInfo, field.Property) {
				case structFieldMissing:
					return sourceError(
						field.Source,
						"resultMap %s property %q is not an exported field of %s",
						resultMap.ID,
						field.Property,
						structInfo.Name,
					)
				case structFieldIndirect:
					return sourceError(
						field.Source,
						"resultMap %s property %q is promoted through an embedded pointer in %s; the generated zero-value result cannot address it safely",
						resultMap.ID,
						field.Property,
						structInfo.Name,
					)
				}
			}
		}
	}
	return nil
}

type structFieldLookup uint8

const (
	structFieldMissing structFieldLookup = iota
	structFieldFound
	structFieldIndirect
	structFieldUnknown
)

func lookupStructField(
	goPackage *GoPackage,
	typeName string,
	info *StructInfo,
	name string,
) structFieldLookup {
	for _, field := range info.Fields {
		if field.Name == name {
			return structFieldFound
		}
	}
	if !info.HasEmbedded {
		return structFieldMissing
	}

	typePackage, complete := goPackage.loadTypePackage()
	if typePackage == nil {
		return structFieldUnknown
	}
	typeObject, ok := typePackage.Scope().Lookup(typeName).(*types.TypeName)
	if !ok {
		return structFieldUnknown
	}
	object, _, indirect := types.LookupFieldOrMethod(
		typeObject.Type(),
		true,
		typePackage,
		name,
	)
	field, ok := object.(*types.Var)
	if ok && field.IsField() && field.Exported() {
		if indirect {
			return structFieldIndirect
		}
		return structFieldFound
	}
	if complete {
		return structFieldMissing
	}
	return structFieldUnknown
}

func structInfoForNamedType(goPackage *GoPackage, name string) *StructInfo {
	visited := make(map[string]bool)
	for name != "" && !visited[name] {
		visited[name] = true
		if info := goPackage.Structs[name]; info != nil {
			return info
		}
		expression := goPackage.Types[name]
		identifier, ok := expression.(*ast.Ident)
		if !ok {
			return nil
		}
		name = identifier.Name
	}
	return nil
}

func resultMapProperties(resultMap *ir.ResultMap) map[string]string {
	properties := make(map[string]string, len(resultMap.Fields))
	for _, field := range resultMap.Fields {
		properties[strings.ToLower(field.Column)] = field.Property
	}
	return properties
}

func structProperties(info *StructInfo, source ir.Span) (map[string]string, error) {
	properties := make(map[string]string, len(info.Fields))
	for _, field := range info.Fields {
		columnKey := strings.ToLower(field.Column)
		if previous := properties[columnKey]; previous != "" {
			return nil, sourceError(
				source,
				"type %s maps column %q to both %s and %s; make db/sqlmap tags unique",
				info.Name,
				field.Column,
				previous,
				field.Name,
			)
		}
		properties[columnKey] = field.Name
	}
	return properties, nil
}

func validateUniqueProjectionColumns(
	columns []string,
	statement *ir.Statement,
	projectionName string,
) error {
	seen := make(map[string]string, len(columns))
	for _, column := range columns {
		key := strings.ToLower(column)
		if previous := seen[key]; previous != "" {
			return sourceError(
				statement.Source,
				"%s projection exposes column %q more than once; use unique aliases",
				projectionName,
				previous,
			)
		}
		seen[key] = column
	}
	return nil
}

func declaredResultTypeMatches(declared, actual string) bool {
	declared = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(declared), "*"))
	actual = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(actual), "*"))
	if strings.Contains(declared, ".") {
		return declared == actual
	}
	return simpleTypeName(actual) == declared
}

func simpleTypeName(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "*"))
	if dot := strings.LastIndex(value, "."); dot >= 0 {
		return value[dot+1:]
	}
	return value
}

func isKnownNonStructResult(goPackage *GoPackage, expression ast.Expr) bool {
	for {
		switch current := expression.(type) {
		case *ast.StarExpr:
			expression = current.X
		case *ast.ParenExpr:
			expression = current.X
		default:
			goto resolved
		}
	}

resolved:
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		// Imported types are validated by the Go compiler because their
		// declarations are outside the package loaded by sqlmapgen.
		return false
	}
	if _, isStruct := goPackage.Structs[identifier.Name]; isStruct {
		return false
	}
	if _, localType := goPackage.Types[identifier.Name]; localType {
		return true
	}
	return isBuiltinScalar(identifier.Name)
}

func isMapResult(goPackage *GoPackage, expression ast.Expr) bool {
	visited := make(map[string]bool)
	for expression != nil {
		switch current := expression.(type) {
		case *ast.StarExpr:
			expression = current.X
		case *ast.ParenExpr:
			expression = current.X
		case *ast.MapType:
			return true
		case *ast.Ident:
			if visited[current.Name] {
				return false
			}
			visited[current.Name] = true
			expression = goPackage.Types[current.Name]
		default:
			return false
		}
	}
	return false
}

func isBuiltinScalar(name string) bool {
	switch name {
	case "any",
		"bool",
		"string",
		"byte",
		"rune",
		"int",
		"int8",
		"int16",
		"int32",
		"int64",
		"uint",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"uintptr",
		"float32",
		"float64",
		"complex64",
		"complex128":
		return true
	default:
		return false
	}
}
