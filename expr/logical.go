package expr

import (
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/sql/scanner"
	"github.com/genjidb/genji/stringutil"
)

// AndOp is the And operator.
type AndOp struct {
	*simpleOperator
}

// And creates an expression that evaluates a And b And returns true if both are truthy.
func And(a, b Expr) Expr {
	return &AndOp{&simpleOperator{a, b, scanner.AND}}
}

// Eval implements the Expr interface. It evaluates a and b and returns true if both evaluate
// to true.
func (op *AndOp) Eval(env *Environment) (document.Value, error) {
	s, err := op.a.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err := s.IsTruthy()
	if !isTruthy || err != nil {
		return falseLitteral, err
	}

	s, err = op.b.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err = s.IsTruthy()
	if !isTruthy || err != nil {
		return falseLitteral, err
	}

	return trueLitteral, nil
}

// String implements the stringutil.Stringer interface.
func (op *AndOp) String() string {
	return stringutil.Sprintf("%v AND %v", op.a, op.b)
}

// OrOp is the And operator.
type OrOp struct {
	*simpleOperator
}

// Or creates an expression that first evaluates a, returns true if truthy, then evaluates b, returns true if truthy Or false if falsy.
func Or(a, b Expr) Expr {
	return &OrOp{&simpleOperator{a, b, scanner.OR}}
}

// Eval implements the Expr interface. It evaluates a and b and returns true if a or b evalutate
// to true.
func (op *OrOp) Eval(env *Environment) (document.Value, error) {
	s, err := op.a.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err := s.IsTruthy()
	if err != nil {
		return falseLitteral, err
	}
	if isTruthy {
		return trueLitteral, nil
	}

	s, err = op.b.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err = s.IsTruthy()
	if err != nil {
		return falseLitteral, err
	}
	if isTruthy {
		return trueLitteral, nil
	}

	return falseLitteral, nil
}

// String implements the stringutil.Stringer interface.
func (op *OrOp) String() string {
	return stringutil.Sprintf("%v OR %v", op.a, op.b)
}
