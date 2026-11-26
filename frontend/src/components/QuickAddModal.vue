<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { DrawflowNodeType } from '../ast/types'
import { NODE_DEFINITIONS } from '../nodes/nodeDefinitions'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'select', type: DrawflowNodeType): void
}>()

const query = ref('')
const selectedIndex = ref(0)
const inputEl = ref<HTMLInputElement | null>(null)

const matches = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return NODE_DEFINITIONS
  return NODE_DEFINITIONS.filter(
    (d) =>
      d.label.toLowerCase().includes(q) ||
      d.type.toLowerCase().includes(q) ||
      d.description.toLowerCase().includes(q) ||
      d.category.toLowerCase().includes(q),
  )
})

watch(
  () => props.open,
  async (isOpen) => {
    if (isOpen) {
      query.value = ''
      selectedIndex.value = 0
      await nextTick()
      inputEl.value?.focus()
    }
  },
)

watch(matches, () => {
  selectedIndex.value = 0
})

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (matches.value.length > 0) {
      selectedIndex.value = (selectedIndex.value + 1) % matches.value.length
    }
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (matches.value.length > 0) {
      selectedIndex.value = (selectedIndex.value - 1 + matches.value.length) % matches.value.length
    }
  } else if (event.key === 'Enter') {
    event.preventDefault()
    const item = matches.value[selectedIndex.value]
    if (item) {
      emit('select', item.type)
      emit('close')
    }
  } else if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
  }
}
</script>

<template>
  <div v-if="open" class="modal-backdrop" @click.self="emit('close')">
    <div class="modal modal--compact" role="dialog" aria-modal="true" aria-label="Quick add node">
      <div class="modal__header">
        <h2>Quick Add Node</h2>
        <span class="muted-meta">Use ↑↓ and Enter</span>
      </div>

      <div class="modal__body">
        <input
          ref="inputEl"
          v-model="query"
          type="text"
          class="modal__search"
          placeholder="Type a node name (e.g., math, if, for, assign, block)..."
          @keydown="handleKeydown"
        />

        <ul class="quick-list">
          <li
            v-for="(item, idx) in matches"
            :key="item.type"
            class="quick-list__item"
            :class="{ 'quick-list__item--selected': idx === selectedIndex }"
            @mouseenter="selectedIndex = idx"
            @click="
              emit('select', item.type);
              emit('close')
            "
          >
            <div>
              <strong>{{ item.label }}</strong>
              <span class="quick-list__desc">{{ item.description }}</span>
            </div>
            <span class="muted-meta">{{ item.category }}</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
