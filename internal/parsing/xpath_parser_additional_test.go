package parsing

import "testing"

func TestXPathParserShouldRejectUnsupportedUnaryPlusAndScientificNotationLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item id="1">A</item><item id="2">B</item></root>`,
		XPathParserConfig{Resource: "xpath_invalid_numeric_tokens.xml"},
	)

	assertPanicContains(t, "A location path was expected, but the following token was encountered", func() {
		_ = parser.EvalDouble("+1")
	})
	assertPanicContains(t, "A location path was expected, but the following token was encountered", func() {
		_ = parser.EvalDouble("(+1)")
	})
	assertPanicContains(t, "A location path was expected, but the following token was encountered", func() {
		_ = parser.EvalDouble("count(/root/item[+1=1])")
	})
	assertPanicContains(t, "A location path was expected, but the following token was encountered", func() {
		_ = parser.EvalDouble("count(/root/item[+1])")
	})
	assertPanicContains(t, "A location path was expected, but the following token was encountered", func() {
		_ = parser.EvalDouble("count(/root/item[position()=+2])")
	})
	assertPanicContains(t, "A location path was expected, but the following token was encountered", func() {
		_ = parser.EvalDouble("count(/root/item[position()<=+2])")
	})
	assertPanicContains(t, "could not be formatted to a number!", func() {
		_ = parser.EvalDouble("1e1")
	})
	assertPanicContains(t, "could not be formatted to a number!", func() {
		_ = parser.EvalDouble("1e-1")
	})
}

func TestXPathParserShouldSupportAdditionalNodeSetComparisonSemanticsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1">A</item>
		  <item id="2">B</item>
		  <item id="3">C</item>
		  <n>2</n>
		</root>
	`, XPathParserConfig{Resource: "xpath_node_set_comparisons.xml"})

	if actual := parser.EvalBoolean("/root/item = /root/item[1]"); actual != true {
		t.Fatalf("EvalBoolean(/root/item = /root/item[1]) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/item != /root/item[1]"); actual != true {
		t.Fatalf("EvalBoolean(/root/item != /root/item[1]) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/item[1] != /root/item[1]"); actual != false {
		t.Fatalf("EvalBoolean(/root/item[1] != /root/item[1]) = %t, want false", actual)
	}
	if actual := parser.EvalString("/root/item = /root/item[1]"); actual != "true" {
		t.Fatalf("EvalString(/root/item = /root/item[1]) = %q, want %q", actual, "true")
	}
	if actual := parser.EvalDouble("/root/item[1] != /root/item[1]"); actual != 0 {
		t.Fatalf("EvalDouble(/root/item[1] != /root/item[1]) = %v, want 0", actual)
	}

	if actual := parser.EvalBoolean("/root/item = true()"); actual != true {
		t.Fatalf("EvalBoolean(/root/item = true()) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/item = false()"); actual != false {
		t.Fatalf("EvalBoolean(/root/item = false()) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("/root/missing = false()"); actual != true {
		t.Fatalf("EvalBoolean(/root/missing = false()) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/missing != false()"); actual != false {
		t.Fatalf("EvalBoolean(/root/missing != false()) = %t, want false", actual)
	}

	if actual := parser.EvalBoolean("/root/n < /root/item/@id"); actual != true {
		t.Fatalf("EvalBoolean(/root/n < /root/item/@id) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/n >= /root/item/@id"); actual != true {
		t.Fatalf("EvalBoolean(/root/n >= /root/item/@id) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/missing < /root/item/@id"); actual != false {
		t.Fatalf("EvalBoolean(/root/missing < /root/item/@id) = %t, want false", actual)
	}
}

func TestXPathParserShouldSupportReversedPositionComparisonsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item>A</item><item>B</item><item>C</item></root>`,
		XPathParserConfig{Resource: "xpath_reversed_position_predicates.xml"},
	)

	if actual := parser.EvalDouble("count(/root/item[last() > position()])"); actual != 2 {
		t.Fatalf("count(/root/item[last() > position()]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[last() > position()]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[last() > position()]) = %q, want %q", actual, "A")
	}

	if actual := parser.EvalDouble("count(/root/item[2 >= position()])"); actual != 2 {
		t.Fatalf("count(/root/item[2 >= position()]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[2 >= position()]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[2 >= position()]) = %q, want %q", actual, "A")
	}

	if actual := parser.EvalDouble("count(/root/item[2 < position()])"); actual != 1 {
		t.Fatalf("count(/root/item[2 < position()]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[2 < position()]"); actual != "C" {
		t.Fatalf("EvalString(/root/item[2 < position()]) = %q, want %q", actual, "C")
	}

	if actual := parser.EvalDouble("count(/root/item[last() = position()])"); actual != 1 {
		t.Fatalf("count(/root/item[last() = position()]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[last() = position()]"); actual != "C" {
		t.Fatalf("EvalString(/root/item[last() = position()]) = %q, want %q", actual, "C")
	}

	if actual := parser.EvalDouble("count(/root/item[last() != position()])"); actual != 2 {
		t.Fatalf("count(/root/item[last() != position()]) = %v, want 2", actual)
	}
	if actual := parser.EvalString("/root/item[last() != position()]"); actual != "A" {
		t.Fatalf("EvalString(/root/item[last() != position()]) = %q, want %q", actual, "A")
	}

	if actual := parser.EvalDouble("count(/root/item[last() <= position()])"); actual != 1 {
		t.Fatalf("count(/root/item[last() <= position()]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("/root/item[last() <= position()]"); actual != "C" {
		t.Fatalf("EvalString(/root/item[last() <= position()]) = %q, want %q", actual, "C")
	}
}

func TestXPathParserShouldSupportGroupedNodeSetPredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item>A</item><item>B</item><item>C</item></root>`,
		XPathParserConfig{Resource: "xpath_grouped_node_set_predicates.xml"},
	)

	if actual := parser.EvalDouble("count((/root/item))"); actual != 3 {
		t.Fatalf("count((/root/item)) = %v, want 3", actual)
	}
	if actual := parser.EvalDouble("count((/root/item)[1])"); actual != 1 {
		t.Fatalf("count((/root/item)[1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("string((/root/item)[1])"); actual != "A" {
		t.Fatalf("string((/root/item)[1]) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count((/root/item)[last()])"); actual != 1 {
		t.Fatalf("count((/root/item)[last()]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("string((/root/item)[last()])"); actual != "C" {
		t.Fatalf("string((/root/item)[last()]) = %q, want %q", actual, "C")
	}
	if actual := parser.EvalBoolean("(/root/item)[1] = /root/item[1]"); actual != true {
		t.Fatalf("EvalBoolean((/root/item)[1] = /root/item[1]) = %t, want true", actual)
	}
}

func TestXPathParserShouldRespectTopLevelUnionBoundariesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item>A</item><item>B</item><alt>C</alt><item>A|B</item></root>`,
		XPathParserConfig{Resource: "xpath_union_boundaries.xml"},
	)

	if actual := parser.EvalDouble("count((/root/item | /root/alt))"); actual != 4 {
		t.Fatalf("count((/root/item | /root/alt)) = %v, want 4", actual)
	}
	if actual := parser.EvalDouble("count((/root/item | /root/alt)[1])"); actual != 1 {
		t.Fatalf("count((/root/item | /root/alt)[1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("string((/root/item | /root/alt)[last()])"); actual != "A|B" {
		t.Fatalf("string((/root/item | /root/alt)[last()]) = %q, want %q", actual, "A|B")
	}
	if actual := parser.EvalDouble(`count(/root/item[.='A|B'])`); actual != 1 {
		t.Fatalf("count(/root/item[.='A|B']) = %v, want 1", actual)
	}
}

func TestXPathParserShouldSupportGroupedNodeSetPathContinuationsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item id="1"><name>A</name></item>
		  <item id="2"><name>B</name></item>
		  <item id="3"><name>C</name></item>
		  <alt id="4"><name>D</name></alt>
		</root>
	`, XPathParserConfig{Resource: "xpath_grouped_node_set_path_continuations.xml"})

	if actual := parser.EvalString("string((/root/item)[1]/@id)"); actual != "1" {
		t.Fatalf("string((/root/item)[1]/@id) = %q, want %q", actual, "1")
	}
	if actual := parser.EvalDouble("count((/root/item)[position() < last()]/name)"); actual != 2 {
		t.Fatalf("count((/root/item)[position() < last()]/name) = %v, want 2", actual)
	}
	if actual := parser.EvalString("string((/root/item)[last()]/name)"); actual != "C" {
		t.Fatalf("string((/root/item)[last()]/name) = %q, want %q", actual, "C")
	}
	if actual := parser.EvalString("string((/root/item | /root/alt)[last()]/@id)"); actual != "4" {
		t.Fatalf("string((/root/item | /root/alt)[last()]/@id) = %q, want %q", actual, "4")
	}
}

func TestXPathParserShouldTreatNumericPredicateExpressionsAsPositionTestsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		  <alt>D</alt>
		</root>
	`, XPathParserConfig{Resource: "xpath_numeric_predicate_expressions.xml"})

	if actual := parser.EvalDouble("count(/root/item[last()-1])"); actual != 1 {
		t.Fatalf("count(/root/item[last()-1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("string(/root/item[last()-1])"); actual != "B" {
		t.Fatalf("string(/root/item[last()-1]) = %q, want %q", actual, "B")
	}

	if actual := parser.EvalDouble("count((/root/item | /root/alt)[last()-1])"); actual != 1 {
		t.Fatalf("count((/root/item | /root/alt)[last()-1]) = %v, want 1", actual)
	}
	if actual := parser.EvalString("string((/root/item | /root/alt)[last()-1])"); actual != "C" {
		t.Fatalf("string((/root/item | /root/alt)[last()-1]) = %q, want %q", actual, "C")
	}
}

func TestXPathParserShouldRejectBareGroupedUnionBooleanComparisonsOnLeftLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><a>A</a><b>B</b></root>`,
		XPathParserConfig{Resource: "xpath_grouped_union_boolean_comparison.xml"},
	)

	assertPanicContains(t, "cannot be cast to class com.sun.org.apache.xpath.internal.objects.XNodeSet", func() {
		_ = parser.EvalBoolean("(/root/a | /root/b) = true()")
	})
	assertPanicContains(t, "cannot be cast to class com.sun.org.apache.xpath.internal.objects.XNodeSet", func() {
		_ = parser.EvalBoolean("(/root/a | /root/b) != false()")
	})
	assertPanicContains(t, "cannot be cast to class com.sun.org.apache.xpath.internal.objects.XNodeSet", func() {
		_ = parser.EvalBoolean("(/root/a | /root/b) < true()")
	})
	assertPanicContains(t, "cannot be cast to class com.sun.org.apache.xpath.internal.objects.XNodeSet", func() {
		_ = parser.EvalBoolean("(/root/a | /root/b) <= true()")
	})
	assertPanicContains(t, "cannot be cast to class com.sun.org.apache.xpath.internal.objects.XNodeSet", func() {
		_ = parser.EvalBoolean("(/root/a | /root/b) > false()")
	})
	assertPanicContains(t, "cannot be cast to class com.sun.org.apache.xpath.internal.objects.XNodeSet", func() {
		_ = parser.EvalBoolean("(/root/a | /root/b) >= false()")
	})

	if actual := parser.EvalBoolean("true() = (/root/a | /root/b)"); actual != true {
		t.Fatalf("true() = (/root/a | /root/b) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("(/root/a | /root/b)[1] = true()"); actual != true {
		t.Fatalf("(/root/a | /root/b)[1] = true() = %t, want true", actual)
	}
}

func TestXPathParserShouldUseOriginalGroupedResultSizeAcrossSequentialGroupedPredicatesLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <item>C</item>
		  <alt>D</alt>
		</root>
	`, XPathParserConfig{Resource: "xpath_grouped_sequential_predicates.xml"})

	if actual := parser.EvalDouble("count((/root/item)[1][last()])"); actual != 0 {
		t.Fatalf("count((/root/item)[1][last()]) = %v, want 0", actual)
	}
	if actual := parser.EvalDouble("count((/root/item)[1][position()=last()])"); actual != 0 {
		t.Fatalf("count((/root/item)[1][position()=last()]) = %v, want 0", actual)
	}
	if actual := parser.EvalString("string((/root/item)[1][position()<last()])"); actual != "A" {
		t.Fatalf("string((/root/item)[1][position()<last()]) = %q, want %q", actual, "A")
	}
	if actual := parser.EvalDouble("count((/root/item)[last()][last()])"); actual != 0 {
		t.Fatalf("count((/root/item)[last()][last()]) = %v, want 0", actual)
	}
	if actual := parser.EvalString("string((/root/item)[last()][position()<last()])"); actual != "C" {
		t.Fatalf("string((/root/item)[last()][position()<last()]) = %q, want %q", actual, "C")
	}
	if actual := parser.EvalString("string((/root/item | /root/alt)[last()][position()<last()])"); actual != "D" {
		t.Fatalf("string((/root/item | /root/alt)[last()][position()<last()]) = %q, want %q", actual, "D")
	}
}

func TestXPathParserShouldCoerceNodeSetsToBooleanForRelationalComparisonsLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t,
		`<root><item>A</item></root>`,
		XPathParserConfig{Resource: "xpath_node_set_boolean_relational_comparisons.xml"},
	)

	if actual := parser.EvalBoolean("/root/item <= true()"); actual != true {
		t.Fatalf("EvalBoolean(/root/item <= true()) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("true() <= /root/item"); actual != true {
		t.Fatalf("EvalBoolean(true() <= /root/item) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("false() < /root/item"); actual != true {
		t.Fatalf("EvalBoolean(false() < /root/item) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/item > false()"); actual != true {
		t.Fatalf("EvalBoolean(/root/item > false()) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/missing < true()"); actual != true {
		t.Fatalf("EvalBoolean(/root/missing < true()) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("/root/missing >= false()"); actual != true {
		t.Fatalf("EvalBoolean(/root/missing >= false()) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("true() < /root/item"); actual != false {
		t.Fatalf("EvalBoolean(true() < /root/item) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("/root/item < true()"); actual != false {
		t.Fatalf("EvalBoolean(/root/item < true()) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("/root/missing > false()"); actual != false {
		t.Fatalf("EvalBoolean(/root/missing > false()) = %t, want false", actual)
	}
}

func TestXPathParserShouldUseRightNodeSetTruthinessForBareGroupedUnionEqualitiesOnLeftLikeJDKXPath(t *testing.T) {
	parser := mustNewXPathParser(t, `
		<root>
		  <item>A</item>
		  <item>B</item>
		  <group attr1="A" attr2="B">
		    <leaf>X</leaf>
		    <leaf>Y</leaf>
		  </group>
		</root>
	`, XPathParserConfig{Resource: "xpath_grouped_union_nodeset_equality.xml"})

	if actual := parser.EvalBoolean("(/root/group/@* | /root/group/*) = /root/item[last()]"); actual != true {
		t.Fatalf("EvalBoolean((/root/group/@* | /root/group/*) = /root/item[last()]) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("(/root/group/leaf | /root/group/leaf[2]) = /root/item"); actual != true {
		t.Fatalf("EvalBoolean((/root/group/leaf | /root/group/leaf[2]) = /root/item) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("(/root/missing | /root/missing2) = /root/item"); actual != true {
		t.Fatalf("EvalBoolean((/root/missing | /root/missing2) = /root/item) = %t, want true", actual)
	}
	if actual := parser.EvalBoolean("(/root/group/@* | /root/group/*) = /root/missing"); actual != false {
		t.Fatalf("EvalBoolean((/root/group/@* | /root/group/*) = /root/missing) = %t, want false", actual)
	}
	if actual := parser.EvalBoolean("(/root/missing | /root/missing2) = /root/missing"); actual != false {
		t.Fatalf("EvalBoolean((/root/missing | /root/missing2) = /root/missing) = %t, want false", actual)
	}
}
