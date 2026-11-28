package program

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	identPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

	validKinds = map[NodeKind]bool{
		KindStart:     true,
		KindCodeBlock: true,
		KindNumber:    true,
		KindVariable:  true,
		KindAssign:    true,
		KindMath:      true,
		KindIfElse:    true,
		KindFor:       true,
		KindOutput:    true,
	}

	validMathOps = map[string]bool{
		"+": true,
		"-": true,
		"*": true,
		"/": true,
	}

	validCompareOps = map[string]bool{
		"==": true,
		"!=": true,
		"<":  true,
		"<=": true,
		">":  true,
		">=": true,
	}

	// Python 3.12 keywords plus disallowed builtins/modules that cannot be shadowed or used as identifiers.
	reservedIdentifiers = map[string]bool{
		"False": true, "None": true, "True": true, "and": true, "as": true,
		"assert": true, "async": true, "await": true, "break": true, "class": true,
		"continue": true, "def": true, "del": true, "elif": true, "else": true,
		"except": true, "finally": true, "for": true, "from": true, "global": true,
		"if": true, "import": true, "in": true, "is": true, "lambda": true,
		"nonlocal": true, "not": true, "or": true, "pass": true, "raise": true,
		"return": true, "try": true, "while": true, "with": true, "yield": true,
		// Disallowed builtins & sensitive module names
		"os": true, "subprocess": true, "socket": true, "sys": true, "open": true,
		"eval": true, "exec": true, "compile": true, "input": true, "breakpoint": true,
		"getattr": true, "setattr": true, "delattr": true, "globals": true, "locals": true,
		"vars": true, "dir": true, "type": true, "object": true, "print": true,
		"range": true, "int": true, "float": true, "bool": true, "exit": true, "quit": true,
		"builtins": true,
	}
)

// ValidateDraft checks draft-save invariants without requiring a complete runnable graph.
// It returns the trimmed program name on success.
func ValidateDraft(name string, g Graph) (string, *ValidationError) {
	var issues []ValidationIssue

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		issues = append(issues, ValidationIssue{
			Code:    "invalid_name",
			Message: "program name must not be blank",
		})
	} else if utf8.RuneCountInString(trimmed) > MaxNameLength {
		issues = append(issues, ValidationIssue{
			Code:    "invalid_name",
			Message: fmt.Sprintf("program name must be at most %d characters", MaxNameLength),
		})
	}

	issues = append(issues, validateStructure(g)...)
	if len(issues) > 0 {
		return "", &ValidationError{Issues: issues}
	}
	return trimmed, nil
}

// ValidateForRun verifies that a visual graph is complete, acyclic, free of floating blocks,
// and safe to translate into Python 3.12.
func ValidateForRun(g Graph) (ValidatedAST, *ValidationError) {
	issues := validateStructure(g)
	if len(issues) > 0 {
		return ValidatedAST{}, &ValidationError{Issues: issues}
	}

	if len(g.Nodes) == 0 {
		if len(g.Edges) > 0 {
			edgeIssues := make([]ValidationIssue, 0, len(g.Edges))
			for _, e := range g.Edges {
				edgeIssues = append(edgeIssues, ValidationIssue{
					BlockID: e.Source,
					Code:    "invalid_edge",
					Message: fmt.Sprintf("edge %q references unknown blocks in an empty program", e.ID),
				})
			}
			return ValidatedAST{}, &ValidationError{Issues: edgeIssues}
		}
		return ValidatedAST{}, nil
	}

	v := newValidator(g)
	ast := v.validate()
	if len(v.issues) > 0 {
		return ValidatedAST{}, &ValidationError{Issues: v.issues}
	}
	return ast, nil
}

func validateStructure(g Graph) []ValidationIssue {
	var issues []ValidationIssue
	if len(g.Nodes) > MaxNodes {
		issues = append(issues, ValidationIssue{
			Code:    "max_nodes_exceeded",
			Message: fmt.Sprintf("program cannot exceed %d nodes (got %d)", MaxNodes, len(g.Nodes)),
		})
		return issues
	}

	seen := make(map[string]bool, len(g.Nodes))
	for _, n := range g.Nodes {
		if strings.TrimSpace(n.ID) == "" {
			issues = append(issues, ValidationIssue{
				Code:    "invalid_node_id",
				Message: "every block must have a non-empty id",
			})
			continue
		}
		if seen[n.ID] {
			issues = append(issues, ValidationIssue{
				BlockID: n.ID,
				Code:    "duplicate_node_id",
				Message: fmt.Sprintf("duplicate block id %q", n.ID),
			})
			continue
		}
		seen[n.ID] = true

		if !validKinds[n.Kind] {
			issues = append(issues, ValidationIssue{
				BlockID: n.ID,
				Code:    "invalid_node_kind",
				Message: fmt.Sprintf("unsupported block kind %q", n.Kind),
			})
		}
	}
	return issues
}

const maxExpandedExprNodes = 1000

type portKey struct {
	nodeID string
	port   string
}

type indexedEdge struct {
	idx  int
	edge Edge
}

type validator struct {
	graph      Graph
	nodes      map[string]Node
	out        map[portKey][]indexedEdge
	in         map[portKey][]indexedEdge
	visited    map[string]bool
	usedEdges  map[int]bool
	exprOps    int
	exprCapped bool
	issues     []ValidationIssue
}

func newValidator(g Graph) *validator {
	v := &validator{
		graph:     g,
		nodes:     make(map[string]Node, len(g.Nodes)),
		out:       make(map[portKey][]indexedEdge),
		in:        make(map[portKey][]indexedEdge),
		visited:   make(map[string]bool, len(g.Nodes)),
		usedEdges: make(map[int]bool, len(g.Edges)),
	}
	for _, n := range g.Nodes {
		v.nodes[n.ID] = n
	}
	for i, e := range g.Edges {
		if _, ok := v.nodes[e.Source]; !ok {
			v.addIssue(e.Source, "invalid_edge", fmt.Sprintf("edge %q references unknown source block %q", e.ID, e.Source))
			continue
		}
		if _, ok := v.nodes[e.Target]; !ok {
			v.addIssue(e.Target, "invalid_edge", fmt.Sprintf("edge %q references unknown target block %q", e.ID, e.Target))
			continue
		}
		if e.SourcePort == "" || e.TargetPort == "" {
			v.addIssue(e.Source, "invalid_edge", fmt.Sprintf("edge %q must specify source_port and target_port", e.ID))
			continue
		}
		ie := indexedEdge{idx: i, edge: e}
		v.out[portKey{nodeID: e.Source, port: e.SourcePort}] = append(v.out[portKey{nodeID: e.Source, port: e.SourcePort}], ie)
		v.in[portKey{nodeID: e.Target, port: e.TargetPort}] = append(v.in[portKey{nodeID: e.Target, port: e.TargetPort}], ie)
	}
	return v
}

func (v *validator) addIssue(blockID, code, msg string) {
	v.issues = append(v.issues, ValidationIssue{
		BlockID: blockID,
		Code:    code,
		Message: msg,
	})
}

func (v *validator) validate() ValidatedAST {
	if len(v.issues) > 0 {
		return ValidatedAST{}
	}

	var starts []Node
	for _, n := range v.graph.Nodes {
		if n.Kind == KindStart {
			starts = append(starts, n)
		}
	}
	if len(starts) == 0 {
		v.addIssue("", "missing_start", "program must have a start block")
		return ValidatedAST{}
	}
	if len(starts) > 1 {
		for _, s := range starts[1:] {
			v.addIssue(s.ID, "multiple_start", "program can only have one start block")
		}
		return ValidatedAST{}
	}

	startNode := starts[0]
	v.visited[startNode.ID] = true

	stmtStack := make(map[string]bool)
	visitedStmts := make(map[string]bool)
	firstID, ok := v.singleTarget(startNode.ID, "next")
	var stmts []Stmt
	if ok {
		stmts = v.walkChain(firstID, stmtStack, visitedStmts)
	}

	for _, n := range v.graph.Nodes {
		if !v.visited[n.ID] {
			v.addIssue(n.ID, "floating_block", "block is not connected to the program entry flow")
		}
	}

	// Never silently drop edges: any edge whose endpoints were visited yet was not consumed
	// by a valid port traversal represents an invalid or misspelled port connection.
	for i, e := range v.graph.Edges {
		if v.usedEdges[i] {
			continue
		}
		if v.visited[e.Source] && v.visited[e.Target] {
			v.addIssue(e.Source, "invalid_connection", fmt.Sprintf("edge %q (%s.%s -> %s.%s) uses an invalid or unused port", e.ID, e.Source, e.SourcePort, e.Target, e.TargetPort))
		}
	}

	return ValidatedAST{Statements: stmts}
}

func (v *validator) singleTarget(nodeID, port string) (string, bool) {
	edges := v.out[portKey{nodeID: nodeID, port: port}]
	if len(edges) == 0 {
		return "", false
	}
	for _, ie := range edges {
		v.usedEdges[ie.idx] = true
	}
	if len(edges) > 1 {
		v.addIssue(nodeID, "ambiguous_connection", fmt.Sprintf("port %q has multiple outgoing connections", port))
		return "", false
	}
	if edges[0].edge.TargetPort != "in" {
		v.addIssue(nodeID, "invalid_connection", fmt.Sprintf("statement port %q must connect to target port \"in\"", port))
		return "", false
	}
	return edges[0].edge.Target, true
}

func (v *validator) walkChain(startID string, stmtStack, visitedStmts map[string]bool) []Stmt {
	var result []Stmt
	var pushed []string
	defer func() {
		for _, id := range pushed {
			delete(stmtStack, id)
		}
	}()

	curr := startID
	for curr != "" {
		if stmtStack[curr] {
			v.addIssue(curr, "cycle_detected", "statement flow contains a cycle")
			return result
		}
		if visitedStmts[curr] {
			v.addIssue(curr, "invalid_connection", "statement block is entered from multiple control paths")
			return result
		}

		node := v.nodes[curr]
		v.visited[curr] = true
		stmtStack[curr] = true
		pushed = append(pushed, curr)
		visitedStmts[curr] = true

		switch node.Kind {
		case KindCodeBlock:
			if bodyID, ok := v.singleTarget(node.ID, "body"); ok {
				bodyStmts := v.walkChain(bodyID, stmtStack, visitedStmts)
				result = append(result, bodyStmts...)
			}
		case KindAssign:
			if !isValidIdentifier(node.Data.Name) {
				v.addIssue(node.ID, "invalid_identifier", fmt.Sprintf("invalid or reserved variable name %q", node.Data.Name))
			}
			expr := v.resolveInput(node.ID, "value", node.Data.Value, make(map[string]bool))
			if expr != nil {
				result = append(result, AssignStmt{
					BlockID: node.ID,
					Name:    node.Data.Name,
					Value:   expr,
				})
			}
		case KindIfElse:
			if !validCompareOps[node.Data.Operator] {
				v.addIssue(node.ID, "invalid_operator", fmt.Sprintf("unsupported comparison operator %q", node.Data.Operator))
			}
			left := v.resolveInput(node.ID, "left", node.Data.Left, make(map[string]bool))
			right := v.resolveInput(node.ID, "right", node.Data.Right, make(map[string]bool))
			var thenStmts, elseStmts []Stmt
			if thenID, ok := v.singleTarget(node.ID, "then"); ok {
				thenStmts = v.walkChain(thenID, stmtStack, visitedStmts)
			}
			if elseID, ok := v.singleTarget(node.ID, "else"); ok {
				elseStmts = v.walkChain(elseID, stmtStack, visitedStmts)
			}
			if left != nil && right != nil {
				result = append(result, IfElseStmt{
					BlockID: node.ID,
					Condition: CompareExpr{
						Operator: node.Data.Operator,
						Left:     left,
						Right:    right,
					},
					Then: thenStmts,
					Else: elseStmts,
				})
			}
		case KindFor:
			if !isValidIdentifier(node.Data.Name) {
				v.addIssue(node.ID, "invalid_identifier", fmt.Sprintf("invalid or reserved loop variable name %q", node.Data.Name))
			}
			fromExpr := v.resolveInput(node.ID, "from", node.Data.From, make(map[string]bool))
			toExpr := v.resolveInput(node.ID, "to", node.Data.To, make(map[string]bool))
			v.checkIntegerLiteralBound(node.ID, "from", fromExpr)
			v.checkIntegerLiteralBound(node.ID, "to", toExpr)
			var bodyStmts []Stmt
			if bodyID, ok := v.singleTarget(node.ID, "body"); ok {
				bodyStmts = v.walkChain(bodyID, stmtStack, visitedStmts)
			}
			if fromExpr != nil && toExpr != nil {
				result = append(result, ForStmt{
					BlockID:  node.ID,
					Variable: node.Data.Name,
					From:     fromExpr,
					To:       toExpr,
					Body:     bodyStmts,
				})
			}
		case KindOutput:
			expr := v.resolveInput(node.ID, "value", node.Data.Value, make(map[string]bool))
			if expr != nil {
				result = append(result, OutputStmt{
					BlockID: node.ID,
					Value:   expr,
				})
			}
		default:
			v.addIssue(node.ID, "invalid_connection", fmt.Sprintf("block kind %q cannot appear in statement flow", node.Kind))
		}

		nextID, ok := v.singleTarget(node.ID, "next")
		if !ok {
			curr = ""
		} else {
			curr = nextID
		}
	}
	return result
}

func (v *validator) resolveInput(targetID, port string, inline *float64, exprStack map[string]bool) Expr {
	edges := v.in[portKey{nodeID: targetID, port: port}]
	for _, ie := range edges {
		v.usedEdges[ie.idx] = true
	}
	if len(edges) > 1 {
		v.addIssue(targetID, "ambiguous_connection", fmt.Sprintf("input port %q has multiple incoming connections", port))
		return nil
	}
	if len(edges) == 1 {
		edge := edges[0].edge
		if edge.SourcePort != "value" {
			v.addIssue(targetID, "invalid_connection", fmt.Sprintf("input port %q must connect from source port \"value\"", port))
			return nil
		}
		return v.resolveExprNode(edge.Source, exprStack)
	}
	if inline != nil {
		if math.IsNaN(*inline) || math.IsInf(*inline, 0) {
			v.addIssue(targetID, "invalid_number", fmt.Sprintf("input %q must be a finite number", port))
			return nil
		}
		return NumberExpr{Value: *inline}
	}
	v.addIssue(targetID, "missing_input", fmt.Sprintf("required input %q is not connected or provided", port))
	return nil
}

func (v *validator) resolveExprNode(nodeID string, exprStack map[string]bool) Expr {
	if exprStack[nodeID] {
		v.addIssue(nodeID, "cycle_detected", "expression connection contains a cycle")
		return nil
	}
	v.exprOps++
	if v.exprOps > maxExpandedExprNodes {
		if !v.exprCapped {
			v.exprCapped = true
			v.addIssue(nodeID, "expression_too_complex", fmt.Sprintf("expression expansion exceeds maximum complexity limit (%d)", maxExpandedExprNodes))
		}
		return nil
	}
	node := v.nodes[nodeID]
	v.visited[nodeID] = true
	exprStack[nodeID] = true
	defer delete(exprStack, nodeID)

	switch node.Kind {
	case KindNumber:
		if node.Data.Value == nil {
			v.addIssue(node.ID, "missing_input", "number block is missing a numeric value")
			return nil
		}
		if math.IsNaN(*node.Data.Value) || math.IsInf(*node.Data.Value, 0) {
			v.addIssue(node.ID, "invalid_number", "number block must contain a finite number")
			return nil
		}
		return NumberExpr{Value: *node.Data.Value}
	case KindVariable:
		if !isValidIdentifier(node.Data.Name) {
			v.addIssue(node.ID, "invalid_identifier", fmt.Sprintf("invalid or reserved variable name %q", node.Data.Name))
			return nil
		}
		return VarExpr{Name: node.Data.Name}
	case KindMath:
		if !validMathOps[node.Data.Operator] {
			v.addIssue(node.ID, "invalid_operator", fmt.Sprintf("unsupported math operator %q", node.Data.Operator))
		}
		left := v.resolveInput(node.ID, "left", node.Data.Left, exprStack)
		right := v.resolveInput(node.ID, "right", node.Data.Right, exprStack)
		if left == nil || right == nil || !validMathOps[node.Data.Operator] {
			return nil
		}
		return MathExpr{
			Operator: node.Data.Operator,
			Left:     left,
			Right:    right,
		}
	default:
		v.addIssue(node.ID, "invalid_connection", fmt.Sprintf("block kind %q cannot be used as a numeric expression", node.Kind))
		return nil
	}
}

func (v *validator) checkIntegerLiteralBound(blockID, port string, expr Expr) {
	num, ok := expr.(NumberExpr)
	if !ok {
		return
	}
	if math.Trunc(num.Value) != num.Value {
		v.addIssue(blockID, "non_integer_bound", fmt.Sprintf("for loop %q literal bound must be an integer", port))
	}
}

func isValidIdentifier(name string) bool {
	if !identPattern.MatchString(name) {
		return false
	}
	if strings.HasPrefix(name, "__") {
		return false
	}
	return !reservedIdentifiers[name]
}
