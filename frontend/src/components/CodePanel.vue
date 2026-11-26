<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Diagnostic } from '../ast/types'
import type { RunResult } from '../services/api'

const props = defineProps<{
  code: string
  diagnostics: Diagnostic[]
  runResult: RunResult | null
  runError: string | null
  isRunning: boolean
}>()

const emit = defineEmits<{
  (e: 'run'): void
}>()

const copied = ref(false)

const codeLines = computed(() => {
  const raw = props.code.endsWith('\n') ? props.code.slice(0, -1) : props.code
  return raw.split('\n')
})

async function copyCode() {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch {
    copied.value = false
  }
}
</script>

<template>
  <aside class="code-panel">
    <section class="panel-section panel-section--code">
      <div class="panel-header">
        <div class="panel-header__title">
          <h2>Generated Python</h2>
          <span class="badge badge--python">Python 3</span>
        </div>
        <div class="panel-header__actions">
          <button type="button" class="btn btn--ghost btn--sm" @click="copyCode">
            {{ copied ? 'Copied!' : 'Copy' }}
          </button>
          <button
            type="button"
            class="btn btn--primary btn--sm"
            :disabled="isRunning"
            @click="emit('run')"
          >
            {{ isRunning ? 'Running...' : 'Run' }}
          </button>
        </div>
      </div>

      <div class="code-viewer" role="region" aria-label="Generated Python code">
        <div v-for="(line, idx) in codeLines" :key="idx" class="code-line">
          <span class="code-line__num">{{ idx + 1 }}</span>
          <span class="code-line__text">{{ line || ' ' }}</span>
        </div>
      </div>

      <div v-if="diagnostics.length > 0" class="diagnostics">
        <div
          v-for="(diag, index) in diagnostics"
          :key="index"
          class="diagnostic"
          :class="`diagnostic--${diag.level}`"
        >
          <strong>{{ diag.level.toUpperCase() }}:</strong> {{ diag.message }}
        </div>
      </div>
    </section>

    <section class="panel-section panel-section--console">
      <div class="panel-header">
        <div class="panel-header__title">
          <h2>Output</h2>
          <span
            v-if="runResult"
            class="badge"
            :class="runResult.exitCode === 0 ? 'badge--ok' : 'badge--err'"
          >
            {{ runResult.timedOut ? 'TIMEOUT' : `exit ${runResult.exitCode}` }}
          </span>
        </div>
        <span v-if="runResult" class="muted-meta">{{ runResult.durationMs }} ms</span>
      </div>

      <div class="console-body">
        <div v-if="isRunning" class="console-placeholder">Executing Python script...</div>
        <div v-else-if="runError" class="console-stderr">{{ runError }}</div>
        <template v-else-if="runResult">
          <pre v-if="runResult.stdout" class="console-stdout">{{ runResult.stdout }}</pre>
          <pre v-if="runResult.stderr" class="console-stderr">{{ runResult.stderr }}</pre>
          <div v-if="!runResult.stdout && !runResult.stderr" class="console-placeholder">
            Program finished with no output. Add a Print node to display values.
          </div>
        </template>
        <div v-else class="console-placeholder">
          Press <strong>Run</strong> or <kbd class="kbd">Ctrl+Enter</kbd> to evaluate your Python
          program on the backend.
        </div>
      </div>
    </section>
  </aside>
</template>
