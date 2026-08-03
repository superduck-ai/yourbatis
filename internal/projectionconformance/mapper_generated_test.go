package projectionconformance

import (
	"os"
	"strings"
	"testing"
)

func TestProjectionFixtureGeneratedShape(t *testing.T) {
	generated, err := os.ReadFile("projection_mapper.sqlmap.gen.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)

	for _, expected := range []string{
		"func scanProjectionMapperGet(row yourbatis.Scanner) (OrderProjection, error)",
		"&result.Price",
		"&result.Discount",
		"&result.FinalPrice",
		"price - discount AS final_price",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated source does not contain %q:\n%s", expected, source)
		}
	}
}
