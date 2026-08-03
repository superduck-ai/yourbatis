package parsing

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type enumTest string

const (
	enumTestYES enumTest = "YES"
	enumTestNO  enumTest = "NO"
)

func TestXPathParserFromString(t *testing.T) {
	parser := mustLoadParser(t)
	testEvalMethod(t, parser)
}

func TestXPathParserFromReader(t *testing.T) {
	xmlText := mustLoadNodeletXML(t)
	parser, err := NewXPathParserFromReader(strings.NewReader(xmlText), XPathParserConfig{Resource: "nodelet_test.xml"})
	if err != nil {
		t.Fatalf("NewXPathParserFromReader failed: %v", err)
	}
	testEvalMethod(t, parser)
}

func TestXPathParserShouldMatchJavaReaderAndInputStreamEncodingSemantics(t *testing.T) {
	latin1XML := `<?xml version="1.0" encoding="ISO-8859-1"?><root>é</root>`

	stringParser, err := NewXPathParser(latin1XML)
	if err != nil {
		t.Fatalf("NewXPathParser with decoded ISO declaration failed: %v", err)
	}
	if actual := stringParser.EvalString("/root"); actual != "é" {
		t.Fatalf("string parser body = %q, want %q", actual, "é")
	}

	readerParser, err := NewXPathParserFromReader(strings.NewReader(latin1XML))
	if err != nil {
		t.Fatalf("NewXPathParserFromReader with decoded ISO declaration failed: %v", err)
	}
	if actual := readerParser.EvalString("/root"); actual != "é" {
		t.Fatalf("reader parser body = %q, want %q", actual, "é")
	}

	inputStreamParser, err := NewXPathParserFromInputStream(strings.NewReader(encodeISO88591ForTest(t, latin1XML)))
	if err != nil {
		t.Fatalf("NewXPathParserFromInputStream with ISO-8859-1 bytes failed: %v", err)
	}
	if actual := inputStreamParser.EvalString("/root"); actual != "é" {
		t.Fatalf("ISO input stream parser body = %q, want %q", actual, "é")
	}

	utf16XML := `<?xml version="1.0" encoding="UTF-16"?><root>雪</root>`
	utf16Parser, err := NewXPathParserFromInputStream(strings.NewReader(encodeUTF16BEForTest(utf16XML)))
	if err != nil {
		t.Fatalf("NewXPathParserFromInputStream with UTF-16 bytes failed: %v", err)
	}
	if actual := utf16Parser.EvalString("/root"); actual != "雪" {
		t.Fatalf("UTF-16 input stream parser body = %q, want %q", actual, "雪")
	}
}

func TestXPathParserInputStreamShouldDecodeCommonIANAEncodings(t *testing.T) {
	testCases := []struct {
		name     string
		charset  string
		encoding encoding.Encoding
		body     string
	}{
		{
			name:     "Windows-1252",
			charset:  "windows-1252",
			encoding: charmap.Windows1252,
			body:     "“€”",
		},
		{
			name:     "Shift_JIS",
			charset:  "Shift_JIS",
			encoding: japanese.ShiftJIS,
			body:     "日本語",
		},
		{
			name:     "GBK",
			charset:  "GBK",
			encoding: simplifiedchinese.GBK,
			body:     "中文",
		},
		{
			name:     "GB18030",
			charset:  "GB18030",
			encoding: simplifiedchinese.GB18030,
			body:     "中文𠀀",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			xmlText := `<?xml version="1.0" encoding="` + testCase.charset + `"?><root>` + testCase.body + `</root>`
			encoded, err := testCase.encoding.NewEncoder().String(xmlText)
			if err != nil {
				t.Fatalf("encoding fixture as %s failed: %v", testCase.charset, err)
			}

			parser, err := NewXPathParserFromInputStream(strings.NewReader(encoded))
			if err != nil {
				t.Fatalf("NewXPathParserFromInputStream with %s bytes failed: %v", testCase.charset, err)
			}
			if actual := parser.EvalString("/root"); actual != testCase.body {
				t.Fatalf("%s input stream parser body = %q, want %q", testCase.charset, actual, testCase.body)
			}
		})
	}
}

func TestXPathParserInputStreamShouldBootstrapEBCDICEncoding(t *testing.T) {
	for _, charset := range []string{"CP037", "ebcdic-cp-us"} {
		t.Run(charset, func(t *testing.T) {
			xmlText := `<?xml version="1.0" encoding="` + charset + `"?><root>price ¢1</root>`
			encoded, err := charmap.CodePage037.NewEncoder().String(xmlText)
			if err != nil {
				t.Fatalf("encoding CP037 fixture failed: %v", err)
			}
			if prefix := []byte(encoded[:4]); !reflect.DeepEqual(prefix, []byte{0x4c, 0x6f, 0xa7, 0x94}) {
				t.Fatalf("EBCDIC XML magic = % X", prefix)
			}

			parser, err := NewXPathParserFromInputStream(strings.NewReader(encoded))
			if err != nil {
				t.Fatalf("NewXPathParserFromInputStream with %s bytes failed: %v", charset, err)
			}
			if actual := parser.EvalString("/root"); actual != "price ¢1" {
				t.Fatalf("%s input stream parser body = %q, want %q", charset, actual, "price ¢1")
			}
		})
	}
}

func TestXPathParserInputStreamShouldRequireEBCDICEncodingDeclaration(t *testing.T) {
	xmlText := `<?xml version="1.0"?><root>ok</root>`
	encoded, err := charmap.CodePage037.NewEncoder().String(xmlText)
	if err != nil {
		t.Fatalf("encoding CP037 fixture failed: %v", err)
	}

	_, err = NewXPathParserFromInputStream(strings.NewReader(encoded))
	if err == nil {
		t.Fatal("EBCDIC input without an encoding declaration should fail")
	}
	if !strings.Contains(err.Error(), "encoding declaration is required for EBCDIC input") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestXPathParserInputStreamShouldDetectUTF32ByteOrder(t *testing.T) {
	testCases := []struct {
		name      string
		charset   string
		byteOrder binary.ByteOrder
		withBOM   bool
	}{
		{
			name:      "big endian with BOM",
			charset:   "UTF-32",
			byteOrder: binary.BigEndian,
			withBOM:   true,
		},
		{
			name:      "little endian with BOM",
			charset:   "UTF-32",
			byteOrder: binary.LittleEndian,
			withBOM:   true,
		},
		{
			name:      "big endian without BOM",
			charset:   "UTF-32BE",
			byteOrder: binary.BigEndian,
		},
		{
			name:      "little endian without BOM",
			charset:   "UTF-32LE",
			byteOrder: binary.LittleEndian,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			const body = "雪𠀀"
			xmlText := `<?xml version="1.0" encoding="` + testCase.charset + `"?><root>` + body + `</root>`
			encoded := encodeUTF32ForTest(xmlText, testCase.byteOrder, testCase.withBOM)

			parser, err := NewXPathParserFromInputStream(strings.NewReader(encoded))
			if err != nil {
				t.Fatalf("NewXPathParserFromInputStream failed: %v", err)
			}
			if actual := parser.EvalString("/root"); actual != body {
				t.Fatalf("UTF-32 input stream parser body = %q, want %q", actual, body)
			}
		})
	}
}

func TestXPathParserInputStreamShouldRejectConflictingUTF32Declaration(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="UTF-32BE"?><root>ok</root>`
	encoded := encodeUTF32ForTest(xmlText, binary.LittleEndian, true)

	_, err := NewXPathParserFromInputStream(strings.NewReader(encoded))
	if err == nil {
		t.Fatal("conflicting UTF-32 BOM and encoding declaration should fail")
	}
	if !strings.Contains(err.Error(), `encoding declaration "UTF-32BE" conflicts`) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestXPathParserInputStreamShouldRejectMalformedUTF32(t *testing.T) {
	t.Run("partial code unit", func(t *testing.T) {
		encoded := []byte(encodeUTF32ForTest(
			`<?xml version="1.0" encoding="UTF-32"?><root>ok</root>`,
			binary.BigEndian,
			true,
		))
		encoded = append(encoded, 0x00)

		_, err := NewXPathParserFromInputStream(strings.NewReader(string(encoded)))
		if err == nil {
			t.Fatal("partial UTF-32 code unit should fail")
		}
		if !strings.Contains(err.Error(), "is not a multiple of 4") {
			t.Fatalf("error = %q", err.Error())
		}
	})

	t.Run("surrogate code point", func(t *testing.T) {
		prefix := []byte(encodeUTF32ForTest(
			`<?xml version="1.0" encoding="UTF-32"?><root>`,
			binary.BigEndian,
			true,
		))
		suffix := []byte(encodeUTF32ForTest(`</root>`, binary.BigEndian, false))
		encoded := append(prefix, 0x00, 0x00, 0xd8, 0x00)
		encoded = append(encoded, suffix...)

		_, err := NewXPathParserFromInputStream(strings.NewReader(string(encoded)))
		if err == nil {
			t.Fatal("UTF-32 surrogate code point should fail")
		}
		if !strings.Contains(err.Error(), "invalid code point U+D800") {
			t.Fatalf("error = %q", err.Error())
		}
	})
}

func TestXPathParserInputStreamShouldClassifyCDATAAfterLatin1TextUsingDecodedOffsets(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="ISO-8859-1"?><root>é<![CDATA[snow]]>after</root>`
	parser, err := NewXPathParserFromInputStream(strings.NewReader(encodeISO88591ForTest(t, xmlText)))
	if err != nil {
		t.Fatalf("NewXPathParserFromInputStream failed: %v", err)
	}

	root := parser.EvalNode("/root")
	if root == nil {
		t.Fatal("root node should exist")
	}
	children := root.GetNode().GetChildNodes()
	if len(children) != 3 {
		t.Fatalf("child node count = %d, want 3", len(children))
	}
	if children[0].GetNodeType() != TEXT_NODE || children[0].GetNodeValue() != "é" {
		t.Fatalf("first child = (%d, %q), want TEXT_NODE %q", children[0].GetNodeType(), children[0].GetNodeValue(), "é")
	}
	if children[1].GetNodeType() != CDATA_SECTION_NODE || children[1].GetNodeValue() != "snow" {
		t.Fatalf(
			"second child = (%d, %q), want CDATA_SECTION_NODE %q",
			children[1].GetNodeType(),
			children[1].GetNodeValue(),
			"snow",
		)
	}
	if children[2].GetNodeType() != TEXT_NODE || children[2].GetNodeValue() != "after" {
		t.Fatalf(
			"third child = (%d, %q), want TEXT_NODE %q",
			children[2].GetNodeType(),
			children[2].GetNodeValue(),
			"after",
		)
	}
}

func TestXPathParserInputStreamShouldMatchJavaBOMAndMalformedUTF16Behavior(t *testing.T) {
	t.Run("UTF-8 BOM does not override a non-UTF-8 declaration", func(t *testing.T) {
		xmlText := `<?xml version="1.0" encoding="ISO-8859-1"?><root>é</root>`
		encoded := append([]byte{0xef, 0xbb, 0xbf}, []byte(encodeISO88591ForTest(t, xmlText))...)
		parser, err := NewXPathParserFromInputStream(strings.NewReader(string(encoded)))
		if err != nil {
			t.Fatalf("NewXPathParserFromInputStream failed: %v", err)
		}
		if actual := parser.EvalString("/root"); actual != "é" {
			t.Fatalf("root body = %q, want %q", actual, "é")
		}
	})

	t.Run("UTF-16 BOM rejects a conflicting declaration", func(t *testing.T) {
		xmlText := `<?xml version="1.0" encoding="ISO-8859-1"?><root>ok</root>`
		_, err := NewXPathParserFromInputStream(strings.NewReader(encodeUTF16BEForTest(xmlText)))
		if err == nil {
			t.Fatal("conflicting UTF-16 BOM and encoding declaration should fail")
		}
		if !strings.Contains(err.Error(), `encoding declaration "ISO-8859-1" conflicts`) {
			t.Fatalf("error = %q", err.Error())
		}
	})

	t.Run("isolated UTF-16 surrogate is malformed", func(t *testing.T) {
		prefix := []byte(encodeUTF16BEForTest(`<?xml version="1.0" encoding="UTF-16"?><root>`))
		suffix := []byte(encodeUTF16BEForTest(`</root>`))[2:]
		encoded := append(prefix, 0xd8, 0x00)
		encoded = append(encoded, suffix...)

		_, err := NewXPathParserFromInputStream(strings.NewReader(string(encoded)))
		if err == nil {
			t.Fatal("isolated UTF-16 surrogate should fail")
		}
		if !strings.Contains(err.Error(), "high surrogate 0xD800") {
			t.Fatalf("error = %q", err.Error())
		}
	})
}

func TestXPathParserFromDocument(t *testing.T) {
	xmlText := mustLoadNodeletXML(t)
	document, err := ParseDocument(strings.NewReader(xmlText), "nodelet_test.xml")
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}
	parser, err := NewXPathParserFromDocument(document, XPathParserConfig{Resource: "nodelet_test.xml"})
	if err != nil {
		t.Fatalf("NewXPathParserFromDocument failed: %v", err)
	}
	testEvalMethod(t, parser)
}

func encodeISO88591ForTest(t *testing.T, value string) string {
	t.Helper()

	encoded := make([]byte, 0, len(value))
	for _, current := range value {
		if current > 0xff {
			t.Fatalf("rune %U cannot be encoded as ISO-8859-1", current)
		}
		encoded = append(encoded, byte(current))
	}
	return string(encoded)
}

func encodeUTF16BEForTest(value string) string {
	codeUnits := utf16.Encode([]rune(value))
	encoded := make([]byte, 2+len(codeUnits)*2)
	encoded[0] = 0xfe
	encoded[1] = 0xff
	for index, codeUnit := range codeUnits {
		binary.BigEndian.PutUint16(encoded[2+index*2:], codeUnit)
	}
	return string(encoded)
}

func encodeUTF32ForTest(value string, byteOrder binary.ByteOrder, withBOM bool) string {
	offset := 0
	if withBOM {
		offset = 4
	}
	encoded := make([]byte, offset+len([]rune(value))*4)
	if withBOM {
		byteOrder.PutUint32(encoded, 0x0000feff)
	}
	for index, current := range []rune(value) {
		byteOrder.PutUint32(encoded[offset+index*4:], uint32(current))
	}
	return string(encoded)
}

func TestXPathParserShouldApplyUpdatedVariablesToEvalStringAndNewXNodesLikeMyBatis(t *testing.T) {
	variables := NewProperties()
	variables.Put("statementId", "selectUsers")
	variables.Put("tableName", "users")

	parser := mustNewXPathParser(
		t,
		`<mapper><sql id="${statementId}">select * from ${tableName}</sql></mapper>`,
		XPathParserConfig{
			Resource:  "xpath_set_variables.xml",
			Variables: variables,
		},
	)

	originalNode := parser.EvalNode("/mapper/sql")
	if originalNode == nil {
		t.Fatal("original sql node should not be nil")
	}
	if actual := parser.EvalString("/mapper/sql/@id"); actual != "selectUsers" {
		t.Fatalf("initial eval string = %q, want %q", actual, "selectUsers")
	}
	if actual := originalNode.GetStringAttribute("id"); actual != "selectUsers" {
		t.Fatalf("original node id = %q, want %q", actual, "selectUsers")
	}
	if actual := originalNode.GetStringBody(""); actual != "select * from users" {
		t.Fatalf("original node body = %q, want %q", actual, "select * from users")
	}

	updatedVariables := NewProperties()
	updatedVariables.Put("statementId", "selectAuthors")
	updatedVariables.Put("tableName", "authors")
	parser.SetVariables(updatedVariables)

	if actual := parser.EvalString("/mapper/sql/@id"); actual != "selectAuthors" {
		t.Fatalf("updated eval string = %q, want %q", actual, "selectAuthors")
	}
	if actual := originalNode.GetStringAttribute("id"); actual != "selectUsers" {
		t.Fatalf("original node id after SetVariables = %q, want %q", actual, "selectUsers")
	}
	if actual := originalNode.GetStringBody(""); actual != "select * from users" {
		t.Fatalf("original node body after SetVariables = %q, want %q", actual, "select * from users")
	}

	updatedNode := parser.EvalNode("/mapper/sql")
	if updatedNode == nil {
		t.Fatal("updated sql node should not be nil")
	}
	if actual := updatedNode.GetStringAttribute("id"); actual != "selectAuthors" {
		t.Fatalf("updated node id = %q, want %q", actual, "selectAuthors")
	}
	if actual := updatedNode.GetStringBody(""); actual != "select * from authors" {
		t.Fatalf("updated node body = %q, want %q", actual, "select * from authors")
	}
}

func TestXPathParserShouldExposeCDATASectionNodesLikeMyBatisDOM(t *testing.T) {
	parser := mustNewXPathParser(t, `<script>before<![CDATA[ and #{id} ]]>after</script>`, XPathParserConfig{
		Resource: "cdata_script.xml",
	})

	script := parser.EvalNode("/script")
	if script == nil {
		t.Fatal("script node should not be nil")
	}

	children := script.GetNode().GetChildNodes()
	if len(children) != 3 {
		t.Fatalf("child node count = %d, want 3", len(children))
	}
	if children[0].GetNodeType() != TEXT_NODE {
		t.Fatalf("first child type = %d, want %d", children[0].GetNodeType(), TEXT_NODE)
	}
	if children[1].GetNodeType() != CDATA_SECTION_NODE {
		t.Fatalf("second child type = %d, want %d", children[1].GetNodeType(), CDATA_SECTION_NODE)
	}
	if children[2].GetNodeType() != TEXT_NODE {
		t.Fatalf("third child type = %d, want %d", children[2].GetNodeType(), TEXT_NODE)
	}

	cdataNode := script.NewXNode(children[1])
	if cdataNode.GetName() != "#cdata-section" {
		t.Fatalf("cdata node name = %q, want %q", cdataNode.GetName(), "#cdata-section")
	}
	if cdataNode.GetStringBody("") != " and #{id} " {
		t.Fatalf("cdata body = %q, want %q", cdataNode.GetStringBody(""), " and #{id} ")
	}
	if cdataNode.GetSourceXML() != "<![CDATA[ and #{id} ]]>" {
		t.Fatalf("cdata source xml = %q", cdataNode.GetSourceXML())
	}
	if cdataNode.GetPosition().Source != "cdata_script.xml" {
		t.Fatalf("cdata source = %q, want %q", cdataNode.GetPosition().Source, "cdata_script.xml")
	}
}

func TestXPathParserShouldMatchQualifiedNamesByLocalPartLikeMyBatisDOM(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<m:mapper
		  xmlns:m="http://mybatis.org/schema/mybatis-mapper"
		  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
		  xsi:schemaLocation="http://mybatis.org/schema/mybatis-mapper https://mybatis.org/schema/mybatis-mapper.xsd">
		  <m:select id="selectOne">select 1</m:select>
		</m:mapper>
	`, XPathParserConfig{Resource: "qualified_names_mapper.xml"})

	mapper := parser.EvalNode("/mapper")
	if mapper == nil {
		t.Fatal("/mapper should match a prefixed root element by local part like MyBatis")
	}
	if mapper.GetName() != "m:mapper" {
		t.Fatalf("mapper name = %q, want %q", mapper.GetName(), "m:mapper")
	}

	if parser.EvalNode("/m:mapper") != nil {
		t.Fatal("/m:mapper should not match a prefixed root element in namespace-unaware XPath")
	}

	if actual := parser.EvalString("/mapper/@schemaLocation"); actual != "http://mybatis.org/schema/mybatis-mapper https://mybatis.org/schema/mybatis-mapper.xsd" {
		t.Fatalf("schemaLocation = %q", actual)
	}
	if actual := parser.EvalString("/mapper/@xsi:schemaLocation"); actual != "" {
		t.Fatalf("xsi:schemaLocation = %q, want empty", actual)
	}
	if actual := parser.EvalString("/mapper/@xmlns:m"); actual != "" {
		t.Fatalf("xmlns:m = %q, want empty", actual)
	}

	selectNode := parser.EvalNode("/mapper/select")
	if selectNode == nil {
		t.Fatal("/mapper/select should match the prefixed child element by local part")
	}
	if selectNode.GetName() != "m:select" {
		t.Fatalf("select name = %q, want %q", selectNode.GetName(), "m:select")
	}
	if parser.EvalNode("/m:mapper/m:select") != nil {
		t.Fatal("/m:mapper/m:select should not match prefixed path steps in namespace-unaware XPath")
	}
	if actual := parser.EvalString("name(/*)"); actual != "m:mapper" {
		t.Fatalf("name(/*) = %q, want %q", actual, "m:mapper")
	}
}

func TestXPathParserShouldMatchMultiplePrefixedElementsByLocalPartLikeMyBatisDOM(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <a:item xmlns:a="urn:a">A</a:item>
		  <b:item xmlns:b="urn:b">B</b:item>
		</root>
	`, XPathParserConfig{Resource: "qualified_names_multiple_prefixes.xml"})

	items := parser.EvalNodes("/root/item")
	if len(items) != 2 {
		t.Fatalf("item count = %d, want 2", len(items))
	}
	if items[0].GetName() != "a:item" {
		t.Fatalf("first item name = %q, want %q", items[0].GetName(), "a:item")
	}
	if items[1].GetName() != "b:item" {
		t.Fatalf("second item name = %q, want %q", items[1].GetName(), "b:item")
	}

	if actual := parser.EvalString("/root/item"); actual != "A" {
		t.Fatalf("EvalString(/root/item) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count(/root/item)"); actual != 2 {
		t.Fatalf("count(/root/item) = %v, want 2", actual)
	}
	if actual := parser.EvalString("name(/root/item[1])"); actual != "a:item" {
		t.Fatalf("name(/root/item[1]) = %q, want %q", actual, "a:item")
	}
	if actual := parser.EvalString("name(/root/item[2])"); actual != "b:item" {
		t.Fatalf("name(/root/item[2]) = %q, want %q", actual, "b:item")
	}
	if actual := parser.EvalString("name(/root/item[3])"); actual != "" {
		t.Fatalf("name(/root/item[3]) = %q, want empty", actual)
	}
	secondItem := parser.EvalNode("/root/item[2]")
	if secondItem == nil {
		t.Fatal("/root/item[2] should match the second item")
	}
	if secondItem.GetName() != "b:item" {
		t.Fatalf("second item node name = %q, want %q", secondItem.GetName(), "b:item")
	}
	if len(parser.EvalNodes("/root/a:item")) != 0 {
		t.Fatal("/root/a:item should not match prefixed path steps in namespace-unaware XPath")
	}
}

func TestXPathParserShouldSupportDescendantSelfParentAndAttributeWildcardLikeMyBatisDOM(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root attr1="A" attr2="B">
		  <parent>
		    <child id="1">before<![CDATA[cdata]]><leaf />after</child>
		    <child id="2">two</child>
		  </parent>
		</root>
	`, XPathParserConfig{Resource: "xpath_next_slice_axes.xml"})

	if actual := parser.EvalDouble("count(/root/parent//leaf)"); actual != 1 {
		t.Fatalf("count(/root/parent//leaf) = %v, want 1", actual)
	}
	if actual := parser.EvalDouble("count(//leaf)"); actual != 1 {
		t.Fatalf("count(//leaf) = %v, want 1", actual)
	}
	leaf := parser.EvalNode("/root/parent//leaf")
	if leaf == nil {
		t.Fatal("/root/parent//leaf should match the nested leaf element")
	}
	if leaf.GetName() != "leaf" {
		t.Fatalf("leaf name = %q, want %q", leaf.GetName(), "leaf")
	}

	if actual := parser.EvalString("/root/parent/child/."); actual != "beforecdataafter" {
		t.Fatalf("EvalString(/root/parent/child/.) = %q, want %q", actual, "beforecdataafter")
	}
	if actual := parser.EvalDouble("count(/root/parent/child/.)"); actual != 2 {
		t.Fatalf("count(/root/parent/child/.) = %v, want 2", actual)
	}
	firstChild := parser.EvalNode("/root/parent/child/.")
	if firstChild == nil {
		t.Fatal("/root/parent/child/. should resolve to the child element itself")
	}
	if actual := firstChild.GetStringAttribute("id"); actual != "1" {
		t.Fatalf("first child id = %q, want %q", actual, "1")
	}

	if actual := parser.EvalString("/root/parent/child/../child[2]"); actual != "two" {
		t.Fatalf("EvalString(/root/parent/child/../child[2]) = %q, want %q", actual, "two")
	}
	if actual := parser.EvalDouble("count(/root/parent/child/../child[2])"); actual != 1 {
		t.Fatalf("count(/root/parent/child/../child[2]) = %v, want 1", actual)
	}
	secondChild := parser.EvalNode("/root/parent/child/../child[2]")
	if secondChild == nil {
		t.Fatal("/root/parent/child/../child[2] should match the second child element")
	}
	if actual := secondChild.GetStringAttribute("id"); actual != "2" {
		t.Fatalf("second child id = %q, want %q", actual, "2")
	}

	if actual := parser.EvalString("/root/@*"); actual != "A" {
		t.Fatalf("EvalString(/root/@*) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count(/root/@*)"); actual != 2 {
		t.Fatalf("count(/root/@*) = %v, want 2", actual)
	}
	attributes := parser.EvalNodes("/root/@*")
	if len(attributes) != 2 {
		t.Fatalf("/root/@* returned %d nodes, want 2", len(attributes))
	}
	if attributes[0].GetName() != "attr1" || attributes[0].GetStringBody("") != "A" {
		t.Fatalf("first attribute = (%q, %q), want (%q, %q)", attributes[0].GetName(), attributes[0].GetStringBody(""), "attr1", "A")
	}
	if attributes[1].GetName() != "attr2" || attributes[1].GetStringBody("") != "B" {
		t.Fatalf("second attribute = (%q, %q), want (%q, %q)", attributes[1].GetName(), attributes[1].GetStringBody(""), "attr2", "B")
	}

	if actual := parser.EvalString("name(/root/@*[1])"); actual != "attr1" {
		t.Fatalf("name(/root/@*[1]) = %q, want %q", actual, "attr1")
	}
	if actual := parser.EvalString("name(/root/@*[1]/..)"); actual != "root" {
		t.Fatalf("name(/root/@*[1]/..) = %q, want %q", actual, "root")
	}
	if actual := parser.EvalString("string(/root/parent/child[1])"); actual != "beforecdataafter" {
		t.Fatalf("string(/root/parent/child[1]) = %q, want %q", actual, "beforecdataafter")
	}
}

func TestXPathParserShouldSupportPositionAndLastPredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item>A</item><item>B</item><item>C</item></root>`,
		XPathParserConfig{Resource: "xpath_position_last_predicates.xml"},
	)

	if actual := parser.EvalString("/root/item[position()=2]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[position()=2]) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[position()=2])"); actual != 1 {
		t.Fatalf("count(/root/item[position()=2]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[last()]"); actual != "C" {
		t.Fatalf("EvalString(/root/item[last()]) = %q, want %q", actual, "C")
	}
	if actual := parser.EvalDouble("count(/root/item[last()])"); actual != 1 {
		t.Fatalf("count(/root/item[last()]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[position() = last()]"); actual != "C" {
		t.Fatalf("EvalString(/root/item[position() = last()]) = %q, want %q", actual, "C")
	}
	if actual := parser.EvalString("/root/item[position()=4]"); actual != "" {
		t.Fatalf("EvalString(/root/item[position()=4]) = %q, want empty", actual)
	}
}

func TestXPathParserShouldSupportAttributePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root xmlns:x="urn:x">
		  <item x:id="1" flag="yes">A</item>
		  <item x:id="2">B</item>
		  <item>C</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_attribute_predicates.xml"})

	if actual := parser.EvalString("/root/item[@id='2']"); actual != "B" {
		t.Fatalf("EvalString(/root/item[@id='2']) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[@id])"); actual != 2 {
		t.Fatalf("count(/root/item[@id]) = %v, want 2", actual)
	}
	if actual := parser.EvalDouble("count(/root/item[@*])"); actual != 2 {
		t.Fatalf("count(/root/item[@*]) = %v, want 2", actual)
	}
	if actual := parser.EvalDouble("count(/root/item[@missing])"); actual != 0 {
		t.Fatalf("count(/root/item[@missing]) = %v, want 0", actual)
	}
	if actual := parser.EvalString("/root/item[@id='2']/@id"); actual != "2" {
		t.Fatalf("EvalString(/root/item[@id='2']/@id) = %q, want %q", actual, "2")
	}
	if actual := parser.EvalDouble("count(/root/item[@id!='2'])"); actual != 1 {
		t.Fatalf("count(/root/item[@id!='2']) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[@id!='2']"); actual != "A" {
		t.Fatalf("EvalString(/root/item[@id!='2']) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count(/root/item[@id>1])"); actual != 1 {
		t.Fatalf("count(/root/item[@id>1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[@id>1]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[@id>1]) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[@id<2])"); actual != 1 {
		t.Fatalf("count(/root/item[@id<2]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[@id<2]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[@id<2]) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count(/root/item[@x:id])"); actual != 0 {
		t.Fatalf("count(/root/item[@x:id]) = %v, want 0", actual)
	}
}

func TestXPathParserShouldSupportStringValuePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item>A</item><item>B</item><item>before<![CDATA[cdata]]>after</item></root>`,
		XPathParserConfig{Resource: "xpath_string_value_predicates.xml"},
	)

	if actual := parser.EvalString("/root/item[.='B']"); actual != "B" {
		t.Fatalf("EvalString(/root/item[.='B']) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[.='B'])"); actual != 1 {
		t.Fatalf("count(/root/item[.='B']) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[.='beforecdataafter']"); actual != "beforecdataafter" {
		t.Fatalf("EvalString(/root/item[.='beforecdataafter']) = %q, want %q", actual, "beforecdataafter")
	}
	if actual := parser.EvalString("/root/item[.='missing']"); actual != "" {
		t.Fatalf("EvalString(/root/item[.='missing']) = %q, want empty", actual)
	}
}

func TestXPathParserShouldSupportTextPredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>before<![CDATA[cdata]]>after</item>
		  <split>before<![CDATA[cdata]]><leaf />after</split>
		</root>
	`, XPathParserConfig{Resource: "xpath_text_predicates.xml"})

	if actual := parser.EvalString("/root/item[text()='B']"); actual != "B" {
		t.Fatalf("EvalString(/root/item[text()='B']) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[text()='B'])"); actual != 1 {
		t.Fatalf("count(/root/item[text()='B']) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[text()='beforecdataafter']"); actual != "beforecdataafter" {
		t.Fatalf("EvalString(/root/item[text()='beforecdataafter']) = %q, want %q", actual, "beforecdataafter")
	}
	if actual := parser.EvalString("/root/split[text()='beforecdata']"); actual != "beforecdataafter" {
		t.Fatalf("EvalString(/root/split[text()='beforecdata']) = %q, want %q", actual, "beforecdataafter")
	}
	if actual := parser.EvalString("/root/split[text()='after']"); actual != "beforecdataafter" {
		t.Fatalf("EvalString(/root/split[text()='after']) = %q, want %q", actual, "beforecdataafter")
	}
	if actual := parser.EvalString("/root/split[text()='beforecdataafter']"); actual != "" {
		t.Fatalf("EvalString(/root/split[text()='beforecdataafter']) = %q, want empty", actual)
	}
	if actual := parser.EvalString("/root/split['after' = text()]"); actual != "beforecdataafter" {
		t.Fatalf("EvalString(/root/split['after' = text()]) = %q, want %q", actual, "beforecdataafter")
	}
}

func TestXPathParserShouldSupportBooleanPredicateExpressionsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item>B</item>
		  <empty />
		  <split>before<![CDATA[cdata]]><leaf />after</split>
		</root>
	`, XPathParserConfig{Resource: "xpath_boolean_predicate_expressions.xml"})

	if actual := parser.EvalDouble("count(/root/*[text()])"); actual != 3 {
		t.Fatalf("count(/root/*[text()]) = %v, want 3", actual)
	}
	if actual := parser.EvalDouble("count(/root/*[not(@id)])"); actual != 3 {
		t.Fatalf("count(/root/*[not(@id)]) = %v, want 3", actual)
	}
	if actual := parser.EvalString("/root/*[not(@id)][1]"); actual != "B" {
		t.Fatalf("EvalString(/root/*[not(@id)][1]) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/*[boolean(@id)])"); actual != 1 {
		t.Fatalf("count(/root/*[boolean(@id)]) = %v, want 1", actual)
	}
}

func TestXPathParserShouldCollapseLogicalTextRunsForTextAndNodeStepsLikeMyBatisDOM(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root attr1="A" attr2="B">
		  <parent>
		    <child id="1">before<![CDATA[cdata]]><leaf />after</child>
		    <child id="2">two</child>
		  </parent>
		</root>
	`, XPathParserConfig{Resource: "xpath_next_slice_text_runs.xml"})

	if actual := parser.EvalString("/root/parent/child/text()"); actual != "beforecdata" {
		t.Fatalf("EvalString(/root/parent/child/text()) = %q, want %q", actual, "beforecdata")
	}
	if actual := parser.EvalDouble("count(/root/parent/child/text())"); actual != 3 {
		t.Fatalf("count(/root/parent/child/text()) = %v, want 3", actual)
	}
	if actual := parser.EvalString("/root/parent/child/node()"); actual != "beforecdata" {
		t.Fatalf("EvalString(/root/parent/child/node()) = %q, want %q", actual, "beforecdata")
	}
	if actual := parser.EvalDouble("count(/root/parent/child/node())"); actual != 4 {
		t.Fatalf("count(/root/parent/child/node()) = %v, want 4", actual)
	}

	textNodes := parser.EvalNodes("/root/parent/child/text()")
	if len(textNodes) != 3 {
		t.Fatalf("text() node count = %d, want 3", len(textNodes))
	}
	if textNodes[0].GetName() != "#text" || textNodes[0].GetStringBody("") != "before" {
		t.Fatalf("first text() node = (%q, %q), want (%q, %q)", textNodes[0].GetName(), textNodes[0].GetStringBody(""), "#text", "before")
	}
	if textNodes[1].GetName() != "#text" || textNodes[1].GetStringBody("") != "after" {
		t.Fatalf("second text() node = (%q, %q), want (%q, %q)", textNodes[1].GetName(), textNodes[1].GetStringBody(""), "#text", "after")
	}
	if textNodes[2].GetName() != "#text" || textNodes[2].GetStringBody("") != "two" {
		t.Fatalf("third text() node = (%q, %q), want (%q, %q)", textNodes[2].GetName(), textNodes[2].GetStringBody(""), "#text", "two")
	}

	nodeNodes := parser.EvalNodes("/root/parent/child/node()")
	if len(nodeNodes) != 4 {
		t.Fatalf("node() node count = %d, want 4", len(nodeNodes))
	}
	if nodeNodes[0].GetName() != "#text" || nodeNodes[0].GetStringBody("") != "before" {
		t.Fatalf("first node() result = (%q, %q), want (%q, %q)", nodeNodes[0].GetName(), nodeNodes[0].GetStringBody(""), "#text", "before")
	}
	if nodeNodes[1].GetName() != "leaf" {
		t.Fatalf("second node() result name = %q, want %q", nodeNodes[1].GetName(), "leaf")
	}
	if nodeNodes[2].GetName() != "#text" || nodeNodes[2].GetStringBody("") != "after" {
		t.Fatalf("third node() result = (%q, %q), want (%q, %q)", nodeNodes[2].GetName(), nodeNodes[2].GetStringBody(""), "#text", "after")
	}
	if nodeNodes[3].GetName() != "#text" || nodeNodes[3].GetStringBody("") != "two" {
		t.Fatalf("fourth node() result = (%q, %q), want (%q, %q)", nodeNodes[3].GetName(), nodeNodes[3].GetStringBody(""), "#text", "two")
	}

	if actual := parser.EvalString("name(/root/parent/child/text())"); actual != "" {
		t.Fatalf("name(/root/parent/child/text()) = %q, want empty", actual)
	}
}

func TestXPathParserShouldCollapseEntityReferenceRunsForTextAndNodeStepsLikeMyBatisDOM(t *testing.T) {
	parser := mustNewXPathParser(t, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper [
  <!ENTITY custom "HELLO">
]>
<mapper>
  <select>pre&custom;post</select>
</mapper>`, XPathParserConfig{Resource: "xpath_next_slice_entity_runs.xml"})

	if actual := parser.EvalString("/mapper/select/text()"); actual != "prepost" {
		t.Fatalf("EvalString(/mapper/select/text()) = %q, want %q", actual, "prepost")
	}
	if actual := parser.EvalDouble("count(/mapper/select/text())"); actual != 1 {
		t.Fatalf("count(/mapper/select/text()) = %v, want 1", actual)
	}
	if actual := parser.EvalDouble("count(/mapper/select/node())"); actual != 1 {
		t.Fatalf("count(/mapper/select/node()) = %v, want 1", actual)
	}
	if actual := parser.EvalString("string(/mapper/select/text())"); actual != "prepost" {
		t.Fatalf("string(/mapper/select/text()) = %q, want %q", actual, "prepost")
	}

	textNodes := parser.EvalNodes("/mapper/select/text()")
	if len(textNodes) != 1 {
		t.Fatalf("text() node count = %d, want 1", len(textNodes))
	}
	if textNodes[0].GetName() != "#text" || textNodes[0].GetStringBody("") != "pre" {
		t.Fatalf("text() node = (%q, %q), want (%q, %q)", textNodes[0].GetName(), textNodes[0].GetStringBody(""), "#text", "pre")
	}

	nodeNodes := parser.EvalNodes("/mapper/select/node()")
	if len(nodeNodes) != 1 {
		t.Fatalf("node() node count = %d, want 1", len(nodeNodes))
	}
	if nodeNodes[0].GetName() != "#text" || nodeNodes[0].GetStringBody("") != "pre" {
		t.Fatalf("node() node = (%q, %q), want (%q, %q)", nodeNodes[0].GetName(), nodeNodes[0].GetStringBody(""), "#text", "pre")
	}
}

func TestXPathParserShouldResolveDoctypeWithEntityResolver(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper
  PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
  "https://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="test.Mapper">
  <select id="selectOne" resultType="string">select 1</select>
</mapper>`
	resolver := &recordingEntityResolver{}

	parser, err := NewXPathParser(xmlText, XPathParserConfig{
		Validation:     true,
		EntityResolver: resolver,
		Resource:       "doctype_mapper.xml",
	})
	if err != nil {
		t.Fatalf("NewXPathParser failed: %v", err)
	}
	if parser == nil {
		t.Fatal("parser should not be nil")
	}

	if len(resolver.calls) != 1 {
		t.Fatalf("resolver call count = %d, want 1", len(resolver.calls))
	}
	if resolver.calls[0].publicID != "-//mybatis.org//DTD Mapper 3.0//EN" {
		t.Fatalf("publicID = %q", resolver.calls[0].publicID)
	}
	if resolver.calls[0].systemID != "https://mybatis.org/dtd/mybatis-3-mapper.dtd" {
		t.Fatalf("systemID = %q", resolver.calls[0].systemID)
	}
}

func TestXPathParserShouldResolveDoctypeMetadataWhenPublicKeywordStartsOnNewLine(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper
PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
"https://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="test.Mapper">
  <select id="selectOne" resultType="string">select 1</select>
</mapper>`
	resolver := &recordingEntityResolver{}

	parser, err := NewXPathParser(xmlText, XPathParserConfig{
		Validation:     true,
		EntityResolver: resolver,
		Resource:       "doctype_mapper_newline_public.xml",
	})
	if err != nil {
		t.Fatalf("NewXPathParser failed: %v", err)
	}

	if len(resolver.calls) != 1 {
		t.Fatalf("resolver call count = %d, want 1", len(resolver.calls))
	}
	if resolver.calls[0].publicID != "-//mybatis.org//DTD Mapper 3.0//EN" {
		t.Fatalf("publicID = %q", resolver.calls[0].publicID)
	}
	if resolver.calls[0].systemID != "https://mybatis.org/dtd/mybatis-3-mapper.dtd" {
		t.Fatalf("systemID = %q", resolver.calls[0].systemID)
	}

	documentType := parser.GetDocumentType()
	if !documentType.Present {
		t.Fatal("document type should be present")
	}
	if documentType.RootName != "mapper" {
		t.Fatalf("rootName = %q, want %q", documentType.RootName, "mapper")
	}
	if documentType.PublicID != "-//mybatis.org//DTD Mapper 3.0//EN" {
		t.Fatalf("document publicID = %q", documentType.PublicID)
	}
	if documentType.SystemID != "https://mybatis.org/dtd/mybatis-3-mapper.dtd" {
		t.Fatalf("document systemID = %q", documentType.SystemID)
	}
}

func TestXPathParserShouldExposeDocumentTypeMetadata(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper
  PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
  "https://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="test.Mapper">
  <select id="selectOne" resultType="string">select 1</select>
</mapper>`
	resolver := &recordingEntityResolver{}

	parser, err := NewXPathParser(xmlText, XPathParserConfig{
		Validation:     true,
		EntityResolver: resolver,
		Resource:       "doctype_mapper.xml",
	})
	if err != nil {
		t.Fatalf("NewXPathParser failed: %v", err)
	}

	documentType := parser.GetDocumentType()
	if !documentType.Present {
		t.Fatal("document type should be present")
	}
	if documentType.RootName != "mapper" {
		t.Fatalf("rootName = %q, want %q", documentType.RootName, "mapper")
	}
	if documentType.PublicID != "-//mybatis.org//DTD Mapper 3.0//EN" {
		t.Fatalf("publicID = %q", documentType.PublicID)
	}
	if documentType.SystemID != "https://mybatis.org/dtd/mybatis-3-mapper.dtd" {
		t.Fatalf("systemID = %q", documentType.SystemID)
	}
}

func TestXPathParserShouldWrapEntityResolverErrors(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper
  PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
  "https://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="test.Mapper">
  <select id="selectOne" resultType="string">select 1</select>
</mapper>`
	resolver := &recordingEntityResolver{err: errors.New("resolver boom")}

	_, err := NewXPathParser(xmlText, XPathParserConfig{
		Validation:     true,
		EntityResolver: resolver,
		Resource:       "doctype_mapper.xml",
	})
	if err == nil {
		t.Fatal("expected NewXPathParser to fail when entity resolver returns an error")
	}
	if !strings.Contains(err.Error(), "resolver boom") {
		t.Fatalf("error = %q, want resolver error", err.Error())
	}
}

func TestXPathParserShouldResolveDoctypeWithEntityResolverWhenValidationDisabledLikeMyBatisDOM(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper
  PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
  "https://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="test.Mapper">
  <select id="selectOne" resultType="string">select 1</select>
</mapper>`
	resolver := &recordingEntityResolver{}

	parser, err := NewXPathParser(xmlText, XPathParserConfig{
		Validation:     false,
		EntityResolver: resolver,
		Resource:       "doctype_mapper_validation_disabled.xml",
	})
	if err != nil {
		t.Fatalf("NewXPathParser failed: %v", err)
	}
	if parser == nil {
		t.Fatal("parser should not be nil")
	}

	if len(resolver.calls) != 1 {
		t.Fatalf("resolver call count = %d, want 1", len(resolver.calls))
	}
	if resolver.calls[0].publicID != "-//mybatis.org//DTD Mapper 3.0//EN" {
		t.Fatalf("publicID = %q", resolver.calls[0].publicID)
	}
	if resolver.calls[0].systemID != "https://mybatis.org/dtd/mybatis-3-mapper.dtd" {
		t.Fatalf("systemID = %q", resolver.calls[0].systemID)
	}
}

func TestXPathParserShouldWrapEntityResolverErrorsWhenValidationDisabledLikeMyBatisDOM(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper
  PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
  "https://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="test.Mapper">
  <select id="selectOne" resultType="string">select 1</select>
</mapper>`
	resolver := &recordingEntityResolver{err: errors.New("resolver boom")}

	_, err := NewXPathParser(xmlText, XPathParserConfig{
		Validation:     false,
		EntityResolver: resolver,
		Resource:       "doctype_mapper_validation_disabled.xml",
	})
	if err == nil {
		t.Fatal("expected NewXPathParser to fail when entity resolver returns an error")
	}
	if !strings.Contains(err.Error(), "resolver boom") {
		t.Fatalf("error = %q, want resolver error", err.Error())
	}
}

func TestXPathParserShouldPreserveEntityReferenceNodesFromResolvedDTDLikeMyBatisDOM(t *testing.T) {
	resolver := &recordingEntityResolver{
		dtd: []byte(`<!ELEMENT mapper (select)>
<!ELEMENT select (#PCDATA)>
<!ATTLIST select id CDATA #REQUIRED mode CDATA #IMPLIED>
<!ENTITY custom "HELLO">`),
	}

	parser, err := NewXPathParser(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper
  PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
  "https://example.com/custom-mapper.dtd">
<mapper>
  <select id="selectOne" mode="pre&custom;post">pre&custom;post</select>
</mapper>`, XPathParserConfig{
		Validation:     false,
		EntityResolver: resolver,
		Resource:       "entity_reference_external_dtd.xml",
	})
	if err != nil {
		t.Fatalf("NewXPathParser failed: %v", err)
	}

	selectNode := parser.EvalNode("/mapper/select")
	if selectNode == nil {
		t.Fatal("select node should not be nil")
	}

	children := selectNode.GetNode().GetChildNodes()
	if len(children) != 3 {
		t.Fatalf("child node count = %d, want 3", len(children))
	}
	if children[0].GetNodeType() != TEXT_NODE {
		t.Fatalf("first child type = %d, want %d", children[0].GetNodeType(), TEXT_NODE)
	}
	if children[1].GetNodeType() != ENTITY_REFERENCE_NODE {
		t.Fatalf("second child type = %d, want %d", children[1].GetNodeType(), ENTITY_REFERENCE_NODE)
	}
	if children[1].GetNodeName() != "custom" {
		t.Fatalf("entity reference name = %q, want %q", children[1].GetNodeName(), "custom")
	}
	if children[2].GetNodeType() != TEXT_NODE {
		t.Fatalf("third child type = %d, want %d", children[2].GetNodeType(), TEXT_NODE)
	}

	if actual := parser.EvalString("/mapper/select"); actual != "prepost" {
		t.Fatalf("EvalString(/mapper/select) = %q, want %q", actual, "prepost")
	}
	if actual := parser.EvalString("/mapper/select/@mode"); actual != "preHELLOpost" {
		t.Fatalf("EvalString(/mapper/select/@mode) = %q, want %q", actual, "preHELLOpost")
	}
	if actual := selectNode.GetStringBody(""); actual != "pre" {
		t.Fatalf("GetStringBody = %q, want %q", actual, "pre")
	}
}

func TestXPathParserShouldPreserveEntityReferenceNodesFromInternalSubsetLikeMyBatisDOM(t *testing.T) {
	parser, err := NewXPathParser(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper [
  <!ENTITY custom "HELLO">
]>
<mapper>
  <select id="selectOne">pre&custom;post</select>
</mapper>`, XPathParserConfig{
		Resource: "entity_reference_internal_subset.xml",
	})
	if err != nil {
		t.Fatalf("NewXPathParser failed: %v", err)
	}

	selectNode := parser.EvalNode("/mapper/select")
	if selectNode == nil {
		t.Fatal("select node should not be nil")
	}

	children := selectNode.GetNode().GetChildNodes()
	if len(children) != 3 {
		t.Fatalf("child node count = %d, want 3", len(children))
	}
	if children[1].GetNodeType() != ENTITY_REFERENCE_NODE {
		t.Fatalf("second child type = %d, want %d", children[1].GetNodeType(), ENTITY_REFERENCE_NODE)
	}
	if children[1].GetNodeName() != "custom" {
		t.Fatalf("entity reference name = %q, want %q", children[1].GetNodeName(), "custom")
	}
	if actual := parser.EvalString("/mapper/select"); actual != "prepost" {
		t.Fatalf("EvalString(/mapper/select) = %q, want %q", actual, "prepost")
	}
}

func testEvalMethod(t *testing.T, parser *XPathParser) {
	t.Helper()

	if actual := parser.EvalLong("/employee/birth_date/year"); actual != 1970 {
		t.Fatalf("EvalLong mismatch: %d", actual)
	}
	if actual := parser.EvalNode("/employee/birth_date/year").GetLongBody(); actual != 1970 {
		t.Fatalf("GetLongBody mismatch: %d", actual)
	}
	if actual := parser.EvalShort("/employee/birth_date/month"); actual != 6 {
		t.Fatalf("EvalShort mismatch: %d", actual)
	}
	if actual := parser.EvalInteger("/employee/birth_date/day"); actual != 15 {
		t.Fatalf("EvalInteger mismatch: %d", actual)
	}
	if actual := parser.EvalNode("/employee/birth_date/day").GetIntBody(); actual != 15 {
		t.Fatalf("GetIntBody mismatch: %d", actual)
	}
	if actual := parser.EvalFloat("/employee/height"); actual != 5.8 {
		t.Fatalf("EvalFloat mismatch: %f", actual)
	}
	if actual := parser.EvalNode("/employee/height").GetFloatBody(); actual != 5.8 {
		t.Fatalf("GetFloatBody mismatch: %f", actual)
	}
	if actual := parser.EvalDouble("/employee/height"); actual != 5.8 {
		t.Fatalf("EvalDouble mismatch: %f", actual)
	}
	if actual := parser.EvalNode("/employee/height").GetDoubleBody(); actual != 5.8 {
		t.Fatalf("GetDoubleBody mismatch: %f", actual)
	}
	if actual := parser.EvalNode("/employee").EvalDouble("height"); actual != 5.8 {
		t.Fatalf("EvalDouble on node mismatch: %f", actual)
	}
	if actual := parser.EvalString("/employee/@id"); actual != "${id_var}" {
		t.Fatalf("EvalString mismatch: %q", actual)
	}
	if actual := parser.EvalNode("/employee/@id").GetStringBody(); actual != "${id_var}" {
		t.Fatalf("GetStringBody mismatch: %q", actual)
	}
	if actual := parser.EvalNode("/employee").EvalString("@id"); actual != "${id_var}" {
		t.Fatalf("EvalString on node mismatch: %q", actual)
	}
	if actual := parser.EvalBoolean("/employee/active"); actual != true {
		t.Fatalf("EvalBoolean mismatch: %t", actual)
	}
	if actual := parser.EvalNode("/employee/active").GetBooleanBody(); actual != true {
		t.Fatalf("GetBooleanBody mismatch: %t", actual)
	}
	if actual := parser.EvalNode("/employee").EvalBoolean("active"); actual != true {
		t.Fatalf("EvalBoolean on node mismatch: %t", actual)
	}
	enumValue := parser.EvalNode("/employee/active").GetEnumAttribute(reflect.TypeOf(enumTest("")), "bot")
	typedEnumValue, ok := enumValue.(enumTest)
	if !ok {
		t.Fatalf("enum attribute type = %T, want %T", enumValue, enumTest(""))
	}
	if typedEnumValue != enumTestYES {
		t.Fatalf("enum attribute mismatch: %q", typedEnumValue)
	}
	if actual := parser.EvalNode("/employee/active").GetFloatAttribute("score"); actual != 3.2 {
		t.Fatalf("GetFloatAttribute mismatch: %f", actual)
	}
	if actual := parser.EvalNode("/employee/active").GetDoubleAttribute("score"); actual != 3.2 {
		t.Fatalf("GetDoubleAttribute mismatch: %f", actual)
	}
	if actual := strings.TrimSpace(parser.EvalNode("/employee/@id").String()); actual != "<id>\n  ${id_var}\n</id>" {
		t.Fatalf("attribute string mismatch:\n%s", actual)
	}
	if actual := len(parser.EvalNodes("/employee/*")); actual != 7 {
		t.Fatalf("EvalNodes size mismatch: %d", actual)
	}

	node := parser.EvalNode("/employee/height")
	if actual := node.GetPath(); actual != "employee/height" {
		t.Fatalf("GetPath mismatch: %q", actual)
	}
	if actual := node.GetValueBasedIdentifier(); actual != "employee[${id_var}]_height" {
		t.Fatalf("GetValueBasedIdentifier mismatch: %q", actual)
	}

	position := node.GetPosition()
	if position.Source != "nodelet_test.xml" {
		t.Fatalf("position source mismatch: %q", position.Source)
	}
	if position.Line <= 0 || position.Column <= 0 {
		t.Fatalf("position not recorded: %+v", position)
	}
}

func TestXPathParserShouldReturnDefaultEnumAttributeWhenMissing(t *testing.T) {
	parser := mustLoadParser(t)

	enumValue := parser.EvalNode("/employee/active").GetEnumAttribute(
		reflect.TypeOf(enumTest("")),
		"missing",
		enumTestNO,
	)
	typedEnumValue, ok := enumValue.(enumTest)
	if !ok {
		t.Fatalf("enum attribute type = %T, want %T", enumValue, enumTest(""))
	}
	if typedEnumValue != enumTestNO {
		t.Fatalf("enum attribute = %q, want %q", typedEnumValue, enumTestNO)
	}
}

func TestXPathParserShouldFailFastOnInvalidNumericValues(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root badInt="abc" badLong="abc" badFloat="abc" badDouble="abc">
		  <badShort>abc</badShort>
		  <badIntBody>abc</badIntBody>
		  <badLongBody>abc</badLongBody>
		  <badFloatBody>abc</badFloatBody>
		  <badDoubleBody>abc</badDoubleBody>
		</root>
	`, XPathParserConfig{Resource: "invalid_numeric_values.xml"})

	root := parser.EvalNode("/root")
	testCases := []struct {
		name string
		fn   func()
	}{
		{name: "EvalShort", fn: func() { _ = parser.EvalShort("/root/badShort") }},
		{name: "EvalInteger", fn: func() { _ = parser.EvalInteger("/root/badIntBody") }},
		{name: "EvalLong", fn: func() { _ = parser.EvalLong("/root/badLongBody") }},
		{name: "EvalFloat", fn: func() { _ = parser.EvalFloat("/root/badFloatBody") }},
		{name: "GetIntBody", fn: func() { _ = parser.EvalNode("/root/badIntBody").GetIntBody() }},
		{name: "GetLongBody", fn: func() { _ = parser.EvalNode("/root/badLongBody").GetLongBody() }},
		{name: "GetFloatBody", fn: func() { _ = parser.EvalNode("/root/badFloatBody").GetFloatBody() }},
		{name: "GetDoubleBody", fn: func() { _ = parser.EvalNode("/root/badDoubleBody").GetDoubleBody() }},
		{name: "GetIntAttribute", fn: func() { _ = root.GetIntAttribute("badInt") }},
		{name: "GetLongAttribute", fn: func() { _ = root.GetLongAttribute("badLong") }},
		{name: "GetFloatAttribute", fn: func() { _ = root.GetFloatAttribute("badFloat") }},
		{name: "GetDoubleAttribute", fn: func() { _ = root.GetDoubleAttribute("badDouble") }},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertPanicContains(t, "invalid syntax", testCase.fn)
		})
	}

	for _, expression := range []string{"/root/badDoubleBody", "/root/missing"} {
		t.Run("EvalDouble/"+expression, func(t *testing.T) {
			if actual := parser.EvalDouble(expression); !math.IsNaN(actual) {
				t.Fatalf("EvalDouble(%q) = %v, want NaN", expression, actual)
			}
		})
	}
}

func TestXPathParserShouldUseXPathNumberSemanticsForEvalDoubleLikeMyBatis(t *testing.T) {
	variables := NewProperties()
	variables.Put("value", "5.8")

	parser := mustNewXPathParser(t, `
		<root>
		  <good>5.8</good>
		  <bad>abc</bad>
		  <templated>${value}</templated>
		</root>
	`, XPathParserConfig{
		Resource:  "eval_double_semantics.xml",
		Variables: variables,
	})

	if actual := parser.EvalDouble("/root/good"); actual != 5.8 {
		t.Fatalf("EvalDouble(good) = %v, want 5.8", actual)
	}
	if actual := parser.EvalNode("/root").EvalDouble("good"); actual != 5.8 {
		t.Fatalf("EvalDouble(good) from node = %v, want 5.8", actual)
	}

	for _, testCase := range []struct {
		name       string
		expression string
	}{
		{name: "bad", expression: "/root/bad"},
		{name: "missing", expression: "/root/missing"},
		{name: "templated", expression: "/root/templated"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if actual := parser.EvalDouble(testCase.expression); !math.IsNaN(actual) {
				t.Fatalf("EvalDouble(%q) = %v, want NaN", testCase.expression, actual)
			}
		})
	}

	if actual := parser.EvalNode("/root").EvalDouble("templated"); !math.IsNaN(actual) {
		t.Fatalf("EvalDouble(templated) from node = %v, want NaN", actual)
	}
}

func TestXPathParserShouldUseXPathBooleanSemanticsForSupportedExpressionsLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <empty></empty>
		</root>
	`, XPathParserConfig{Resource: "eval_boolean_semantics.xml"})

	if actual := parser.EvalBoolean("count(/root/item)"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("count(/root/missing)"); actual != false {
		t.Fatalf("EvalBoolean(count(/root/missing)) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("string(/root/item)"); actual != true {
		t.Fatalf("EvalBoolean(string(/root/item)) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("string(/root/missing)"); actual != false {
		t.Fatalf("EvalBoolean(string(/root/missing)) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("boolean(count(/root/item))"); actual != true {
		t.Fatalf("EvalBoolean(boolean(count(/root/item))) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("boolean(count(/root/missing))"); actual != false {
		t.Fatalf("EvalBoolean(boolean(count(/root/missing))) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("true()"); actual != true {
		t.Fatalf("EvalBoolean(true()) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("false()"); actual != false {
		t.Fatalf("EvalBoolean(false()) = %t, want false", actual)
	}

	if actual := parser.EvalString("true()"); actual != "true" {
		t.Fatalf("EvalString(true()) = %q, want %q", actual, "true")
	}
	if actual := parser.EvalString("false()"); actual != "false" {
		t.Fatalf("EvalString(false()) = %q, want %q", actual, "false")
	}
	if actual := parser.EvalString("boolean(count(/root/item))"); actual != "true" {
		t.Fatalf("EvalString(boolean(count(/root/item))) = %q, want %q", actual, "true")
	}

	if actual := parser.EvalDouble("true()"); actual != 1 {
		t.Fatalf("EvalDouble(true()) = %v, want 1", actual)
	}
	if actual := parser.EvalDouble("false()"); actual != 0 {
		t.Fatalf("EvalDouble(false()) = %v, want 0", actual)
	}
	if actual := parser.EvalDouble("boolean(count(/root/item))"); actual != 1 {
		t.Fatalf("EvalDouble(boolean(count(/root/item))) = %v, want 1", actual)
	}
}

func TestXPathParserShouldSupportEqualityComparisonsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2" flag="yes">B</item>
		  <n>2</n>
		</root>
	`, XPathParserConfig{Resource: "eval_equality_comparisons.xml"})

	if actual := parser.EvalBoolean("count(/root/item)=2"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)=2) = %t, want true", actual)
	}
	if actual := parser.EvalString("count(/root/item)=2"); actual != "true" {
		t.Fatalf("EvalString(count(/root/item)=2) = %q, want %q", actual, "true")
	}
	if actual := parser.EvalDouble("count(/root/item)=2"); actual != 1 {
		t.Fatalf("EvalDouble(count(/root/item)=2) = %v, want 1", actual)
	}
	if actual := parser.EvalBoolean("/root/item='A'"); actual != true {
		t.Fatalf("EvalBoolean(/root/item='A') = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/item='Z'"); actual != false {
		t.Fatalf("EvalBoolean(/root/item='Z') = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("/root/item/@id='2'"); actual != true {
		t.Fatalf("EvalBoolean(/root/item/@id='2') = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/item/@id=2"); actual != true {
		t.Fatalf("EvalBoolean(/root/item/@id=2) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/n=2"); actual != true {
		t.Fatalf("EvalBoolean(/root/n=2) = %t, want true", actual)
	}
}

func TestXPathParserShouldSupportEqualityComparisonsInsidePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2" flag="yes">B</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_equality_predicates.xml"})

	if actual := parser.EvalDouble("count(/root/item[count(@*)=1])"); actual != 1 {
		t.Fatalf("count(/root/item[count(@*)=1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[count(@*)=1]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[count(@*)=1]) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalString("/root/item[count(@*)=2]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[count(@*)=2]) = %q, want %q", actual, "B")
	}
}

func TestXPathParserShouldSupportNonEqualityComparisonsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2" flag="yes">B</item>
		  <n>2</n>
		</root>
	`, XPathParserConfig{Resource: "eval_non_equality_comparisons.xml"})

	if actual := parser.EvalBoolean("count(/root/item)!=1"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)!=1) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)>1"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)>1) = %t, want true", actual)
	}
	if actual := parser.EvalString("count(/root/item)>1"); actual != "true" {
		t.Fatalf("EvalString(count(/root/item)>1) = %q, want %q", actual, "true")
	}
	if actual := parser.EvalDouble("count(/root/item)>1"); actual != 1 {
		t.Fatalf("EvalDouble(count(/root/item)>1) = %v, want 1", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)>=2"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)>=2) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)<2"); actual != false {
		t.Fatalf("EvalBoolean(count(/root/item)<2) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)<=2"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)<=2) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/n>1"); actual != true {
		t.Fatalf("EvalBoolean(/root/n>1) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/n<1"); actual != false {
		t.Fatalf("EvalBoolean(/root/n<1) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("/root/item/@id>1"); actual != true {
		t.Fatalf("EvalBoolean(/root/item/@id>1) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/item/@id<2"); actual != true {
		t.Fatalf("EvalBoolean(/root/item/@id<2) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean(`/root/item!="Z"`); actual != true {
		t.Fatalf(`EvalBoolean(/root/item!="Z") = %t, want true`, actual)
	}
	if actual := parser.EvalBoolean(`/root/item!="A"`); actual != true {
		t.Fatalf(`EvalBoolean(/root/item!="A") = %t, want true`, actual)
	}
}

func TestXPathParserShouldSupportNonEqualityComparisonsInsidePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2" flag="yes">B</item>
		  <item id="3" flag="yes" kind="third">C</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_non_equality_predicates.xml"})

	if actual := parser.EvalDouble("count(/root/item[count(@*)>1])"); actual != 2 {
		t.Fatalf("count(/root/item[count(@*)>1]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[count(@*)>1]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[count(@*)>1]) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[count(@*)!=1])"); actual != 2 {
		t.Fatalf("count(/root/item[count(@*)!=1]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[count(@*)!=1]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[count(@*)!=1]) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[position()<last()])"); actual != 2 {
		t.Fatalf("count(/root/item[position()<last()]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[position()<last()]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[position()<last()]) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count(/root/item[position()!=last()])"); actual != 2 {
		t.Fatalf("count(/root/item[position()!=last()]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[position()!=last()]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[position()!=last()]) = %q, want %q", actual, "A")
	}
}

func TestXPathParserShouldSupportLogicalExpressionsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2">B</item>
		  <n>2</n>
		</root>
	`, XPathParserConfig{Resource: "eval_logical_expressions.xml"})

	if actual := parser.EvalBoolean("count(/root/item)=2 and /root/n=2"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)=2 and /root/n=2) = %t, want true", actual)
	}
	if actual := parser.EvalString("count(/root/item)=2 and /root/n=2"); actual != "true" {
		t.Fatalf("EvalString(count(/root/item)=2 and /root/n=2) = %q, want %q", actual, "true")
	}
	if actual := parser.EvalDouble("count(/root/item)=2 and /root/n=2"); actual != 1 {
		t.Fatalf("EvalDouble(count(/root/item)=2 and /root/n=2) = %v, want 1", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)=2 and /root/n=3"); actual != false {
		t.Fatalf("EvalBoolean(count(/root/item)=2 and /root/n=3) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)=1 or /root/n=2"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)=1 or /root/n=2) = %t, want true", actual)
	}
}

func TestXPathParserShouldSupportLogicalExpressionsInsidePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2">B</item>
		  <item id="3">C</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_logical_predicates.xml"})

	if actual := parser.EvalDouble(`/root/item[@id="1" or @id="2"]`); !math.IsNaN(actual) {
		t.Fatalf(`EvalDouble(/root/item[@id="1" or @id="2"]) = %v, want NaN`, actual)
	}
	if actual := parser.EvalDouble(`count(/root/item[@id="1" or @id="2"])`); actual != 2 {
		t.Fatalf(`count(/root/item[@id="1" or @id="2"]) = %v, want 2`, actual)
	}
	if actual := parser.EvalString(`/root/item[@id="1" or @id="2"]`); actual != "A" {
		t.Fatalf(`EvalString(/root/item[@id="1" or @id="2"]) = %q, want %q`, actual, "A")
	}
	if actual := parser.EvalString(`/root/item[@id="1" and .="A"]`); actual != "A" {
		t.Fatalf(`EvalString(/root/item[@id="1" and .="A"]) = %q, want %q`, actual, "A")
	}
	if actual := parser.EvalString(`/root/item[@id="1" and .="B"]`); actual != "" {
		t.Fatalf(`EvalString(/root/item[@id="1" and .="B"]) = %q, want empty`, actual)
	}
}

func TestXPathParserShouldSupportAdditiveArithmeticExpressionsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		</root>
	`, XPathParserConfig{Resource: "eval_additive_arithmetic.xml"})

	if actual := parser.EvalBoolean("1+1=2"); actual != true {
		t.Fatalf("EvalBoolean(1+1=2) = %t, want true", actual)
	}
	if actual := parser.EvalString("1+1=2"); actual != "true" {
		t.Fatalf("EvalString(1+1=2) = %q, want %q", actual, "true")
	}
	if actual := parser.EvalDouble("1+1"); actual != 2 {
		t.Fatalf("EvalDouble(1+1) = %v, want 2", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)+1=4"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)+1=4) = %t, want true", actual)
	}
	if actual := parser.EvalDouble("count(/root/item)+1"); actual != 4 {
		t.Fatalf("EvalDouble(count(/root/item)+1) = %v, want 4", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)-1=2"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)-1=2) = %t, want true", actual)
	}
	if actual := parser.EvalDouble("count(/root/item)-1"); actual != 2 {
		t.Fatalf("EvalDouble(count(/root/item)-1) = %v, want 2", actual)
	}
}

func TestXPathParserShouldSupportAdditiveArithmeticInsidePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_additive_arithmetic_predicates.xml"})

	if actual := parser.EvalDouble("count(/root/item[position()=last()-1])"); actual != 1 {
		t.Fatalf("count(/root/item[position()=last()-1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[position()=last()-1]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[position()=last()-1]) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[position()=1+1])"); actual != 1 {
		t.Fatalf("count(/root/item[position()=1+1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[position()=1+1]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[position()=1+1]) = %q, want %q", actual, "B")
	}
}

func TestXPathParserShouldSupportMultiplicativeArithmeticExpressionsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		</root>
	`, XPathParserConfig{Resource: "eval_multiplicative_arithmetic.xml"})

	if actual := parser.EvalBoolean("2*3=6"); actual != true {
		t.Fatalf("EvalBoolean(2*3=6) = %t, want true", actual)
	}
	if actual := parser.EvalDouble("2*3"); actual != 6 {
		t.Fatalf("EvalDouble(2*3) = %v, want 6", actual)
	}
	if actual := parser.EvalBoolean("8 div 2=4"); actual != true {
		t.Fatalf("EvalBoolean(8 div 2=4) = %t, want true", actual)
	}
	if actual := parser.EvalDouble("8 div 2"); actual != 4 {
		t.Fatalf("EvalDouble(8 div 2) = %v, want 4", actual)
	}
	if actual := parser.EvalBoolean("5 mod 2=1"); actual != true {
		t.Fatalf("EvalBoolean(5 mod 2=1) = %t, want true", actual)
	}
	if actual := parser.EvalDouble("5 mod 2"); actual != 1 {
		t.Fatalf("EvalDouble(5 mod 2) = %v, want 1", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)*2=6"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)*2=6) = %t, want true", actual)
	}
}

func TestXPathParserShouldSupportMultiplicativeArithmeticInsidePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_multiplicative_arithmetic_predicates.xml"})

	if actual := parser.EvalDouble("count(/root/item[position()*2=4])"); actual != 1 {
		t.Fatalf("count(/root/item[position()*2=4]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[position()*2=4]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[position()*2=4]) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/item[position() mod 2=1])"); actual != 2 {
		t.Fatalf("count(/root/item[position() mod 2=1]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[position() mod 2=1]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[position() mod 2=1]) = %q, want %q", actual, "A")
	}
}

func TestXPathParserShouldSupportGroupedAndUnaryArithmeticExpressionsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		</root>
	`, XPathParserConfig{Resource: "eval_grouped_unary_arithmetic.xml"})

	if actual := parser.EvalBoolean("(1+2)*3=9"); actual != true {
		t.Fatalf("EvalBoolean((1+2)*3=9) = %t, want true", actual)
	}
	if actual := parser.EvalDouble("(1+2)*3"); actual != 9 {
		t.Fatalf("EvalDouble((1+2)*3) = %v, want 9", actual)
	}
	if actual := parser.EvalBoolean("1+(2*3)=7"); actual != true {
		t.Fatalf("EvalBoolean(1+(2*3)=7) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("count(/root/item)+(1)=4"); actual != true {
		t.Fatalf("EvalBoolean(count(/root/item)+(1)=4) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("-(1)+2=1"); actual != true {
		t.Fatalf("EvalBoolean(-(1)+2=1) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("-count(/root/item)+4=1"); actual != true {
		t.Fatalf("EvalBoolean(-count(/root/item)+4=1) = %t, want true", actual)
	}
}

func TestXPathParserShouldSupportGroupedAndUnaryArithmeticInsidePredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_grouped_unary_arithmetic_predicates.xml"})

	if actual := parser.EvalDouble("count(/root/item[position()=(last()-1)])"); actual != 1 {
		t.Fatalf("count(/root/item[position()=(last()-1)]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[position()=(last()-1)]"); actual != "B" {
		t.Fatalf("EvalString(/root/item[position()=(last()-1)]) = %q, want %q", actual, "B")
	}
}

func TestXPathParserShouldNotTreatHyphenatedStepNamesAsArithmeticLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <cache-ref>A</cache-ref>
		  <item-1>B</item-1>
		</root>
	`, XPathParserConfig{Resource: "xpath_hyphenated_step_names.xml"})

	if actual := parser.EvalString("/root/cache-ref"); actual != "A" {
		t.Fatalf("EvalString(/root/cache-ref) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalString("/root/item-1"); actual != "B" {
		t.Fatalf("EvalString(/root/item-1) = %q, want %q", actual, "B")
	}
	if actual := parser.EvalDouble("count(/root/*)"); actual != 2 {
		t.Fatalf("count(/root/*) = %v, want 2", actual)
	}
}

func TestXPathParserShouldSupportNumberFunctionLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><n>5.8</n><empty></empty></root>`,
		XPathParserConfig{Resource: "eval_number_function.xml"},
	)

	if actual := parser.EvalDouble("number('5.8')"); actual != 5.8 {
		t.Fatalf("EvalDouble(number('5.8')) = %v, want 5.8", actual)
	}
	if actual := parser.EvalDouble("number('abc')"); !math.IsNaN(actual) {
		t.Fatalf("EvalDouble(number('abc')) = %v, want NaN", actual)
	}
	if actual := parser.EvalDouble("number()"); actual != 5.8 {
		t.Fatalf("EvalDouble(number()) = %v, want 5.8", actual)
	}
	if actual := parser.EvalString("string(number(true()))"); actual != "1" {
		t.Fatalf("EvalString(string(number(true()))) = %q, want %q", actual, "1")
	}
	if actual := parser.EvalString("string(number(/root/n))"); actual != "5.8" {
		t.Fatalf("EvalString(string(number(/root/n))) = %q, want %q", actual, "5.8")
	}
}

func TestXPathParserShouldSupportNotFunctionLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item>A</item></root>`,
		XPathParserConfig{Resource: "eval_not_function.xml"},
	)

	if actual := parser.EvalBoolean("not(false())"); actual != true {
		t.Fatalf("EvalBoolean(not(false())) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("not('')"); actual != true {
		t.Fatalf("EvalBoolean(not('')) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("not('x')"); actual != false {
		t.Fatalf("EvalBoolean(not('x')) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("not(/root/unknown)"); actual != true {
		t.Fatalf("EvalBoolean(not(/root/unknown)) = %t, want true", actual)
	}
	if actual := parser.EvalString("not(/root/item)"); actual != "false" {
		t.Fatalf("EvalString(not(/root/item)) = %q, want %q", actual, "false")
	}
}

func TestXPathParserShouldRejectScalarExpressionsWhenNodeSetIsRequiredLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		</root>
	`, XPathParserConfig{Resource: "eval_nodes_scalar_conversion.xml"})

	assertPanicContains(t, "Error evaluating XPath. Cause: Can not convert #BOOLEAN to a NodeList!", func() {
		_ = parser.EvalNode("true()")
	})
	assertPanicContains(t, "Error evaluating XPath. Cause: Can not convert #BOOLEAN to a NodeList!", func() {
		_ = parser.EvalNodes("true()")
	})
	assertPanicContains(t, "Error evaluating XPath. Cause: Can not convert #NUMBER to a NodeList!", func() {
		_ = parser.EvalNode("1")
	})
	assertPanicContains(t, "Error evaluating XPath. Cause: Can not convert #NUMBER to a NodeList!", func() {
		_ = parser.EvalNodes("1")
	})
	assertPanicContains(t, "Error evaluating XPath. Cause: Can not convert #STRING to a NodeList!", func() {
		_ = parser.EvalNode("string(/root/item)")
	})
	assertPanicContains(t, "Error evaluating XPath. Cause: Can not convert #STRING to a NodeList!", func() {
		_ = parser.EvalNodes("'x'")
	})
}

func TestXPathParserShouldOnlyTrimXPathWhitespaceForEvalDoubleLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <space> 5.8 </space>
		  <nbsp> 5.8 </nbsp>
		</root>
	`, XPathParserConfig{Resource: "eval_double_xpath_whitespace.xml"})

	if actual := parser.EvalDouble("/root/space"); actual != 5.8 {
		t.Fatalf("EvalDouble(space) = %v, want 5.8", actual)
	}
	if actual := parser.EvalNode("/root").EvalDouble("space"); actual != 5.8 {
		t.Fatalf("EvalDouble(space) from node = %v, want 5.8", actual)
	}

	if actual := parser.EvalDouble("/root/nbsp"); !math.IsNaN(actual) {
		t.Fatalf("EvalDouble(nbsp) = %v, want NaN", actual)
	}
	if actual := parser.EvalNode("/root").EvalDouble("nbsp"); !math.IsNaN(actual) {
		t.Fatalf("EvalDouble(nbsp) from node = %v, want NaN", actual)
	}
}

func TestXPathParserShouldUseJavaFloatWhitespaceSemanticsLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root goodFloatAttr=" 3.2 " goodDoubleAttr=" 4.6 " badFloatAttr=" 3.2 " badDoubleAttr=" 4.6 ">
		  <goodFloatBody> 5.8 </goodFloatBody>
		  <goodDoubleBody> 6.7 </goodDoubleBody>
		  <badFloatBody> 5.8 </badFloatBody>
		  <badDoubleBody> 6.7 </badDoubleBody>
		</root>
	`, XPathParserConfig{Resource: "java_float_whitespace.xml"})

	root := parser.EvalNode("/root")
	if root == nil {
		t.Fatal("root should not be nil")
	}

	if actual := parser.EvalFloat("/root/goodFloatBody"); actual != 5.8 {
		t.Fatalf("EvalFloat(goodFloatBody) = %v, want 5.8", actual)
	}
	if actual := parser.EvalNode("/root/goodFloatBody").GetFloatBody(); actual != 5.8 {
		t.Fatalf("GetFloatBody(goodFloatBody) = %v, want 5.8", actual)
	}
	if actual := parser.EvalNode("/root/goodDoubleBody").GetDoubleBody(); actual != 6.7 {
		t.Fatalf("GetDoubleBody(goodDoubleBody) = %v, want 6.7", actual)
	}
	if actual := root.GetFloatAttribute("goodFloatAttr"); actual != 3.2 {
		t.Fatalf("GetFloatAttribute(goodFloatAttr) = %v, want 3.2", actual)
	}
	if actual := root.GetDoubleAttribute("goodDoubleAttr"); actual != 4.6 {
		t.Fatalf("GetDoubleAttribute(goodDoubleAttr) = %v, want 4.6", actual)
	}
	if actual, ok := root.LookupFloatAttribute("goodFloatAttr"); !ok || actual != 3.2 {
		t.Fatalf("LookupFloatAttribute(goodFloatAttr) = (%v, %t), want (3.2, true)", actual, ok)
	}
	if actual, ok := root.LookupDoubleAttribute("goodDoubleAttr"); !ok || actual != 4.6 {
		t.Fatalf("LookupDoubleAttribute(goodDoubleAttr) = (%v, %t), want (4.6, true)", actual, ok)
	}

	assertPanicContains(t, "invalid syntax", func() { _ = parser.EvalFloat("/root/badFloatBody") })
	assertPanicContains(t, "invalid syntax", func() { _ = parser.EvalNode("/root/badFloatBody").GetFloatBody() })
	assertPanicContains(t, "invalid syntax", func() { _ = parser.EvalNode("/root/badDoubleBody").GetDoubleBody() })
	assertPanicContains(t, "invalid syntax", func() { _ = root.GetFloatAttribute("badFloatAttr") })
	assertPanicContains(t, "invalid syntax", func() { _ = root.GetDoubleAttribute("badDoubleAttr") })

	if _, ok := root.LookupFloatAttribute("badFloatAttr"); ok {
		t.Fatal("LookupFloatAttribute(badFloatAttr) should fail")
	}
	if _, ok := root.LookupDoubleAttribute("badDoubleAttr"); ok {
		t.Fatal("LookupDoubleAttribute(badDoubleAttr) should fail")
	}
}

func TestXPathParserShouldRejectNonXPathWhitespaceAroundUnionLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <a>1</a>
		  <b>2</b>
		</root>
	`, XPathParserConfig{Resource: "xpath_union_whitespace.xml"})

	if actual := len(parser.EvalNodes(" /root/a | /root/b ")); actual != 2 {
		t.Fatalf("EvalNodes with ASCII whitespace returned %d nodes, want 2", actual)
	}

	assertPanicContains(t, "Error evaluating XPath. Cause: A location path was expected", func() {
		_ = parser.EvalNodes("/root/a| /root/b")
	})
}

func mustLoadParser(t *testing.T) *XPathParser {
	t.Helper()
	return mustNewXPathParser(t, mustLoadNodeletXML(t), XPathParserConfig{Resource: "nodelet_test.xml"})
}

func mustLoadNodeletXML(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/nodelet_test.xml")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	return string(data)
}

func mustNewXPathParser(t *testing.T, xmlText string, configs ...XPathParserConfig) *XPathParser {
	t.Helper()
	parser, err := NewXPathParser(xmlText, configs...)
	if err != nil {
		t.Fatalf("NewXPathParser failed: %v", err)
	}
	return parser
}

type entityResolutionCall struct {
	publicID string
	systemID string
}

type recordingEntityResolver struct {
	calls []entityResolutionCall
	err   error
	dtd   []byte
}

func (resolver *recordingEntityResolver) ResolveEntity(publicID, systemID string) ([]byte, error) {
	resolver.calls = append(resolver.calls, entityResolutionCall{
		publicID: publicID,
		systemID: systemID,
	})
	if resolver.err != nil {
		return nil, resolver.err
	}
	if resolver.dtd != nil {
		return resolver.dtd, nil
	}
	return []byte("<!ELEMENT mapper ANY>\n"), nil
}

func assertPanicContains(t *testing.T, expectedSubstring string, fn func()) {
	t.Helper()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("panic = nil, want substring %q", expectedSubstring)
		}
		if !strings.Contains(panicMessage(recovered), expectedSubstring) {
			t.Fatalf("panic = %q, want substring %q", panicMessage(recovered), expectedSubstring)
		}
	}()

	fn()
}

func panicMessage(recovered any) string {
	switch value := recovered.(type) {
	case error:
		return value.Error()
	case string:
		return value
	default:
		return ""
	}
}
