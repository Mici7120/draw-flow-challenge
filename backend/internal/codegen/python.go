package codegen

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
)

const indentUnit = "    "

// Generate deterministically translates a validated visual AST into readable Python 3.12 code.
func Generate(ast program.ValidatedAST) (string, error) {
	if len(ast.Statements) == 0 {
		return "", nil
	}
	var lines []string
	if err := emitStatements(&lines, ast.Statements, 0); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func emitStatements(lines *[]string, stmts []program.Stmt, depth int) error {
	prefix := strings.Repeat(indentUnit, depth)
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case program.AssignStmt:
			expr, err := emitExpr(s.Value, false)
			if err != nil {
				return err
			}
			*lines = append(*lines, fmt.Sprintf("%s%s = %s", prefix, s.Name, expr))
		case program.OutputStmt:
			expr, err := emitExpr(s.Value, false)
			if err != nil {
				return err
			}
			*lines = append(*lines, fmt.Sprintf("%sprint(%s)", prefix, expr))
		case program.IfElseStmt:
			left, err := emitExpr(s.Condition.Left, false)
			if err != nil {
				return err
			}
			right, err := emitExpr(s.Condition.Right, false)
			if err != nil {
				return err
			}
			*lines = append(*lines, fmt.Sprintf("%sif %s %s %s:", prefix, left, s.Condition.Operator, right))
			if len(s.Then) == 0 {
				*lines = append(*lines, fmt.Sprintf("%s%spass", prefix, indentUnit))
			} else if err := emitStatements(lines, s.Then, depth+1); err != nil {
				return err
			}
			if len(s.Else) > 0 {
				*lines = append(*lines, fmt.Sprintf("%selse:", prefix))
				if err := emitStatements(lines, s.Else, depth+1); err != nil {
					return err
				}
			}
		case program.ForStmt:
			fromExpr, err := emitExpr(s.From, false)
			if err != nil {
				return err
			}
			toExpr, err := emitExpr(s.To, true)
			if err != nil {
				return err
			}
			*lines = append(*lines, fmt.Sprintf("%sfor %s in range(%s, %s + 1):", prefix, s.Variable, fromExpr, toExpr))
			if len(s.Body) == 0 {
				*lines = append(*lines, fmt.Sprintf("%s%spass", prefix, indentUnit))
			} else if err := emitStatements(lines, s.Body, depth+1); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported statement type %T", stmt)
		}
	}
	return nil
}

func emitExpr(expr program.Expr, wrapMath bool) (string, error) {
	switch e := expr.(type) {
	case program.NumberExpr:
		return formatNumber(e.Value), nil
	case program.VarExpr:
		return e.Name, nil
	case program.MathExpr:
		left, err := emitExpr(e.Left, true)
		if err != nil {
			return "", err
		}
		right, err := emitExpr(e.Right, true)
		if err != nil {
			return "", err
		}
		rendered := fmt.Sprintf("%s %s %s", left, e.Operator, right)
		if wrapMath {
			return "(" + rendered + ")", nil
		}
		return rendered, nil
	default:
		return "", fmt.Errorf("unsupported expression type %T", expr)
	}
}

func formatNumber(v float64) string {
	if v == 0 {
		return "0"
	}
	if math.Trunc(v) == v && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
