import type { DrawflowNodeType } from '../ast/types'

export interface NodeDefinition {
  type: DrawflowNodeType
  label: string
  category: 'Control Flow' | 'Statements' | 'Expressions'
  shortcut: string
  description: string
  inputs: number
  outputs: number
  defaultData: Record<string, unknown>
  html: string
}

export const NODE_DEFINITIONS: NodeDefinition[] = [
  {
    type: 'start',
    label: 'Start',
    category: 'Control Flow',
    shortcut: '1',
    description: 'Program entry point',
    inputs: 0,
    outputs: 1,
    defaultData: {},
    html: `
      <div class="df-node df-node--start">
        <div class="df-node__header">Start</div>
        <div class="df-node__body">
          <div class="df-port-row df-port-row--right"><span>Flow</span></div>
        </div>
      </div>
    `,
  },
  {
    type: 'block',
    label: 'Code Block',
    category: 'Control Flow',
    shortcut: '2',
    description: 'Group sequential statements',
    inputs: 1,
    outputs: 2,
    defaultData: { label: 'Main Block' },
    html: `
      <div class="df-node df-node--block">
        <div class="df-node__header">Code Block</div>
        <div class="df-node__body">
          <label class="df-field">
            <span>Label</span>
            <input type="text" df-label placeholder="Block name" />
          </label>
          <div class="df-port-hints">
            <span>In: Flow</span>
            <span>Out 1: Body · Out 2: Next</span>
          </div>
        </div>
      </div>
    `,
  },
  {
    type: 'if_else',
    label: 'If / Else',
    category: 'Control Flow',
    shortcut: '3',
    description: 'Conditional branch (then / else)',
    inputs: 2,
    outputs: 3,
    defaultData: { condition: 'x > 0' },
    html: `
      <div class="df-node df-node--control">
        <div class="df-node__header">If / Else</div>
        <div class="df-node__body">
          <label class="df-field">
            <span>Condition (or wire In 2)</span>
            <input type="text" df-condition placeholder="x > 0" />
          </label>
          <div class="df-port-hints">
            <span>In 1: Flow · In 2: Cond</span>
            <span>Out 1: Then · 2: Else · 3: Next</span>
          </div>
        </div>
      </div>
    `,
  },
  {
    type: 'for_loop',
    label: 'For (From-To)',
    category: 'Control Flow',
    shortcut: '4',
    description: 'Loop from start to end value',
    inputs: 3,
    outputs: 2,
    defaultData: { iterator: 'i', from: '1', to: '5', inclusive: 'true' },
    html: `
      <div class="df-node df-node--control">
        <div class="df-node__header">For (From-To)</div>
        <div class="df-node__body">
          <div class="df-grid-3">
            <label class="df-field">
              <span>Var</span>
              <input type="text" df-iterator placeholder="i" />
            </label>
            <label class="df-field">
              <span>From</span>
              <input type="text" df-from placeholder="1" />
            </label>
            <label class="df-field">
              <span>To</span>
              <input type="text" df-to placeholder="5" />
            </label>
          </div>
          <label class="df-field">
            <span>Range End</span>
            <select df-inclusive>
              <option value="true">Inclusive (to + 1)</option>
              <option value="false">Exclusive (to)</option>
            </select>
          </label>
          <div class="df-port-hints">
            <span>In 1: Flow · 2: From · 3: To</span>
            <span>Out 1: Body · Out 2: Next</span>
          </div>
        </div>
      </div>
    `,
  },
  {
    type: 'assign',
    label: 'Assign',
    category: 'Statements',
    shortcut: '5',
    description: 'Assign value to a variable',
    inputs: 2,
    outputs: 1,
    defaultData: { variable: 'x', value: '0' },
    html: `
      <div class="df-node df-node--statement">
        <div class="df-node__header">Assign</div>
        <div class="df-node__body">
          <div class="df-grid-2">
            <label class="df-field">
              <span>Variable</span>
              <input type="text" df-variable placeholder="x" />
            </label>
            <label class="df-field">
              <span>Value (or In 2)</span>
              <input type="text" df-value placeholder="0" />
            </label>
          </div>
          <div class="df-port-hints">
            <span>In 1: Flow · In 2: Value</span>
            <span>Out 1: Next</span>
          </div>
        </div>
      </div>
    `,
  },
  {
    type: 'print',
    label: 'Print',
    category: 'Statements',
    shortcut: '6',
    description: 'Print value to output console',
    inputs: 2,
    outputs: 1,
    defaultData: { value: 'x' },
    html: `
      <div class="df-node df-node--statement">
        <div class="df-node__header">Print</div>
        <div class="df-node__body">
          <label class="df-field">
            <span>Expression (or In 2)</span>
            <input type="text" df-value placeholder="x" />
          </label>
          <div class="df-port-hints">
            <span>In 1: Flow · In 2: Value</span>
            <span>Out 1: Next</span>
          </div>
        </div>
      </div>
    `,
  },
  {
    type: 'math',
    label: 'Math Op',
    category: 'Expressions',
    shortcut: '7',
    description: 'Basic math (+, -, *, /, %)',
    inputs: 2,
    outputs: 1,
    defaultData: { operator: '+', left: '1', right: '2' },
    html: `
      <div class="df-node df-node--expr">
        <div class="df-node__header">Math Operation</div>
        <div class="df-node__body">
          <div class="df-grid-3">
            <label class="df-field">
              <span>A (In 1)</span>
              <input type="text" df-left placeholder="1" />
            </label>
            <label class="df-field">
              <span>Op</span>
              <select df-operator>
                <option value="+">+</option>
                <option value="-">-</option>
                <option value="*">*</option>
                <option value="/">/</option>
                <option value="%">%</option>
              </select>
            </label>
            <label class="df-field">
              <span>B (In 2)</span>
              <input type="text" df-right placeholder="2" />
            </label>
          </div>
          <div class="df-port-hints">
            <span>Out 1: Result</span>
          </div>
        </div>
      </div>
    `,
  },
  {
    type: 'compare',
    label: 'Compare',
    category: 'Expressions',
    shortcut: '8',
    description: 'Comparison (==, !=, <, <=, >, >=)',
    inputs: 2,
    outputs: 1,
    defaultData: { operator: '>=', left: 'x', right: '10' },
    html: `
      <div class="df-node df-node--expr">
        <div class="df-node__header">Compare</div>
        <div class="df-node__body">
          <div class="df-grid-3">
            <label class="df-field">
              <span>A (In 1)</span>
              <input type="text" df-left placeholder="x" />
            </label>
            <label class="df-field">
              <span>Op</span>
              <select df-operator>
                <option value="==">==</option>
                <option value="!=">!=</option>
                <option value="<">&lt;</option>
                <option value="<=">&lt;=</option>
                <option value=">">&gt;</option>
                <option value=">=">&gt;=</option>
              </select>
            </label>
            <label class="df-field">
              <span>B (In 2)</span>
              <input type="text" df-right placeholder="10" />
            </label>
          </div>
          <div class="df-port-hints">
            <span>Out 1: Bool</span>
          </div>
        </div>
      </div>
    `,
  },
  {
    type: 'number',
    label: 'Number',
    category: 'Expressions',
    shortcut: '9',
    description: 'Numeric literal value',
    inputs: 0,
    outputs: 1,
    defaultData: { value: '10' },
    html: `
      <div class="df-node df-node--expr">
        <div class="df-node__header">Number</div>
        <div class="df-node__body">
          <label class="df-field">
            <span>Value</span>
            <input type="text" df-value placeholder="10" />
          </label>
        </div>
      </div>
    `,
  },
  {
    type: 'variable',
    label: 'Variable',
    category: 'Expressions',
    shortcut: '0',
    description: 'Read a variable value',
    inputs: 0,
    outputs: 1,
    defaultData: { name: 'x' },
    html: `
      <div class="df-node df-node--expr">
        <div class="df-node__header">Variable</div>
        <div class="df-node__body">
          <label class="df-field">
            <span>Name</span>
            <input type="text" df-name placeholder="x" />
          </label>
        </div>
      </div>
    `,
  },
]

export function getNodeDefinition(type: DrawflowNodeType): NodeDefinition {
  const found = NODE_DEFINITIONS.find((d) => d.type === type)
  if (!found) {
    throw new Error(`Unknown node type: ${type}`)
  }
  return found
}
