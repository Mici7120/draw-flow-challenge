export type MathOperator = '+' | '-' | '*' | '/' | '%'
export type CompareOperator = '==' | '!=' | '<' | '<=' | '>' | '>='

export type DrawflowNodeType =
  | 'start'
  | 'block'
  | 'number'
  | 'variable'
  | 'math'
  | 'compare'
  | 'assign'
  | 'if_else'
  | 'for_loop'
  | 'print'

export interface DrawflowConnection {
  node: string
  input?: string
  output?: string
}

export interface DrawflowPort {
  connections: DrawflowConnection[]
}

export interface DrawflowNode {
  id: number
  name: string
  data: Record<string, unknown>
  class: string
  html: string
  typenode: boolean
  inputs: Record<string, DrawflowPort>
  outputs: Record<string, DrawflowPort>
  pos_x: number
  pos_y: number
}

export interface DrawflowExport {
  drawflow: {
    Home: {
      data: Record<string, DrawflowNode>
    }
  }
}

export type ExpressionNode =
  | { kind: 'NumberLiteral'; value: string }
  | { kind: 'VariableRef'; name: string }
  | { kind: 'RawExpression'; code: string }
  | {
      kind: 'BinaryOp'
      operator: MathOperator | CompareOperator
      left: ExpressionNode
      right: ExpressionNode
    }

export type StatementNode =
  | {
      kind: 'Block'
      label: string
      body: StatementNode[]
    }
  | {
      kind: 'Assign'
      variable: string
      value: ExpressionNode
    }
  | {
      kind: 'IfElse'
      condition: ExpressionNode
      thenBranch: StatementNode[]
      elseBranch: StatementNode[]
    }
  | {
      kind: 'ForLoop'
      iterator: string
      from: ExpressionNode
      to: ExpressionNode
      inclusive: boolean
      body: StatementNode[]
    }
  | {
      kind: 'Print'
      value: ExpressionNode
    }

export interface ProgramAST {
  kind: 'Program'
  body: StatementNode[]
}

export interface Diagnostic {
  level: 'error' | 'warning'
  nodeId?: string
  message: string
}

export interface CompileResult {
  ast: ProgramAST
  code: string
  diagnostics: Diagnostic[]
}
