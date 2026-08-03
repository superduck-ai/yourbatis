package xmlmapper

import (
	"fmt"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

var fakeSpan = ir.Span{File: "x.xml", Line: 1, Column: 1}

func TestProbeEscape(t *testing.T) {
	cases := []string{
		`\#{id}`,
		`\\#{id}`,
		`\\\\#{id}`,
		`\${id}`,
		`\\${id}`,
		`SELECT \#{literal}, #{id}`,
		`\#{`,
		`\#{}`,
		`\${}`,
		`\${`,
		`#{a{b}}`,
		`#{a{b}c}`,
		`#{a"{b}"}`,
	}
	for _, c := range cases {
		fmt.Printf("INPUT: %q\n", c)
		nodes, err := parseSQLText(c, fakeSpan)
		if err != nil {
			fmt.Printf("  ERR: %v\n", err)
			continue
		}
		for i, n := range nodes {
			fmt.Printf("  [%d] %T %+v\n", i, n, n)
		}
	}
}
