package returningconformance

import (
	"os"
	"strings"
	"testing"
)

func TestReturningFixtureGeneratedShape(t *testing.T) {
	generated, err := os.ReadFile("returning_mapper.sqlmap.gen.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)

	for _, expected := range []string{
		"Kind:   yourbatis.StatementInsert",
		"Kind:   yourbatis.StatementUpdate",
		"Kind:   yourbatis.StatementDelete",
		"return yourbatis.QueryOne(",
		"return yourbatis.QueryMany(",
		"return yourbatis.SelectOptional(",
		"func scanReturningMapperCreateID(row yourbatis.Scanner) (int64, error)",
		"func scanReturningMapperCreate(row yourbatis.Scanner) (CreatedUser, error)",
		"func scanReturningMapperCreateMany(row yourbatis.Scanner) (CreatedUser, error)",
		"&result.ID",
		"&result.CreatedAt",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}
