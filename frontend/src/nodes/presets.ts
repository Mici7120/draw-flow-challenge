import type { DrawflowExport, DrawflowNode } from '../ast/types'
import { getNodeDefinition } from './nodeDefinitions'
import type { DrawflowNodeType } from '../ast/types'

export interface ProgramPreset {
  id: string
  name: string
  description: string
  graph: DrawflowExport
}

function node(
  id: number,
  type: DrawflowNodeType,
  pos_x: number,
  pos_y: number,
  data: Record<string, unknown>,
  inputs: Record<string, Array<{ node: string; input: string }>>,
  outputs: Record<string, Array<{ node: string; output: string }>>,
): DrawflowNode {
  const def = getNodeDefinition(type)
  const fullInputs: DrawflowNode['inputs'] = {}
  for (let i = 1; i <= def.inputs; i++) {
    const key = `input_${i}`
    fullInputs[key] = { connections: inputs[key] ?? [] }
  }
  const fullOutputs: DrawflowNode['outputs'] = {}
  for (let i = 1; i <= def.outputs; i++) {
    const key = `output_${i}`
    fullOutputs[key] = { connections: outputs[key] ?? [] }
  }

  return {
    id,
    name: type,
    data: { ...def.defaultData, ...data },
    class: `node-${type}`,
    html: def.html,
    typenode: false,
    inputs: fullInputs,
    outputs: fullOutputs,
    pos_x,
    pos_y,
  }
}

function buildExport(nodes: DrawflowNode[]): DrawflowExport {
  const data: Record<string, DrawflowNode> = {}
  for (const n of nodes) {
    data[String(n.id)] = n
  }
  return { drawflow: { Home: { data } } }
}

export const PROGRAM_PRESETS: ProgramPreset[] = [
  {
    id: 'add-assign',
    name: 'Add & Assign Demo',
    description: 'Connects a Number and Math Add node into an Assign node and prints the result.',
    graph: buildExport([
      node(1, 'start', 40, 80, {}, {}, { output_1: [{ node: '4', output: 'input_1' }] }),
      node(2, 'number', 40, 250, { value: '12' }, {}, { output_1: [{ node: '3', output: 'input_1' }] }),
      node(
        3,
        'math',
        260,
        240,
        { operator: '+', left: '0', right: '30' },
        { input_1: [{ node: '2', input: 'output_1' }] },
        { output_1: [{ node: '4', output: 'input_2' }] },
      ),
      node(
        4,
        'assign',
        530,
        90,
        { variable: 'answer', value: '0' },
        {
          input_1: [{ node: '1', input: 'output_1' }],
          input_2: [{ node: '3', input: 'output_1' }],
        },
        { output_1: [{ node: '5', output: 'input_1' }] },
      ),
      node(
        5,
        'print',
        810,
        90,
        { value: 'answer' },
        { input_1: [{ node: '4', input: 'output_1' }] },
        {},
      ),
    ]),
  },
  {
    id: 'if-else-check',
    name: 'If-Else Condition',
    description: 'Assigns a score, compares it against 60, and branches with If/Else.',
    graph: buildExport([
      node(1, 'start', 30, 60, {}, {}, { output_1: [{ node: '2', output: 'input_1' }] }),
      node(
        2,
        'assign',
        210,
        60,
        { variable: 'score', value: '85' },
        { input_1: [{ node: '1', input: 'output_1' }] },
        { output_1: [{ node: '4', output: 'input_1' }] },
      ),
      node(
        3,
        'compare',
        210,
        240,
        { operator: '>=', left: 'score', right: '60' },
        {},
        { output_1: [{ node: '4', output: 'input_2' }] },
      ),
      node(
        4,
        'if_else',
        500,
        100,
        { condition: 'score >= 60' },
        {
          input_1: [{ node: '2', input: 'output_1' }],
          input_2: [{ node: '3', input: 'output_1' }],
        },
        {
          output_1: [{ node: '5', output: 'input_1' }],
          output_2: [{ node: '6', output: 'input_1' }],
          output_3: [{ node: '7', output: 'input_1' }],
        },
      ),
      node(
        5,
        'assign',
        810,
        30,
        { variable: 'passed', value: '1' },
        { input_1: [{ node: '4', input: 'output_1' }] },
        {},
      ),
      node(
        6,
        'assign',
        810,
        180,
        { variable: 'passed', value: '0' },
        { input_1: [{ node: '4', input: 'output_2' }] },
        {},
      ),
      node(
        7,
        'print',
        810,
        330,
        { value: 'passed' },
        { input_1: [{ node: '4', input: 'output_3' }] },
        {},
      ),
    ]),
  },
  {
    id: 'for-loop-sum',
    name: 'For Loop Sum (1..5)',
    description: 'Uses a Code Block and a For (from-to) loop to sum integers from 1 to 5.',
    graph: buildExport([
      node(1, 'start', 30, 80, {}, {}, { output_1: [{ node: '2', output: 'input_1' }] }),
      node(
        2,
        'block',
        210,
        80,
        { label: 'Accumulate Sum' },
        { input_1: [{ node: '1', input: 'output_1' }] },
        {
          output_1: [{ node: '3', output: 'input_1' }],
          output_2: [{ node: '6', output: 'input_1' }],
        },
      ),
      node(
        3,
        'assign',
        470,
        40,
        { variable: 'total', value: '0' },
        { input_1: [{ node: '2', input: 'output_1' }] },
        { output_1: [{ node: '4', output: 'input_1' }] },
      ),
      node(
        4,
        'for_loop',
        740,
        40,
        { iterator: 'i', from: '1', to: '5', inclusive: 'true' },
        { input_1: [{ node: '3', input: 'output_1' }] },
        { output_1: [{ node: '5', output: 'input_1' }] },
      ),
      node(
        5,
        'assign',
        1040,
        40,
        { variable: 'total', value: 'total + i' },
        { input_1: [{ node: '4', input: 'output_1' }] },
        {},
      ),
      node(
        6,
        'print',
        470,
        240,
        { value: 'total' },
        { input_1: [{ node: '2', input: 'output_2' }] },
        {},
      ),
    ]),
  },
]
