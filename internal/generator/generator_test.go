package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateResolvesDefaultImportFromDeclaredPackageName(t *testing.T) {
	directory := t.TempDir()
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	writeGeneratorFixture(t, directory, "go.mod", fmt.Sprintf(`module fixture

go 1.26.2

require github.com/superduck-ai/yourbatis v0.0.0

replace github.com/superduck-ai/yourbatis => %q
`, projectRoot))
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import (
	"context"
	"github.com/superduck-ai/yourbatis"
)

type UserMapper interface {
	Delete(ctx context.Context, column yourbatis.Identifier) error
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <delete id="Delete">DELETE FROM users WHERE name = #{column}</delete>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatalf("Generate() with an unaliased runtime import: %v", err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "column yourbatis.Identifier") {
		t.Fatalf("generated source lost the declared package qualifier:\n%s", generated)
	}
}

func TestGenerateRejectsUnknownMapperParameterField(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64
}

type SearchParams struct {
	ID int64
}

type UserMapper interface {
	Get(ctx context.Context, params SearchParams) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT id FROM users WHERE id = #{params.Missing}
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), `type SearchParams has no exported field "Missing"`) {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not be written, stat error = %v", statErr)
	}
}

func TestGenerateRejectsUnresolvedIncludePropertyPlaceholder(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64
}

type UserMapper interface {
	Get(ctx context.Context) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <sql id="column">${column}</sql>
  <select id="Get" resultType="User">
    SELECT <include refid="column">
      <property name="alias" value="u"/>
      <property name="column" value="${alias}.id"/>
    </include>
    FROM users u
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(
		err.Error(),
		`include property "column" contains unresolved placeholder "${alias}"`,
	) {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not be written, stat error = %v", statErr)
	}
}

func TestGenerateRejectsScalarFieldChain(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	Name string
}

type UserMapper interface {
	Get(ctx context.Context, u User) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    SELECT name FROM users WHERE name = #{u.Name.Length}
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), `cannot select field "Length" from scalar type string`) {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not be written, stat error = %v", statErr)
	}
}

func TestGenerateRejectsNullComparisonForNonNilableField(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type SearchParams struct {
	Name string
}

type UserMapper interface {
	Search(ctx context.Context, params SearchParams) error
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Search">
    SELECT id FROM users
    <if test="params.Name != null">WHERE name = #{params.Name}</if>
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), "type string cannot be compared with nil") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not be written, stat error = %v", statErr)
	}
}

func TestGenerateAvoidsInternalIdentifierCollisions(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64
}

type UserMapper interface {
	Get(
		ctx context.Context,
		dialect int64,
		builder string,
		mapper int64,
	) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get" resultType="User">
    <bind name="trim1" value="builder"/>
    SELECT id FROM users
    <where>
      <if test="dialect > 0">AND id = #{mapper}</if>
    </where>
    AND name = #{trim1}
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	for _, expected := range []string{
		"func buildUserMapperGet(dialect2 yourbatis.Dialect, dialect int64, builder string, mapper int64)",
		"builder2 := yourbatis.NewBuilder(dialect2)",
		"trim2 := yourbatis.NewBuilder(dialect2)",
		"func (mapper2 *userMapper) Get(",
		"mapper2.executor",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestGeneratePassesDatabaseIDUsedOnlyAsParameter(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type Row struct {
	ID int64
}

type UserMapper interface {
	Get(ctx context.Context) (Row, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get" resultType="Row">
    SELECT id FROM rows WHERE note = #{_databaseId}
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	for _, expected := range []string{
		"func buildUserMapperGet(dialect yourbatis.Dialect, _databaseId string) yourbatis.BoundSQL",
		`builder.Arg("_databaseId", _databaseId)`,
		"yourbatis.ExecutorDatabaseID(mapper.executor)",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestGenerateAvoidsBoundSQLIdentifierCollisions(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Get(ctx context.Context, bound int64) (int64, error)
	GetWithBoundContext(bound context.Context, id int64) (int64, error)
	GetWithPrefixedCollision(
		ctx context.Context,
		bound int64,
		__yourbatisBound int64,
	) (int64, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get">
    SELECT id FROM users WHERE id = #{bound}
  </select>
  <select id="GetWithBoundContext">
    SELECT id FROM users WHERE id = #{id}
  </select>
  <select id="GetWithPrefixedCollision">
    SELECT id FROM users WHERE id = #{bound} OR id = #{__yourbatisBound}
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	for _, expected := range []string{
		"__yourbatisBound := buildUserMapperGet(mapper.executor.Dialect(), bound)",
		"return yourbatis.SelectOne(ctx, mapper.executor, userMapperGetStatement, __yourbatisBound, scanUserMapperGet)",
		"__yourbatisBound := buildUserMapperGetWithBoundContext(mapper.executor.Dialect(), id)",
		"return yourbatis.SelectOne(bound, mapper.executor, userMapperGetWithBoundContextStatement, __yourbatisBound, scanUserMapperGetWithBoundContext)",
		"__yourbatisBound2 := buildUserMapperGetWithPrefixedCollision(mapper.executor.Dialect(), bound, __yourbatisBound)",
		"return yourbatis.SelectOne(ctx, mapper.executor, userMapperGetWithPrefixedCollisionStatement, __yourbatisBound2, scanUserMapperGetWithPrefixedCollision)",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestGenerateRejectsRuntimeImportNameAsParameter(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Delete(ctx context.Context, yourbatis int64) error
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <delete id="Delete">DELETE FROM users WHERE id = #{yourbatis}</delete>
</mapper>
`)

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), `parameter name "yourbatis" is reserved`) {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateRejectsInterfaceBackedForeachWithoutReflection(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Delete(ctx context.Context, values any) error
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <delete id="Delete">
    DELETE FROM users WHERE id IN
    <foreach collection="values" index="index" item="value" open="(" close=")" separator=",">
      #{value}
    </foreach>
  </delete>
</mapper>
`)

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), "interface-backed foreach requires runtime reflection") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateUsesMyBatisOpenCloseSemanticsForFilteredForeach(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Search(ctx context.Context, ids []int64) ([]int64, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Search" resultType="int64">
    SELECT id FROM users
    <where>
      <if test="len(ids) > 0">
        AND id IN
        <foreach collection="ids" item="id" open="(" separator="," close=")">
          <if test="id > 0">#{id}</if>
        </foreach>
      </if>
    </where>
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	if !strings.Contains(source, ".HasIterations()") {
		t.Fatalf("generated foreach should guard open/close by source iterations:\n%s", source)
	}
	if strings.Contains(source, ".HasContent()") {
		t.Fatalf("generated foreach must not guard open/close by rendered content:\n%s", source)
	}
}

func TestGenerateValidatesFlatResultMappings(t *testing.T) {
	testCases := []struct {
		name      string
		models    string
		statement string
		message   string
	}{
		{
			name: "unknown resultMap property",
			models: `type User struct {
	ID int64
}`,
			statement: `
  <resultMap id="UserResult" type="User">
    <id column="id" property="Missing"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">SELECT id FROM users</select>`,
			message: `property "Missing" is not an exported field of User`,
		},
		{
			name: "unknown resultMap type",
			models: `type User struct {
	ID int64
}`,
			statement: `
  <resultMap id="UserResult" type="MissingUser">
    <id column="id" property="ID"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">SELECT id FROM users</select>`,
			message: `references unknown local Go type MissingUser`,
		},
		{
			name: "duplicate resultMap column",
			models: `type User struct {
	ID int64
	Name string
}`,
			statement: `
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
    <result column="ID" property="Name"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">SELECT id FROM users</select>`,
			message: `maps column "ID" more than once`,
		},
		{
			name: "duplicate resultMap property",
			models: `type User struct {
	ID int64
}`,
			statement: `
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
    <result column="legacy_id" property="ID"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">SELECT id FROM users</select>`,
			message: `maps property "ID" more than once`,
		},
		{
			name: "duplicate struct column tags",
			models: `type User struct {
	ID int64 ` + "`db:\"id\"`" + `
	LegacyID int64 ` + "`db:\"ID\"`" + `
}`,
			statement: `
  <select id="Get" resultType="User">SELECT id FROM users</select>`,
			message: `maps column "ID" to both ID and LegacyID`,
		},
		{
			name: "duplicate projection aliases",
			models: `type User struct {
	ID int64
}`,
			statement: `
  <select id="Get" resultType="User">SELECT id, legacy_id AS ID FROM users</select>`,
			message: `projection exposes column "id" more than once`,
		},
		{
			name: "dynamic projection",
			models: `type User struct {
	ID int64
	Name string
}

type Params struct {
	IncludeName bool
}`,
			statement: `
  <select id="Get" resultType="User">
    SELECT id
    <if test="params.IncludeName">, name</if>
    FROM users
  </select>`,
			message: `dynamic <if> cannot change the SELECT projection`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			parameter := ""
			if testCase.name == "dynamic projection" {
				parameter = ", params Params"
			}
			writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

`+testCase.models+`

type UserMapper interface {
	Get(ctx context.Context`+parameter+`) (User, error)
}
`)
			writeGeneratorFixture(
				t,
				directory,
				"user_mapper.xml",
				`<mapper namespace="UserMapper">`+testCase.statement+`
</mapper>`,
			)
			output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

			err := Generate(Config{
				PackageDir:    directory,
				Mapper:        "UserMapper",
				SQLFile:       "user_mapper.xml",
				Output:        output,
				RuntimeImport: "github.com/superduck-ai/yourbatis",
			})
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Fatalf("output should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestGenerateDoesNotResolveForeignResultMapByBareName(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64
}

type UserMapper interface {
	Get(ctx context.Context) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
  </resultMap>
  <select id="Get" resultMap="OtherMapper.UserResult">SELECT id FROM users</select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), `resultMap "OtherMapper.UserResult" not found`) {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not be written, stat error = %v", statErr)
	}
}

func TestGenerateMapsComputedProjectionAliases(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserLabelFields struct {
	UserID int64 `+"`db:\"user_id\"`"+`
	DisplayName string `+"`db:\"display_name\"`"+`
}

type UserLabel UserLabelFields

type UserMapper interface {
	Get(ctx context.Context) (UserLabel, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get" resultType="UserLabel">
    SELECT id AS user_id, UPPER(name) AS display_name
    FROM users
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	for _, expected := range []string{"&result.UserID", "&result.DisplayName"} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestGenerateRejectsAmbiguousBinaryProjection(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type Order struct {
	Price int64 `+"`db:\"price\"`"+`
	Discount int64 `+"`db:\"discount\"`"+`
}

type OrderMapper interface {
	Get(ctx context.Context) (Order, error)
}
`)
	writeGeneratorFixture(t, directory, "order_mapper.xml", `<mapper namespace="OrderMapper">
  <select id="Get" resultType="Order">
    SELECT price - discount FROM orders
  </select>
</mapper>
`)
	output := filepath.Join(directory, "order_mapper.sqlmap.gen.go")

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "OrderMapper",
		SQLFile:       "order_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), "requires an explicit AS alias") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not be written, stat error = %v", statErr)
	}
}

func TestGenerateValidatesScalarResultShape(t *testing.T) {
	testCases := []struct {
		name       string
		imports    string
		resultType string
		resultMode string
		projection string
		message    string
	}{
		{
			name:       "inferred scalar with multiple columns",
			resultType: "int64",
			projection: "id, version",
			message:    "requires exactly one SELECT column, found 2",
		},
		{
			name:       "explicit scalar with multiple columns",
			resultType: "int64",
			resultMode: ` result="scalar"`,
			projection: "id, version",
			message:    "result=scalar requires exactly one SELECT column, found 2",
		},
		{
			name:       "map result",
			resultType: "map[string]any",
			projection: "payload",
			message:    "map return type map[string]any is not supported",
		},
		{
			name:       "explicit scalar map result",
			resultType: "map[string]any",
			resultMode: ` result="scalar"`,
			projection: "payload",
			message:    "map return type map[string]any is not supported",
		},
		{
			name:       "external type with multiple columns",
			imports:    "\nimport \"time\"\n",
			resultType: "time.Time",
			projection: "created_at, updated_at",
			message:    "requires exactly one SELECT column, found 2",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			importBlock := `import "context"`
			if testCase.imports != "" {
				importBlock = `import (
	"context"
	"time"
)`
			}
			writeGeneratorFixture(t, directory, "mapper.go", `package fixture

`+importBlock+`

type UserMapper interface {
	Get(ctx context.Context) (`+testCase.resultType+`, error)
}
`)
			writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get"`+testCase.resultMode+`>
    SELECT `+testCase.projection+` FROM users
  </select>
</mapper>
`)
			output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
			err := Generate(Config{
				PackageDir:    directory,
				Mapper:        "UserMapper",
				SQLFile:       "user_mapper.xml",
				Output:        output,
				RuntimeImport: "github.com/superduck-ai/yourbatis",
			})
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestGenerateAllowsUnaliasedComputedScalarProjection(t *testing.T) {
	testCases := []struct {
		name       string
		resultMode string
		projection string
	}{
		{
			name:       "explicit scalar exists",
			resultMode: ` result="scalar"`,
			projection: `EXISTS (SELECT 1 FROM users WHERE active)`,
		},
		{
			name:       "inferred scalar count",
			projection: `COUNT(*)`,
		},
		{
			name:       "explicit scalar expression with AS inside",
			resultMode: ` result="scalar"`,
			projection: `CAST(COUNT(*) AS BOOLEAN)`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Get(ctx context.Context) (bool, error)
}
`)
			writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get"`+testCase.resultMode+`>
    SELECT `+testCase.projection+`
  </select>
</mapper>
`)
			output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
			if err := Generate(Config{
				PackageDir:    directory,
				Mapper:        "UserMapper",
				SQLFile:       "user_mapper.xml",
				Output:        output,
				RuntimeImport: "github.com/superduck-ai/yourbatis",
			}); err != nil {
				t.Fatal(err)
			}
			generated, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			source := string(generated)
			if !strings.Contains(source, "func scanUserMapperGet(row yourbatis.Scanner) (bool, error)") ||
				!strings.Contains(source, "row.Scan(&result)") {
				t.Fatalf("scalar scanner was not generated correctly:\n%s", source)
			}
		})
	}
}

func TestGenerateRejectsWildcardScalarProjection(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Get(ctx context.Context) (int64, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get" result="scalar">SELECT * FROM users</select>
</mapper>
`)
	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), "SELECT * cannot be statically mapped") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateInfersByteSliceAsOneScalarValue(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Get(ctx context.Context) ([]byte, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get">SELECT payload FROM blobs</select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	if !strings.Contains(source, "func scanUserMapperGet(row yourbatis.Scanner) ([]byte, error)") ||
		!strings.Contains(source, "return yourbatis.SelectOne(") {
		t.Fatalf("[]byte should be generated as one scalar result:\n%s", source)
	}
}

func TestGenerateScansPointerScalarWithoutPreallocation(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Get(ctx context.Context) (*string, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Get" result="scalar">SELECT name FROM users</select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	if !strings.Contains(source, "var result *string") || !strings.Contains(source, "row.Scan(&result)") {
		t.Fatalf("pointer scalar scanner was not generated correctly:\n%s", source)
	}
}

func TestGenerateInfersAndEmitsOptionalSelects(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64
}

type UserMapper interface {
	Find(ctx context.Context, id int64) (User, bool, error)
	FindName(ctx context.Context, id int64) (string, bool, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <select id="Find" resultType="User">
    SELECT id FROM users WHERE id = #{id}
  </select>
  <select id="FindName" result="optional">
    SELECT name FROM users WHERE id = #{id}
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	if count := strings.Count(source, "return yourbatis.SelectOptional("); count != 2 {
		t.Fatalf("SelectOptional call count = %d, want 2:\n%s", count, source)
	}
	for _, expected := range []string{
		"func scanUserMapperFind(row yourbatis.Scanner) (User, error)",
		"func scanUserMapperFindName(row yourbatis.Scanner) (string, error)",
		"func (mapper *userMapper) Find(ctx context.Context, id int64) (User, bool, error)",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestGenerateValidatesOptionalSelectSignature(t *testing.T) {
	testCases := []struct {
		name       string
		results    string
		resultMode string
		message    string
	}{
		{
			name:    "inferred optional requires bool",
			results: "(User, int, error)",
			message: "optional select methods must return (T, bool, error)",
		},
		{
			name:       "explicit optional requires presence result",
			results:    "(User, error)",
			resultMode: ` result="optional"`,
			message:    "result=optional requires (T, bool, error)",
		},
		{
			name:       "explicit optional requires bool",
			results:    "(User, string, error)",
			resultMode: ` result="optional"`,
			message:    "result=optional requires (T, bool, error)",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64
}

type UserMapper interface {
	Find(ctx context.Context) `+testCase.results+`
}
`)
			writeGeneratorFixture(
				t,
				directory,
				"user_mapper.xml",
				`<mapper namespace="UserMapper"><select id="Find"`+
					testCase.resultMode+
					` resultType="User">SELECT id FROM users</select></mapper>`,
			)
			err := Generate(Config{
				PackageDir:    directory,
				Mapper:        "UserMapper",
				SQLFile:       "user_mapper.xml",
				Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
				RuntimeImport: "github.com/superduck-ai/yourbatis",
			})
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestGenerateAlignsForeachMapNestedFilteredAndBindScopes(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type Pair struct {
	Key string
	Value int64
}

type UserMapper interface {
	InsertMap(ctx context.Context, values map[string]int64) error
	InsertNested(ctx context.Context, groups map[string][]int64) error
	InsertBound(ctx context.Context, values []int64) error
	InsertFiltered(ctx context.Context, ids []int64) error
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <insert id="InsertMap">
    INSERT INTO pairs (key, value) VALUES
    <foreach collection="values" index="key" item="value" separator=",">
      (#{key}, #{value})
    </foreach>
  </insert>
  <insert id="InsertNested">
    INSERT INTO memberships (group_name, id) VALUES
    <foreach collection="groups" index="groupName" item="ids" separator=",">
      <foreach collection="ids" index="index" item="id" separator=",">
        (#{groupName}, #{id})
      </foreach>
    </foreach>
  </insert>
  <insert id="InsertBound">
    INSERT INTO values_table (position, value) VALUES
    <foreach collection="values" index="index" item="value" separator=",">
      <bind name="next" value="value + 1"/>
      (#{index}, #{next})
    </foreach>
  </insert>
  <insert id="InsertFiltered">
    INSERT INTO ids (id) VALUES
    <foreach collection="ids" index="index" item="id" separator=",">
      <if test="id > 0">(#{id})</if>
    </foreach>
  </insert>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	for _, expected := range []string{
		"for key, value := range values",
		"for groupName, ids := range groups",
		"for index, id := range ids",
		"next := value + 1",
		"if id > 0",
		".AppendSeparated(",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestGenerateEmitsTypedDMLReturningQueries(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64 `+"`db:\"id\"`"+`
	Name string `+"`db:\"name\"`"+`
}

type UserMapper interface {
	CreateID(ctx context.Context, name string) (int64, error)
	Create(ctx context.Context, name string) (User, error)
	CreateMany(ctx context.Context, names []string) ([]User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <insert id="CreateID" result="returning">
    INSERT INTO users (name) VALUES (#{name}) RETURNING id
  </insert>
  <insert id="Create" result="returning" resultType="User">
    INSERT INTO users (name) VALUES (#{name}) RETURNING id, name
  </insert>
  <insert id="CreateMany" result="returning" resultType="User">
    INSERT INTO users (name) VALUES
    <foreach collection="names" item="name" separator=",">(#{name})</foreach>
    RETURNING id, name
  </insert>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")

	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
		Dialect:       "sqlite",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	for _, expected := range []string{
		"return yourbatis.QueryOne(",
		"return yourbatis.QueryMany(",
		"func scanUserMapperCreateID(row yourbatis.Scanner) (int64, error)",
		"func scanUserMapperCreate(row yourbatis.Scanner) (User, error)",
		"func scanUserMapperCreateMany(row yourbatis.Scanner) (User, error)",
		"&result.ID",
		"&result.Name",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestGenerateValidatesReturningShapeAndTargetDialect(t *testing.T) {
	testCases := []struct {
		name      string
		signature string
		statement string
		dialect   string
		message   string
	}{
		{
			name:      "error only returning",
			signature: "Create(ctx context.Context, name string) error",
			statement: `<insert id="Create" result="returning">INSERT INTO users (name) VALUES (#{name}) RETURNING id</insert>`,
			dialect:   "sqlite",
			message:   "result=returning requires one value and error",
		},
		{
			name:      "missing returning clause",
			signature: "Create(ctx context.Context, name string) (int64, error)",
			statement: `<insert id="Create" result="returning">INSERT INTO users (name) VALUES (#{name})</insert>`,
			dialect:   "sqlite",
			message:   "top-level RETURNING keyword not found",
		},
		{
			name:      "mysql returning",
			signature: "Create(ctx context.Context, name string) (int64, error)",
			statement: `<insert id="Create" result="returning">INSERT INTO users (name) VALUES (#{name}) RETURNING id</insert>`,
			dialect:   "mysql",
			message:   "target dialect mysql does not support DML returning",
		},
		{
			name:      "postgres lastid",
			signature: "Create(ctx context.Context, name string) (int64, error)",
			statement: `<insert id="Create" result="lastid">INSERT INTO users (name) VALUES (#{name})</insert>`,
			dialect:   "postgres",
			message:   "target dialect postgres does not support LastInsertId",
		},
		{
			name:      "unsupported target",
			signature: "Create(ctx context.Context, name string) error",
			statement: `<insert id="Create">INSERT INTO users (name) VALUES (#{name})</insert>`,
			dialect:   "oracle",
			message:   `unsupported target dialect "oracle"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	`+testCase.signature+`
}
`)
			writeGeneratorFixture(
				t,
				directory,
				"user_mapper.xml",
				`<mapper namespace="UserMapper">`+testCase.statement+`</mapper>`,
			)
			err := Generate(Config{
				PackageDir:    directory,
				Mapper:        "UserMapper",
				SQLFile:       "user_mapper.xml",
				Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
				RuntimeImport: "github.com/superduck-ai/yourbatis",
				Dialect:       testCase.dialect,
			})
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestGenerateAcceptsMySQLLastInsertID(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	Create(ctx context.Context, name string) (int64, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <insert id="Create" result="lastid">
    INSERT INTO users (name) VALUES (#{name})
  </insert>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
		Dialect:       "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if source := string(generated); !strings.Contains(source, "return yourbatis.ExecuteLastInsertID(") {
		t.Fatalf("generated MySQL lastid method is missing:\n%s", source)
	}
}

func TestGenerateEmitsYieldSelectsAndExcludesControlParameter(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64 `+"`db:\"id\"`"+`
	Name string `+"`db:\"name\"`"+`
}

type SearchParams struct {
	ID int64
}

type UserMapper interface {
	SearchEach(ctx context.Context, params SearchParams, yield func(User) error) error
	NamesEach(ctx context.Context, yield func(string) error) error
	MappedEach(ctx context.Context, yield func(User) error) error
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
    <result column="name" property="Name"/>
  </resultMap>
  <select id="SearchEach" resultType="User">
    SELECT id, name FROM users WHERE id = #{_parameter.ID}
  </select>
  <select id="NamesEach">
    SELECT name FROM users
  </select>
  <select id="MappedEach" resultMap="UserResult">
    SELECT id, name FROM users
  </select>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
		Dialect:       "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	for _, expected := range []string{
		"func scanUserMapperSearchEach(row yourbatis.Scanner) (User, error)",
		"func scanUserMapperNamesEach(row yourbatis.Scanner) (string, error)",
		"func scanUserMapperMappedEach(row yourbatis.Scanner) (User, error)",
		"func buildUserMapperSearchEach(dialect yourbatis.Dialect, params SearchParams) yourbatis.BoundSQL",
		"__yourbatisBound := buildUserMapperSearchEach(mapper.executor.Dialect(), params)",
		"return yourbatis.SelectEach(ctx, mapper.executor, userMapperSearchEachStatement, __yourbatisBound, scanUserMapperSearchEach, yield)",
		"return yourbatis.SelectEach(ctx, mapper.executor, userMapperNamesEachStatement, __yourbatisBound, scanUserMapperNamesEach, yield)",
		"&result.ID",
		"&result.Name",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
	if strings.Contains(source, "buildUserMapperSearchEach(mapper.executor.Dialect(), params, yield)") ||
		strings.Contains(source, "func buildUserMapperSearchEach(dialect yourbatis.Dialect, params SearchParams, yield") {
		t.Fatalf("yield control parameter leaked into the SQL builder:\n%s", source)
	}
}

func TestGenerateRejectsInvalidYieldMethods(t *testing.T) {
	testCases := []struct {
		name      string
		signature string
		statement string
		message   string
	}{
		{
			name:      "wrong callback result",
			signature: "SearchEach(ctx context.Context, yield func(User) bool) error",
			statement: `<select id="SearchEach">SELECT id FROM users</select>`,
			message:   "yield callback must have signature func(T) error",
		},
		{
			name:      "callback is not final",
			signature: "SearchEach(ctx context.Context, yield func(User) error, limit int) error",
			statement: `<select id="SearchEach">SELECT id FROM users LIMIT #{limit}</select>`,
			message:   "yield callback must be the final mapper parameter",
		},
		{
			name:      "non-select statement",
			signature: "SearchEach(ctx context.Context, yield func(User) error) error",
			statement: `<insert id="SearchEach">INSERT INTO users (id) VALUES (1)</insert>`,
			message:   "yield callbacks are only supported by select statements",
		},
		{
			name:      "value return",
			signature: "SearchEach(ctx context.Context, yield func(User) error) ([]User, error)",
			statement: `<select id="SearchEach">SELECT id FROM users</select>`,
			message:   "yield select methods must return error only",
		},
		{
			name:      "explicit result mode",
			signature: "SearchEach(ctx context.Context, yield func(User) error) error",
			statement: `<select id="SearchEach" result="many">SELECT id FROM users</select>`,
			message:   "yield select methods infer their result mode",
		},
		{
			name:      "callback referenced by XML",
			signature: "SearchEach(ctx context.Context, yield func(User) error) error",
			statement: `<select id="SearchEach">SELECT id FROM users WHERE id = #{yield}</select>`,
			message:   `unknown identifier "yield"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type User struct {
	ID int64 `+"`db:\"id\"`"+`
}

type UserMapper interface {
	`+testCase.signature+`
}
`)
			writeGeneratorFixture(
				t,
				directory,
				"user_mapper.xml",
				`<mapper namespace="UserMapper">`+testCase.statement+`</mapper>`,
			)
			err := Generate(Config{
				PackageDir:    directory,
				Mapper:        "UserMapper",
				SQLFile:       "user_mapper.xml",
				Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
				RuntimeImport: "github.com/superduck-ai/yourbatis",
			})
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestGenerateSupportsMyBatisDTDChooseWithoutWhen(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type UserMapper interface {
	OtherwiseOnly(ctx context.Context, id int64) error
	EmptyChoose(ctx context.Context) error
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <update id="OtherwiseOnly">
    UPDATE users SET active = TRUE
    <choose><otherwise>WHERE id = #{id}</otherwise></choose>
  </update>
  <update id="EmptyChoose">
    UPDATE users SET active = TRUE
    <choose/>
  </update>
</mapper>
`)
	output := filepath.Join(directory, "user_mapper.sqlmap.gen.go")
	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), `builder.Arg("id", id)`) {
		t.Fatalf("otherwise branch was not generated:\n%s", generated)
	}
}

func writeGeneratorFixture(t *testing.T, directory, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
