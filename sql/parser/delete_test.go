package parser

import (
	"testing"

	"github.com/genjidb/genji/planner"
	"github.com/genjidb/genji/stream"
	"github.com/stretchr/testify/require"
)

func TestParserDelete(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected *stream.Stream
	}{
		{"NoCond", "DELETE FROM test", stream.New(stream.SeqScan("test")).Pipe(stream.TableDelete("test"))},
		{"WithCond", "DELETE FROM test WHERE age = 10",
			stream.New(stream.SeqScan("test")).
				Pipe(stream.Filter(MustParseExpr("age = 10"))).
				Pipe(stream.TableDelete("test")),
		},
		{"WithOffset", "DELETE FROM test WHERE age = 10 OFFSET 20",
			stream.New(stream.SeqScan("test")).
				Pipe(stream.Filter(MustParseExpr("age = 10"))).
				Pipe(stream.Skip(20)).
				Pipe(stream.TableDelete("test")),
		},
		{"WithLimit", "DELETE FROM test LIMIT 10",
			stream.New(stream.SeqScan("test")).
				Pipe(stream.Take(10)).
				Pipe(stream.TableDelete("test")),
		},
		{"WithOrderByThenOffset", "DELETE FROM test WHERE age = 10 ORDER BY age OFFSET 20",
			stream.New(stream.SeqScan("test")).
				Pipe(stream.Filter(MustParseExpr("age = 10"))).
				Pipe(stream.Sort(MustParseExpr("age"))).
				Pipe(stream.Skip(20)).
				Pipe(stream.TableDelete("test")),
		},
		{"WithOrderByThenLimitThenOffset", "DELETE FROM test WHERE age = 10 ORDER BY age LIMIT 10 OFFSET 20",
			stream.New(stream.SeqScan("test")).
				Pipe(stream.Filter(MustParseExpr("age = 10"))).
				Pipe(stream.Sort(MustParseExpr("age"))).
				Pipe(stream.Skip(20)).
				Pipe(stream.Take(10)).
				Pipe(stream.TableDelete("test")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := ParseQuery(test.s)
			require.NoError(t, err)
			require.Len(t, q.Statements, 1)
			require.EqualValues(t, &planner.Statement{Stream: test.expected}, q.Statements[0])
		})
	}
}
