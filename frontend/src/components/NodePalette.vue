<script setup lang="ts">
import { computed } from 'vue'
import type { DrawflowNodeType } from '../ast/types'
import { NODE_DEFINITIONS, type NodeDefinition } from '../nodes/nodeDefinitions'

const emit = defineEmits<{
  (e: 'add-node', type: DrawflowNodeType): void
}>()

const categories = computed(() => {
  const groups: Array<{ title: NodeDefinition['category']; items: NodeDefinition[] }> = [
    { title: 'Control Flow', items: [] },
    { title: 'Statements', items: [] },
    { title: 'Expressions', items: [] },
  ]
  for (const def of NODE_DEFINITIONS) {
    const group = groups.find((g) => g.title === def.category)
    group?.items.push(def)
  }
  return groups
})

function onDragStart(event: DragEvent, type: DrawflowNodeType) {
  event.dataTransfer?.setData('application/drawflow-node', type)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'copy'
  }
}
</script>

<template>
  <aside class="palette">
    <div class="palette__header">
      <h2>Nodes</h2>
      <span class="palette__hint">Drag or click · Ctrl+K</span>
    </div>

    <div class="palette__sections">
      <section v-for="group in categories" :key="group.title" class="palette__group">
        <h3>{{ group.title }}</h3>
        <button
          v-for="item in group.items"
          :key="item.type"
          type="button"
          class="palette__item"
          :class="`palette__item--${item.category.toLowerCase().replace(' ', '-')}`"
          draggable="true"
          :title="item.description"
          @dragstart="onDragStart($event, item.type)"
          @click="emit('add-node', item.type)"
        >
          <div class="palette__item-top">
            <span class="palette__item-label">{{ item.label }}</span>
            <kbd class="kbd">{{ item.shortcut }}</kbd>
          </div>
          <span class="palette__item-desc">{{ item.description }}</span>
        </button>
      </section>
    </div>

    <div class="palette__shortcuts">
      <div class="shortcut-row"><kbd class="kbd">Ctrl+K</kbd><span>Quick add node</span></div>
      <div class="shortcut-row"><kbd class="kbd">Ctrl+Enter</kbd><span>Run Python</span></div>
      <div class="shortcut-row"><kbd class="kbd">Ctrl+S</kbd><span>Save program</span></div>
      <div class="shortcut-row"><kbd class="kbd">Del</kbd><span>Delete selected</span></div>
    </div>
  </aside>
</template>
