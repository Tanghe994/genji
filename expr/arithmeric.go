package expr

import (
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/sql/scanner"
	"github.com/genjidb/genji/stringutil"
)

// IsArithmeticOperator returns true if e is one of
// +, -, *, /, %, &, |, or ^ operators.
func IsArithmeticOperator(op Operator) bool {
	switch op.(type) {
	case *addOp, *subOp, *mulOp, *divOp, *modOp,
		*bitwiseAndOp, *bitwiseOrOp, *bitwiseXorOp:
		return true
	}

	return false
}

type addOp struct {
	*simpleOperator
}

// Add creates an expression thats evaluates to the result of a + b.
func Add(a, b Expr) Expr {
	return &addOp{&simpleOperator{a, b, scanner.ADD}}
}

func (op addOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.Add(b)
}

func (op addOp) String() string {
	return stringutil.Sprintf("%v + %v", op.a, op.b)
}

type subOp struct {
	*simpleOperator
}

// Sub creates an expression thats evaluates to the result of a - b.
func Sub(a, b Expr) Expr {
	return &subOp{&simpleOperator{a, b, scanner.SUB}}
}

func (op subOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.Sub(b)
}

func (op subOp) String() string {
	return stringutil.Sprintf("%v - %v", op.a, op.b)
}

type mulOp struct {
	*simpleOperator
}

// Mul creates an expression thats evaluates to the result of a * b.
func Mul(a, b Expr) Expr {
	return &mulOp{&simpleOperator{a, b, scanner.MUL}}
}

func (op mulOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.Mul(b)
}

func (op mulOp) String() string {
	return stringutil.Sprintf("%v * %v", op.a, op.b)
}

type divOp struct {
	*simpleOperator
}

// Div creates an expression thats evaluates to the result of a / b.
func Div(a, b Expr) Expr {
	return &divOp{&simpleOperator{a, b, scanner.DIV}}
}

func (op divOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.Div(b)
}

func (op divOp) String() string {
	return stringutil.Sprintf("%v / %v", op.a, op.b)
}

type modOp struct {
	*simpleOperator
}

// Mod creates an expression thats evaluates to the result of a % b.
func Mod(a, b Expr) Expr {
	return &modOp{&simpleOperator{a, b, scanner.MOD}}
}

func (op modOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.Mod(b)
}

func (op modOp) String() string {
	return stringutil.Sprintf("%v %% %v", op.a, op.b)
}

type bitwiseAndOp struct {
	*simpleOperator
}

// BitwiseAnd creates an expression thats evaluates to the result of a & b.
func BitwiseAnd(a, b Expr) Expr {
	return &bitwiseAndOp{&simpleOperator{a, b, scanner.BITWISEAND}}
}

func (op bitwiseAndOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.BitwiseAnd(b)
}

func (op bitwiseAndOp) String() string {
	return stringutil.Sprintf("%v & %v", op.a, op.b)
}

type bitwiseOrOp struct {
	*simpleOperator
}

// BitwiseOr creates an expression thats evaluates to the result of a | b.
func BitwiseOr(a, b Expr) Expr {
	return &bitwiseOrOp{&simpleOperator{a, b, scanner.BITWISEOR}}
}

func (op bitwiseOrOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.BitwiseOr(b)
}

func (op bitwiseOrOp) String() string {
	return stringutil.Sprintf("%v | %v", op.a, op.b)
}

type bitwiseXorOp struct {
	*simpleOperator
}

// BitwiseXor creates an expression thats evaluates to the result of a ^ b.
func BitwiseXor(a, b Expr) Expr {
	return &bitwiseXorOp{&simpleOperator{a, b, scanner.BITWISEXOR}}
}

func (op bitwiseXorOp) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	return a.BitwiseXor(b)
}

func (op bitwiseXorOp) String() string {
	return stringutil.Sprintf("%v ^ %v", op.a, op.b)
}
