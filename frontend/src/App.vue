<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Drawflow from 'drawflow'
import 'drawflow/dist/drawflow.min.css'

import {
  compileDrawflowToPython,
  type Diagnostic,
  type DrawflowExport,
  type DrawflowNodeType,
} from './ast'
import { getNodeDefinition, NODE_DEFINITIONS } from './nodes/nodeDefinitions'
import { PROGRAM_PRESETS } from './nodes/presets'
import {
  checkHealth,
  deleteProgram,
  getProgram,
  listPrograms,
  runCode,
  saveProgram,
  updateProgram,
  type ProgramSummary,
  type RunResult,
} from './services/api'

import NodePalette from './components/NodePalette.vue'
import CodePanel from './components/CodePanel.vue'
import ProgramModal from './components/ProgramModal.vue'
import QuickAddModal from './components/QuickAddModal.vue'

const canvasEl = ref<HTMLElement | null>(null)
let editor: Drawflow | null = null
let placementStep = 0

const programId = ref<string | null>(null)
const programName = ref('Add & Assign Demo')
const selectedPresetId = ref('add-assign')

const generatedCode = ref('')
const diagnostics = ref<Diagnostic[]>([])

const runResult = ref<RunResult | null>(null)
const runError = ref<string | null>(null)
const isRunning = ref(false)
const isSaving = ref(false)
const backendOnline = ref<boolean | null>(null)

const showProgramsModal = ref(false)
const savedPrograms = ref<ProgramSummary[]>([])
const loadingPrograms = ref(false)
const programsError = ref<string | null>(null)

const showQuickAdd = ref(false)
const statusToast = ref<{ kind: 'info' | 'error'; text: string } | null>(null)
let toastTimer: number | undefined

function notify(text: string, kind: 'info' | 'error' = 'info') {
  statusToast.value = { kind, text }
  if (toastTimer) window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => {
    statusToast.value = null
  }, 3000)
}

function recompile() {
  if (!editor) return
  const exported = editor.export() as unknown as DrawflowExport
  const result = compileDrawflowToPython(exported)
  generatedCode.value = result.code
  diagnostics.value = result.diagnostics
}

function addNodeAt(type: DrawflowNodeType, x?: number, y?: number) {
  if (!editor) return
  const def = getNodeDefinition(type)
  const posX = x ?? 120 + (placementStep % 5) * 45
  const posY = y ?? 90 + (placementStep % 5) * 40
  placementStep += 1

  editor.addNode(
    def.type,
    def.inputs,
    def.outputs,
    posX,
    posY,
    `node-${def.type}`,
    { ...def.defaultData },
    def.html,
    false,
  )
  recompile()
}

function onCanvasDragOver(event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'copy'
  }
}

function onCanvasDrop(event: DragEvent) {
  event.preventDefault()
  const rawType = event.dataTransfer?.getData('application/drawflow-node') as DrawflowNodeType
  if (!rawType || !canvasEl.value || !editor) return

  const rect = canvasEl.value.getBoundingClientRect()
  const zoom = editor.zoom || 1
  const x = (event.clientX - rect.left) / zoom
  const y = (event.clientY - rect.top) / zoom
  addNodeAt(rawType, x, y)
}

function loadPreset(presetId: string) {
  const preset = PROGRAM_PRESETS.find((p) => p.id === presetId)
  if (!preset || !editor) return
  editor.clear()
  editor.import(structuredClone(preset.graph) as unknown as Record<string, unknown>)
  programId.value = null
  programName.value = preset.name
  runResult.value = null
  runError.value = null
  recompile()
  notify(`Loaded example: ${preset.name}`)
}

function createNewProgram() {
  if (!editor) return
  editor.clear()
  programId.value = null
  programName.value = 'Untitled Program'
  runResult.value = null
  runError.value = null
  placementStep = 0
  addNodeAt('start', 80, 120)
  notify('Started new program')
}

async function handleSave() {
  if (!editor || isSaving.value) return
  const name = programName.value.trim()
  if (!name) {
    notify('Please enter a program name before saving.', 'error')
    return
  }

  isSaving.value = true
  try {
    recompile()
    const payload = {
      name,
      graph: editor.export() as unknown as DrawflowExport,
      code: generatedCode.value,
    }
    const saved = programId.value
      ? await updateProgram(programId.value, payload)
      : await saveProgram(payload)
    programId.value = saved.id
    programName.value = saved.name
    backendOnline.value = true
    notify(`Saved "${saved.name}"`)
  } catch (err) {
    notify(err instanceof Error ? err.message : 'Failed to save program', 'error')
  } finally {
    isSaving.value = false
  }
}

async function openLoadModal() {
  showProgramsModal.value = true
  await refreshPrograms()
}

async function refreshPrograms() {
  loadingPrograms.value = true
  programsError.value = null
  try {
    savedPrograms.value = await listPrograms()
    backendOnline.value = true
  } catch (err) {
    programsError.value = err instanceof Error ? err.message : 'Failed to load programs'
  } finally {
    loadingPrograms.value = false
  }
}

async function handleLoadProgram(id: string) {
  if (!editor) return
  try {
    const prog = await getProgram(id)
    editor.clear()
    editor.import(prog.graph as unknown as Record<string, unknown>)
    programId.value = prog.id
    programName.value = prog.name
    runResult.value = null
    runError.value = null
    showProgramsModal.value = false
    recompile()
    notify(`Loaded "${prog.name}"`)
  } catch (err) {
    notify(err instanceof Error ? err.message : 'Failed to load program', 'error')
  }
}

async function handleDeleteProgram(id: string) {
  try {
    await deleteProgram(id)
    if (programId.value === id) {
      programId.value = null
    }
    savedPrograms.value = savedPrograms.value.filter((p) => p.id !== id)
    notify('Program deleted')
  } catch (err) {
    notify(err instanceof Error ? err.message : 'Failed to delete program', 'error')
  }
}

async function handleRun() {
  if (isRunning.value) return
  recompile()
  if (!generatedCode.value.trim()) {
    runError.value = 'No Python code to run.'
    return
  }

  isRunning.value = true
  runError.value = null
  try {
    runResult.value = await runCode(generatedCode.value)
    backendOnline.value = true
  } catch (err) {
    runError.value = err instanceof Error ? err.message : 'Execution request failed'
  } finally {
    isRunning.value = false
  }
}

function zoomIn() {
  editor?.zoom_in()
}

function zoomOut() {
  editor?.zoom_out()
}

function zoomReset() {
  editor?.zoom_reset()
}

function isEditableElement(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  const tag = target.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable
}

function handleGlobalKeydown(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    showQuickAdd.value = !showQuickAdd.value
    return
  }

  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
    event.preventDefault()
    void handleSave()
    return
  }

  if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
    event.preventDefault()
    void handleRun()
    return
  }

  if (event.key === 'Escape') {
    showQuickAdd.value = false
    showProgramsModal.value = false
    return
  }

  if (showQuickAdd.value || showProgramsModal.value || isEditableElement(event.target)) {
    return
  }

  if (event.key === '/') {
    event.preventDefault()
    showQuickAdd.value = true
    return
  }

  const shortcutDef = NODE_DEFINITIONS.find((d) => d.shortcut === event.key)
  if (shortcutDef) {
    event.preventDefault()
    addNodeAt(shortcutDef.type)
    return
  }

  if ((event.key === 'Delete' || event.key === 'Backspace') && editor?.node_selected) {
    event.preventDefault()
    const selectedId = editor.node_selected.id
    if (selectedId) {
      editor.removeNodeId(selectedId)
      recompile()
    }
  }
}

onMounted(async () => {
  if (canvasEl.value) {
    editor = new Drawflow(canvasEl.value)
    editor.reroute = true
    editor.curvature = 0.45
    editor.start()

    const syncEvents = [
      'nodeCreated',
      'nodeRemoved',
      'nodeDataChanged',
      'connectionCreated',
      'connectionRemoved',
      'import',
    ]
    for (const evt of syncEvents) {
      editor.on(evt, () => recompile())
    }

    canvasEl.value.addEventListener('input', recompile)
    canvasEl.value.addEventListener('change', recompile)

    loadPreset(selectedPresetId.value)
  }

  window.addEventListener('keydown', handleGlobalKeydown)
  backendOnline.value = await checkHealth()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
  canvasEl.value?.removeEventListener('input', recompile)
  canvasEl.value?.removeEventListener('change', recompile)
  if (toastTimer) window.clearTimeout(toastTimer)
})
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="topbar__left">
        <div class="brand">
          <span class="brand__dot" />
          <span class="brand__name">Draw-Flow</span>
        </div>

        <input
          v-model="programName"
          type="text"
          class="program-title-input"
          aria-label="Program name"
          placeholder="Program name..."
        />

        <span
          class="health-pill"
          :class="
            backendOnline === true
              ? 'health-pill--ok'
              : backendOnline === false
                ? 'health-pill--offline'
                : ''
          "
          :title="
            backendOnline
              ? 'Backend API connected'
              : 'Backend API unreachable — start Go server on :8081'
          "
        >
          {{ backendOnline === true ? 'API Online' : backendOnline === false ? 'API Offline' : 'Checking API...' }}
        </span>
      </div>

      <div class="topbar__right">
        <select
          v-model="selectedPresetId"
          class="preset-select"
          aria-label="Load example program"
          @change="loadPreset(selectedPresetId)"
        >
          <option v-for="preset in PROGRAM_PRESETS" :key="preset.id" :value="preset.id">
            Example: {{ preset.name }}
          </option>
        </select>

        <div class="zoom-group" role="group" aria-label="Zoom controls">
          <button type="button" class="btn btn--ghost btn--sm" title="Zoom out" @click="zoomOut">
            -
          </button>
          <button type="button" class="btn btn--ghost btn--sm" title="Reset zoom" @click="zoomReset">
            100%
          </button>
          <button type="button" class="btn btn--ghost btn--sm" title="Zoom in" @click="zoomIn">
            +
          </button>
        </div>

        <button type="button" class="btn btn--ghost btn--sm" @click="createNewProgram">New</button>
        <button type="button" class="btn btn--ghost btn--sm" @click="openLoadModal">Load</button>
        <button
          type="button"
          class="btn btn--secondary btn--sm"
          :disabled="isSaving"
          @click="handleSave"
        >
          {{ isSaving ? 'Saving...' : programId ? 'Update' : 'Save' }}
        </button>
        <button
          type="button"
          class="btn btn--primary btn--sm"
          :disabled="isRunning"
          @click="handleRun"
        >
          {{ isRunning ? 'Running...' : 'Run' }}
        </button>
      </div>
    </header>

    <main class="workspace">
      <NodePalette @add-node="addNodeAt" />

      <section class="canvas-stage" aria-label="Visual node canvas">
        <div
          ref="canvasEl"
          class="drawflow-canvas"
          @dragover="onCanvasDragOver"
          @drop="onCanvasDrop"
        />
        <div v-if="statusToast" class="toast" :class="`toast--${statusToast.kind}`">
          {{ statusToast.text }}
        </div>
      </section>

      <CodePanel
        :code="generatedCode"
        :diagnostics="diagnostics"
        :run-result="runResult"
        :run-error="runError"
        :is-running="isRunning"
        @run="handleRun"
      />
    </main>

    <ProgramModal
      :open="showProgramsModal"
      :programs="savedPrograms"
      :loading="loadingPrograms"
      :error="programsError"
      :active-program-id="programId"
      @close="showProgramsModal = false"
      @load="handleLoadProgram"
      @delete="handleDeleteProgram"
      @refresh="refreshPrograms"
    />

    <QuickAddModal
      :open="showQuickAdd"
      @close="showQuickAdd = false"
      @select="addNodeAt"
    />
  </div>
</template>
