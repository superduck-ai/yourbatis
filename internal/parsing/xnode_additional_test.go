package parsing

import (
	"strings"
	"testing"
)

func TestXNodeNilReceiversAndDefaultsShouldReturnZeroValuesLikeMyBatis(t *testing.T) {
	var xnode *XNode

	if actual := xnode.EvalString("id"); actual != "" {
		t.Fatalf("EvalString() = %q, want empty string", actual)
	}
	if actual := xnode.EvalBoolean("id"); actual {
		t.Fatalf("EvalBoolean() = %t, want false", actual)
	}
	if actual := xnode.EvalDouble("id"); actual != 0 {
		t.Fatalf("EvalDouble() = %f, want 0", actual)
	}
	if actual := xnode.EvalNodes("id"); actual != nil {
		t.Fatalf("EvalNodes() = %#v, want nil", actual)
	}
	if actual := xnode.EvalNode("id"); actual != nil {
		t.Fatalf("EvalNode() = %#v, want nil", actual)
	}
	if actual := xnode.GetNode(); actual != nil {
		t.Fatalf("GetNode() = %#v, want nil", actual)
	}
	if actual := xnode.GetName(); actual != "" {
		t.Fatalf("GetName() = %q, want empty string", actual)
	}
	if xnode.IsValidationEnabled() {
		t.Fatal("IsValidationEnabled() = true, want false")
	}
	if actual := xnode.GetPosition(); actual != (Position{}) {
		t.Fatalf("GetPosition() = %#v, want zero position", actual)
	}
	if actual := xnode.GetAttributes(); actual != nil {
		t.Fatalf("GetAttributes() = %#v, want nil", actual)
	}
	if actual, ok := xnode.LookupStringBody(); ok || actual != "" {
		t.Fatalf("LookupStringBody() = (%q, %t), want (\"\", false)", actual, ok)
	}
	if actual := xnode.GetStringBody("fallback"); actual != "fallback" {
		t.Fatalf("GetStringBody() = %q, want %q", actual, "fallback")
	}
	if actual := xnode.GetBooleanBody(true); !actual {
		t.Fatalf("GetBooleanBody() = %t, want true", actual)
	}
	if actual := xnode.GetIntBody(7); actual != 7 {
		t.Fatalf("GetIntBody() = %d, want 7", actual)
	}
	if actual := xnode.GetLongBody(8); actual != 8 {
		t.Fatalf("GetLongBody() = %d, want 8", actual)
	}
	if actual := xnode.GetDoubleBody(1.25); actual != 1.25 {
		t.Fatalf("GetDoubleBody() = %f, want 1.25", actual)
	}
	if actual := xnode.GetFloatBody(2.5); actual != 2.5 {
		t.Fatalf("GetFloatBody() = %f, want 2.5", actual)
	}
	if actual, ok := xnode.LookupStringAttribute("id"); ok || actual != "" {
		t.Fatalf("LookupStringAttribute() = (%q, %t), want (\"\", false)", actual, ok)
	}
	if actual := xnode.GetStringAttribute("id", "fallback"); actual != "fallback" {
		t.Fatalf("GetStringAttribute() = %q, want %q", actual, "fallback")
	}
	if actual := xnode.GetDoubleAttribute("scale", 1.5); actual != 1.5 {
		t.Fatalf("GetDoubleAttribute() = %f, want 1.5", actual)
	}
	if actual := xnode.GetFloatAttribute("ratio", 2.5); actual != 2.5 {
		t.Fatalf("GetFloatAttribute() = %f, want 2.5", actual)
	}
	if actual := xnode.String(); actual != "" {
		t.Fatalf("String() = %q, want empty string", actual)
	}
	if actual := xnode.GetSourceXML(); actual != "" {
		t.Fatalf("GetSourceXML() = %q, want empty string", actual)
	}

	var builder strings.Builder
	xnode.buildSourceXML(&builder, 0)
	if builder.Len() != 0 {
		t.Fatalf("buildSourceXML() wrote %q, want empty output", builder.String())
	}
}

func TestXNodeShouldParseTypedBodiesAndFloatingPointAttributesLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<mapper stringAttr="alpha" doubleAttr=" 1.25 " floatAttr=" 2.5 ">
		  <stringNode>value</stringNode>
		  <booleanNode>TrUe</booleanNode>
		  <intNode>7</intNode>
		  <longNode>9223372036854775807</longNode>
		  <doubleNode> 3.75 </doubleNode>
		  <floatNode> 4.5 </floatNode>
		  <emptyNode />
		</mapper>
	`, XPathParserConfig{Resource: "xnode_additional_types.xml"})

	mapperNode := parser.EvalNode("/mapper")
	if mapperNode == nil {
		t.Fatal("mapper node should not be nil")
	}
	if actual := mapperNode.GetStringAttribute("stringAttr", "fallback"); actual != "alpha" {
		t.Fatalf("GetStringAttribute(stringAttr) = %q, want %q", actual, "alpha")
	}
	if actual := mapperNode.GetStringAttribute("missing", "fallback"); actual != "fallback" {
		t.Fatalf("GetStringAttribute(missing) = %q, want %q", actual, "fallback")
	}
	if actual := mapperNode.GetDoubleAttribute("doubleAttr", 0); actual != 1.25 {
		t.Fatalf("GetDoubleAttribute(doubleAttr) = %f, want 1.25", actual)
	}
	if actual := mapperNode.GetFloatAttribute("floatAttr", 0); actual != 2.5 {
		t.Fatalf("GetFloatAttribute(floatAttr) = %f, want 2.5", actual)
	}

	stringNode := parser.EvalNode("/mapper/stringNode")
	if actual, ok := stringNode.LookupStringBody(); !ok || actual != "value" {
		t.Fatalf("LookupStringBody(stringNode) = (%q, %t), want (%q, true)", actual, ok, "value")
	}
	if actual := stringNode.GetStringBody("fallback"); actual != "value" {
		t.Fatalf("GetStringBody(stringNode) = %q, want %q", actual, "value")
	}

	booleanNode := parser.EvalNode("/mapper/booleanNode")
	if actual := booleanNode.GetBooleanBody(false); !actual {
		t.Fatalf("GetBooleanBody(booleanNode) = %t, want true", actual)
	}

	intNode := parser.EvalNode("/mapper/intNode")
	if actual := intNode.GetIntBody(0); actual != 7 {
		t.Fatalf("GetIntBody(intNode) = %d, want 7", actual)
	}

	longNode := parser.EvalNode("/mapper/longNode")
	if actual := longNode.GetLongBody(0); actual != 9223372036854775807 {
		t.Fatalf("GetLongBody(longNode) = %d, want 9223372036854775807", actual)
	}

	doubleNode := parser.EvalNode("/mapper/doubleNode")
	if actual := doubleNode.GetDoubleBody(0); actual != 3.75 {
		t.Fatalf("GetDoubleBody(doubleNode) = %f, want 3.75", actual)
	}

	floatNode := parser.EvalNode("/mapper/floatNode")
	if actual := floatNode.GetFloatBody(0); actual != 4.5 {
		t.Fatalf("GetFloatBody(floatNode) = %f, want 4.5", actual)
	}

	emptyNode := parser.EvalNode("/mapper/emptyNode")
	if actual := emptyNode.GetStringBody("fallback"); actual != "fallback" {
		t.Fatalf("GetStringBody(emptyNode) = %q, want %q", actual, "fallback")
	}
	if actual := emptyNode.GetStringBody(); actual != "" {
		t.Fatalf("GetStringBody(emptyNode) = %q, want empty string", actual)
	}
}

func TestXNodeShouldRenderSourceXMLForTextCDATAEntityAndAttributeNodesLikeMyBatis(t *testing.T) {
	t.Run("text node", func(t *testing.T) {
		parser := mustNewXPathParser(t, `<script>select 1</script>`, XPathParserConfig{Resource: "xnode_source_text.xml"})
		script := parser.EvalNode("/script")

		textNode := mustXNodeChildOfType(t, script, TEXT_NODE)
		if actual := textNode.GetSourceXML(); actual != "select 1" {
			t.Fatalf("text GetSourceXML() = %q, want %q", actual, "select 1")
		}
	})

	t.Run("cdata node", func(t *testing.T) {
		parser := mustNewXPathParser(t, `<script><![CDATA[select 1]]></script>`, XPathParserConfig{Resource: "xnode_source_cdata.xml"})
		script := parser.EvalNode("/script")

		cdataNode := mustXNodeChildOfType(t, script, CDATA_SECTION_NODE)
		if actual := cdataNode.GetSourceXML(); actual != "<![CDATA[select 1]]>" {
			t.Fatalf("cdata GetSourceXML() = %q, want %q", actual, "<![CDATA[select 1]]>")
		}
	})

	t.Run("entity reference node", func(t *testing.T) {
		parser := mustNewXPathParser(t, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper [
  <!ENTITY custom "HELLO">
]>
<mapper>
  <select id="selectOne">pre&custom;post</select>
</mapper>`, XPathParserConfig{Resource: "xnode_source_entity.xml"})
		selectNode := parser.EvalNode("/mapper/select")

		entityNode := mustXNodeChildOfType(t, selectNode, ENTITY_REFERENCE_NODE)
		if actual := entityNode.GetSourceXML(); actual != "&custom;" {
			t.Fatalf("entity GetSourceXML() = %q, want %q", actual, "&custom;")
		}
	})

	t.Run("attribute node", func(t *testing.T) {
		attributeNode := mustLoadParser(t).EvalNode("/employee/@id")
		if attributeNode == nil {
			t.Fatal("attribute node should not be nil")
		}
		if actual := attributeNode.GetSourceXML(); actual != "<id>\n  ${id_var}\n</id>\n" {
			t.Fatalf("attribute GetSourceXML() = %q, want %q", actual, "<id>\n  ${id_var}\n</id>\n")
		}
	})
}

func mustXNodeChildOfType(t *testing.T, parent *XNode, nodeType NodeType) *XNode {
	t.Helper()

	if parent == nil || parent.GetNode() == nil {
		t.Fatal("parent node should not be nil")
	}

	for _, childNode := range parent.GetNode().GetChildNodes() {
		if childNode.GetNodeType() == nodeType {
			return parent.NewXNode(childNode)
		}
	}

	t.Fatalf("child node type %d should exist", nodeType)
	return nil
}
