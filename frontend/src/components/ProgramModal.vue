<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ProgramSummary } from '../services/api'

const props = defineProps<{
  open: boolean
  programs: ProgramSummary[]
  loading: boolean
  error: string | null
  activeProgramId: string | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'load', id: string): void
  (e: 'delete', id: string): void
  (e: 'refresh'): void
}>()

const search = ref('')

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return props.programs
  return props.programs.filter((p) => p.name.toLowerCase().includes(q))
})

function formatDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}
</script>

<template>
  <div v-if="open" class="modal-backdrop" @click.self="emit('close')">
    <div class="modal" role="dialog" aria-modal="true" aria-label="Saved programs">
      <div class="modal__header">
        <h2>Saved Programs</h2>
        <div class="modal__header-actions">
          <button type="button" class="btn btn--ghost btn--sm" @click="emit('refresh')">
            Refresh
          </button>
          <button type="button" class="btn btn--ghost btn--sm" @click="emit('close')">Close</button>
        </div>
      </div>

      <div class="modal__body">
        <input
          v-model="search"
          type="text"
          class="modal__search"
          placeholder="Filter saved programs..."
        />

        <div v-if="loading" class="modal__empty">Loading programs from backend...</div>
        <div v-else-if="error" class="diagnostic diagnostic--error">{{ error }}</div>
        <div v-else-if="filtered.length === 0" class="modal__empty">
          No saved programs yet. Save your current canvas with Ctrl+S.
        </div>

        <ul v-else class="program-list">
          <li
            v-for="item in filtered"
            :key="item.id"
            class="program-list__item"
            :class="{ 'program-list__item--active': item.id === activeProgramId }"
          >
            <div class="program-list__info">
              <div class="program-list__name">{{ item.name }}</div>
              <div class="program-list__meta">Updated {{ formatDate(item.updatedAt) }}</div>
            </div>
            <div class="program-list__actions">
              <button type="button" class="btn btn--primary btn--sm" @click="emit('load', item.id)">
                Load
              </button>
              <button
                type="button"
                class="btn btn--danger btn--sm"
                @click="emit('delete', item.id)"
              >
                Delete
              </button>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
