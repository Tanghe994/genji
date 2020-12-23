package expr

import (
	"fmt"
	"strings"

	"github.com/genjidb/genji/document"
)

// A LiteralValue represents a litteral value of any type defined by the value package.
type LiteralValue document.Value

// IsEqual compares this expression with the other expression and returns
// true if they are equal.
func (v LiteralValue) IsEqual(other Expr) bool {
	o, ok := other.(LiteralValue)
	if !ok {
		return false
	}
	ok, err := document.Value(v).IsEqual(document.Value(o))
	return ok && err == nil
}

// String implements the fmt.Stringer interface.
func (v LiteralValue) String() string {
	return document.Value(v).String()
}

// BlobValue creates a litteral value of type Blob.
func BlobValue(v []byte) LiteralValue {
	return LiteralValue(document.NewBlobValue(v))
}

// TextValue creates a litteral value of type Text.
func TextValue(v string) LiteralValue {
	return LiteralValue(document.NewTextValue(v))
}

// BoolValue creates a litteral value of type Bool.
func BoolValue(v bool) LiteralValue {
	return LiteralValue(document.NewBoolValue(v))
}

// IntegerValue creates a litteral value of type Integer.
func IntegerValue(v int64) LiteralValue {
	return LiteralValue(document.NewIntegerValue(v))
}

// DoubleValue creates a litteral value of type Double.
func DoubleValue(v float64) LiteralValue {
	return LiteralValue(document.NewDoubleValue(v))
}

// NullValue creates a litteral value of type Null.
func NullValue() LiteralValue {
	return LiteralValue(document.NewNullValue())
}

// DocumentValue creates a litteral value of type Document.
func DocumentValue(d document.Document) LiteralValue {
	return LiteralValue(document.NewDocumentValue(d))
}

// ArrayValue creates a litteral value of type Array.
func ArrayValue(a document.Array) LiteralValue {
	return LiteralValue(document.NewArrayValue(a))
}

// Eval returns l. It implements the Expr interface.
func (v LiteralValue) Eval(*Environment) (document.Value, error) {
	return document.Value(v), nil
}

// LiteralExprList is a list of expressions.
type LiteralExprList []Expr

// IsEqual compares this expression with the other expression and returns
// true if they are equal.
func (l LiteralExprList) IsEqual(other Expr) bool {
	o, ok := other.(LiteralExprList)
	if !ok {
		return false
	}
	if len(l) != len(o) {
		return false
	}

	for i := range l {
		if !Equal(l[i], o[i]) {
			return false
		}
	}

	return true
}

// String implements the fmt.Stringer interface.
func (l LiteralExprList) String() string {
	var b strings.Builder

	b.WriteRune('[')
	for i, e := range l {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%v", e))
	}
	b.WriteRune(']')

	return b.String()
}

// Eval evaluates all the expressions and returns a litteralValueList. It implements the Expr interface.
func (l LiteralExprList) Eval(env *Environment) (document.Value, error) {
	var err error
	values := make([]document.Value, len(l))
	for i, e := range l {
		values[i], err = e.Eval(env)
		if err != nil {
			return nullLitteral, err
		}
	}

	return document.NewArrayValue(document.NewValueBuffer(values...)), nil
}

// KVPair associates an identifier with an expression.
type KVPair struct {
	K string
	V Expr
}

// String implements the fmt.Stringer interface.
func (p KVPair) String() string {
	return fmt.Sprintf("%q: %v", p.K, p.V)
}

// KVPairs is a list of KVPair.
type KVPairs []KVPair

// IsEqual compares this expression with the other expression and returns
// true if they are equal.
func (kvp KVPairs) IsEqual(other Expr) bool {
	o, ok := other.(KVPairs)
	if !ok {
		return false
	}
	if len(kvp) != len(o) {
		return false
	}

	for i := range kvp {
		if kvp[i].K != o[i].K {
			return false
		}
		if !Equal(kvp[i].V, o[i].V) {
			return false
		}
	}

	return true
}

// Eval turns a list of KVPairs into a document.
func (kvp KVPairs) Eval(env *Environment) (document.Value, error) {
	var fb document.FieldBuffer
	doc := document.NewDocumentValue(&fb)
	if _, ok := env.GetCurrentValue(); !ok {
		env.SetCurrentValue(doc)
	}

	for _, kv := range kvp {
		v, err := kv.V.Eval(env)
		if err != nil {
			return document.Value{}, err
		}

		fb.Add(kv.K, v)
	}

	return doc, nil
}

// String implements the fmt.Stringer interface.
func (kvp KVPairs) String() string {
	var b strings.Builder

	b.WriteRune('{')
	for i, p := range kvp {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%s", p))
	}
	b.WriteRune('}')

	return b.String()
}
