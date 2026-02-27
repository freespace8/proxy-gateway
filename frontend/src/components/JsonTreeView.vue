<template>
  <div class="json-node">
    <details v-if="isCollapsible" :open="defaultOpen">
      <summary class="json-summary">
        <span class="json-brace">{{ isArrayValue ? '[' : '{' }}</span>
        <span class="json-ellipsis">...</span>
        <span class="json-brace">{{ isArrayValue ? ']' : '}' }}</span>
      </summary>
      <div class="json-children">
        <div v-for="([key, child], index) in entries" :key="key" class="json-row">
          <span v-if="!isArrayValue" class="json-key">{{ formatKey(key) }}</span>
          <span v-if="!isArrayValue" class="json-separator">: </span>
          <JsonTreeView :value="child" :level="level + 1" />
          <span v-if="index < entries.length - 1" class="json-comma">,</span>
        </div>
      </div>
      <div class="json-close">{{ isArrayValue ? ']' : '}' }}</div>
    </details>
    <span v-else :class="primitiveClass">{{ primitiveText }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

defineOptions({ name: 'JsonTreeView' })

const props = withDefaults(defineProps<{
  value: unknown
  level?: number
}>(), {
  level: 0
})

const level = computed(() => props.level ?? 0)
const isArrayValue = computed(() => Array.isArray(props.value))
const isObjectValue = computed(
  () => props.value !== null && typeof props.value === 'object' && !Array.isArray(props.value)
)
const isCollapsible = computed(() => isArrayValue.value || isObjectValue.value)
const defaultOpen = computed(() => level.value >= 0)

const entries = computed<Array<[string, unknown]>>(() => {
  if (Array.isArray(props.value)) {
    return props.value.map((item, index) => [String(index), item])
  }
  if (props.value !== null && typeof props.value === 'object') {
    return Object.entries(props.value as Record<string, unknown>)
  }
  return []
})

const primitiveText = computed(() => formatPrimitive(props.value))
const primitiveClass = computed(() => {
  if (props.value === null) return 'json-null'
  switch (typeof props.value) {
    case 'string':
      return 'json-string'
    case 'number':
      return 'json-number'
    case 'boolean':
      return 'json-boolean'
    default:
      return 'json-plain'
  }
})

function formatKey(key: string): string {
  return isArrayValue.value ? key : JSON.stringify(key)
}

function formatPrimitive(value: unknown): string {
  if (typeof value === 'string') return JSON.stringify(value)
  if (value === undefined) return 'undefined'
  return JSON.stringify(value)
}
</script>

<style scoped>
.json-node {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
}

.json-summary {
  cursor: pointer;
  user-select: none;
}

.json-summary::-webkit-details-marker {
  margin-right: 6px;
}

.json-brace {
  color: rgba(var(--v-theme-on-surface), 0.9);
}

.json-ellipsis {
  color: rgba(var(--v-theme-on-surface), 0.45);
  margin: 0 4px;
}

.json-children {
  margin-left: 16px;
  border-left: 1px dashed rgba(var(--v-theme-on-surface), 0.16);
  padding-left: 10px;
}

.json-row {
  white-space: nowrap;
}

.json-key {
  color: rgb(var(--v-theme-primary));
}

.json-separator {
  color: rgba(var(--v-theme-on-surface), 0.7);
}

.json-comma {
  color: rgba(var(--v-theme-on-surface), 0.7);
}

.json-close {
  color: rgba(var(--v-theme-on-surface), 0.9);
}

.json-string {
  color: rgb(var(--v-theme-success));
}

.json-number {
  color: rgb(var(--v-theme-warning));
}

.json-boolean {
  color: rgb(var(--v-theme-info));
}

.json-null,
.json-plain {
  color: rgba(var(--v-theme-on-surface), 0.82);
}
</style>
