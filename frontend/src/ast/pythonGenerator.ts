import type { ExpressionNode, ProgramAST, StatementNode } from './types'

const INDENT = '    '

export function generatePython(ast: ProgramAST): string {
  if (ast.body.length === 0) {
    return '# Connect nodes starting from Start to generate Python code\n'
  }
  const lines = emitStatements(ast.body, 0)
  return lines.join('\n') + '\n'
}

function emitStatements(statements: StatementNode[], depth: number): string[] {
  const lines: string[] = []
  const pad = INDENT.repeat(depth)

  for (const stmt of statements) {
    switch (stmt.kind) {
      case 'Block': {
        if (stmt.label) {
          lines.push(`${pad}# Block: ${stmt.label}`)
        }
        if (stmt.body.length > 0) {
          lines.push(...emitStatements(stmt.body, depth))
        } else {
          lines.push(`${pad}pass`)
        }
        break
      }

      case 'Assign': {
        lines.push(`${pad}${stmt.variable} = ${emitExpression(stmt.value, true)}`)
        break
      }

      case 'IfElse': {
        lines.push(`${pad}if ${emitExpression(stmt.condition, true)}:`)
        if (stmt.thenBranch.length > 0) {
          lines.push(...emitStatements(stmt.thenBranch, depth + 1))
        } else {
          lines.push(`${pad}${INDENT}pass`)
        }
        if (stmt.elseBranch.length > 0) {
          lines.push(`${pad}else:`)
          lines.push(...emitStatements(stmt.elseBranch, depth + 1))
        }
        break
      }

      case 'ForLoop': {
        const fromCode = emitExpression(stmt.from, true)
        const toCode = formatRangeEnd(stmt.to, stmt.inclusive)
        lines.push(`${pad}for ${stmt.iterator} in range(${fromCode}, ${toCode}):`)
        if (stmt.body.length > 0) {
          lines.push(...emitStatements(stmt.body, depth + 1))
        } else {
          lines.push(`${pad}${INDENT}pass`)
        }
        break
      }

      case 'Print': {
        lines.push(`${pad}print(${emitExpression(stmt.value, true)})`)
        break
      }
    }
  }

  return lines
}

function formatRangeEnd(expr: ExpressionNode, inclusive: boolean): string {
  if (!inclusive) {
    return emitExpression(expr, true)
  }
  if (expr.kind === 'NumberLiteral' && /^-?\d+$/.test(expr.value)) {
    return String(Number.parseInt(expr.value, 10) + 1)
  }
  return `${emitExpression(expr, false)} + 1`
}

function emitExpression(expr: ExpressionNode, isTopLevel: boolean): string {
  switch (expr.kind) {
    case 'NumberLiteral':
      return expr.value
    case 'VariableRef':
      return expr.name
    case 'RawExpression':
      return expr.code
    case 'BinaryOp': {
      const left = emitExpression(expr.left, false)
      const right = emitExpression(expr.right, false)
      const combined = `${left} ${expr.operator} ${right}`
      return isTopLevel ? combined : `(${combined})`
    }
  }
}
