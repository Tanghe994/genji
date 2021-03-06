package planner_test

import (
	"testing"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/expr"
	"github.com/genjidb/genji/planner"
	"github.com/genjidb/genji/sql/parser"
	st "github.com/genjidb/genji/stream"
	"github.com/genjidb/genji/testutil"
	"github.com/stretchr/testify/require"
)

func TestSplitANDConditionRule(t *testing.T) {
	tests := []struct {
		name         string
		in, expected *st.Stream
	}{
		{
			"no and",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(expr.BoolValue(true))),
			st.New(st.SeqScan("foo")).Pipe(st.Filter(expr.BoolValue(true))),
		},
		{
			"and / top-level selection node",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(
				expr.And(
					expr.BoolValue(true),
					expr.BoolValue(false),
				),
			)),
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(expr.BoolValue(true))).
				Pipe(st.Filter(expr.BoolValue(false))),
		},
		{
			"and / middle-level selection node",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(
					expr.And(
						expr.BoolValue(true),
						expr.BoolValue(false),
					),
				)).
				Pipe(st.Take(1)),
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(expr.BoolValue(true))).
				Pipe(st.Filter(expr.BoolValue(false))).
				Pipe(st.Take(1)),
		},
		{
			"multi and",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(
					expr.And(
						expr.And(
							expr.IntegerValue(1),
							expr.IntegerValue(2),
						),
						expr.And(
							expr.IntegerValue(3),
							expr.IntegerValue(4),
						),
					),
				)).
				Pipe(st.Take(10)),
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(expr.IntegerValue(1))).
				Pipe(st.Filter(expr.IntegerValue(2))).
				Pipe(st.Filter(expr.IntegerValue(3))).
				Pipe(st.Filter(expr.IntegerValue(4))).
				Pipe(st.Take(10)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := planner.SplitANDConditionRule(test.in, nil, nil)
			require.NoError(t, err)
			require.Equal(t, res.String(), test.expected.String())
		})
	}
}

func TestPrecalculateExprRule(t *testing.T) {
	tests := []struct {
		name        string
		e, expected expr.Expr
		params      []expr.Param
	}{
		{
			"constant expr: 3 -> 3",
			expr.IntegerValue(3),
			expr.IntegerValue(3),
			nil,
		},
		{
			"operator with two constant operands: 3 + 2.4 -> 5.4",
			expr.Add(expr.IntegerValue(3), expr.PositionalParam(1)),
			expr.DoubleValue(5.4),
			[]expr.Param{{Value: 2.4}},
		},
		{
			"operator with constant nested operands: 3 > 1 - 40 -> true",
			expr.Gt(expr.DoubleValue(3), expr.Sub(expr.IntegerValue(1), expr.DoubleValue(40))),
			expr.BoolValue(true),
			nil,
		},
		{
			"constant sub-expr: a > 1 - 40 -> a > -39",
			expr.Gt(expr.Path{document.PathFragment{FieldName: "a"}}, expr.Sub(expr.IntegerValue(1), expr.DoubleValue(40))),
			expr.Gt(expr.Path{document.PathFragment{FieldName: "a"}}, expr.DoubleValue(-39)),
			nil,
		},
		{
			"constant sub-expr: a IN [1, 2] -> a IN array([1, 2])",
			expr.In(expr.Path{document.PathFragment{FieldName: "a"}}, expr.LiteralExprList{expr.IntegerValue(1), expr.IntegerValue(2)}),
			expr.In(expr.Path{document.PathFragment{FieldName: "a"}}, expr.LiteralValue(document.NewArrayValue(document.NewValueBuffer().
				Append(document.NewIntegerValue(1)).
				Append(document.NewIntegerValue(2))))),
			nil,
		},
		{
			"non-constant expr list: [a, 1 - 40] -> [a, -39]",
			expr.LiteralExprList{
				expr.Path{document.PathFragment{FieldName: "a"}},
				expr.Sub(expr.IntegerValue(1), expr.DoubleValue(40)),
			},
			expr.LiteralExprList{
				expr.Path{document.PathFragment{FieldName: "a"}},
				expr.DoubleValue(-39),
			},
			nil,
		},
		{
			"constant expr list: [3, 1 - 40] -> array([3, -39])",
			expr.LiteralExprList{
				expr.IntegerValue(3),
				expr.Sub(expr.IntegerValue(1), expr.DoubleValue(40)),
			},
			expr.LiteralValue(document.NewArrayValue(document.NewValueBuffer().
				Append(document.NewIntegerValue(3)).
				Append(document.NewDoubleValue(-39)))),
			nil,
		},
		{
			`non-constant kvpair: {"a": d, "b": 1 - 40} -> {"a": 3, "b": -39}`,
			&expr.KVPairs{Pairs: []expr.KVPair{
				{K: "a", V: expr.Path{document.PathFragment{FieldName: "d"}}},
				{K: "b", V: expr.Sub(expr.IntegerValue(1), expr.DoubleValue(40))},
			}},
			&expr.KVPairs{Pairs: []expr.KVPair{
				{K: "a", V: expr.Path{document.PathFragment{FieldName: "d"}}},
				{K: "b", V: expr.DoubleValue(-39)},
			}},
			nil,
		},
		{
			`constant kvpair: {"a": 3, "b": 1 - 40} -> document({"a": 3, "b": -39})`,
			&expr.KVPairs{Pairs: []expr.KVPair{
				{K: "a", V: expr.IntegerValue(3)},
				{K: "b", V: expr.Sub(expr.IntegerValue(1), expr.DoubleValue(40))},
			}},
			expr.LiteralValue(document.NewDocumentValue(document.NewFieldBuffer().
				Add("a", document.NewIntegerValue(3)).
				Add("b", document.NewDoubleValue(-39)),
			)),
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := st.New(st.SeqScan("foo")).
				Pipe(st.Filter(test.e))
			res, err := planner.PrecalculateExprRule(s, nil, test.params)
			require.NoError(t, err)
			require.Equal(t, st.New(st.SeqScan("foo")).Pipe(st.Filter(test.expected)).String(), res.String())
		})
	}
}

func TestRemoveUnnecessarySelectionNodesRule(t *testing.T) {
	tests := []struct {
		name           string
		root, expected *st.Stream
	}{
		{
			"non-constant expr",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("a"))),
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("a"))),
		},
		{
			"truthy constant expr",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("10"))),
			st.New(st.SeqScan("foo")),
		},
		{
			"truthy constant expr with IN",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(expr.In(
				expr.Path(document.NewPath("a")),
				expr.ArrayValue(document.NewValueBuffer()),
			))),
			&st.Stream{},
		},
		{
			"falsy constant expr",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("0"))),
			&st.Stream{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := planner.RemoveUnnecessaryFilterNodesRule(test.root, nil, nil)
			require.NoError(t, err)
			if test.expected != nil {
				require.Equal(t, test.expected.String(), res.String())
			} else {
				require.Equal(t, test.expected, res)
			}
		})
	}
}

func TestRemoveUnnecessaryDedupNodeRule(t *testing.T) {
	tests := []struct {
		name           string
		root, expected *st.Stream
	}{
		{
			"non-unique key",
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("b"))).
				Pipe(st.Distinct()),
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("b"))).
				Pipe(st.Distinct()),
		},
		{
			"primary key",
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("a"))).
				Pipe(st.Distinct()),
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("a"))),
		},
		{
			"primary key with alias",
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("a AS A"))).
				Pipe(st.Distinct()),
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("a AS A"))),
		},
		{
			"unique index",
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("c"))).
				Pipe(st.Distinct()),
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("c"))),
		},
		{
			"pk() function",
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("pk()"))).
				Pipe(st.Distinct()),
			st.New(st.SeqScan("foo")).
				Pipe(st.Project(parser.MustParseExpr("pk()"))),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := genji.Open(":memory:")
			require.NoError(t, err)
			defer db.Close()

			tx, err := db.Begin(true)
			require.NoError(t, err)
			defer tx.Rollback()

			err = tx.Exec(`
				CREATE TABLE foo(a integer PRIMARY KEY, b integer, c integer);
				CREATE UNIQUE INDEX idx_foo_idx ON foo(c);
				INSERT INTO foo (a, b, c) VALUES
					(1, 1, 1),
					(2, 2, 2),
					(3, 3, 3)
			`)
			require.NoError(t, err)

			res, err := planner.RemoveUnnecessaryDistinctNodeRule(test.root, tx.Transaction, nil)
			require.NoError(t, err)
			require.Equal(t, test.expected.String(), res.String())
		})
	}
}

func TestUseIndexBasedOnSelectionNodeRule(t *testing.T) {
	tests := []struct {
		name           string
		root, expected *st.Stream
	}{
		{
			"non-indexed path",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("d = 1"))),
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("d = 1"))),
		},
		{
			"FROM foo WHERE a = 1",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("a = 1"))),
			st.New(st.IndexScan("idx_foo_a", st.Range{Min: document.NewIntegerValue(1), Exact: true})),
		},
		{
			"FROM foo WHERE a = 1 AND b = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("a = 1"))).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))),
			st.New(st.IndexScan("idx_foo_b", st.Range{Min: document.NewIntegerValue(2), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("a = 1"))),
		},
		{
			"FROM foo WHERE c = 3 AND b = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("c = 3"))).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))),
			st.New(st.IndexScan("idx_foo_c", st.Range{Min: document.NewIntegerValue(3), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))),
		},
		{
			"FROM foo WHERE c > 3 AND b = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("c > 3"))).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))),
			st.New(st.IndexScan("idx_foo_b", st.Range{Min: document.NewIntegerValue(2), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("c > 3"))),
		},
		{
			"SELECT a FROM foo WHERE c = 3 AND b = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("c = 3"))).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))).
				Pipe(st.Project(parser.MustParseExpr("a"))),
			st.New(st.IndexScan("idx_foo_c", st.Range{Min: document.NewIntegerValue(3), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))).
				Pipe(st.Project(parser.MustParseExpr("a"))),
		},
		{
			"SELECT a FROM foo WHERE c = 'hello' AND b = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("c = 'hello'"))).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))).
				Pipe(st.Project(parser.MustParseExpr("a"))),
			st.New(st.IndexScan("idx_foo_b", st.Range{Min: document.NewIntegerValue(2), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("c = 'hello'"))).
				Pipe(st.Project(parser.MustParseExpr("a"))),
		},
		{
			"SELECT a FROM foo WHERE c = 'hello' AND d = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("c = 'hello'"))).
				Pipe(st.Filter(parser.MustParseExpr("d = 2"))).
				Pipe(st.Project(parser.MustParseExpr("a"))),
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("c = 'hello'"))).
				Pipe(st.Filter(parser.MustParseExpr("d = 2"))).
				Pipe(st.Project(parser.MustParseExpr("a"))),
		},
		{
			"FROM foo WHERE a IN [1, 2]",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(
				expr.In(
					parser.MustParseExpr("a"),
					expr.ArrayValue(document.NewValueBuffer(document.NewIntegerValue(1), document.NewIntegerValue(2))),
				),
			)),
			st.New(st.IndexScan("idx_foo_a", st.Range{Min: document.NewIntegerValue(1), Exact: true}, st.Range{Min: document.NewIntegerValue(2), Exact: true})),
		},
		{
			"FROM foo WHERE 1 IN a",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("1 IN a"))),
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("1 IN a"))),
		},
		{
			"FROM foo WHERE a >= 10",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("a >= 10"))),
			st.New(st.IndexScan("idx_foo_a", st.Range{Min: document.NewIntegerValue(10)})),
		},
		{
			"FROM foo WHERE k = 1",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("k = 1"))),
			st.New(st.PkScan("foo", st.Range{Min: document.NewIntegerValue(1), Exact: true})),
		},
		{
			"FROM foo WHERE k = 1 AND b = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("k = 1"))).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))),
			st.New(st.PkScan("foo", st.Range{Min: document.NewIntegerValue(1), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("b = 2"))),
		},
		{
			"FROM foo WHERE a = 1 AND k = 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("a = 1"))).
				Pipe(st.Filter(parser.MustParseExpr("2 = k"))),
			st.New(st.PkScan("foo", st.Range{Min: document.NewIntegerValue(2), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("a = 1"))),
		},
		{
			"FROM foo WHERE a = 1 AND k < 2",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("a = 1"))).
				Pipe(st.Filter(parser.MustParseExpr("k < 2"))),
			st.New(st.IndexScan("idx_foo_a", st.Range{Min: document.NewIntegerValue(1), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("k < 2"))),
		},
		{
			"FROM foo WHERE a = 1 AND k = 'hello'",
			st.New(st.SeqScan("foo")).
				Pipe(st.Filter(parser.MustParseExpr("a = 1"))).
				Pipe(st.Filter(parser.MustParseExpr("k = 'hello'"))),
			st.New(st.IndexScan("idx_foo_a", st.Range{Min: document.NewIntegerValue(1), Exact: true})).
				Pipe(st.Filter(parser.MustParseExpr("k = 'hello'"))),
		},
		{ // c is an INT, 1.1 cannot be converted to int without precision loss, don't use the index
			"FROM foo WHERE c < 1.1",
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("c < 1.1"))),
			st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("c < 1.1"))),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := genji.Open(":memory:")
			require.NoError(t, err)
			defer db.Close()

			tx, err := db.Begin(true)
			require.NoError(t, err)
			defer tx.Rollback()

			err = tx.Exec(`
				CREATE TABLE foo (k INT PRIMARY KEY, c INT);
				CREATE INDEX idx_foo_a ON foo(a);
				CREATE INDEX idx_foo_b ON foo(b);
				CREATE UNIQUE INDEX idx_foo_c ON foo(c);
				INSERT INTO foo (k, a, b, c, d) VALUES
					(1, 1, 1, 1, 1),
					(2, 2, 2, 2, 2),
					(3, 3, 3, 3, 3)
			`)
			require.NoError(t, err)

			res, err := planner.UseIndexBasedOnFilterNodeRule(test.root, tx.Transaction, nil)
			require.NoError(t, err)
			require.Equal(t, test.expected.String(), res.String())
		})
	}

	t.Run("array indexes", func(t *testing.T) {
		tests := []struct {
			name           string
			root, expected *st.Stream
		}{
			{
				"non-indexed path",
				st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("b = [1, 1]"))),
				st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("b = [1, 1]"))),
			},
			{
				"FROM foo WHERE k = [1, 1]",
				st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("k = [1, 1]"))),
				st.New(st.PkScan("foo", st.Range{Min: document.NewArrayValue(testutil.MakeArray(t, `[1, 1]`)), Exact: true})),
			},
			{ // constraint on k[0] INT should not modify the operand
				"FROM foo WHERE k = [1.5, 1.5]",
				st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("k = [1.5, 1.5]"))),
				st.New(st.PkScan("foo", st.Range{Min: document.NewArrayValue(testutil.MakeArray(t, `[1.5, 1.5]`)), Exact: true})),
			},
			{
				"FROM foo WHERE a = [1, 1]",
				st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("a = [1, 1]"))),
				st.New(st.IndexScan("idx_foo_a", st.Range{Min: document.NewArrayValue(testutil.MakeArray(t, `[1, 1]`)), Exact: true})),
			},
			{ // constraint on a[0] DOUBLE should modify the operand because it's lossless
				"FROM foo WHERE a = [1, 1.5]",
				st.New(st.SeqScan("foo")).Pipe(st.Filter(parser.MustParseExpr("a = [1, 1.5]"))),
				st.New(st.IndexScan("idx_foo_a", st.Range{Min: document.NewArrayValue(testutil.MakeArray(t, `[1.0, 1.5]`)), Exact: true})),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				db, err := genji.Open(":memory:")
				require.NoError(t, err)
				defer db.Close()

				tx, err := db.Begin(true)
				require.NoError(t, err)
				defer tx.Rollback()

				err = tx.Exec(`
					CREATE TABLE foo (
						k ARRAY PRIMARY KEY,
						k[0] INT,
						a ARRAY,
						a[0] DOUBLE
					);
					CREATE INDEX idx_foo_a ON foo(a);
					CREATE INDEX idx_foo_a0 ON foo(a[0]);
					INSERT INTO foo (k, a, b) VALUES
						([1, 1], [1, 1], [1, 1]),
						([2, 2], [2, 2], [2, 2]),
						([3, 3], [3, 3], [3, 3])
				`)
				require.NoError(t, err)

				res, err := planner.PrecalculateExprRule(test.root, tx.Transaction, nil)
				require.NoError(t, err)

				res, err = planner.UseIndexBasedOnFilterNodeRule(res, tx.Transaction, nil)
				require.NoError(t, err)
				require.Equal(t, test.expected.String(), res.String())
			})
		}
	})
}
