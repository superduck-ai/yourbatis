package parsing

import "testing"

func TestXNodeFormatXNodeToString(t *testing.T) {
	parser := mustNewXPathParser(t,
		"<users><user><id>100</id><name>Tom</name><age>30</age><cars><car index=\"1\">BMW</car><car index=\"2\">Audi</car><car index=\"3\">Benz</car></cars></user></users>",
	)

	usersNodeToString := parser.EvalNode("/users").String()
	userNodeToString := parser.EvalNode("/users/user").String()
	carsNodeToString := parser.EvalNode("/users/user/cars").String()

	usersNodeToStringExpect := "<users>\n" +
		"  <user>\n" +
		"    <id>\n" +
		"      100\n" +
		"    </id>\n" +
		"    <name>\n" +
		"      Tom\n" +
		"    </name>\n" +
		"    <age>\n" +
		"      30\n" +
		"    </age>\n" +
		"    <cars>\n" +
		"      <car index=\"1\">\n" +
		"        BMW\n" +
		"      </car>\n" +
		"      <car index=\"2\">\n" +
		"        Audi\n" +
		"      </car>\n" +
		"      <car index=\"3\">\n" +
		"        Benz\n" +
		"      </car>\n" +
		"    </cars>\n" +
		"  </user>\n" +
		"</users>\n"

	userNodeToStringExpect := "<user>\n" +
		"  <id>\n" +
		"    100\n" +
		"  </id>\n" +
		"  <name>\n" +
		"    Tom\n" +
		"  </name>\n" +
		"  <age>\n" +
		"    30\n" +
		"  </age>\n" +
		"  <cars>\n" +
		"    <car index=\"1\">\n" +
		"      BMW\n" +
		"    </car>\n" +
		"    <car index=\"2\">\n" +
		"      Audi\n" +
		"    </car>\n" +
		"    <car index=\"3\">\n" +
		"      Benz\n" +
		"    </car>\n" +
		"  </cars>\n" +
		"</user>\n"

	carsNodeToStringExpect := "<cars>\n" +
		"  <car index=\"1\">\n" +
		"    BMW\n" +
		"  </car>\n" +
		"  <car index=\"2\">\n" +
		"    Audi\n" +
		"  </car>\n" +
		"  <car index=\"3\">\n" +
		"    Benz\n" +
		"  </car>\n" +
		"</cars>\n"

	if usersNodeToString != usersNodeToStringExpect {
		t.Fatalf("users string mismatch:\n%s", usersNodeToString)
	}
	if userNodeToString != userNodeToStringExpect {
		t.Fatalf("user string mismatch:\n%s", userNodeToString)
	}
	if carsNodeToString != carsNodeToStringExpect {
		t.Fatalf("cars string mismatch:\n%s", carsNodeToString)
	}
}

func TestXNodeToString(t *testing.T) {
	xmlText := "<mapper>\n" +
		"  <select id='select' resultType='map'>\n" +
		"    select\n" +
		"    <var set='foo' value='bar' />\n" +
		"      ID,\n" +
		"      NAME\n" +
		"    from STUDENT\n" +
		"    <where>\n" +
		"      <if test=\"name != null\">\n" +
		"        NAME = #{name}\n" +
		"      </if>\n" +
		"      and DISABLED = false\n" +
		"    </where>\n" +
		"    order by ID\n" +
		"    <choose>\n" +
		"      <when test='limit10'>\n" +
		"        limit 10\n" +
		"      </when>\n" +
		"      <otherwise>limit 20</otherwise>\n" +
		"    </choose>\n" +
		"  </select>\n" +
		"</mapper>\n"

	expected := "<select id=\"select\" resultType=\"map\">\n" +
		"  select\n" +
		"  <var set=\"foo\" value=\"bar\" />\n" +
		"  ID,\n" +
		"      NAME\n" +
		"    from STUDENT\n" +
		"  <where>\n" +
		"    <if test=\"name != null\">\n" +
		"      NAME = #{name}\n" +
		"    </if>\n" +
		"    and DISABLED = false\n" +
		"  </where>\n" +
		"  order by ID\n" +
		"  <choose>\n" +
		"    <when test=\"limit10\">\n" +
		"      limit 10\n" +
		"    </when>\n" +
		"    <otherwise>\n" +
		"      limit 20\n" +
		"    </otherwise>\n" +
		"  </choose>\n" +
		"</select>\n"

	parser := mustNewXPathParser(t, xmlText)
	selectNode := parser.EvalNode("/mapper/select")
	if actual := selectNode.String(); actual != expected {
		t.Fatalf("select string mismatch:\n%s", actual)
	}
}

func TestXNodeShouldPreserveQualifiedNamesInStringAndSourceXMLLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<m:mapper
		  xmlns:m="http://mybatis.org/schema/mybatis-mapper"
		  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
		  xsi:schemaLocation="http://mybatis.org/schema/mybatis-mapper https://mybatis.org/schema/mybatis-mapper.xsd">
		  <m:select id="selectOne">select 1</m:select>
		</m:mapper>
	`, XPathParserConfig{Resource: "qualified_names_xnode.xml"})

	mapper := parser.EvalNode("/mapper")
	if mapper == nil {
		t.Fatal("mapper node should not be nil")
	}
	if mapper.GetName() != "m:mapper" {
		t.Fatalf("mapper name = %q, want %q", mapper.GetName(), "m:mapper")
	}

	expected := "<m:mapper xmlns:m=\"http://mybatis.org/schema/mybatis-mapper\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:schemaLocation=\"http://mybatis.org/schema/mybatis-mapper https://mybatis.org/schema/mybatis-mapper.xsd\">\n" +
		"  <m:select id=\"selectOne\">\n" +
		"    select 1\n" +
		"  </m:select>\n" +
		"</m:mapper>\n"

	if actual := mapper.String(); actual != expected {
		t.Fatalf("mapper string mismatch:\n%s", actual)
	}
	if actual := mapper.GetSourceXML(); actual != expected {
		t.Fatalf("mapper source xml mismatch:\n%s", actual)
	}
}

func TestXNodeToStringVariables(t *testing.T) {
	src := "<root attr='${x}'>y = ${y}<sub attr='${y}'>x = ${x}</sub></root>"
	expected := "<root attr=\"foo\">\n  y = bar\n  <sub attr=\"bar\">\n    x = foo\n  </sub>\n</root>\n"
	vars := NewProperties()
	vars.Put("x", "foo")
	vars.Put("y", "bar")
	parser := mustNewXPathParser(t, src, XPathParserConfig{Variables: vars})
	selectNode := parser.EvalNode("/root")
	if actual := selectNode.String(); actual != expected {
		t.Fatalf("root string mismatch:\n%s", actual)
	}
}

func TestXNodeShouldEvaluateRelativeNodesLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
<mapper>
  <resultMap id="blogResult" type="Blog">
    <id property="id" column="id"/>
    <result property="title" column="title"/>
    <association property="author">
      <id property="id" column="author_id"/>
    </association>
  </resultMap>
</mapper>
`, XPathParserConfig{Resource: "xnode_relative_eval.xml"})

	resultMapNode := parser.EvalNode("/mapper/resultMap")
	if resultMapNode == nil {
		t.Fatal("resultMap node should not be nil")
	}

	associationNode := resultMapNode.EvalNode("association")
	if associationNode == nil {
		t.Fatal("association node should be resolved relative to resultMap")
	}
	if associationNode.GetStringAttribute("property") != "author" {
		t.Fatalf("association property = %q, want %q", associationNode.GetStringAttribute("property"), "author")
	}

	associationIDNode := associationNode.EvalNode("id")
	if associationIDNode == nil {
		t.Fatal("association id node should be resolved relative to association")
	}
	if associationIDNode.GetStringAttribute("column") != "author_id" {
		t.Fatalf("association id column = %q, want %q", associationIDNode.GetStringAttribute("column"), "author_id")
	}

	parentIDNode := associationNode.EvalNode("../id")
	if parentIDNode == nil {
		t.Fatal("parent id node should be resolved with parent axis")
	}
	if parentIDNode.GetStringAttribute("column") != "id" {
		t.Fatalf("parent id column = %q, want %q", parentIDNode.GetStringAttribute("column"), "id")
	}

	relativeNodes := resultMapNode.EvalNodes("id|association/id")
	if len(relativeNodes) != 2 {
		t.Fatalf("relative node count = %d, want %d", len(relativeNodes), 2)
	}
	if relativeNodes[0].GetStringAttribute("column") != "id" {
		t.Fatalf("relativeNodes[0] column = %q, want %q", relativeNodes[0].GetStringAttribute("column"), "id")
	}
	if relativeNodes[1].GetStringAttribute("column") != "author_id" {
		t.Fatalf("relativeNodes[1] column = %q, want %q", relativeNodes[1].GetStringAttribute("column"), "author_id")
	}
}

func TestXNodeShouldPreserveRelativeDocumentOrderForGroupedUnionsFromAttributeContextLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
<mapper namespace="test.Mapper">
  <resultMap id="blogResult" type="Blog">
    <id property="id" column="id"/>
    <result property="title" column="title"/>
    <association property="author">
      <id property="id" column="author_id"/>
    </association>
    <collection property="posts">
      <id property="id" column="post_id"/>
    </collection>
  </resultMap>
</mapper>
`, XPathParserConfig{Resource: "xnode_relative_attribute_union_eval.xml"})

	attributeNode := parser.EvalNode("/mapper/resultMap/association/@property")
	if attributeNode == nil {
		t.Fatal("association property attribute should not be nil")
	}

	relativeNodes := attributeNode.EvalNodes("../@*|../*")
	if len(relativeNodes) != 2 {
		t.Fatalf("relative node count = %d, want %d", len(relativeNodes), 2)
	}
	if relativeNodes[0].GetNode().GetNodeType() != ATTRIBUTE_NODE {
		t.Fatalf("relativeNodes[0] node type = %d, want %d", relativeNodes[0].GetNode().GetNodeType(), ATTRIBUTE_NODE)
	}
	if relativeNodes[0].GetName() != "property" || relativeNodes[0].GetStringBody("") != "author" {
		t.Fatalf(
			"relativeNodes[0] = (%q, %q), want (%q, %q)",
			relativeNodes[0].GetName(),
			relativeNodes[0].GetStringBody(""),
			"property",
			"author",
		)
	}
	if relativeNodes[1].GetName() != "id" || relativeNodes[1].GetStringAttribute("column") != "author_id" {
		t.Fatalf(
			"relativeNodes[1] = (%q, %q), want (%q, %q)",
			relativeNodes[1].GetName(),
			relativeNodes[1].GetStringAttribute("column"),
			"id",
			"author_id",
		)
	}

	firstGroupedNode := attributeNode.EvalNode("(.|..)[1]")
	if firstGroupedNode == nil {
		t.Fatal("first grouped node should not be nil")
	}
	if firstGroupedNode.GetName() != "association" || firstGroupedNode.GetStringAttribute("property") != "author" {
		t.Fatalf(
			"first grouped node = (%q, %q), want (%q, %q)",
			firstGroupedNode.GetName(),
			firstGroupedNode.GetStringAttribute("property"),
			"association",
			"author",
		)
	}

	lastGroupedNode := attributeNode.EvalNode("(.|..)[last()]")
	if lastGroupedNode == nil {
		t.Fatal("last grouped node should not be nil")
	}
	if lastGroupedNode.GetNode().GetNodeType() != ATTRIBUTE_NODE {
		t.Fatalf("last grouped node type = %d, want %d", lastGroupedNode.GetNode().GetNodeType(), ATTRIBUTE_NODE)
	}
	if lastGroupedNode.GetName() != "property" || lastGroupedNode.GetStringBody("") != "author" {
		t.Fatalf(
			"last grouped node = (%q, %q), want (%q, %q)",
			lastGroupedNode.GetName(),
			lastGroupedNode.GetStringBody(""),
			"property",
			"author",
		)
	}

	secondGroupedNode := attributeNode.EvalNode("(../../@id|../../id|../id)[2]")
	if secondGroupedNode == nil {
		t.Fatal("second grouped node should not be nil")
	}
	if secondGroupedNode.GetName() != "id" || secondGroupedNode.GetStringAttribute("column") != "id" {
		t.Fatalf(
			"second grouped node = (%q, %q), want (%q, %q)",
			secondGroupedNode.GetName(),
			secondGroupedNode.GetStringAttribute("column"),
			"id",
			"id",
		)
	}

	lastUnionNode := attributeNode.EvalNode("(../../@id|../../id|../id)[last()]")
	if lastUnionNode == nil {
		t.Fatal("last union node should not be nil")
	}
	if lastUnionNode.GetName() != "id" || lastUnionNode.GetStringAttribute("column") != "author_id" {
		t.Fatalf(
			"last union node = (%q, %q), want (%q, %q)",
			lastUnionNode.GetName(),
			lastUnionNode.GetStringAttribute("column"),
			"id",
			"author_id",
		)
	}
}

func TestXNodeStringShouldRenderEntityReferenceChildren(t *testing.T) {
	parser := mustNewXPathParser(t, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper [
  <!ENTITY custom "HELLO">
]>
<mapper>
  <select id="selectOne">pre&custom;post</select>
</mapper>`, XPathParserConfig{Resource: "xnode_entity_reference_string.xml"})

	selectNode := parser.EvalNode("/mapper/select")
	if selectNode == nil {
		t.Fatal("select node should not be nil")
	}

	const want = "<select id=\"selectOne\">\n  pre\n  &custom;\n  post\n</select>\n"
	if actual := selectNode.String(); actual != want {
		t.Fatalf("String() = %q, want %q", actual, want)
	}
}

func TestXNodeShouldGetChildrenAsPropertiesAndCloneAttributes(t *testing.T) {
	vars := NewProperties()
	vars.Put("user", "arthur")
	vars.Put("password", "secret")

	parser := mustNewXPathParser(t, `
		<configuration>
		  <properties resource="jdbc.properties" owner="${user}">
		    <property name="username" value="${user}" />
		    <property name="password" value="${password}" />
		    <property name="ignoredWithoutValue" />
		    <property value="ignoredWithoutName" />
		  </properties>
		</configuration>
	`, XPathParserConfig{
		Resource:  "xnode_children_as_properties.xml",
		Variables: vars,
	})

	propertiesNode := parser.EvalNode("/configuration/properties")
	if propertiesNode == nil {
		t.Fatal("properties node should not be nil")
	}

	attributes := propertiesNode.GetAttributes()
	if len(attributes) != 2 {
		t.Fatalf("attribute count = %d, want 2", len(attributes))
	}
	if actual := attributes.GetProperty("owner"); actual != "arthur" {
		t.Fatalf("owner attribute = %q, want %q", actual, "arthur")
	}

	attributes.Put("owner", "mutated")
	if actual := propertiesNode.GetStringAttribute("owner"); actual != "arthur" {
		t.Fatalf("GetAttributes should return a clone, owner = %q", actual)
	}

	childProperties := propertiesNode.GetChildrenAsProperties()
	if len(childProperties) != 2 {
		t.Fatalf("child property count = %d, want 2", len(childProperties))
	}
	if actual := childProperties.GetProperty("username"); actual != "arthur" {
		t.Fatalf("username property = %q, want %q", actual, "arthur")
	}
	if actual := childProperties.GetProperty("password"); actual != "secret" {
		t.Fatalf("password property = %q, want %q", actual, "secret")
	}

	childProperties.Put("username", "changed")
	if actual := propertiesNode.GetChildrenAsProperties().GetProperty("username"); actual != "arthur" {
		t.Fatalf("GetChildrenAsProperties should return a fresh map, username = %q", actual)
	}
}

func TestXNodeShouldUseStringAttributeSupplierLazilyLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<mapper>
		  <resultMap id="blogResult" type="Blog">
		    <association property="author" javaType="Author" />
		  </resultMap>
		</mapper>
	`, XPathParserConfig{Resource: "xnode_supplier_fallback.xml"})

	resultMapNode := parser.EvalNode("/mapper/resultMap")
	if resultMapNode == nil {
		t.Fatal("resultMap node should not be nil")
	}

	supplierCalled := false
	if actual := resultMapNode.GetStringAttributeWithSupplier("id", func() string {
		supplierCalled = true
		return "fallback"
	}); actual != "blogResult" {
		t.Fatalf("explicit id = %q, want %q", actual, "blogResult")
	}
	if supplierCalled {
		t.Fatal("supplier should not be called when the attribute exists")
	}

	associationNode := parser.EvalNode("/mapper/resultMap/association")
	if associationNode == nil {
		t.Fatal("association node should not be nil")
	}

	supplierCalls := 0
	if actual := associationNode.GetStringAttributeWithSupplier("id", func() string {
		supplierCalls++
		return associationNode.GetValueBasedIdentifier()
	}); actual != "mapper_resultMap[blogResult]_association[author]" {
		t.Fatalf("fallback id = %q, want %q", actual, "mapper_resultMap[blogResult]_association[author]")
	}
	if supplierCalls != 1 {
		t.Fatalf("supplier call count = %d, want 1", supplierCalls)
	}
}

func TestXNodeGetStringAttributeWithSupplierShouldPanicWhenSupplierNil(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<mapper>
		  <resultMap id="blogResult" type="Blog">
		    <association property="author" javaType="Author" />
		  </resultMap>
		</mapper>
	`, XPathParserConfig{Resource: "xnode_supplier_nil.xml"})

	resultMapNode := parser.EvalNode("/mapper/resultMap")
	if resultMapNode == nil {
		t.Fatal("resultMap node should not be nil")
	}
	if actual := resultMapNode.GetStringAttributeWithSupplier("id", nil); actual != "blogResult" {
		t.Fatalf("existing id = %q, want %q", actual, "blogResult")
	}

	associationNode := parser.EvalNode("/mapper/resultMap/association")
	if associationNode == nil {
		t.Fatal("association node should not be nil")
	}

	assertPanicContains(
		t,
		`parsing: nil default supplier for missing attribute "id"`,
		func() {
			_ = associationNode.GetStringAttributeWithSupplier("id", nil)
		},
	)
}

func TestXNodeShouldPropagateValidationAndOnlyExposeElementChildren(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<mapper>
		  ignored text
		  <select id="selectOne">select 1</select>
		  <sql id="fragment">id</sql>
		</mapper>
	`, XPathParserConfig{
		Validation: true,
		Resource:   "xnode_validation_enabled.xml",
	})

	mapperNode := parser.EvalNode("/mapper")
	if mapperNode == nil {
		t.Fatal("mapper node should not be nil")
	}
	if !mapperNode.IsValidationEnabled() {
		t.Fatal("mapper node should report validation enabled")
	}

	children := mapperNode.GetChildren()
	if len(children) != 2 {
		t.Fatalf("child count = %d, want 2", len(children))
	}
	if children[0].GetName() != "select" || children[1].GetName() != "sql" {
		t.Fatalf("child names = (%q, %q), want (%q, %q)", children[0].GetName(), children[1].GetName(), "select", "sql")
	}

	for _, child := range children {
		if !child.IsValidationEnabled() {
			t.Fatalf("%s child should inherit validation flag", child.GetName())
		}
		parent := child.GetParent()
		if parent == nil {
			t.Fatalf("%s parent should not be nil", child.GetName())
		}
		if parent.GetName() != "mapper" {
			t.Fatalf("%s parent name = %q, want %q", child.GetName(), parent.GetName(), "mapper")
		}
		if !parent.IsValidationEnabled() {
			t.Fatalf("%s parent should retain validation flag", child.GetName())
		}
	}
}

func TestXNodeShouldUseDOMAttributeParentSemanticsLikeMyBatis(t *testing.T) {
	parser := mustLoadParser(t)

	attributeNode := parser.EvalNode("/employee/@id")
	if attributeNode == nil {
		t.Fatal("attribute node should not be nil")
	}
	if attributeNode.GetNode().GetParentNode() != nil {
		t.Fatal("attribute GetParentNode should be nil like DOM Attr nodes")
	}
	if attributeNode.GetParent() != nil {
		t.Fatal("attribute XNode parent should be nil like DOM Attr nodes")
	}
	if actual := attributeNode.GetPath(); actual != "" {
		t.Fatalf("attribute path = %q, want empty", actual)
	}
	if actual := attributeNode.GetValueBasedIdentifier(); actual != "id" {
		t.Fatalf("attribute identifier = %q, want %q", actual, "id")
	}
}

func TestXNodeShouldDistinguishMissingAndExplicitEmptyBooleanAttributesLikeMyBatis(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<mapper flushCache="false" useCache="TrUe" resultOrdered="">
		  <select id="selectOne">select 1</select>
		</mapper>
	`, XPathParserConfig{Resource: "xnode_boolean_attributes.xml"})

	mapperNode := parser.EvalNode("/mapper")
	if mapperNode == nil {
		t.Fatal("mapper node should not be nil")
	}

	if actual, ok := mapperNode.LookupBooleanAttribute("flushCache"); !ok || actual {
		t.Fatalf("LookupBooleanAttribute(flushCache) = (%t, %t), want (false, true)", actual, ok)
	}
	if actual := mapperNode.GetBooleanAttribute("flushCache", true); actual {
		t.Fatalf("GetBooleanAttribute(flushCache, true) = %t, want false", actual)
	}

	if actual, ok := mapperNode.LookupBooleanAttribute("useCache"); !ok || !actual {
		t.Fatalf("LookupBooleanAttribute(useCache) = (%t, %t), want (true, true)", actual, ok)
	}
	if actual := mapperNode.GetBooleanAttribute("useCache", false); !actual {
		t.Fatalf("GetBooleanAttribute(useCache, false) = %t, want true", actual)
	}

	if actual, ok := mapperNode.LookupBooleanAttribute("resultOrdered"); !ok || actual {
		t.Fatalf("LookupBooleanAttribute(resultOrdered) = (%t, %t), want (false, true)", actual, ok)
	}
	if actual := mapperNode.GetBooleanAttribute("resultOrdered", true); actual {
		t.Fatalf("GetBooleanAttribute(resultOrdered, true) = %t, want false", actual)
	}

	if actual, ok := mapperNode.LookupBooleanAttribute("missing"); ok || actual {
		t.Fatalf("LookupBooleanAttribute(missing) = (%t, %t), want (false, false)", actual, ok)
	}
	if actual := mapperNode.GetBooleanAttribute("missing", true); !actual {
		t.Fatalf("GetBooleanAttribute(missing, true) = %t, want true", actual)
	}
}

func TestXNodeShouldFailFastOnInvalidNumericAttributesAndPreserveLookupDefaults(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<mapper fetchSize="200" timeout="-10" invalidInt="abc" longTimeout="9223372036854775807" invalidLong="abc">
		  <select id="selectOne">select 1</select>
		</mapper>
	`, XPathParserConfig{Resource: "xnode_numeric_attributes.xml"})

	mapperNode := parser.EvalNode("/mapper")
	if mapperNode == nil {
		t.Fatal("mapper node should not be nil")
	}

	if actual, ok := mapperNode.LookupIntAttribute("fetchSize"); !ok || actual != 200 {
		t.Fatalf("LookupIntAttribute(fetchSize) = (%d, %t), want (200, true)", actual, ok)
	}
	if actual := mapperNode.GetIntAttribute("fetchSize", 1); actual != 200 {
		t.Fatalf("GetIntAttribute(fetchSize, 1) = %d, want 200", actual)
	}
	if actual, ok := mapperNode.LookupIntAttribute("missingInt"); ok || actual != 0 {
		t.Fatalf("LookupIntAttribute(missingInt) = (%d, %t), want (0, false)", actual, ok)
	}
	if actual := mapperNode.GetIntAttribute("missingInt", 1); actual != 1 {
		t.Fatalf("GetIntAttribute(missingInt, 1) = %d, want 1", actual)
	}
	if actual, ok := mapperNode.LookupIntAttribute("invalidInt"); ok || actual != 0 {
		t.Fatalf("LookupIntAttribute(invalidInt) = (%d, %t), want (0, false)", actual, ok)
	}
	assertPanicContains(t, "invalid syntax", func() {
		_ = mapperNode.GetIntAttribute("invalidInt")
	})

	if actual, ok := mapperNode.LookupLongAttribute("longTimeout"); !ok || actual != 9223372036854775807 {
		t.Fatalf("LookupLongAttribute(longTimeout) = (%d, %t), want (9223372036854775807, true)", actual, ok)
	}
	if actual := mapperNode.GetLongAttribute("longTimeout", 1); actual != 9223372036854775807 {
		t.Fatalf("GetLongAttribute(longTimeout, 1) = %d, want 9223372036854775807", actual)
	}
	if actual, ok := mapperNode.LookupLongAttribute("missingLong"); ok || actual != 0 {
		t.Fatalf("LookupLongAttribute(missingLong) = (%d, %t), want (0, false)", actual, ok)
	}
	if actual := mapperNode.GetLongAttribute("missingLong", 1); actual != 1 {
		t.Fatalf("GetLongAttribute(missingLong, 1) = %d, want 1", actual)
	}
	if actual, ok := mapperNode.LookupLongAttribute("invalidLong"); ok || actual != 0 {
		t.Fatalf("LookupLongAttribute(invalidLong) = (%d, %t), want (0, false)", actual, ok)
	}
	assertPanicContains(t, "invalid syntax", func() {
		_ = mapperNode.GetLongAttribute("invalidLong")
	})
}
