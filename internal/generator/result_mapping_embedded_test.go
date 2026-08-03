package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAcceptsPromotedResultMapField(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorModule(t, directory)
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type Base struct {
	ID int64
}

type Details struct {
	Base
}

type User struct {
	Details
	Name string
}

type UserMapper interface {
	Get(ctx context.Context, id int64) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
    <result column="name" property="Name"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">
    SELECT id, name FROM users WHERE id = #{id}
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
		t.Fatalf("generate resultMap with promoted field: %v", err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "&result.ID") {
		t.Fatalf("generated source does not scan the promoted ID field:\n%s", generated)
	}
}

func TestGenerateAcceptsPromotedResultMapFieldFromImportedType(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorModule(t, directory)
	if err := os.Mkdir(filepath.Join(directory, "model"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGeneratorFixture(t, filepath.Join(directory, "model"), "base.go", `package model

type Base struct {
	ID int64
}
`)
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import (
	"context"
	"fixture/model"
)

type User struct {
	model.Base
	Name string
}

type UserMapper interface {
	Get(ctx context.Context, id int64) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
    <result column="name" property="Name"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">
    SELECT id, name FROM users WHERE id = #{id}
  </select>
</mapper>
`)

	if err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	}); err != nil {
		t.Fatalf("generate resultMap with imported promoted field: %v", err)
	}
}

func TestGenerateRejectsAmbiguousPromotedResultMapField(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorModule(t, directory)
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type Left struct {
	ID int64
}

type Right struct {
	ID int64
}

type User struct {
	Left
	Right
}

type UserMapper interface {
	Get(ctx context.Context, id int64) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">SELECT id FROM users WHERE id = #{id}</select>
</mapper>
`)

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), `property "ID" is not an exported field of User`) {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateRejectsPromotedResultMapFieldThroughEmbeddedPointer(t *testing.T) {
	directory := t.TempDir()
	writeGeneratorModule(t, directory)
	writeGeneratorFixture(t, directory, "mapper.go", `package fixture

import "context"

type Base struct {
	ID int64
}

type User struct {
	*Base
}

type UserMapper interface {
	Get(ctx context.Context, id int64) (User, error)
}
`)
	writeGeneratorFixture(t, directory, "user_mapper.xml", `<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id column="id" property="ID"/>
  </resultMap>
  <select id="Get" resultMap="UserResult">SELECT id FROM users WHERE id = #{id}</select>
</mapper>
`)

	err := Generate(Config{
		PackageDir:    directory,
		Mapper:        "UserMapper",
		SQLFile:       "user_mapper.xml",
		Output:        filepath.Join(directory, "user_mapper.sqlmap.gen.go"),
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), "promoted through an embedded pointer") {
		t.Fatalf("error = %v", err)
	}
}

func writeGeneratorModule(t *testing.T, directory string) {
	t.Helper()
	writeGeneratorFixture(t, directory, "go.mod", "module fixture\n\ngo 1.26\n")
}
