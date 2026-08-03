package parsing

import "strings"

const KEY_PREFIX = "org.apache.ibatis.parsing.PropertyParser."

const KEY_ENABLE_DEFAULT_VALUE = KEY_PREFIX + "enable-default-value"

const KEY_DEFAULT_VALUE_SEPARATOR = KEY_PREFIX + "default-value-separator"

const ENABLE_DEFAULT_VALUE = "false"

const DEFAULT_VALUE_SEPARATOR = ":"

type propertyParser struct{}

var PropertyParser propertyParser

func (propertyParser) Parse(text string, variables Properties) string {
	variableTokenHandler := newVariableTokenHandler(variables)
	genericTokenParser := NewGenericTokenParser("${", "}", variableTokenHandler)
	return genericTokenParser.Parse(text)
}

type variableTokenHandler struct {
	variables             Properties
	enableDefaultValue    bool
	defaultValueSeparator string
}

func newVariableTokenHandler(variables Properties) *variableTokenHandler {
	handler := &variableTokenHandler{
		variables: variables,
	}
	handler.enableDefaultValue = strings.EqualFold(
		handler.getPropertyValue(KEY_ENABLE_DEFAULT_VALUE, ENABLE_DEFAULT_VALUE),
		"true",
	)
	handler.defaultValueSeparator = handler.getPropertyValue(KEY_DEFAULT_VALUE_SEPARATOR, DEFAULT_VALUE_SEPARATOR)
	return handler
}

func (handler *variableTokenHandler) getPropertyValue(key, defaultValue string) string {
	if handler == nil || handler.variables == nil {
		return defaultValue
	}
	return handler.variables.GetProperty(key, defaultValue)
}

func (handler *variableTokenHandler) HandleToken(content string) *string {
	if handler == nil {
		return stringPointer("${" + content + "}")
	}

	if handler.variables != nil {
		key := content
		if handler.enableDefaultValue {
			separatorIndex := strings.Index(content, handler.defaultValueSeparator)
			var defaultValue *string
			if separatorIndex >= 0 {
				key = content[:separatorIndex]
				defaultValueText := content[separatorIndex+len(handler.defaultValueSeparator):]
				defaultValue = &defaultValueText
			}
			if defaultValue != nil {
				return stringPointer(handler.variables.GetProperty(key, *defaultValue))
			}
		}

		if handler.variables.ContainsKey(key) {
			return stringPointer(handler.variables.GetProperty(key))
		}
	}

	return stringPointer("${" + content + "}")
}

func stringPointer(value string) *string {
	return &value
}
