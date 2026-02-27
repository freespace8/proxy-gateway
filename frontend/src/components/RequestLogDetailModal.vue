<template>
  <v-dialog v-model="open" max-width="1100">
    <v-card>
      <v-card-title class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-text-box-search-outline</v-icon>
          <span class="text-subtitle-1">请求详情</span>
          <v-chip v-if="detail?.requestId" size="x-small" variant="tonal" class="font-mono">
            {{ detail.requestId }}
          </v-chip>
        </div>
        <div class="d-flex align-center ga-2">
          <v-btn size="small" variant="text" :disabled="!curlCommand" title="复制 cURL" @click="copyCurl">
            <v-icon start size="small">{{ copiedCurl ? 'mdi-check' : 'mdi-content-copy' }}</v-icon>
            复制 cURL
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

        <div v-else-if="detail">
          <div class="text-caption text-medium-emphasis mb-3">
            {{ (detail.requestMethod || 'POST').toUpperCase() }} {{ detail.requestUrl || '--' }}
          </div>

          <v-row dense>
            <v-col cols="12" md="5">
              <div class="d-flex align-center justify-space-between mb-2">
                <span class="text-subtitle-2">请求头</span>
                <v-btn size="x-small" variant="text" :disabled="!headersText" @click="copyHeaders">
                  <v-icon start size="small">{{ copiedHeaders ? 'mdi-check' : 'mdi-content-copy' }}</v-icon>
                  复制
                </v-btn>
              </div>
              <div class="code-container">
                <pre class="code-pre">{{ headersText || '无' }}</pre>
              </div>
            </v-col>

            <v-col cols="12" md="7">
              <div class="d-flex align-center justify-space-between mb-2">
                <div class="d-flex align-center ga-2">
                  <span class="text-subtitle-2">请求 Body</span>
                  <v-chip
                    v-if="detail.requestBodyTruncated"
                    size="x-small"
                    color="warning"
                    variant="tonal"
                  >
                    已截断
                  </v-chip>
                </div>
                <v-btn size="x-small" variant="text" :disabled="!rawBody" @click="copyBody">
                  <v-icon start size="small">{{ copiedBody ? 'mdi-check' : 'mdi-content-copy' }}</v-icon>
                  复制
                </v-btn>
              </div>
              <div class="code-container">
                <JsonTreeView v-if="parsedBody.ok" :value="parsedBody.value" class="code-tree" />
                <pre v-else class="code-pre">{{ bodyText || '无' }}</pre>
              </div>
            </v-col>

            <v-col cols="12">
              <div class="d-flex align-center justify-space-between mb-2">
                <span class="text-subtitle-2">cURL</span>
                <v-btn size="x-small" variant="text" :disabled="!curlCommand" @click="copyCurl">
                  <v-icon start size="small">{{ copiedCurl ? 'mdi-check' : 'mdi-content-copy' }}</v-icon>
                  复制
                </v-btn>
              </div>
              <div class="code-container">
                <pre class="code-pre">{{ curlCommand || '无' }}</pre>
              </div>
            </v-col>
          </v-row>
        </div>

        <div v-else class="py-6 text-center text-caption text-medium-emphasis">暂无数据</div>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api, type ApiType, type RequestLogRecord } from '@/services/api'
import JsonTreeView from '@/components/JsonTreeView.vue'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  apiType: ApiType
  logId: number | null
}>()

const loading = ref(false)
const error = ref('')
const detail = ref<RequestLogRecord | null>(null)
let requestSeq = 0

const copiedCurl = ref(false)
const copiedHeaders = ref(false)
const copiedBody = ref(false)

const headersObject = computed<Record<string, string>>(() => {
  const headers = detail.value?.requestHeaders || {}
  const keys = Object.keys(headers).sort((a, b) => a.localeCompare(b))
  const out: Record<string, string> = {}
  for (const k of keys) out[k] = headers[k] ?? ''
  return out
})

const headersText = computed(() => {
  const obj = headersObject.value
  if (!obj || Object.keys(obj).length === 0) return ''
  return JSON.stringify(obj, null, 2)
})

const rawBody = computed(() => detail.value?.requestBody || '')

const parsedBody = computed<{ ok: true; value: unknown } | { ok: false; value: null }>(() => {
  if (!rawBody.value) return { ok: false, value: null }
  try {
    return { ok: true, value: JSON.parse(rawBody.value) as unknown }
  } catch {
    return { ok: false, value: null }
  }
})

const bodyText = computed(() => {
  if (!rawBody.value) return ''
  if (parsedBody.value.ok) return JSON.stringify(parsedBody.value.value, null, 2)
  return rawBody.value
})

function shellQuote(value: string): string {
  if (value === '') return "''"
  return `'${value.replace(/'/g, `'\"'\"'`)}'`
}

function buildCurlBody(raw: string): string {
  if (!raw) return raw
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    if (!parsed || typeof parsed !== 'object') return raw

    const model = typeof parsed.model === 'string' ? parsed.model.toLowerCase() : ''
    if (!model.includes('codex') || !('input' in parsed)) return raw

    const replaced: Record<string, unknown> = { ...parsed }
    replaced.input = [
      {
        role: 'user',
        content: [{ type: 'input_text', text: '你好' }],
      },
    ]
    return JSON.stringify(replaced)
  } catch {
    return raw
  }
}

const curlCommand = computed(() => {
  if (!detail.value) return ''

  const method = (detail.value.requestMethod || 'POST').toUpperCase()
  const requestUrl = detail.value.requestUrl || ''
  const url = /^https?:\/\//i.test(requestUrl)
    ? requestUrl
    : `${window.location.origin}${requestUrl.startsWith('/') ? requestUrl : `/${requestUrl || ''}`}`

  const lines: string[] = [`curl -X ${method} ${shellQuote(url)}`]

  const original = (detail.value.requestHeaders || {}) as Record<string, string>
  const filtered: Record<string, string> = {}
  for (const [k, v] of Object.entries(original)) {
    const lower = k.toLowerCase()
    if (!v) continue
    if (lower === 'content-length' || lower === 'host' || lower === 'connection') continue
    filtered[k] = v
  }

  const headerKeys = Object.keys(filtered).sort((a, b) => a.localeCompare(b))
  for (const k of headerKeys) {
    lines.push(`-H ${shellQuote(`${k}: ${filtered[k]}`)}`)
  }

  if (rawBody.value && method !== 'GET' && method !== 'HEAD') {
    lines.push(`--data-binary ${shellQuote(buildCurlBody(rawBody.value))}`)
  }

  return lines.join(' ')
})

async function fetchDetail() {
  if (!props.logId) {
    detail.value = null
    return
  }
  const seq = ++requestSeq
  loading.value = true
  error.value = ''
  try {
    const resp = await api.getRequestLogDetail(props.apiType, props.logId)
    if (seq !== requestSeq) return
    detail.value = resp.log || null
  } catch (e) {
    if (seq !== requestSeq) return
    error.value = e instanceof Error ? e.message : '加载失败'
    detail.value = null
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

watch(
  () => [open.value, props.apiType, props.logId],
  ([isOpen]) => {
    if (isOpen) fetchDetail()
  }
)

watch(open, v => {
  if (!v) {
    copiedCurl.value = false
    copiedHeaders.value = false
    copiedBody.value = false
  }
})

async function copyText(text: string, flag: { value: boolean }) {
  if (!text) return
  await navigator.clipboard.writeText(text)
  flag.value = true
  window.setTimeout(() => {
    flag.value = false
  }, 1200)
}

const copyCurl = () => copyText(curlCommand.value, copiedCurl)
const copyHeaders = () => copyText(headersText.value, copiedHeaders)
const copyBody = () => copyText(rawBody.value, copiedBody)
</script>

<style scoped>
.font-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}
.code-container {
  max-height: 55vh;
  overflow: auto;
  background: rgb(var(--v-theme-surface));
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 8px;
}
.code-pre {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.4;
  white-space: pre;
  margin: 0;
  padding: 12px;
}

.code-tree {
  padding: 12px;
}
</style>
