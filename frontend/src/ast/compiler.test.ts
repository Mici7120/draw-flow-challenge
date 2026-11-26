import { describe, expect, it } from 'vitest'
import { compileDrawflowToPython, type DrawflowExport, type DrawflowNode } from './index'

function makeNode(
  id: number,
  name: string,
  data: Record<string, unknown>,
  inputs: Record<string, Array<{ node: string; input?: string }>> = {},
  outputs: Record<string, Array<{ node: string; output?: string }>> = {},
): DrawflowNode {
  const mappedInputs: DrawflowNode['inputs'] = {}
  for (const [k, conns] of Object.entries(inputs)) {
    mappedInputs[k] = { connections: conns }
  }
  const mappedOutputs: DrawflowNode['outputs'] = {}
  for (const [k, conns] of Object.entries(outputs)) {
    mappedOutputs[k] = { connections: conns }
  }
  return {
    id,
    name,
    data,
    class: name,
    html: '',
    typenode: false,
    inputs: mappedInputs,
    outputs: mappedOutputs,
    pos_x: id * 100,
    pos_y: 100,
  }
}

function wrap(nodes: DrawflowNode[]): DrawflowExport {
  const data: Record<string, DrawflowNode> = {}
  for (const n of nodes) {
    data[String(n.id)] = n
  }
  return { drawflow: { Home: { data } } }
}

describe('Drawflow -> AST -> Python compiler', () => {
  it('compiles Add connected to Assign and Print (README example)', () => {
    const graph = wrap([
      makeNode(1, 'start', {}, {}, { output_1: [{ node: '4' }] }),
      makeNode(2, 'number', { value: '10' }, {}, { output_1: [{ node: '3' }] }),
      makeNode(
        3,
        'math',
        { operator: '+', left: '0', right: '5' },
        { input_1: [{ node: '2' }] },
        { output_1: [{ node: '4' }] },
      ),
      makeNode(
        4,
        'assign',
        { variable: 'result', value: '0' },
        { input_1: [{ node: '1' }], input_2: [{ node: '3' }] },
        { output_1: [{ node: '5' }] },
      ),
      makeNode(
        5,
        'print',
        { value: 'result' },
        { input_1: [{ node: '4' }] },
        {},
      ),
    ])

    const res = compileDrawflowToPython(graph)
    expect(res.diagnostics).toHaveLength(0)
    expect(res.code).toBe('result = 10 + 5\nprint(result)\n')
  })

  it('compiles If-Else control structure with comparison node', () => {
    const graph = wrap([
      makeNode(1, 'start', {}, {}, { output_1: [{ node: '2' }] }),
      makeNode(
        2,
        'assign',
        { variable: 'score', value: '85' },
        { input_1: [{ node: '1' }] },
        { output_1: [{ node: '4' }] },
      ),
      makeNode(
        3,
        'compare',
        { operator: '>=', left: 'score', right: '60' },
        {},
        { output_1: [{ node: '4' }] },
      ),
      makeNode(
        4,
        'if_else',
        { condition: 'True' },
        { input_1: [{ node: '2' }], input_2: [{ node: '3' }] },
        {
          output_1: [{ node: '5' }],
          output_2: [{ node: '6' }],
          output_3: [{ node: '7' }],
        },
      ),
      makeNode(5, 'assign', { variable: 'passed', value: '1' }, { input_1: [{ node: '4' }] }),
      makeNode(6, 'assign', { variable: 'passed', value: '0' }, { input_1: [{ node: '4' }] }),
      makeNode(7, 'print', { value: 'passed' }, { input_1: [{ node: '4' }] }),
    ])

    const res = compileDrawflowToPython(graph)
    expect(res.diagnostics).toHaveLength(0)
    expect(res.code).toBe(
      [
        'score = 85',
        'if score >= 60:',
        '    passed = 1',
        'else:',
        '    passed = 0',
        'print(passed)',
        '',
      ].join('\n'),
    )
  })

  it('compiles For (from-to) loop inside a Code Block', () => {
    const graph = wrap([
      makeNode(1, 'start', {}, {}, { output_1: [{ node: '2' }] }),
      makeNode(
        2,
        'block',
        { label: 'Sum 1..5' },
        { input_1: [{ node: '1' }] },
        { output_1: [{ node: '3' }], output_2: [{ node: '6' }] },
      ),
      makeNode(
        3,
        'assign',
        { variable: 'total', value: '0' },
        { input_1: [{ node: '2' }] },
        { output_1: [{ node: '4' }] },
      ),
      makeNode(
        4,
        'for_loop',
        { iterator: 'i', from: '1', to: '5', inclusive: true },
        { input_1: [{ node: '3' }] },
        { output_1: [{ node: '5' }] },
      ),
      makeNode(
        5,
        'assign',
        { variable: 'total', value: 'total + i' },
        { input_1: [{ node: '4' }] },
      ),
      makeNode(6, 'print', { value: 'total' }, { input_1: [{ node: '2' }] }),
    ])

    const res = compileDrawflowToPython(graph)
    expect(res.diagnostics).toHaveLength(0)
    expect(res.code).toBe(
      [
        '# Block: Sum 1..5',
        'total = 0',
        'for i in range(1, 6):',
        '    total = total + i',
        'print(total)',
        '',
      ].join('\n'),
    )
  })

  it('detects control-flow cycles gracefully without infinite recursion', () => {
    const graph = wrap([
      makeNode(1, 'start', {}, {}, { output_1: [{ node: '2' }] }),
      makeNode(
        2,
        'assign',
        { variable: 'a', value: '1' },
        { input_1: [{ node: '1' }] },
        { output_1: [{ node: '2' }] },
      ),
    ])

    const res = compileDrawflowToPython(graph)
    expect(res.diagnostics.some((d) => d.level === 'error' && d.message.includes('Cycle'))).toBe(true)
    expect(res.code).toBe('a = 1\n')
  })
})
