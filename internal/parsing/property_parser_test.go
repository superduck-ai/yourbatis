package parsing

import "testing"

func TestPropertyParserReplaceToVariableValue(t *testing.T) {
	props := NewProperties()
	props.Put(KEY_ENABLE_DEFAULT_VALUE, "true")
	props.Put("key", "value")
	props.Put("tableName", "members")
	props.Put("orderColumn", "member_id")
	props.Put("a:b", "c")

	assertPropertyParse(t, "value", "${key}", props)
	assertPropertyParse(t, "value", "${key:aaaa}", props)
	assertPropertyParse(t, "SELECT * FROM members ORDER BY member_id",
		"SELECT * FROM ${tableName:users} ORDER BY ${orderColumn:id}", props)

	props.Put(KEY_ENABLE_DEFAULT_VALUE, "false")
	assertPropertyParse(t, "c", "${a:b}", props)

	delete(props, KEY_ENABLE_DEFAULT_VALUE)
	assertPropertyParse(t, "c", "${a:b}", props)
}

func TestPropertyParserNotReplace(t *testing.T) {
	props := NewProperties()
	props.Put(KEY_ENABLE_DEFAULT_VALUE, "true")
	assertPropertyParse(t, "${key}", "${key}", props)
	assertPropertyParse(t, "${key}", "${key}", nil)

	props.Put(KEY_ENABLE_DEFAULT_VALUE, "false")
	assertPropertyParse(t, "${a:b}", "${a:b}", props)

	delete(props, KEY_ENABLE_DEFAULT_VALUE)
	assertPropertyParse(t, "${a:b}", "${a:b}", props)
}

func TestPropertyParserApplyDefaultValue(t *testing.T) {
	props := NewProperties()
	props.Put(KEY_ENABLE_DEFAULT_VALUE, "true")
	assertPropertyParse(t, "default", "${key:default}", props)
	assertPropertyParse(t, "SELECT * FROM users ORDER BY id",
		"SELECT * FROM ${tableName:users} ORDER BY ${orderColumn:id}", props)
	assertPropertyParse(t, "", "${key:}", props)
	assertPropertyParse(t, " ", "${key: }", props)
	assertPropertyParse(t, ":", "${key::}", props)
}

func TestPropertyParserApplyCustomSeparator(t *testing.T) {
	props := NewProperties()
	props.Put(KEY_ENABLE_DEFAULT_VALUE, "true")
	props.Put(KEY_DEFAULT_VALUE_SEPARATOR, "?:")
	assertPropertyParse(t, "default", "${key?:default}", props)
	assertPropertyParse(t,
		"SELECT * FROM prod.${tableName == null ? 'users' : tableName} ORDER BY ${orderColumn}",
		"SELECT * FROM ${schema?:prod}.${tableName == null ? 'users' : tableName} ORDER BY ${orderColumn}",
		props,
	)
	assertPropertyParse(t, "", "${key?:}", props)
	assertPropertyParse(t, " ", "${key?: }", props)
	assertPropertyParse(t, ":", "${key?::}", props)
}

func assertPropertyParse(t *testing.T, expected, text string, props Properties) {
	t.Helper()
	if actual := PropertyParser.Parse(text, props); actual != expected {
		t.Fatalf("parse(%q) = %q, want %q", text, actual, expected)
	}
}
