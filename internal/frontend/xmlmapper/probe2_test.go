package xmlmapper

import (
	"fmt"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func TestProbe2(t *testing.T) {
	// Case: escaped backslash then real marker
	nodes, err := parseSQLText(`SELECT \\#{real}`, ir.Span{File: "p.xml"})
	fmt.Printf("ESC nodes=%#v err=%v\n", nodes, err)
	for i, n := range nodes {
		switch v := n.(type) {
		case *ir.Text:
			fmt.Printf("  [%d] Text %q\n", i, v.Value)
		case *ir.Parameter:
			fmt.Printf("  [%d] Param path=%q opts=%v\n", i, v.Path, v.Options)
		case *ir.TextExpression:
			fmt.Printf("  [%d] TextExpr %q\n", i, v.Expression)
		}
	}

	// Case: brace inside unclosed parens
	nodes, err = parseSQLText(`#{ f(a}b) }`, ir.Span{File: "p.xml"})
	fmt.Printf("PAREN nodes=%#v err=%v\n", nodes, err)
	for i, n := range nodes {
		switch v := n.(type) {
		case *ir.Text:
			fmt.Printf("  [%d] Text %q\n", i, v.Value)
		case *ir.Parameter:
			fmt.Printf("  [%d] Param path=%q opts=%v\n", i, v.Path, v.Options)
		}
	}

	// UTF-8 multi-byte path
	nodes, err = parseSQLText(`#{ café }`, ir.Span{File: "p.xml"})
	fmt.Printf("UTF8 nodes=%#v err=%v\n", nodes, err)
	for i, n := range nodes {
		switch v := n.(type) {
		case *ir.Parameter:
			fmt.Printf("  [%d] Param path=%q (bytes=%v)\n", i, v.Path, []byte(v.Path))
		}
	}

	// jdbcType dead-branch: content with no separator after colon
	p, err := parseParameter(`id:VARCHAR`, ir.Span{File: "p.xml"})
	fmt.Printf("DEAD p=%#v err=%v\n", p, err)

	// The `:` separator finding with jdbcType containing comma in string
	p, err = parseParameter(`id:"a,b"`, ir.Span{File: "p.xml"})
	fmt.Printf("JDBCSTR p=%#v err=%v\n", p, err)

	// unbalanced ] 
	p, err = parseParameter(`a], b`, ir.Span{File: "p.xml"})
	fmt.Printf("UNBRA p=%#v err=%v\n", p, err)

	// line comment inside content with newline
	p, err = parseParameter(`a // x\n , b`, ir.Span{File: "p.xml"})
	fmt.Printf("LINEC p=%#v err=%v\n", p, err)
}
