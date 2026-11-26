import { parseDrawflowToAST } from './parser'
import { generatePython } from './pythonGenerator'
import type { CompileResult, DrawflowExport } from './types'

export function compileDrawflowToPython(exported: DrawflowExport | null | undefined): CompileResult {
  const { ast, diagnostics } = parseDrawflowToAST(exported)
  const code = generatePython(ast)
  return { ast, code, diagnostics }
}

export * from './types'
export { parseDrawflowToAST, generatePython }
