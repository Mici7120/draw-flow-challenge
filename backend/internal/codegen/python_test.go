package codegen_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/codegen"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ast  program.ValidatedAST
		want string
	}{
		{
			name: "empty AST produces empty string",
			ast:  program.ValidatedAST{},
			want: "",
		},
		{
			name: "AC-1 add 2 + 3 assign x and print x",
			ast: program.ValidatedAST{
				Statements: []program.Stmt{
					program.AssignStmt{
						BlockID: "a1",
						Name:    "x",
						Value: program.MathExpr{
							Operator: "+",
							Left:     program.NumberExpr{Value: 2},
							Right:    program.NumberExpr{Value: 3},
						},
					},
					program.OutputStmt{
						BlockID: "o1",
						Value:   program.VarExpr{Name: "x"},
					},
				},
			},
			want: "x = 2 + 3\nprint(x)",
		},
		{
			name: "nested math preserves visual precedence with parentheses",
			ast: program.ValidatedAST{
				Statements: []program.Stmt{
					program.AssignStmt{
						BlockID: "a1",
						Name:    "res",
						Value: program.MathExpr{
							Operator: "*",
							Left: program.MathExpr{
								Operator: "+",
								Left:     program.NumberExpr{Value: 2},
								Right:    program.NumberExpr{Value: 3},
							},
							Right: program.MathExpr{
								Operator: "/",
								Left:     program.NumberExpr{Value: 5},
								Right:    program.NumberExpr{Value: 2},
							},
						},
					},
				},
			},
			want: "res = (2 + 3) * (5 / 2)",
		},
		{
			name: "AC-3 if-else with both branches",
			ast: program.ValidatedAST{
				Statements: []program.Stmt{
					program.IfElseStmt{
						BlockID: "if1",
						Condition: program.CompareExpr{
							Operator: ">=",
							Left:     program.VarExpr{Name: "x"},
							Right:    program.NumberExpr{Value: 10},
						},
						Then: []program.Stmt{
							program.OutputStmt{BlockID: "o1", Value: program.NumberExpr{Value: 1}},
						},
						Else: []program.Stmt{
							program.OutputStmt{BlockID: "o2", Value: program.NumberExpr{Value: 0}},
						},
					},
				},
			},
			want: "if x >= 10:\n    print(1)\nelse:\n    print(0)",
		},
		{
			name: "AC-4 inclusive for with nested if",
			ast: program.ValidatedAST{
				Statements: []program.Stmt{
					program.AssignStmt{BlockID: "a0", Name: "total", Value: program.NumberExpr{Value: 0}},
					program.ForStmt{
						BlockID:  "f1",
						Variable: "i",
						From:     program.NumberExpr{Value: 1},
						To:       program.NumberExpr{Value: 4},
						Body: []program.Stmt{
							program.IfElseStmt{
								BlockID: "if1",
								Condition: program.CompareExpr{
									Operator: ">",
									Left:     program.VarExpr{Name: "i"},
									Right:    program.NumberExpr{Value: 0},
								},
								Then: []program.Stmt{
									program.AssignStmt{
										BlockID: "a1",
										Name:    "total",
										Value: program.MathExpr{
											Operator: "+",
											Left:     program.VarExpr{Name: "total"},
											Right:    program.VarExpr{Name: "i"},
										},
									},
								},
							},
						},
					},
				},
			},
			want: strings.Join([]string{
				"total = 0",
				"for i in range(1, 4 + 1):",
				"    if i > 0:",
				"        total = total + i",
			}, "\n"),
		},
		{
			name: "AC-5 empty if then body and empty for body emit pass",
			ast: program.ValidatedAST{
				Statements: []program.Stmt{
					program.IfElseStmt{
						BlockID: "if1",
						Condition: program.CompareExpr{
							Operator: "==",
							Left:     program.NumberExpr{Value: 1},
							Right:    program.NumberExpr{Value: 1},
						},
					},
					program.ForStmt{
						BlockID:  "f1",
						Variable: "i",
						From:     program.NumberExpr{Value: 5},
						To:       program.NumberExpr{Value: 2},
					},
				},
			},
			want: strings.Join([]string{
				"if 1 == 1:",
				"    pass",
				"for i in range(5, 2 + 1):",
				"    pass",
			}, "\n"),
		},
		{
			name: "float literals and negative zero format cleanly",
			ast: program.ValidatedAST{
				Statements: []program.Stmt{
					program.AssignStmt{BlockID: "a1", Name: "pi", Value: program.NumberExpr{Value: 3.14}},
					program.AssignStmt{BlockID: "a2", Name: "z", Value: program.NumberExpr{Value: -0.0}},
				},
			},
			want: "pi = 3.14\nz = 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := codegen.Generate(tc.ast)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("Generate() =\n%s\n\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestGenerate_100NodesPerformanceBudget(t *testing.T) {
	t.Parallel()

	// Build a 100-node valid graph (1 start + 99 assignments) and verify validate + generate <= 150 ms (AC-12).
	nodes := make([]program.Node, 0, program.MaxNodes)
	edges := make([]program.Edge, 0, program.MaxNodes-1)
	nodes = append(nodes, program.Node{ID: "s", Kind: program.KindStart})
	prev := "s"
	for i := 1; i < program.MaxNodes; i++ {
		id := fmt.Sprintf("a%d", i)
		v := float64(i)
		nodes = append(nodes, program.Node{
			ID:   id,
			Kind: program.KindAssign,
			Data: program.NodeData{Name: fmt.Sprintf("v%d", i), Value: &v},
		})
		edges = append(edges, program.Edge{
			ID:         fmt.Sprintf("e%d", i),
			Source:     prev,
			SourcePort: "next",
			Target:     id,
			TargetPort: "in",
		})
		prev = id
	}

	start := time.Now()
	ast, verr := program.ValidateForRun(program.Graph{Nodes: nodes, Edges: edges})
	if verr != nil {
		t.Fatalf("ValidateForRun failed: %v", verr)
	}
	py, err := codegen.Generate(ast)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if strings.Count(py, "\n")+1 != 99 {
		t.Fatalf("expected 99 generated lines, got %d", strings.Count(py, "\n")+1)
	}
	if elapsed > 150*time.Millisecond {
		t.Errorf("100-node translation took %v, exceeds 150ms budget", elapsed)
	}
}

func TestGenerate_ForExpressionBoundsAndErrors(t *testing.T) {
	t.Parallel()

	t.Run("for loop with math expression to-bound parenthesizes cleanly", func(t *testing.T) {
		t.Parallel()
		ast := program.ValidatedAST{
			Statements: []program.Stmt{
				program.ForStmt{
					BlockID:  "f1",
					Variable: "i",
					From:     program.VarExpr{Name: "a"},
					To: program.MathExpr{
						Operator: "+",
						Left:     program.VarExpr{Name: "b"},
						Right:    program.NumberExpr{Value: 2},
					},
					Body: []program.Stmt{
						program.OutputStmt{BlockID: "o1", Value: program.VarExpr{Name: "i"}},
					},
				},
			},
		}
		got, err := codegen.Generate(ast)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		want := "for i in range(a, (b + 2) + 1):\n    print(i)"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("nil statement or expression returns error", func(t *testing.T) {
		t.Parallel()
		badCases := []program.ValidatedAST{
			{Statements: []program.Stmt{nil}},
			{Statements: []program.Stmt{program.AssignStmt{Name: "x", Value: nil}}},
			{Statements: []program.Stmt{program.OutputStmt{Value: nil}}},
			{Statements: []program.Stmt{program.IfElseStmt{Condition: program.CompareExpr{Operator: "==", Left: nil, Right: program.NumberExpr{Value: 1}}}}},
			{Statements: []program.Stmt{program.ForStmt{Variable: "i", From: nil, To: program.NumberExpr{Value: 1}}}},
			{Statements: []program.Stmt{program.AssignStmt{Name: "x", Value: program.MathExpr{Operator: "+", Left: nil, Right: program.NumberExpr{Value: 1}}}}},
		}
		for i, ast := range badCases {
			if _, err := codegen.Generate(ast); err == nil {
				t.Errorf("case %d: expected error for malformed AST, got nil", i)
			}
		}
	})
}
