<template>
  <v-dialog v-model="open" max-width="960">
    <v-card>
      <v-card-title class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="warning">mdi-alert-circle</v-icon>
          <span class="text-subtitle-1">{{ title || '最后失败日志' }}</span>
          <v-chip v-if="truncated" size="x-small" color="warning" variant="tonal">已截断（中间省略）</v-chip>
        </div>
        <div class="d-flex align-center ga-2">
          <v-btn
            size="small"
            variant="text"
            :disabled="!rawLog"
            @click="copyToClipboard"
            title="复制"
          >
            <v-icon start size="small">mdi-content-copy</v-icon>
            复制
          </v-btn>
          <v-btn icon variant="text" @click="open = false">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </div>
      </v-card-title>

      <v-divider />

      <v-card-text class="pt-3">
        <v-alert v-if="error" type="error" variant="tonal" class="mb-3" density="compact">
          {{ error }}
        </v-alert>

        <div v-if="loading" class="d-flex justify-center py-8">
          <v-progress-circular indeterminate />
        </div>

        <div v-else class="log-container">
          <pre class="log-pre">{{ prettyLog }}</pre>
        </div>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  title?: string
  log?: string
  loading?: boolean
  error?: string
}>()

const rawLog = computed(() => props.log || '')

const parsed = computed(() => {
  try {
    return rawLog.value ? JSON.parse(rawLog.value) : null
  } catch {
    return null
  }
})

const truncated = computed(() => {
  if (parsed.value && typeof parsed.value === 'object') {
    return Boolean((parsed.value as any).truncated)
  }
  return rawLog.value.includes('...(中间省略)...')
})

const prettyLog = computed(() => {
  if (!rawLog.value) return '暂无失败日志'
  if (parsed.value) return JSON.stringify(parsed.value, null, 2)
  return rawLog.value
})

const copyToClipboard = async () => {
  if (!rawLog.value) return
  await navigator.clipboard.writeText(rawLog.value)
}
</script>

<style scoped>
.log-container {
  max-height: 70vh;
  overflow: auto;
  background: rgb(var(--v-theme-surface));
}
.log-pre {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.4;
  white-space: pre;
  margin: 0;
  padding: 12px;
}
</style>
