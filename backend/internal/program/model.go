package program

import (
	"fmt"
	"strings"
	"time"
)

const (
	// MaxNodes is the maximum number of nodes allowed in a single visual program.
	MaxNodes = 100
	// MaxNameLength is the maximum rune length of a trimmed program name.
	MaxNameLength = 120
)

// NodeKind identifies the semantic role of a visual block.
type NodeKind string

const (
	KindStart     NodeKind = "start"
	KindCodeBlock NodeKind = "code_block"
	KindNumber    NodeKind = "number"
	KindVariable  NodeKind = "variable"
	KindAssign    NodeKind = "assign"
	KindMath      NodeKind = "math"
	KindIfElse    NodeKind = "if_else"
	KindFor       NodeKind = "for"
	KindOutput    NodeKind = "output"
)

// Position stores optional visual canvas coordinates preserved verbatim for the editor.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NodeData holds kind-specific block parameters.
type NodeData struct {
	Value    *float64 `json:"value,omitempty"`
	Name     string   `json:"name,omitempty"`
	Operator string   `json:"operator,omitempty"`
	Left     *float64 `json:"left,omitempty"`
	Right    *float64 `json:"right,omitempty"`
	From     *float64 `json:"from,omitempty"`
	To       *float64 `json:"to,omitempty"`
}

// Node represents one visual block inside a program graph.
type Node struct {
	ID       string    `json:"id"`
	Kind     NodeKind  `json:"kind"`
	Data     NodeData  `json:"data"`
	Position *Position `json:"position,omitempty"`
}

// Edge connects a source node port to a target node port.
type Edge struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	SourcePort string `json:"source_port"`
	Target     string `json:"target"`
	TargetPort string `json:"target_port"`
}

// Graph is the normalized visual program representation shared across persistence and validation.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Program is a persisted visual program record.
type Program struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Graph     Graph     `json:"graph"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProgramSummary is the lightweight projection returned when listing saved programs.
type ProgramSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ValidationIssue describes a single validation failure tied to an optional block ID.
type ValidationIssue struct {
	BlockID string `json:"block_id,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationError collects one or more validation issues.
type ValidationError struct {
	Issues []ValidationIssue
}

// Error formats the validation issues into a concise, safe message.
func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(e.Issues))
	for _, iss := range e.Issues {
		if iss.BlockID != "" {
			parts = append(parts, fmt.Sprintf("block %s: %s", iss.BlockID, iss.Message))
		} else {
			parts = append(parts, iss.Message)
		}
	}
	return strings.Join(parts, "; ")
}

// ValidatedAST is the verified, acyclic statement sequence ready for code generation.
type ValidatedAST struct {
	Statements []Stmt
}

// Stmt marks a validated executable statement node.
type Stmt interface {
	stmtNode()
}

// Expr marks a validated numeric expression node.
type Expr interface {
	exprNode()
}

// AssignStmt binds a validated variable identifier to an expression.
type AssignStmt struct {
	BlockID string
	Name    string
	Value   Expr
}

func (AssignStmt) stmtNode() {}

// CompareExpr compares two numeric expressions with one of ==, !=, <, <=, >, >=.
type CompareExpr struct {
	Operator string
	Left     Expr
	Right    Expr
}

// IfElseStmt executes Then when Condition is true, otherwise Else.
type IfElseStmt struct {
	BlockID   string
	Condition CompareExpr
	Then      []Stmt
	Else      []Stmt
}

func (IfElseStmt) stmtNode() {}

// ForStmt iterates Variable inclusively from From through To.
type ForStmt struct {
	BlockID  string
	Variable string
	From     Expr
	To       Expr
	Body     []Stmt
}

func (ForStmt) stmtNode() {}

// OutputStmt prints an expression's evaluated numeric value.
type OutputStmt struct {
	BlockID string
	Value   Expr
}

func (OutputStmt) stmtNode() {}

// NumberExpr represents a numeric literal.
type NumberExpr struct {
	Value float64
}

func (NumberExpr) exprNode() {}

// VarExpr references a validated variable identifier.
type VarExpr struct {
	Name string
}

func (VarExpr) exprNode() {}

// MathExpr combines two expressions with one of +, -, *, /.
type MathExpr struct {
	Operator string
	Left     Expr
	Right    Expr
}

func (MathExpr) exprNode() {}
