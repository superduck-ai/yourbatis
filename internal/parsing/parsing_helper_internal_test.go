package parsing

import (
	"bytes"
	"errors"
	"math"
	"testing"
)

func TestXMLMapperEntityResolverShouldResolveKnownDTDResourcesLikeMyBatis(t *testing.T) {
	resolver := &XMLMapperEntityResolver{}

	testCases := []struct {
		name     string
		systemID string
		want     []byte
	}{
		{
			name:     "mybatis config",
			systemID: "https://mybatis.org/dtd/mybatis-3-config.dtd",
			want:     mybatisConfigDTD,
		},
		{
			name:     "ibatis config case insensitive",
			systemID: "HTTPS://IBATIS.ORG/DTD/IBATIS-3-CONFIG.DTD",
			want:     mybatisConfigDTD,
		},
		{
			name:     "mybatis mapper",
			systemID: "https://mybatis.org/dtd/mybatis-3-mapper.dtd",
			want:     mybatisMapperDTD,
		},
		{
			name:     "ibatis mapper case insensitive",
			systemID: "HTTPS://IBATIS.ORG/DTD/IBATIS-3-MAPPER.DTD",
			want:     mybatisMapperDTD,
		},
		{
			name:     "unknown dtd",
			systemID: "https://example.com/custom.dtd",
		},
		{
			name: "empty system id",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := resolver.ResolveEntity("-//ignored", testCase.systemID)
			if err != nil {
				t.Fatalf("ResolveEntity failed: %v", err)
			}

			if testCase.want == nil {
				if actual != nil {
					t.Fatalf("ResolveEntity(%q) = %q, want nil", testCase.systemID, string(actual))
				}
				return
			}
			if !bytes.Equal(actual, testCase.want) {
				t.Fatalf("ResolveEntity(%q) returned unexpected DTD payload", testCase.systemID)
			}
		})
	}
}

func TestXMLMapperEntityResolverShouldReturnFreshDTDContentLikeJavaResourceStreams(t *testing.T) {
	resolver := &XMLMapperEntityResolver{}

	first, err := resolver.ResolveEntity("", MYBATIS_MAPPER_SYSTEM)
	if err != nil {
		t.Fatalf("first ResolveEntity failed: %v", err)
	}
	second, err := resolver.ResolveEntity("", MYBATIS_MAPPER_SYSTEM)
	if err != nil {
		t.Fatalf("second ResolveEntity failed: %v", err)
	}
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("resolved mapper DTDs should not be empty")
	}

	original := second[0]
	first[0] ^= 0xff
	if second[0] != original {
		t.Fatal("resolved DTD payloads unexpectedly share mutable backing storage")
	}
	if mybatisMapperDTD[0] != original {
		t.Fatal("mutating a resolved DTD payload unexpectedly changed the embedded resource")
	}
}

func TestPropertiesPutAllShouldMergeAndIgnoreNilReceiversLikeJavaProperties(t *testing.T) {
	properties := NewProperties()
	properties.Put("existing", "old")

	properties.PutAll(Properties{
		"existing": "new",
		"added":    "value",
	})
	properties.PutAll(nil)

	if actual := properties.GetProperty("existing"); actual != "new" {
		t.Fatalf("existing property = %q, want %q", actual, "new")
	}
	if actual := properties.GetProperty("added"); actual != "value" {
		t.Fatalf("added property = %q, want %q", actual, "value")
	}
	if len(properties) != 2 {
		t.Fatalf("property count = %d, want 2", len(properties))
	}

	var nilProperties Properties
	nilProperties.PutAll(Properties{"ignored": "value"})
	if nilProperties != nil {
		t.Fatalf("nil properties = %#v, want nil", nilProperties)
	}
}

func TestParsingExceptionShouldExposeMessageAndCauseLikeWrappedErrors(t *testing.T) {
	cause := errors.New("boom")

	implicitMessage := NewParsingException("", cause)
	if actual := implicitMessage.Error(); actual != "boom" {
		t.Fatalf("implicit message = %q, want %q", actual, "boom")
	}
	if actual := implicitMessage.Unwrap(); actual != cause {
		t.Fatalf("Unwrap() = %#v, want %#v", actual, cause)
	}
	if !errors.Is(implicitMessage, cause) {
		t.Fatal("errors.Is should match the wrapped cause")
	}

	explicitMessage := NewParsingException("outer", cause)
	if actual := explicitMessage.Error(); actual != "outer" {
		t.Fatalf("explicit message = %q, want %q", actual, "outer")
	}

	var nilException *ParsingException
	if actual := nilException.Error(); actual != "" {
		t.Fatalf("nil exception Error() = %q, want empty", actual)
	}
	if actual := nilException.Unwrap(); actual != nil {
		t.Fatalf("nil exception Unwrap() = %#v, want nil", actual)
	}
}

func TestNodeHasChildNodesAndRemoveAttributeShouldReflectCurrentDOMStateLikeMyBatis(t *testing.T) {
	document := mustParseTestDocument(t, `<root attr="value"><child /></root>`, "node_has_child_nodes.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}

	var nilNode *Node
	if nilNode.HasChildNodes() {
		t.Fatal("nil node should not report child nodes")
	}
	if !root.HasChildNodes() {
		t.Fatal("root should report child nodes while the child element is attached")
	}

	child := root.GetFirstChild()
	if child == nil {
		t.Fatal("root child should not be nil")
	}
	if removed := root.RemoveChild(child); removed != child {
		t.Fatalf("RemoveChild() = %#v, want %#v", removed, child)
	}
	if root.HasChildNodes() {
		t.Fatal("root should not report child nodes after removing its only child")
	}

	attribute := root.GetAttribute("attr")
	if attribute == nil {
		t.Fatal("attribute should not be nil")
	}
	if !attribute.HasChildNodes() {
		t.Fatal("attribute should still expose its text value child like DOM")
	}

	root.removeAttribute(attribute)
	if actual := root.GetAttribute("attr"); actual != nil {
		t.Fatalf("root attribute after remove = %#v, want nil", actual)
	}
	if len(root.GetAttributes()) != 0 {
		t.Fatalf("attribute count after remove = %d, want 0", len(root.GetAttributes()))
	}
	if attribute.parent != nil {
		t.Fatal("removed attribute parent should be nil")
	}
	if !attribute.HasChildNodes() {
		t.Fatal("removed attribute should retain its value child")
	}
}

func TestXPathParserHelperMethodsShouldMatchMyBatisSemantics(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">1</item>
		  <item id="2">2</item>
		  <value>5.8</value>
		</root>
	`, XPathParserConfig{Resource: "xpath_internal_helpers.xml"})

	if parser.GetDocument() == nil {
		t.Fatal("GetDocument() should return the parsed document")
	}
	var nilParser *XPathParser
	if actual := nilParser.GetDocument(); actual != nil {
		t.Fatalf("nil parser GetDocument() = %#v, want nil", actual)
	}

	root := parser.GetDocument().GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}
	valueNodes := root.childElementsByName("value")
	if len(valueNodes) != 1 {
		t.Fatalf("value node count = %d, want 1", len(valueNodes))
	}
	valueNode := valueNodes[0]

	if actual := parser.evalXPathString(valueNode, ""); actual != "5.8" {
		t.Fatalf("evalXPathString(empty) = %q, want %q", actual, "5.8")
	}
	if actual := parser.evalXPathString(nil, ""); actual != "" {
		t.Fatalf("evalXPathString(nil, empty) = %q, want empty", actual)
	}
	if actual := parser.evalXPathNumber(valueNode, ""); actual != 5.8 {
		t.Fatalf("evalXPathNumber(empty) = %v, want 5.8", actual)
	}
	if actual := parser.evalXPathNumber(nil, ""); !math.IsNaN(actual) {
		t.Fatalf("evalXPathNumber(nil, empty) = %v, want NaN", actual)
	}
	if actual := parser.evalXPathBoolean(root, ""); actual != false {
		t.Fatalf("evalXPathBoolean(empty) = %t, want false", actual)
	}
	if actual := parser.evalXPathComparison(parser.rootNode(), "count(/root/item)", xpathComparisonEqual, "2"); actual != true {
		t.Fatalf("evalXPathComparison(count(/root/item)=2) = %t, want true", actual)
	}
	if actual := parser.evalXPathLogical(parser.rootNode(), "count(/root/item)=2", xpathLogicalAnd, "/root/item/@id=2"); actual != true {
		t.Fatalf("evalXPathLogical(and) = %t, want true", actual)
	}

	nodes := parser.evaluateNodes("/root/item", parser.rootNode())
	if len(nodes) != 2 {
		t.Fatalf("evaluateNodes(/root/item) count = %d, want 2", len(nodes))
	}
	if actual := nodes[0].GetAttribute("id").GetNodeValue(); actual != "1" {
		t.Fatalf("first item id = %q, want %q", actual, "1")
	}
	if actual := nodes[1].GetAttribute("id").GetNodeValue(); actual != "2" {
		t.Fatalf("second item id = %q, want %q", actual, "2")
	}
}

func TestXPathParserShouldSupportReversedRelationalComparisonsAndPositionPredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2">B</item>
		  <item id="3">C</item>
		</root>
	`, XPathParserConfig{Resource: "xpath_reversed_comparisons.xml"})

	if actual := parser.EvalBoolean("1 < /root/item/@id"); actual != true {
		t.Fatalf("EvalBoolean(1 < /root/item/@id) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("3 < /root/item/@id"); actual != false {
		t.Fatalf("EvalBoolean(3 < /root/item/@id) = %t, want false", actual)
	}
	if actual := parser.EvalDouble("count(/root/item[last()>position()])"); actual != 2 {
		t.Fatalf("count(/root/item[last()>position()]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[last()>position()]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[last()>position()]) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count(/root/item[2<position()])"); actual != 1 {
		t.Fatalf("count(/root/item[2<position()]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[2<position()]"); actual != "C" {
		t.Fatalf("EvalString(/root/item[2<position()]) = %q, want %q", actual, "C")
	}
}

func TestApplyXPathPredicateShouldFilterNodeSlicesLikeXPath(t *testing.T) {
	document := mustParseTestDocument(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2">B</item>
		  <item>before<![CDATA[cdata]]><leaf />tail</item>
		</root>
	`, "apply_xpath_predicate.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}
	items := root.childElementsByName("item")
	if len(items) != 3 {
		t.Fatalf("item count = %d, want 3", len(items))
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind:     xpathPredicatePositionIndex,
		position: 2,
	}); len(actual) != 1 || actual[0].stringValue() != "B" {
		t.Fatalf("position index result = %#v, want second item", actual)
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind: xpathPredicatePositionLast,
	}); len(actual) != 1 || actual[0].stringValue() != "beforecdatatail" {
		t.Fatalf("position last result = %#v, want third item", actual)
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind:               xpathPredicatePositionComparison,
		position:           1,
		comparisonOperator: xpathComparisonGreaterThan,
	}); len(actual) != 2 || actual[0].stringValue() != "B" || actual[1].stringValue() != "beforecdatatail" {
		t.Fatalf("position comparison result = %#v, want second and third items", actual)
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind:          xpathPredicateAttributeExists,
		attributeName: "id",
	}); len(actual) != 2 || actual[0].stringValue() != "A" || actual[1].stringValue() != "B" {
		t.Fatalf("attribute exists result = %#v, want first two items", actual)
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind:          xpathPredicateAttributeEqualsString,
		attributeName: "id",
		stringValue:   "2",
	}); len(actual) != 1 || actual[0].stringValue() != "B" {
		t.Fatalf("attribute equals result = %#v, want second item", actual)
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind:        xpathPredicateStringValueEqualsString,
		stringValue: "B",
	}); len(actual) != 1 || actual[0].stringValue() != "B" {
		t.Fatalf("string value result = %#v, want second item", actual)
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind:        xpathPredicateTextEqualsString,
		stringValue: "tail",
	}); len(actual) != 1 || actual[0].stringValue() != "beforecdatatail" {
		t.Fatalf("text value result = %#v, want third item", actual)
	}

	if actual := applyXPathPredicate(items, xpathPredicate{
		kind:       xpathPredicateBooleanExpression,
		expression: "@id",
	}); len(actual) != len(items) {
		t.Fatalf("default predicate result count = %d, want %d", len(actual), len(items))
	}
}
