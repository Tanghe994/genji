package parser

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/genjidb/genji/expr"
	"github.com/genjidb/genji/planner"
	"github.com/genjidb/genji/query"
	"github.com/genjidb/genji/stream"
)

func TestParserMultiStatement(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected []query.Statement
	}{
		{"OnlyCommas", ";;;", nil},
		{"TrailingComma", "SELECT * FROM foo;;;DELETE FROM foo;", []query.Statement{
			&planner.Statement{
				Stream:   stream.New(stream.SeqScan("foo")).Pipe(stream.Project(expr.Wildcard{})),
				ReadOnly: true,
			},
			&planner.Statement{
				Stream: stream.New(stream.SeqScan("foo")).Pipe(stream.TableDelete("foo")),
			},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := ParseQuery(test.s)
			require.NoError(t, err)
			require.EqualValues(t, test.expected, q.Statements)
		})
	}
}

func TestParserDivideByZero(t *testing.T) {
	// See https://github.com/genjidb/genji/issues/268
	require.NotPanics(t, func() {
		_, _ = ParseQuery("SELECT * FROM t LIMIT 0 % .5")
	})
}
