package generator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func TestSelectColumnsHandlesCTESubqueriesCommentsAndAliases(t *testing.T) {
	statement := projectionStatement(`
WITH recent AS (
    SELECT user_id, MAX(created_at) AS created_at
    FROM audit_log
    GROUP BY user_id
)
SELECT DISTINCT
    u.id /* FROM is ignored here */ AS user_id,
    COALESCE(u.name, 'from fallback') AS "display_name",
    (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) AS post_count
FROM users u
JOIN recent r ON r.user_id = u.id
`)

	columns, err := selectColumns(statement)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"user_id", "display_name", "post_count"}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("columns = %#v, want %#v", columns, want)
	}
}

func TestSelectColumnsSupportsSelectWithoutFrom(t *testing.T) {
	statement := projectionStatement(`SELECT 1 AS one, 'from is data' AS label;`)

	columns, err := selectColumns(statement)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"one", "label"}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("columns = %#v, want %#v", columns, want)
	}
}

func TestSelectColumnsRejectsDynamicProjection(t *testing.T) {
	statement := &ir.Statement{
		Body: []ir.Node{
			&ir.Text{Value: "SELECT id", Source: ir.Span{File: "query.xml", Line: 2, Column: 3}},
			&ir.If{
				Test:   "includeEmail",
				Body:   []ir.Node{&ir.Text{Value: ", email"}},
				Source: ir.Span{File: "query.xml", Line: 3, Column: 5},
			},
			&ir.Text{Value: " FROM users"},
		},
		Source: ir.Span{File: "query.xml", Line: 1, Column: 1},
	}

	_, err := selectColumns(statement)
	if err == nil || !strings.Contains(err.Error(), "dynamic <if> cannot change the SELECT projection") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "query.xml:3:5") {
		t.Fatalf("error lacks dynamic element source: %v", err)
	}
}

func TestSelectColumnsAllowsDynamicConditionsAfterProjection(t *testing.T) {
	testCases := []struct {
		name string
		body []ir.Node
	}{
		{
			name: "after FROM",
			body: []ir.Node{
				&ir.Text{Value: "SELECT id FROM users"},
				&ir.If{
					Test:   "enabled",
					Body:   []ir.Node{&ir.Text{Value: " WHERE enabled = TRUE"}},
					Source: ir.Span{File: "query.xml", Line: 3, Column: 5},
				},
			},
		},
		{
			name: "SELECT without FROM",
			body: []ir.Node{
				&ir.Text{Value: "SELECT 1 AS id"},
				&ir.If{
					Test:   "enabled",
					Body:   []ir.Node{&ir.Text{Value: " WHERE TRUE"}},
					Source: ir.Span{File: "query.xml", Line: 3, Column: 5},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			statement := &ir.Statement{
				Body:   testCase.body,
				Source: ir.Span{File: "query.xml", Line: 1, Column: 1},
			}
			columns, err := selectColumns(statement)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(columns, []string{"id"}) {
				t.Fatalf("columns = %#v", columns)
			}
		})
	}
}

func TestSelectColumnsRequiresAliasesForComputedValues(t *testing.T) {
	statement := projectionStatement(`SELECT COUNT(*) FROM users`)

	_, err := selectColumns(statement)
	if err == nil || !strings.Contains(err.Error(), "requires an explicit AS alias") {
		t.Fatalf("error = %v", err)
	}
}

func TestSelectColumnsRequiresExplicitASForAliases(t *testing.T) {
	testCases := []struct {
		name string
		sql  string
	}{
		{
			name: "simple column",
			sql:  `SELECT price amount FROM orders`,
		},
		{
			name: "qualified column",
			sql:  `SELECT orders.price amount FROM orders`,
		},
		{
			name: "binary expression mistaken for alias",
			sql:  `SELECT price - discount FROM orders`,
		},
		{
			name: "computed expression",
			sql:  `SELECT price - discount final_price FROM orders`,
		},
		{
			name: "function call",
			sql:  `SELECT COUNT(*) total FROM orders`,
		},
		{
			name: "scalar subquery",
			sql:  `SELECT (SELECT COUNT(*) FROM orders) total`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := selectColumns(projectionStatement(testCase.sql))
			if err == nil || !strings.Contains(err.Error(), "requires an explicit AS alias") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestSelectColumnsAcceptsExplicitASAndUnaliasedColumns(t *testing.T) {
	statement := projectionStatement(`
SELECT
    price,
    orders.discount,
    price AS original_price,
    price - discount AS final_price,
    COUNT(*) AS total
FROM orders
`)

	columns, err := selectColumns(statement)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"price", "discount", "original_price", "final_price", "total"}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("columns = %#v, want %#v", columns, want)
	}
}

func TestReturningColumnsHandlesStaticExpressionsAndAliases(t *testing.T) {
	statement := returningProjectionStatement(`
WITH source AS (
    SELECT name FROM pending_users
)
INSERT INTO users (name)
SELECT name FROM source
RETURNING id, UPPER(name) AS display_name, created_at;
`)

	columns, err := returningColumns(statement)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"id", "display_name", "created_at"}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("columns = %#v, want %#v", columns, want)
	}
}

func TestReturningColumnsRejectsDynamicProjection(t *testing.T) {
	statement := &ir.Statement{
		Command:    ir.Insert,
		ResultMode: ir.ResultReturning,
		Body: []ir.Node{
			&ir.Text{Value: "INSERT INTO users (name) VALUES (?) RETURNING id"},
			&ir.If{
				Test:   "includeName",
				Body:   []ir.Node{&ir.Text{Value: ", name"}},
				Source: ir.Span{File: "returning.xml", Line: 4, Column: 5},
			},
		},
		Source: ir.Span{File: "returning.xml", Line: 1, Column: 1},
	}

	_, err := returningColumns(statement)
	if err == nil || !strings.Contains(err.Error(), "dynamic <if> cannot change the RETURNING projection") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "returning.xml:4:5") {
		t.Fatalf("error lacks dynamic element source: %v", err)
	}
}

func TestReturningColumnsRequiresReturningClause(t *testing.T) {
	statement := returningProjectionStatement(`INSERT INTO users (name) VALUES ('Arthur')`)

	_, err := returningColumns(statement)
	if err == nil || !strings.Contains(err.Error(), "top-level RETURNING keyword not found") {
		t.Fatalf("error = %v", err)
	}
}

func TestReturningColumnsRejectsWildcard(t *testing.T) {
	statement := returningProjectionStatement(`INSERT INTO users (name) VALUES ('Arthur') RETURNING *`)

	_, err := returningColumns(statement)
	if err == nil || !strings.Contains(err.Error(), "RETURNING * cannot be statically mapped") {
		t.Fatalf("error = %v", err)
	}
}

func TestReturningColumnsRequiresExplicitASForAliases(t *testing.T) {
	testCases := []string{
		`UPDATE orders SET discount = 20 RETURNING price amount`,
		`UPDATE orders SET discount = 20 RETURNING price - discount`,
		`UPDATE orders SET discount = 20 RETURNING price - discount final_price`,
	}

	for _, sql := range testCases {
		t.Run(sql, func(t *testing.T) {
			_, err := returningColumns(returningProjectionStatement(sql))
			if err == nil || !strings.Contains(err.Error(), "requires an explicit AS alias") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func projectionStatement(sql string) *ir.Statement {
	source := ir.Span{File: "query.xml", Line: 7, Column: 3}
	return &ir.Statement{
		Command: ir.Select,
		Body: []ir.Node{
			&ir.Text{Value: sql, Source: source},
		},
		Source: source,
	}
}

func returningProjectionStatement(sql string) *ir.Statement {
	source := ir.Span{File: "returning.xml", Line: 7, Column: 3}
	return &ir.Statement{
		Command:    ir.Insert,
		ResultMode: ir.ResultReturning,
		Body: []ir.Node{
			&ir.Text{Value: sql, Source: source},
		},
		Source: source,
	}
}
