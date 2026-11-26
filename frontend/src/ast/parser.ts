import type {
  CompareOperator,
  Diagnostic,
  DrawflowExport,
  DrawflowNode,
  ExpressionNode,
  MathOperator,
  ProgramAST,
  StatementNode,
} from './types'

const VALID_IDENTIFIER = /^[A-Za-z_][A-Za-z0-9_]*$/
const VALID_MATH_OPS = new Set<MathOperator>(['+', '-', '*', '/', '%'])
const VALID_COMPARE_OPS = new Set<CompareOperator>(['==', '!=', '<', '<=', '>', '>='])

export function parseDrawflowToAST(exported: DrawflowExport | null | undefined): {
  ast: ProgramAST
  diagnostics: Diagnostic[]
} {
  const diagnostics: Diagnostic[] = []
  const nodes = exported?.drawflow?.Home?.data ?? {}
  const nodeEntries = Object.values(nodes)

  if (nodeEntries.length === 0) {
    return {
      ast: { kind: 'Program', body: [] },
      diagnostics,
    }
  }

  const startNodes = nodeEntries.filter((n) => n.name === 'start')
  let entryNodeIds: string[] = []

  if (startNodes.length > 0) {
    if (startNodes.length > 1) {
      diagnostics.push({
        level: 'warning',
        message: 'Multiple Start nodes detected; using the first Start node.',
      })
    }
    const firstStart = startNodes[0]
    entryNodeIds = getOutputTargets(firstStart, 'output_1')
  } else {
    // Fallback: allow flow starting from root statement nodes if no Start node exists
    const statementKinds = new Set(['block', 'assign', 'if_else', 'for_loop', 'print'])
    const rootStatements = nodeEntries
      .filter((n) => statementKinds.has(n.name) && getInputSources(n, 'input_1').length === 0)
      .sort((a, b) => a.pos_y - b.pos_y || a.pos_x - b.pos_x)

    if (rootStatements.length > 0) {
      diagnostics.push({
        level: 'warning',
        message: 'No Start node found; compiling from top-level statement nodes.',
      })
      entryNodeIds = rootStatements.map((n) => String(n.id))
    } else {
      diagnostics.push({
        level: 'warning',
        message: 'Add a Start node and connect statements to generate Python code.',
      })
    }
  }

  const body: StatementNode[] = []
  for (const targetId of entryNodeIds) {
    body.push(...parseFlowChain(targetId, nodes, new Set(), diagnostics))
  }

  return {
    ast: { kind: 'Program', body },
    diagnostics,
  }
}

function parseFlowChain(
  startNodeId: string,
  nodes: Record<string, DrawflowNode>,
  activeStack: Set<string>,
  diagnostics: Diagnostic[],
): StatementNode[] {
  const statements: StatementNode[] = []
  let currentId: string | undefined = startNodeId
  const localVisited = new Set<string>()

  while (currentId) {
    if (activeStack.has(currentId) || localVisited.has(currentId)) {
      diagnostics.push({
        level: 'error',
        nodeId: currentId,
        message: `Cycle detected in control flow at node #${currentId}.`,
      })
      break
    }

    const node: DrawflowNode | undefined = nodes[currentId]
    if (!node) {
      break
    }

    localVisited.add(currentId)
    const nextStack = new Set(activeStack)
    nextStack.add(currentId)

    switch (node.name) {
      case 'block': {
        const label = readString(node.data, 'label', 'Code Block').trim()
        const bodyTargets = getOutputTargets(node, 'output_1')
        const blockBody: StatementNode[] = []
        for (const t of bodyTargets) {
          blockBody.push(...parseFlowChain(t, nodes, nextStack, diagnostics))
        }
        statements.push({
          kind: 'Block',
          label,
          body: blockBody,
        })
        currentId = getOutputTargets(node, 'output_2')[0]
        break
      }

      case 'assign': {
        const variable = sanitizeIdentifier(
          readString(node.data, 'variable', 'x'),
          'x',
          String(node.id),
          diagnostics,
        )
        const valueExpr = resolveInputExpression(
          node,
          'input_2',
          readString(node.data, 'value', '0'),
          nodes,
          new Set(),
          diagnostics,
        )
        statements.push({
          kind: 'Assign',
          variable,
          value: valueExpr,
        })
        currentId = getOutputTargets(node, 'output_1')[0]
        break
      }

      case 'if_else': {
        const condition = resolveInputExpression(
          node,
          'input_2',
          readString(node.data, 'condition', 'True'),
          nodes,
          new Set(),
          diagnostics,
        )
        const thenBranch: StatementNode[] = []
        for (const t of getOutputTargets(node, 'output_1')) {
          thenBranch.push(...parseFlowChain(t, nodes, nextStack, diagnostics))
        }
        const elseBranch: StatementNode[] = []
        for (const t of getOutputTargets(node, 'output_2')) {
          elseBranch.push(...parseFlowChain(t, nodes, nextStack, diagnostics))
        }
        statements.push({
          kind: 'IfElse',
          condition,
          thenBranch,
          elseBranch,
        })
        currentId = getOutputTargets(node, 'output_3')[0]
        break
      }

      case 'for_loop': {
        const iterator = sanitizeIdentifier(
          readString(node.data, 'iterator', 'i'),
          'i',
          String(node.id),
          diagnostics,
        )
        const fromExpr = resolveInputExpression(
          node,
          'input_2',
          readString(node.data, 'from', '1'),
          nodes,
          new Set(),
          diagnostics,
        )
        const toExpr = resolveInputExpression(
          node,
          'input_3',
          readString(node.data, 'to', '5'),
          nodes,
          new Set(),
          diagnostics,
        )
        const inclusive = readBoolean(node.data, 'inclusive', true)
        const body: StatementNode[] = []
        for (const t of getOutputTargets(node, 'output_1')) {
          body.push(...parseFlowChain(t, nodes, nextStack, diagnostics))
        }
        statements.push({
          kind: 'ForLoop',
          iterator,
          from: fromExpr,
          to: toExpr,
          inclusive,
          body,
        })
        currentId = getOutputTargets(node, 'output_2')[0]
        break
      }

      case 'print': {
        const valueExpr = resolveInputExpression(
          node,
          'input_2',
          readString(node.data, 'value', '0'),
          nodes,
          new Set(),
          diagnostics,
        )
        statements.push({
          kind: 'Print',
          value: valueExpr,
        })
        currentId = getOutputTargets(node, 'output_1')[0]
        break
      }

      default: {
        diagnostics.push({
          level: 'warning',
          nodeId: currentId,
          message: `Node #${currentId} (${node.name}) is an expression node connected to a control-flow port.`,
        })
        currentId = undefined
        break
      }
    }
  }

  return statements
}

function resolveInputExpression(
  node: DrawflowNode,
  inputPort: string,
  fallbackRaw: string,
  nodes: Record<string, DrawflowNode>,
  visitedExpr: Set<string>,
  diagnostics: Diagnostic[],
): ExpressionNode {
  const sources = getInputSources(node, inputPort)
  if (sources.length > 0) {
    return parseExpressionNode(sources[0], nodes, visitedExpr, diagnostics)
  }
  const trimmed = fallbackRaw.trim()
  if (trimmed === '') {
    return { kind: 'NumberLiteral', value: '0' }
  }
  if (/^-?\d+(\.\d+)?$/.test(trimmed)) {
    return { kind: 'NumberLiteral', value: trimmed }
  }
  if (VALID_IDENTIFIER.test(trimmed)) {
    return { kind: 'VariableRef', name: trimmed }
  }
  return { kind: 'RawExpression', code: trimmed }
}

function parseExpressionNode(
  nodeId: string,
  nodes: Record<string, DrawflowNode>,
  visitedExpr: Set<string>,
  diagnostics: Diagnostic[],
): ExpressionNode {
  if (visitedExpr.has(nodeId)) {
    diagnostics.push({
      level: 'error',
      nodeId,
      message: `Cycle detected in expression graph at node #${nodeId}.`,
    })
    return { kind: 'NumberLiteral', value: '0' }
  }

  const node = nodes[nodeId]
  if (!node) {
    return { kind: 'NumberLiteral', value: '0' }
  }

  const nextVisited = new Set(visitedExpr)
  nextVisited.add(nodeId)

  switch (node.name) {
    case 'number': {
      const raw = readString(node.data, 'value', '0').trim()
      if (!/^-?\d+(\.\d+)?$/.test(raw)) {
        diagnostics.push({
          level: 'warning',
          nodeId,
          message: `Number node #${nodeId} has invalid value "${raw}"; defaulting to 0.`,
        })
        return { kind: 'NumberLiteral', value: '0' }
      }
      return { kind: 'NumberLiteral', value: raw }
    }

    case 'variable': {
      const name = sanitizeIdentifier(
        readString(node.data, 'name', 'x'),
        'x',
        nodeId,
        diagnostics,
      )
      return { kind: 'VariableRef', name }
    }

    case 'math': {
      const rawOp = readString(node.data, 'operator', '+') as MathOperator
      const operator: MathOperator = VALID_MATH_OPS.has(rawOp) ? rawOp : '+'
      const left = resolveInputExpression(
        node,
        'input_1',
        readString(node.data, 'left', '0'),
        nodes,
        nextVisited,
        diagnostics,
      )
      const right = resolveInputExpression(
        node,
        'input_2',
        readString(node.data, 'right', '0'),
        nodes,
        nextVisited,
        diagnostics,
      )
      return { kind: 'BinaryOp', operator, left, right }
    }

    case 'compare': {
      const rawOp = readString(node.data, 'operator', '==') as CompareOperator
      const operator: CompareOperator = VALID_COMPARE_OPS.has(rawOp) ? rawOp : '=='
      const left = resolveInputExpression(
        node,
        'input_1',
        readString(node.data, 'left', '0'),
        nodes,
        nextVisited,
        diagnostics,
      )
      const right = resolveInputExpression(
        node,
        'input_2',
        readString(node.data, 'right', '0'),
        nodes,
        nextVisited,
        diagnostics,
      )
      return { kind: 'BinaryOp', operator, left, right }
    }

    default: {
      diagnostics.push({
        level: 'warning',
        nodeId,
        message: `Node #${nodeId} (${node.name}) cannot be used as a value expression.`,
      })
      return { kind: 'NumberLiteral', value: '0' }
    }
  }
}

function getOutputTargets(node: DrawflowNode, portName: string): string[] {
  const port = node.outputs?.[portName]
  if (!port || !Array.isArray(port.connections)) {
    return []
  }
  return port.connections.map((c) => String(c.node))
}

function getInputSources(node: DrawflowNode, portName: string): string[] {
  const port = node.inputs?.[portName]
  if (!port || !Array.isArray(port.connections)) {
    return []
  }
  return port.connections.map((c) => String(c.node))
}

function readString(data: Record<string, unknown> | undefined, key: string, fallback: string): string {
  const val = data?.[key]
  if (typeof val === 'string') return val
  if (typeof val === 'number' || typeof val === 'boolean') return String(val)
  return fallback
}

function readBoolean(data: Record<string, unknown> | undefined, key: string, fallback: boolean): boolean {
  const val = data?.[key]
  if (typeof val === 'boolean') return val
  if (val === 'true') return true
  if (val === 'false') return false
  return fallback
}

function sanitizeIdentifier(
  raw: string,
  fallback: string,
  nodeId: string,
  diagnostics: Diagnostic[],
): string {
  const trimmed = raw.trim()
  if (VALID_IDENTIFIER.test(trimmed)) {
    return trimmed
  }
  diagnostics.push({
    level: 'warning',
    nodeId,
    message: `Invalid identifier "${raw}" on node #${nodeId}; using "${fallback}".`,
  })
  return fallback
}
