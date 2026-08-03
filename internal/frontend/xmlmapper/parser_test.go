package xmlmapper

import (
	"reflect"
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func TestParseMapperBuildsDynamicIRAndExpandsInclude(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="columns">id, name</sql>
  <resultMap id="UserResult" type="User">
    <id property="ID" column="id"/>
    <result property="Name" column="name"/>
  </resultMap>
  <select id="Search" resultMap="UserResult">
    SELECT <include refid="columns"/> FROM users
    <where>
      <if test="params.Name != ''">AND name = #{params.Name}</if>
      <foreach collection="params.IDs" item="id" open="(" separator="," close=")">
        #{id}
      </foreach>
    </where>
  </select>
</mapper>`), "user_mapper.xml")
	if err != nil {
		t.Fatal(err)
	}
	if mapper.Namespace != "UserMapper" || len(mapper.Statements) != 1 {
		t.Fatalf("mapper = %#v", mapper)
	}
	if len(mapper.ResultMaps["UserMapper.UserResult"].Fields) != 2 {
		t.Fatalf("result map = %#v", mapper.ResultMaps["UserMapper.UserResult"])
	}

	statement := mapper.Statements[0]
	var foundTrim, foundForeach bool
	for _, node := range statement.Body {
		trim, ok := node.(*ir.Trim)
		if !ok {
			continue
		}
		foundTrim = true
		for _, child := range trim.Body {
			if _, ok := child.(*ir.Foreach); ok {
				foundForeach = true
			}
		}
	}
	if !foundTrim || !foundForeach {
		t.Fatalf("dynamic body = %#v", statement.Body)
	}
	if sqlText := flattenTestNodes(statement.Body); !strings.Contains(sqlText, "id, name") {
		t.Fatalf("include was not expanded: %q", sqlText)
	}
}

func TestParseMapperPreservesTypedTextExpression(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <select id="Unsafe">SELECT * FROM users ORDER BY ${column}</select>
</mapper>`), "unsafe.xml")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(mapper.Statements) != 1 {
		t.Fatalf("statements = %d", len(mapper.Statements))
	}
	var expression *ir.TextExpression
	for _, node := range mapper.Statements[0].Body {
		if current, ok := node.(*ir.TextExpression); ok {
			expression = current
			break
		}
	}
	if expression == nil || expression.Expression != "column" {
		t.Fatalf("text expression = %#v", expression)
	}
}

func TestParseMapperPreservesEscapedDollarPlaceholder(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <select id="Literal" resultType="string">SELECT $$\${literal}$$</select>
</mapper>`), "escaped_dollar.xml")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(mapper.Statements) != 1 {
		t.Fatalf("statements = %d", len(mapper.Statements))
	}
	if sqlText := flattenTestNodes(mapper.Statements[0].Body); !strings.Contains(sqlText, `$${literal}$$`) {
		t.Fatalf("SQL text = %q", sqlText)
	}
	for _, node := range mapper.Statements[0].Body {
		if expression, ok := node.(*ir.TextExpression); ok {
			t.Fatalf("escaped placeholder became text expression %#v", expression)
		}
	}
}

func TestParseMapperDoesNotSubstituteEscapedDollarPlaceholder(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="literal">
    SELECT '${value}' AS expanded, '\${value}' AS literal
  </sql>
  <select id="Get" resultType="string">
    <include refid="literal">
      <property name="value" value="expanded"/>
    </include>
  </select>
</mapper>`), "escaped_property.xml")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sqlText := flattenTestNodes(mapper.Statements[0].Body)
	for _, expected := range []string{`'expanded' AS expanded`, `'${value}' AS literal`} {
		if !strings.Contains(sqlText, expected) {
			t.Fatalf("SQL text %q does not contain %q", sqlText, expected)
		}
	}
	for _, node := range mapper.Statements[0].Body {
		if expression, ok := node.(*ir.TextExpression); ok {
			t.Fatalf("escaped placeholder became text expression %#v", expression)
		}
	}
}

func TestJavaXMLScriptBuilderTestShouldThrowIfUnknownElementFound(t *testing.T) {
	_, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <select id="Search" resultType="User">
    SELECT id FROM users
    <choose>
      <when test="true">AND id = 1</when>
      <otherwize>AND id &gt; 0</otherwize>
    </choose>
  </select>
</mapper>`), "XMLScriptBuilderTest.xml")
	if err == nil || !strings.Contains(err.Error(), "choose only accepts when and otherwise") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseChooseEnforcesMyBatisDTDOrder(t *testing.T) {
	testCases := []struct {
		name    string
		choose  string
		message string
	}{
		{
			name: "when after otherwise",
			choose: `<choose>
  <otherwise>AND id = 0</otherwise>
  <when test="enabled">AND id = 1</when>
</choose>`,
			message: "choose requires when elements to appear before otherwise",
		},
		{
			name: "when after empty otherwise",
			choose: `<choose>
  <otherwise/>
  <when test="enabled">AND id = 1</when>
</choose>`,
			message: "choose requires when elements to appear before otherwise",
		},
		{
			name: "multiple empty otherwise elements",
			choose: `<choose>
  <otherwise/>
  <otherwise/>
</choose>`,
			message: "choose allows only one otherwise",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(`<mapper namespace="UserMapper">
  <select id="Search" resultType="User">
    SELECT id FROM users
    `+testCase.choose+`
  </select>
</mapper>`), "choose_order.xml")
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestParseChooseAllowsMyBatisDTDOptionalBranches(t *testing.T) {
	testCases := []struct {
		name          string
		choose        string
		wantOtherwise bool
	}{
		{
			name:          "otherwise only",
			choose:        `<choose><otherwise>WHERE id = 1</otherwise></choose>`,
			wantOtherwise: true,
		},
		{
			name:   "empty choose",
			choose: `<choose/>`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mapper, err := Parse(strings.NewReader(`<mapper namespace="UserMapper">
  <select id="Search" resultType="User">
    SELECT id FROM users
    `+testCase.choose+`
  </select>
</mapper>`), "choose_optional.xml")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var choose *ir.Choose
			for _, node := range mapper.Statements[0].Body {
				if current, ok := node.(*ir.Choose); ok {
					choose = current
					break
				}
			}
			if choose == nil {
				t.Fatal("choose node was not parsed")
			}
			if len(choose.When) != 0 {
				t.Fatalf("when nodes = %#v, want none", choose.When)
			}
			if got := choose.Otherwise != nil; got != testCase.wantOtherwise {
				t.Fatalf("otherwise present = %v, want %v", got, testCase.wantOtherwise)
			}
		})
	}
}

func TestParseMapperRejectsUnsupportedGoFirstXML(t *testing.T) {
	testCases := []struct {
		name    string
		xml     string
		message string
	}{
		{
			name: "top-level element",
			xml: `
<mapper namespace="UserMapper">
  <cache/>
</mapper>`,
			message: "unsupported mapper element <cache>",
		},
		{
			name: "statement attribute",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" parameterType="int" resultType="User">
    SELECT id FROM users
  </select>
</mapper>`,
			message: `<select> does not support attribute "parameterType"`,
		},
		{
			name: "result mapping attribute",
			xml: `
<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <result property="Name" column="name" typeHandler="StringHandler"/>
  </resultMap>
</mapper>`,
			message: `<result> does not support attribute "typeHandler"`,
		},
		{
			name: "foreach unsupported attribute",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT id FROM users WHERE id IN
    <foreach collection="ids" item="id" optional="true">#{id}</foreach>
  </select>
</mapper>`,
			message: `<foreach> does not support attribute "optional"`,
		},
		{
			name: "when outside choose",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT id FROM users
    <when test="enabled">WHERE enabled = TRUE</when>
  </select>
</mapper>`,
			message: "unsupported SQL element <when>",
		},
		{
			name: "parameter jdbc type",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT id FROM users WHERE id = #{id,jdbcType=BIGINT}
  </select>
</mapper>`,
			message: `parameter option "jdbcType" is not supported`,
		},
		{
			name: "invalid sensitive value",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT id FROM users WHERE email = #{email,sensitive=yes}
  </select>
</mapper>`,
			message: `parameter option "sensitive" must be true or false`,
		},
		{
			name: "include property without value",
			xml: `
<mapper namespace="UserMapper">
  <sql id="columns">${column}</sql>
  <select id="Get" resultType="User">
    SELECT <include refid="columns"><property name="column"/></include>
    FROM users
  </select>
</mapper>`,
			message: "include property requires value",
		},
		{
			name: "result map and type together",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" resultMap="UserResult" resultType="User">
    SELECT id FROM users
  </select>
</mapper>`,
			message: "resultMap and resultType are mutually exclusive",
		},
		{
			name: "select execution result mode",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" result="rows">
    SELECT id FROM users
  </select>
</mapper>`,
			message: `select does not support result mode "rows"`,
		},
		{
			name: "command result type",
			xml: `
<mapper namespace="UserMapper">
  <update id="Update" resultType="User">
    UPDATE users SET active = TRUE
  </update>
</mapper>`,
			message: "update does not support resultMap or resultType",
		},
		{
			name: "scalar result map",
			xml: `
<mapper namespace="UserMapper">
  <select id="Count" result="scalar" resultMap="CountResult">
    SELECT COUNT(*) AS count FROM users
  </select>
</mapper>`,
			message: "result=scalar cannot be combined with resultMap",
		},
		{
			name: "mapper text content",
			xml: `
<mapper namespace="UserMapper">
  unexpected
</mapper>`,
			message: "<mapper> only accepts child elements and whitespace",
		},
		{
			name: "result map text content",
			xml: `
<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    unexpected
    <id property="ID" column="id"/>
  </resultMap>
</mapper>`,
			message: "<resultMap> only accepts child elements and whitespace",
		},
		{
			name: "result leaf content",
			xml: `
<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id property="ID" column="id">unexpected</id>
  </resultMap>
</mapper>`,
			message: "<id> must be empty",
		},
		{
			name: "bind body",
			xml: `
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    <bind name="normalized" value="name">unexpected</bind>
    SELECT id FROM users
  </select>
</mapper>`,
			message: "<bind> must be empty",
		},
		{
			name: "include text content",
			xml: `
<mapper namespace="UserMapper">
  <sql id="column">id</sql>
  <select id="Get" resultType="User">
    SELECT <include refid="column">unexpected</include> FROM users
  </select>
</mapper>`,
			message: "<include> only accepts child elements and whitespace",
		},
		{
			name: "include property body",
			xml: `
<mapper namespace="UserMapper">
  <sql id="column">${column}</sql>
  <select id="Get" resultType="User">
    SELECT <include refid="column"><property name="column" value="id">unexpected</property></include>
    FROM users
  </select>
</mapper>`,
			message: "<property> must be empty",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(testCase.xml), "strict.xml")
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestParseMapperRejectsDuplicateSQLFragments(t *testing.T) {
	_, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="columns">id</sql>
  <sql id="columns">id, name</sql>
</mapper>`), "duplicate_fragment.xml")
	if err == nil || !strings.Contains(err.Error(), `duplicate sql fragment "UserMapper.columns"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestParseMapperRejectsEquivalentSQLFragmentIDsInEitherOrder(t *testing.T) {
	testCases := []struct {
		name      string
		fragments string
	}{
		{
			name: "qualified then bare",
			fragments: `<sql id="UserMapper.columns">qualified</sql>
  <sql id="columns">bare</sql>`,
		},
		{
			name: "bare then qualified",
			fragments: `<sql id="columns">bare</sql>
  <sql id="UserMapper.columns">qualified</sql>`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(`<mapper namespace="UserMapper">
  `+testCase.fragments+`
</mapper>`), "equivalent_fragment_ids.xml")
			if err == nil || !strings.Contains(err.Error(), `duplicate sql fragment "UserMapper.columns"`) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestParseMapperResolvesBareAndQualifiedSQLFragmentReferences(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="columns">id, name</sql>
  <select id="Bare" resultType="User">SELECT <include refid="columns"/> FROM users</select>
  <select id="Qualified" resultType="User">SELECT <include refid="UserMapper.columns"/> FROM users</select>
</mapper>`), "canonical_fragment_references.xml")
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range mapper.Statements {
		if sqlText := flattenTestNodes(statement.Body); !strings.Contains(sqlText, "id, name") {
			t.Fatalf("statement %q SQL = %q", statement.ID, sqlText)
		}
	}
}

func TestParseMapperCanonicalizesResultMapDefinitionsAndReferences(t *testing.T) {
	for _, definitionID := range []string{"UserResult", "UserMapper.UserResult"} {
		t.Run(definitionID, func(t *testing.T) {
			mapper, err := Parse(strings.NewReader(`<mapper namespace="UserMapper">
  <resultMap id="`+definitionID+`" type="User">
    <id column="id" property="ID"/>
  </resultMap>
  <select id="Bare" resultMap="UserResult">SELECT id FROM users</select>
  <select id="Qualified" resultMap="UserMapper.UserResult">SELECT id FROM users</select>
</mapper>`), "canonical_result_map_references.xml")
			if err != nil {
				t.Fatal(err)
			}
			if len(mapper.ResultMaps) != 1 {
				t.Fatalf("result maps = %#v", mapper.ResultMaps)
			}
			resultMap := mapper.ResultMaps["UserMapper.UserResult"]
			if resultMap == nil || resultMap.ID != "UserMapper.UserResult" {
				t.Fatalf("canonical result map = %#v", resultMap)
			}
			for _, statement := range mapper.Statements {
				if statement.ResultMap != "UserMapper.UserResult" {
					t.Fatalf("statement %q resultMap = %q", statement.ID, statement.ResultMap)
				}
			}
		})
	}
}

func TestParseMapperRejectsEquivalentResultMapIDsInEitherOrder(t *testing.T) {
	testCases := []struct {
		name       string
		resultMaps string
	}{
		{
			name: "qualified then bare",
			resultMaps: `<resultMap id="UserMapper.UserResult" type="User"><id column="id" property="ID"/></resultMap>
  <resultMap id="UserResult" type="User"><id column="id" property="ID"/></resultMap>`,
		},
		{
			name: "bare then qualified",
			resultMaps: `<resultMap id="UserResult" type="User"><id column="id" property="ID"/></resultMap>
  <resultMap id="UserMapper.UserResult" type="User"><id column="id" property="ID"/></resultMap>`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(`<mapper namespace="UserMapper">
  `+testCase.resultMaps+`
</mapper>`), "equivalent_result_map_ids.xml")
			if err == nil || !strings.Contains(err.Error(), `duplicate resultMap "UserMapper.UserResult"`) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestParseMapperRejectsResultMapQualifiedWithAnotherNamespace(t *testing.T) {
	_, err := Parse(strings.NewReader(`<mapper namespace="UserMapper">
  <resultMap id="OtherMapper.UserResult" type="User">
    <id column="id" property="ID"/>
  </resultMap>
</mapper>`), "foreign_result_map_namespace.xml")
	if err == nil || !strings.Contains(err.Error(), `dots are not allowed in resultMap ids`) {
		t.Fatalf("error = %v", err)
	}
}

func TestParseMapperRejectsSQLFragmentQualifiedWithAnotherNamespace(t *testing.T) {
	_, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="OtherMapper.columns">id</sql>
</mapper>`), "foreign_fragment_namespace.xml")
	if err == nil || !strings.Contains(err.Error(), `dots are not allowed in SQL fragment ids`) {
		t.Fatalf("error = %v", err)
	}
}

func TestParseMapperRejectsDuplicateStatements(t *testing.T) {
	_, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">SELECT id FROM users</select>
  <select id="Get" resultType="User">SELECT id FROM archived_users</select>
</mapper>`), "duplicate_statement.xml")
	if err == nil || !strings.Contains(err.Error(), `duplicate statement "Get"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestParseMapperAcceptsGoFirstParameterOptions(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT id FROM users WHERE email = #{email,sensitive=true}
  </select>
</mapper>`), "parameter_options.xml")
	if err != nil {
		t.Fatal(err)
	}

	var parameter *ir.Parameter
	for _, node := range mapper.Statements[0].Body {
		if current, ok := node.(*ir.Parameter); ok {
			parameter = current
			break
		}
	}
	if parameter == nil || !parameter.Sensitive {
		t.Fatalf("parameter = %#v", parameter)
	}
}

func TestParseMapperAcceptsDMLReturningResultMapping(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <resultMap id="CreatedUserResult" type="CreatedUser">
    <id property="ID" column="id"/>
    <result property="CreatedAt" column="created_at"/>
  </resultMap>
  <insert id="Create" result="returning" resultMap="CreatedUserResult">
    INSERT INTO users (name) VALUES (#{name})
    RETURNING id, created_at
  </insert>
  <update id="Rename" result="returning" resultType="User">
    UPDATE users SET name = #{name} WHERE id = #{id}
    RETURNING id, name
  </update>
</mapper>`), "returning.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(mapper.Statements) != 2 {
		t.Fatalf("statements = %d", len(mapper.Statements))
	}
	if mapper.Statements[0].ResultMode != ir.ResultReturning ||
		mapper.Statements[0].ResultMap != "UserMapper.CreatedUserResult" {
		t.Fatalf("insert statement = %#v", mapper.Statements[0])
	}
	if mapper.Statements[1].ResultMode != ir.ResultReturning ||
		mapper.Statements[1].ResultType != "User" {
		t.Fatalf("update statement = %#v", mapper.Statements[1])
	}
}

func TestParseMapperAcceptsOptionalSelect(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <select id="Find" result="optional" resultType="User">
    SELECT id FROM users WHERE id = #{id}
  </select>
</mapper>`), "optional.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(mapper.Statements) != 1 {
		t.Fatalf("statements = %d, want 1", len(mapper.Statements))
	}
	statement := mapper.Statements[0]
	if statement.ResultMode != ir.ResultOptional || statement.ResultType != "User" {
		t.Fatalf("optional statement = %#v", statement)
	}
}

func TestParseMapperRejectsInvalidDMLResultModes(t *testing.T) {
	testCases := []struct {
		name    string
		xml     string
		message string
	}{
		{
			name: "select returning",
			xml: `<mapper namespace="UserMapper">
  <select id="Get" result="returning">SELECT id FROM users</select>
</mapper>`,
			message: `select does not support result mode "returning"`,
		},
		{
			name: "update lastid",
			xml: `<mapper namespace="UserMapper">
  <update id="Update" result="lastid">UPDATE users SET active = TRUE</update>
</mapper>`,
			message: "result=lastid is only supported by insert",
		},
		{
			name: "insert optional",
			xml: `<mapper namespace="UserMapper">
  <insert id="Create" result="optional">INSERT INTO users (name) VALUES (#{name})</insert>
</mapper>`,
			message: `insert does not support result mode "optional"`,
		},
		{
			name: "non-returning result type",
			xml: `<mapper namespace="UserMapper">
  <delete id="Delete" resultType="User">DELETE FROM users</delete>
</mapper>`,
			message: "delete does not support resultMap or resultType",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(testCase.xml), "invalid_result.xml")
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestParseMapperExpandsParameterizedInclude(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="column">${alias}.id AS id</sql>
  <select id="Get" resultType="User">
    SELECT <include refid="column"><property name="alias" value="u"/></include>
    FROM users u
  </select>
</mapper>`), "parameterized_include.xml")
	if err != nil {
		t.Fatal(err)
	}
	if sqlText := flattenTestNodes(mapper.Statements[0].Body); !strings.Contains(sqlText, "u.id AS id") {
		t.Fatalf("expanded SQL = %q", sqlText)
	}
}

func TestParseMapperExpandsNestedIncludesWithInheritedProperties(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="idColumn">${prefix}id AS id</sql>
  <sql id="namedColumn">${prefix}${name} AS ${alias}</sql>
  <sql id="nested">
    <include refid="${target}">
      <property name="prefix" value="${tableAlias}."/>
    </include>
  </sql>
  <select id="Get" resultType="User">
    SELECT
    <include refid="nested">
      <property name="target" value="idColumn"/>
      <property name="tableAlias" value="u"/>
    </include>,
    <include refid="namedColumn">
      <property name="prefix" value="u."/>
      <property name="name" value="display_name"/>
      <property name="alias" value="name"/>
    </include>
    FROM users u
  </select>
</mapper>`), "nested_include.xml")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := flattenTestNodes(mapper.Statements[0].Body)
	for _, expected := range []string{"u.id AS id", "u.display_name AS name"} {
		if !strings.Contains(sqlText, expected) {
			t.Fatalf("expanded SQL %q does not contain %q", sqlText, expected)
		}
	}
}

func TestParseMapperAllowsEscapedPlaceholderInIncludeProperty(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="literal">SELECT '${value}' AS placeholder</sql>
  <select id="Get" resultType="User">
    <include refid="literal">
      <property name="value" value="\\\${column}"/>
    </include>
  </select>
</mapper>`), "escaped_include_property.xml")
	if err != nil {
		t.Fatal(err)
	}
	if sqlText := flattenTestNodes(mapper.Statements[0].Body); !strings.Contains(sqlText, `'${column}'`) {
		t.Fatalf("expanded SQL = %q", sqlText)
	}
}

func TestParseMapperSubstitutesIncludePropertiesInDynamicAttributes(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <sql id="conditionalColumn">
    <if test="${enabled}">${column}</if>
  </sql>
  <select id="Get" resultType="User">
    SELECT
    <include refid="conditionalColumn">
      <property name="enabled" value="params.IncludeName"/>
      <property name="column" value="name"/>
    </include>
    FROM users
  </select>
</mapper>`), "attribute_include.xml")
	if err != nil {
		t.Fatal(err)
	}
	var conditional *ir.If
	for _, node := range mapper.Statements[0].Body {
		if current, ok := node.(*ir.If); ok {
			conditional = current
		}
	}
	if conditional == nil || conditional.Test != "params.IncludeName" ||
		!strings.Contains(flattenTestNodes(conditional.Body), "name") {
		t.Fatalf("conditional include = %#v", conditional)
	}
}

func TestParseSetTrimsBothLeadingAndTrailingCommas(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <update id="Update">
    UPDATE users
    <set>, name = #{name},</set>
  </update>
</mapper>`), "set.xml")
	if err != nil {
		t.Fatal(err)
	}
	var trim *ir.Trim
	for _, node := range mapper.Statements[0].Body {
		if current, ok := node.(*ir.Trim); ok {
			trim = current
		}
	}
	if trim == nil || len(trim.PrefixOverrides) != 1 || trim.PrefixOverrides[0] != "," ||
		len(trim.SuffixOverrides) != 1 || trim.SuffixOverrides[0] != "," {
		t.Fatalf("set trim = %#v", trim)
	}
}

func TestParseCustomTrimAddsSQLNodeAffixSpacing(t *testing.T) {
	mapper, err := Parse(strings.NewReader(`
<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT id FROM users
    <trim prefix="WHERE" prefixOverrides="AND |OR " suffix="RETURNING">
      AND id = #{id}
    </trim>
  </select>
</mapper>`), "trim.xml")
	if err != nil {
		t.Fatal(err)
	}
	var trim *ir.Trim
	for _, node := range mapper.Statements[0].Body {
		if current, ok := node.(*ir.Trim); ok {
			trim = current
		}
	}
	if trim == nil || trim.Prefix != "WHERE " || trim.Suffix != " RETURNING" {
		t.Fatalf("trim = %#v", trim)
	}
	if len(trim.PrefixOverrides) != 2 ||
		trim.PrefixOverrides[0] != "AND " ||
		trim.PrefixOverrides[1] != "OR " {
		t.Fatalf("prefix overrides = %#v", trim.PrefixOverrides)
	}
}

func TestSplitOverridesPreservesMyBatisWhitespace(t *testing.T) {
	overrides := splitOverrides("AND |OR\n|| OR ")
	expected := []string{"AND ", "OR\n", " OR "}
	if !reflect.DeepEqual(overrides, expected) {
		t.Fatalf("overrides = %#v, want %#v", overrides, expected)
	}
}

func TestParseMapperRejectsInvalidIncludes(t *testing.T) {
	testCases := []struct {
		name    string
		xml     string
		message string
	}{
		{
			name: "duplicate property",
			xml: `
<mapper namespace="UserMapper">
  <sql id="column">${alias}.id AS id</sql>
  <select id="Get" resultType="User">
    SELECT
    <include refid="column">
      <property name="alias" value="u"/>
      <property name="alias" value="other"/>
    </include>
    FROM users u
  </select>
</mapper>`,
			message: `duplicate include property "alias"`,
		},
		{
			name: "missing outer property",
			xml: `
<mapper namespace="UserMapper">
  <sql id="column">${column}</sql>
  <select id="Get" resultType="User">
    SELECT
    <include refid="column">
      <property name="column" value="${missing}.id"/>
    </include>
    FROM users
  </select>
</mapper>`,
			message: `include property "column" contains unresolved placeholder "${missing}"`,
		},
		{
			name: "circular reference",
			xml: `
<mapper namespace="UserMapper">
  <sql id="a"><include refid="b"/></sql>
  <sql id="b"><include refid="a"/></sql>
  <select id="Get" resultType="User">
    SELECT <include refid="a"/> FROM users
  </select>
</mapper>`,
			message: `circular SQL include`,
		},
		{
			name: "circular reference using bare and qualified ids",
			xml: `
<mapper namespace="UserMapper">
  <sql id="a"><include refid="UserMapper.b"/></sql>
  <sql id="b"><include refid="a"/></sql>
  <select id="Get" resultType="User">
    SELECT <include refid="UserMapper.a"/> FROM users
  </select>
</mapper>`,
			message: `circular SQL include "UserMapper.a"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(testCase.xml), "include.xml")
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func flattenTestNodes(nodes []ir.Node) string {
	var result strings.Builder
	for _, node := range nodes {
		switch current := node.(type) {
		case *ir.Text:
			result.WriteString(current.Value)
		case *ir.If:
			result.WriteString(flattenTestNodes(current.Body))
		case *ir.Foreach:
			result.WriteString(flattenTestNodes(current.Body))
		case *ir.Trim:
			result.WriteString(flattenTestNodes(current.Body))
		}
	}
	return result.String()
}
