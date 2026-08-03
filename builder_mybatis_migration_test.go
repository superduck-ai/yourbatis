package yourbatis

import "testing"

// TestJavaDynamicSqlSourceTest keeps the portable cases from MyBatis'
// DynamicSqlSourceTest visible by their original Java method names. The
// generated-mapper conformance tests exercise the same runtime primitives
// through XML and generated Go code.
func TestJavaDynamicSqlSourceTest(t *testing.T) {
	t.Run("shouldDemonstrateSimpleExpectedTextWithNoLoopsOrConditionals", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		appendJavaSQLNode(builder, "SELECT * FROM BLOG")
		assertJavaSQL(t, builder, "SELECT * FROM BLOG")
	})

	t.Run("shouldDemonstrateMultipartExpectedTextWithNoLoopsOrConditionals", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		appendJavaSQLNode(builder, "SELECT * FROM BLOG")
		appendJavaSQLNode(builder, "WHERE ID = ?")
		assertJavaSQL(t, builder, "SELECT * FROM BLOG WHERE ID = ?")
	})

	t.Run("shouldConditionallyIncludeWhere", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		appendJavaSQLNode(builder, "SELECT * FROM BLOG")
		if true {
			appendJavaSQLNode(builder, "WHERE ID = ?")
		}
		assertJavaSQL(t, builder, "SELECT * FROM BLOG WHERE ID = ?")
	})

	t.Run("shouldConditionallyExcludeWhere", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		appendJavaSQLNode(builder, "SELECT * FROM BLOG")
		if false {
			appendJavaSQLNode(builder, "WHERE ID = ?")
		}
		assertJavaSQL(t, builder, "SELECT * FROM BLOG")
	})

	t.Run("shouldConditionallyDefault", func(t *testing.T) {
		assertJavaSQL(t, buildJavaChoose(false, false), "SELECT * FROM BLOG WHERE CATEGORY = 'DEFAULT'")
	})

	t.Run("shouldConditionallyChooseFirst", func(t *testing.T) {
		assertJavaSQL(t, buildJavaChoose(true, false), "SELECT * FROM BLOG WHERE CATEGORY = ?")
	})

	t.Run("shouldConditionallyChooseSecond", func(t *testing.T) {
		assertJavaSQL(t, buildJavaChoose(false, true), "SELECT * FROM BLOG WHERE CATEGORY = 'NONE'")
	})

	t.Run("shouldTrimWHEREInsteadOfANDForFirstCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   and ID = ?  "),
			"SELECT * FROM BLOG WHERE  ID = ?",
		)
	})

	t.Run("shouldTrimWHEREANDWithLFForFirstCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   and\n ID = ?  "),
			"SELECT * FROM BLOG WHERE \n ID = ?",
		)
	})

	t.Run("shouldTrimWHEREANDWithCRLFForFirstCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   and\r\n ID = ?  "),
			"SELECT * FROM BLOG WHERE \r\n ID = ?",
		)
	})

	t.Run("shouldTrimWHEREANDWithTABForFirstCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   and\t ID = ?  "),
			"SELECT * FROM BLOG WHERE \t ID = ?",
		)
	})

	t.Run("shouldTrimWHEREORWithLFForFirstCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   or\n ID = ?  "),
			"SELECT * FROM BLOG WHERE \n ID = ?",
		)
	})

	t.Run("shouldTrimWHEREORWithCRLFForFirstCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   or\r\n ID = ?  "),
			"SELECT * FROM BLOG WHERE \r\n ID = ?",
		)
	})

	t.Run("shouldTrimWHEREORWithTABForFirstCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   or\t ID = ?  "),
			"SELECT * FROM BLOG WHERE \t ID = ?",
		)
	})

	t.Run("shouldTrimWHEREInsteadOfORForSecondCondition", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   or NAME = ?  "),
			"SELECT * FROM BLOG WHERE  NAME = ?",
		)
	})

	t.Run("shouldTrimWHEREInsteadOfANDForBothConditions", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaWhere("   and ID = ?   ", "OR NAME = ?  "),
			"SELECT * FROM BLOG WHERE  ID = ?    OR NAME = ?",
		)
	})

	t.Run("shouldTrimNoWhereClause", func(t *testing.T) {
		assertJavaSQL(t, buildJavaWhere(), "SELECT * FROM BLOG")
	})

	t.Run("shouldTrimSETInsteadOfCOMMAForBothConditions", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaSet(" ID = ?, ", " NAME = ?, "),
			"UPDATE BLOG SET ID = ?,   NAME = ?",
		)
	})

	t.Run("shouldTrimCommaAfterSET", func(t *testing.T) {
		assertJavaSQL(
			t,
			buildJavaSet(", NAME = ?"),
			"UPDATE BLOG SET  NAME = ?",
		)
	})

	t.Run("shouldTrimNoSetClause", func(t *testing.T) {
		assertJavaSQL(t, buildJavaSet(), "UPDATE BLOG")
	})

	t.Run("shouldIterateOnceForEachItemInCollection", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		appendJavaSQLNode(builder, "SELECT * FROM BLOG WHERE ID in")
		collection := NewBuilder(DialectQuestion)
		for _, value := range []string{"one", "two", "three"} {
			item := NewBuilder(DialectQuestion)
			item.Arg("item", value)
			collection.AppendSeparated(item, " AND ")
		}
		if collection.HasContent() {
			builder.Text(" ( ")
			builder.Append(collection)
			builder.Text(" )")
		}

		bound := builder.Build()
		if bound.SQL != "SELECT * FROM BLOG WHERE ID in ( ? AND ? AND ? )" {
			t.Fatalf("SQL = %q", bound.SQL)
		}
		if len(bound.Args) != 3 {
			t.Fatalf("Args = %#v", bound.Args)
		}
		for index, expected := range []string{"one", "two", "three"} {
			if bound.Args[index].Name != "item" || bound.Args[index].Value != expected {
				t.Fatalf("Args = %#v", bound.Args)
			}
		}
	})

	t.Run("shouldSkipForEachWhenCollectionIsEmpty", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		appendJavaSQLNode(builder, "SELECT * FROM BLOG")
		collection := NewBuilder(DialectQuestion)
		if collection.HasContent() {
			builder.Text(" WHERE id in (")
			builder.Append(collection)
			builder.Text(")")
		}

		bound := builder.Build()
		if bound.SQL != "SELECT * FROM BLOG" || len(bound.Args) != 0 {
			t.Fatalf("bound = %#v", bound)
		}
	})

	t.Run("shouldPerformStrictMatchOnForEachVariableSubstitution", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		builder.Text("INSERT INTO BLOG (ID, NAME, NOTE, COMMENT) VALUES (")
		builder.Arg("uuu.u", "xyz")
		builder.Text(", ")
		builder.Arg("u.id", "bean id")
		builder.Text(", ")
		builder.Arg("u", "bean id")
		builder.Text(", ")
		builder.Arg("u", "bean id")
		builder.Text(")")

		bound := builder.Build()
		expectedNames := []string{"uuu.u", "u.id", "u", "u"}
		if len(bound.Args) != len(expectedNames) {
			t.Fatalf("Args = %#v", bound.Args)
		}
		for index, expected := range expectedNames {
			if bound.Args[index].Name != expected {
				t.Fatalf("Args[%d].Name = %q, want %q", index, bound.Args[index].Name, expected)
			}
		}
	})

	t.Run("cornerCase_ForeachComesFirst", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		collection := NewBuilder(DialectQuestion)
		for _, id := range []string{"bean id 1", "bean id 2"} {
			item := NewBuilder(DialectQuestion)
			item.Arg("b.id", id)
			collection.AppendSeparated(item, ", ")
		}
		if collection.HasContent() {
			builder.Text("(")
			builder.Append(collection)
			builder.Text(")")
		}

		bound := builder.Build()
		if bound.SQL != "(?, ?)" || len(bound.Args) != 2 {
			t.Fatalf("bound = %#v", bound)
		}
		for _, argument := range bound.Args {
			if argument.Name != "b.id" {
				t.Fatalf("Args = %#v", bound.Args)
			}
		}
	})

	t.Run("testShrinkWhitespacesInSql", func(t *testing.T) {
		builder := NewBuilder(DialectQuestion)
		builder.Text("\t\n\n  SELECT * \n        FROM user\n \t        WHERE user_id = 1\n\t  ")
		if actual := builder.Build().SQL; actual != "SELECT * \n        FROM user\n \t        WHERE user_id = 1" {
			t.Fatalf("SQL = %q", actual)
		}
	})
}

func appendJavaSQLNode(builder *Builder, text string) {
	if builder.HasContent() {
		builder.Text(" ")
	}
	builder.Text(text)
}

func buildJavaChoose(first, second bool) *Builder {
	builder := NewBuilder(DialectQuestion)
	appendJavaSQLNode(builder, "SELECT * FROM BLOG")
	switch {
	case first:
		appendJavaSQLNode(builder, "WHERE CATEGORY = ?")
	case second:
		appendJavaSQLNode(builder, "WHERE CATEGORY = 'NONE'")
	default:
		appendJavaSQLNode(builder, "WHERE CATEGORY = 'DEFAULT'")
	}
	return builder
}

func buildJavaWhere(parts ...string) *Builder {
	builder := NewBuilder(DialectQuestion)
	appendJavaSQLNode(builder, "SELECT * FROM BLOG")
	where := NewBuilder(DialectQuestion)
	for _, part := range parts {
		appendJavaSQLNode(where, part)
	}
	builder.AppendTrimmed(where, " WHERE ", myBatisWherePrefixOverrides(), "", nil)
	return builder
}

func buildJavaSet(parts ...string) *Builder {
	builder := NewBuilder(DialectQuestion)
	appendJavaSQLNode(builder, "UPDATE BLOG")
	set := NewBuilder(DialectQuestion)
	for _, part := range parts {
		appendJavaSQLNode(set, part)
	}
	builder.AppendTrimmed(set, " SET ", []string{","}, "", []string{","})
	return builder
}

func assertJavaSQL(t *testing.T, builder *Builder, expected string) {
	t.Helper()
	if actual := builder.Build().SQL; actual != expected {
		t.Fatalf("SQL = %q, want %q", actual, expected)
	}
}
