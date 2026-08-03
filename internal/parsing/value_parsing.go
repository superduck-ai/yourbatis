package parsing

import (
	"fmt"
	"reflect"
	"strconv"
)

func mustParseShort(value string) int16 {
	parsed, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		panic(err)
	}
	return int16(parsed)
}

func mustParseInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func mustParseInt64(value string) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(err)
	}
	return parsed
}

func mustParseFloat32(value string) float32 {
	parsed, err := strconv.ParseFloat(TrimJavaWhitespace(value), 32)
	if err != nil {
		panic(err)
	}
	return float32(parsed)
}

func mustParseFloat64(value string) float64 {
	parsed, err := strconv.ParseFloat(TrimJavaWhitespace(value), 64)
	if err != nil {
		panic(err)
	}
	return parsed
}

func normalizeEnumType(enumType reflect.Type) reflect.Type {
	for enumType != nil && enumType.Kind() == reflect.Pointer {
		enumType = enumType.Elem()
	}
	return enumType
}

func convertStringToEnumType(value string, enumType reflect.Type) any {
	enumType = normalizeEnumType(enumType)
	if enumType == nil {
		return nil
	}

	enumValue := reflect.New(enumType).Elem()
	switch enumType.Kind() {
	case reflect.String:
		enumValue.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		enumValue.SetInt(mustParseInt64(value))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			panic(err)
		}
		enumValue.SetUint(parsed)
	default:
		panic(fmt.Errorf("type %s is not a supported enum attribute type", enumType.String()))
	}
	return enumValue.Interface()
}
