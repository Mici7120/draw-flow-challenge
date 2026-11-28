package program_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
)

func f64(v float64) *float64 { return &v }

func TestValidateDraft(t *testing.T) {
	t.Parallel()

	tooManyNodes := make([]program.Node, program.MaxNodes+1)
	for i := range tooManyNodes {
		tooManyNodes[i] = program.Node{ID: fmt.Sprintf("n%d", i), Kind: program.KindNumber, Data: program.NodeData{Value: f64(1)}}
	}

	tests := []struct {
		name     string
		progName string
		graph    program.Graph
		wantName string
		wantCode string
	}{
		{
			name:     "valid incomplete draft with floating block and missing ports",
			progName: "  My First Draft  ",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "n1", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}},
				},
			},
			wantName: "My First Draft",
		},
		{
			name:     "blank name rejected",
			progName: "   ",
			graph:    program.Graph{},
			wantCode: "invalid_name",
		},
		{
			name:     "name over 120 chars rejected",
			progName: strings.Repeat("a", program.MaxNameLength+1),
			graph:    program.Graph{},
			wantCode: "invalid_name",
		},
		{
			name:     "name exactly 120 runes allowed",
			progName: strings.Repeat("é", program.MaxNameLength),
			graph:    program.Graph{},
			wantName: strings.Repeat("é", program.MaxNameLength),
		},
		{
			name:     "more than 100 nodes rejected",
			progName: "Too Big",
			graph:    program.Graph{Nodes: tooManyNodes},
			wantCode: "max_nodes_exceeded",
		},
		{
			name:     "duplicate node IDs rejected",
			progName: "Dup",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "n1", Kind: program.KindStart},
					{ID: "n1", Kind: program.KindOutput},
				},
			},
			wantCode: "duplicate_node_id",
		},
		{
			name:     "unknown node kind rejected",
			progName: "Unknown",
			graph: program.Graph{
				Nodes: []program.Node{{ID: "n1", Kind: "http_request"}},
			},
			wantCode: "invalid_node_kind",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotName, verr := program.ValidateDraft(tc.progName, tc.graph)
			if tc.wantCode == "" {
				if verr != nil {
					t.Fatalf("expected valid draft, got error: %v", verr)
				}
				if gotName != tc.wantName {
					t.Errorf("trimmed name = %q, want %q", gotName, tc.wantName)
				}
				return
			}
			if verr == nil {
				t.Fatalf("expected validation error code %q, got nil", tc.wantCode)
			}
			if !hasIssueCode(verr, tc.wantCode) {
				t.Errorf("issues = %+v, want code %q", verr.Issues, tc.wantCode)
			}
		})
	}
}

func TestValidateForRun_ValidGraphs(t *testing.T) {
	t.Parallel()

	t.Run("empty graph succeeds with empty AST", func(t *testing.T) {
		t.Parallel()
		ast, verr := program.ValidateForRun(program.Graph{})
		if verr != nil {
			t.Fatalf("unexpected error: %v", verr)
		}
		if len(ast.Statements) != 0 {
			t.Errorf("statements = %d, want 0", len(ast.Statements))
		}
	})

	t.Run("AC-1 add 2 + 3 assign x and print x", func(t *testing.T) {
		t.Parallel()
		g := program.Graph{
			Nodes: []program.Node{
				{ID: "s", Kind: program.KindStart},
				{ID: "n2", Kind: program.KindNumber, Data: program.NodeData{Value: f64(2)}},
				{ID: "n3", Kind: program.KindNumber, Data: program.NodeData{Value: f64(3)}},
				{ID: "add", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}},
				{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x"}},
				{ID: "vx", Kind: program.KindVariable, Data: program.NodeData{Name: "x"}},
				{ID: "out", Kind: program.KindOutput},
			},
			Edges: []program.Edge{
				{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
				{ID: "e2", Source: "n2", SourcePort: "value", Target: "add", TargetPort: "left"},
				{ID: "e3", Source: "n3", SourcePort: "value", Target: "add", TargetPort: "right"},
				{ID: "e4", Source: "add", SourcePort: "value", Target: "asg", TargetPort: "value"},
				{ID: "e5", Source: "asg", SourcePort: "next", Target: "out", TargetPort: "in"},
				{ID: "e6", Source: "vx", SourcePort: "value", Target: "out", TargetPort: "value"},
			},
		}
		ast, verr := program.ValidateForRun(g)
		if verr != nil {
			t.Fatalf("unexpected error: %v", verr)
		}
		if len(ast.Statements) != 2 {
			t.Fatalf("len(Statements) = %d, want 2", len(ast.Statements))
		}
		asg, ok := ast.Statements[0].(program.AssignStmt)
		if !ok || asg.Name != "x" {
			t.Fatalf("stmt[0] = %#v, want AssignStmt x", ast.Statements[0])
		}
		if _, ok := asg.Value.(program.MathExpr); !ok {
			t.Fatalf("AssignStmt.Value = %#v, want MathExpr", asg.Value)
		}
		if _, ok := ast.Statements[1].(program.OutputStmt); !ok {
			t.Fatalf("stmt[1] = %#v, want OutputStmt", ast.Statements[1])
		}
	})

	t.Run("code_block + if_else + inclusive for with variable bound", func(t *testing.T) {
		t.Parallel()
		g := program.Graph{
			Nodes: []program.Node{
				{ID: "s", Kind: program.KindStart},
				{ID: "blk", Kind: program.KindCodeBlock},
				{ID: "n_asg", Kind: program.KindAssign, Data: program.NodeData{Name: "n", Value: f64(4)}},
				{ID: "loop", Kind: program.KindFor, Data: program.NodeData{Name: "i", From: f64(1)}},
				{ID: "vn", Kind: program.KindVariable, Data: program.NodeData{Name: "n"}},
				{ID: "cond", Kind: program.KindIfElse, Data: program.NodeData{Operator: ">=", Right: f64(2)}},
				{ID: "vi", Kind: program.KindVariable, Data: program.NodeData{Name: "i"}},
				{ID: "then_blk", Kind: program.KindCodeBlock},
				{ID: "out", Kind: program.KindOutput, Data: program.NodeData{Value: f64(1)}},
			},
			Edges: []program.Edge{
				{ID: "e1", Source: "s", SourcePort: "next", Target: "blk", TargetPort: "in"},
				{ID: "e2", Source: "blk", SourcePort: "body", Target: "n_asg", TargetPort: "in"},
				{ID: "e3", Source: "n_asg", SourcePort: "next", Target: "loop", TargetPort: "in"},
				{ID: "e4", Source: "vn", SourcePort: "value", Target: "loop", TargetPort: "to"},
				{ID: "e5", Source: "loop", SourcePort: "body", Target: "cond", TargetPort: "in"},
				{ID: "e6", Source: "vi", SourcePort: "value", Target: "cond", TargetPort: "left"},
				{ID: "e7", Source: "cond", SourcePort: "then", Target: "then_blk", TargetPort: "in"},
				{ID: "e8", Source: "then_blk", SourcePort: "body", Target: "out", TargetPort: "in"},
			},
		}
		ast, verr := program.ValidateForRun(g)
		if verr != nil {
			t.Fatalf("unexpected error: %v", verr)
		}
		if len(ast.Statements) != 2 {
			t.Fatalf("len(Statements) = %d, want 2", len(ast.Statements))
		}
		loopStmt, ok := ast.Statements[1].(program.ForStmt)
		if !ok || loopStmt.Variable != "i" || len(loopStmt.Body) != 1 {
			t.Fatalf("stmt[1] = %#v, want ForStmt with 1 body stmt", ast.Statements[1])
		}
		ifStmt, ok := loopStmt.Body[0].(program.IfElseStmt)
		if !ok || len(ifStmt.Then) != 1 || len(ifStmt.Else) != 0 {
			t.Fatalf("loop body[0] = %#v, want IfElseStmt with 1 then, 0 else", loopStmt.Body[0])
		}
	})
}

func TestValidateForRun_InvalidGraphs(t *testing.T) {
	t.Parallel()

	tooMany := make([]program.Node, program.MaxNodes+1)
	for i := range tooMany {
		tooMany[i] = program.Node{ID: fmt.Sprintf("n%d", i), Kind: program.KindNumber, Data: program.NodeData{Value: f64(1)}}
	}

	tests := []struct {
		name        string
		graph       program.Graph
		wantCode    string
		wantBlockID string
	}{
		{
			name:        "more than 100 nodes rejected",
			graph:       program.Graph{Nodes: tooMany},
			wantCode:    "max_nodes_exceeded",
			wantBlockID: "",
		},
		{
			name: "floating unconnected block rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(1)}},
					{ID: "orphan", Kind: program.KindAssign, Data: program.NodeData{Name: "y", Value: f64(2)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
				},
			},
			wantCode:    "floating_block",
			wantBlockID: "orphan",
		},
		{
			name: "missing required math operand rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "m1", Kind: program.KindMath, Data: program.NodeData{Operator: "+", Left: f64(2)}},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x"}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
					{ID: "e2", Source: "m1", SourcePort: "value", Target: "asg", TargetPort: "value"},
				},
			},
			wantCode:    "missing_input",
			wantBlockID: "m1",
		},
		{
			name: "cyclic math expression rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "m1", Kind: program.KindMath, Data: program.NodeData{Operator: "+", Left: f64(1)}},
					{ID: "m2", Kind: program.KindMath, Data: program.NodeData{Operator: "*", Left: f64(2)}},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x"}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
					{ID: "e2", Source: "m1", SourcePort: "value", Target: "asg", TargetPort: "value"},
					{ID: "e3", Source: "m2", SourcePort: "value", Target: "m1", TargetPort: "right"},
					{ID: "e4", Source: "m1", SourcePort: "value", Target: "m2", TargetPort: "right"},
				},
			},
			wantCode:    "cycle_detected",
			wantBlockID: "m1",
		},
		{
			name: "cyclic statement flow rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "a1", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(1)}},
					{ID: "a2", Kind: program.KindAssign, Data: program.NodeData{Name: "y", Value: f64(2)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "a1", TargetPort: "in"},
					{ID: "e2", Source: "a1", SourcePort: "next", Target: "a2", TargetPort: "in"},
					{ID: "e3", Source: "a2", SourcePort: "next", Target: "a1", TargetPort: "in"},
				},
			},
			wantCode:    "cycle_detected",
			wantBlockID: "a1",
		},
		{
			name: "reserved Python keyword variable rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "for", Value: f64(1)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
				},
			},
			wantCode:    "invalid_identifier",
			wantBlockID: "asg",
		},
		{
			name: "unsafe builtin or dunder variable name rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "__import__", Value: f64(1)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
				},
			},
			wantCode:    "invalid_identifier",
			wantBlockID: "asg",
		},
		{
			name: "injection characters in variable name rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x; import os", Value: f64(1)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
				},
			},
			wantCode:    "invalid_identifier",
			wantBlockID: "asg",
		},
		{
			name: "non-integer inline literal for bound rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "loop", Kind: program.KindFor, Data: program.NodeData{Name: "i", From: f64(1.5), To: f64(5)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "loop", TargetPort: "in"},
				},
			},
			wantCode:    "non_integer_bound",
			wantBlockID: "loop",
		},
		{
			name: "non-integer connected number node for bound rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "loop", Kind: program.KindFor, Data: program.NodeData{Name: "i", From: f64(1)}},
					{ID: "badnum", Kind: program.KindNumber, Data: program.NodeData{Value: f64(3.25)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "loop", TargetPort: "in"},
					{ID: "e2", Source: "badnum", SourcePort: "value", Target: "loop", TargetPort: "to"},
				},
			},
			wantCode:    "non_integer_bound",
			wantBlockID: "loop",
		},
		{
			name: "unsupported math operator modulo rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "m1", Kind: program.KindMath, Data: program.NodeData{Operator: "%", Left: f64(5), Right: f64(2)}},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x"}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
					{ID: "e2", Source: "m1", SourcePort: "value", Target: "asg", TargetPort: "value"},
				},
			},
			wantCode:    "invalid_operator",
			wantBlockID: "m1",
		},
		{
			name: "non-finite number literal rejected",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(math.Inf(1))}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
				},
			},
			wantCode:    "invalid_number",
			wantBlockID: "asg",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, verr := program.ValidateForRun(tc.graph)
			if verr == nil {
				t.Fatalf("expected validation error %q, got nil", tc.wantCode)
			}
			if !hasIssueCode(verr, tc.wantCode) {
				t.Errorf("issues = %+v, want code %q", verr.Issues, tc.wantCode)
			}
			if tc.wantBlockID != "" && !hasIssueBlock(verr, tc.wantBlockID) {
				t.Errorf("issues = %+v, want block_id %q", verr.Issues, tc.wantBlockID)
			}
		})
	}
}

func hasIssueCode(verr *program.ValidationError, code string) bool {
	for _, iss := range verr.Issues {
		if iss.Code == code {
			return true
		}
	}
	return false
}

func hasIssueBlock(verr *program.ValidationError, blockID string) bool {
	for _, iss := range verr.Issues {
		if iss.BlockID == blockID {
			return true
		}
	}
	return false
}

func TestValidateForRun_EdgeCasesAndErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		graph    program.Graph
		wantCode string
	}{
		{
			name: "missing start block",
			graph: program.Graph{
				Nodes: []program.Node{{ID: "a1", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(1)}}},
			},
			wantCode: "missing_start",
		},
		{
			name: "multiple start blocks",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s1", Kind: program.KindStart},
					{ID: "s2", Kind: program.KindStart},
				},
			},
			wantCode: "multiple_start",
		},
		{
			name: "edge references unknown node",
			graph: program.Graph{
				Nodes: []program.Node{{ID: "s", Kind: program.KindStart}},
				Edges: []program.Edge{{ID: "e1", Source: "s", SourcePort: "next", Target: "ghost", TargetPort: "in"}},
			},
			wantCode: "invalid_edge",
		},
		{
			name: "edge missing port",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "a1", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(1)}},
				},
				Edges: []program.Edge{{ID: "e1", Source: "s", SourcePort: "", Target: "a1", TargetPort: "in"}},
			},
			wantCode: "invalid_edge",
		},
		{
			name: "ambiguous outgoing statement port",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "a1", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(1)}},
					{ID: "a2", Kind: program.KindAssign, Data: program.NodeData{Name: "y", Value: f64(2)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "a1", TargetPort: "in"},
					{ID: "e2", Source: "s", SourcePort: "next", Target: "a2", TargetPort: "in"},
				},
			},
			wantCode: "ambiguous_connection",
		},
		{
			name: "expression node placed in statement flow",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "n1", Kind: program.KindNumber, Data: program.NodeData{Value: f64(5)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "n1", TargetPort: "in"},
				},
			},
			wantCode: "invalid_connection",
		},
		{
			name: "statement node wired as expression value",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "out", Kind: program.KindOutput},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "out", TargetPort: "in"},
					{ID: "e2", Source: "s", SourcePort: "value", Target: "out", TargetPort: "value"},
				},
			},
			wantCode: "invalid_connection",
		},
		{
			name: "number node missing value",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "n1", Kind: program.KindNumber},
					{ID: "out", Kind: program.KindOutput},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "out", TargetPort: "in"},
					{ID: "e2", Source: "n1", SourcePort: "value", Target: "out", TargetPort: "value"},
				},
			},
			wantCode: "missing_input",
		},
		{
			name: "invalid comparison operator on if_else",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "cond", Kind: program.KindIfElse, Data: program.NodeData{Operator: "and", Left: f64(1), Right: f64(1)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "cond", TargetPort: "in"},
				},
			},
			wantCode: "invalid_operator",
		},
		{
			name: "invalid variable name on variable node and for loop",
			graph: program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "loop", Kind: program.KindFor, Data: program.NodeData{Name: "1bad", From: f64(1), To: f64(3)}},
					{ID: "vbad", Kind: program.KindVariable, Data: program.NodeData{Name: "eval"}},
					{ID: "out", Kind: program.KindOutput},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "loop", TargetPort: "in"},
					{ID: "e2", Source: "loop", SourcePort: "body", Target: "out", TargetPort: "in"},
					{ID: "e3", Source: "vbad", SourcePort: "value", Target: "out", TargetPort: "value"},
				},
			},
			wantCode: "invalid_identifier",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, verr := program.ValidateForRun(tc.graph)
			if verr == nil {
				t.Fatalf("expected error %q, got nil", tc.wantCode)
			}
			if !hasIssueCode(verr, tc.wantCode) {
				t.Errorf("issues = %+v, want %q", verr.Issues, tc.wantCode)
			}
			if verr.Error() == "" {
				t.Error("ValidationError.Error() returned empty string")
			}
		})
	}
}
