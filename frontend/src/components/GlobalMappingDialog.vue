<template>
  <v-dialog :model-value="show" max-width="760" @update:model-value="onDialogToggle">
    <v-card rounded="lg">
      <v-card-title class="d-flex align-center ga-2">
        <v-icon color="primary">{{ icon }}</v-icon>
        <span>{{ title }}</span>
      </v-card-title>

      <v-card-text>
        <div v-if="description" class="text-body-2 text-medium-emphasis mb-4">
          {{ description }}
        </div>

        <v-skeleton-loader
          v-if="loading"
          type="article"
          class="mb-4"
        />

        <template v-else>
          <v-alert
            type="info"
            variant="tonal"
            density="compact"
            class="mb-4"
          >
            当前共 {{ Object.keys(localMappings).length }} 条规则
          </v-alert>

          <v-list
            v-if="mappingEntries.length"
            density="compact"
            class="mb-4 border rounded"
          >
            <v-list-item
              v-for="[source, target] in mappingEntries"
              :key="source"
            >
              <template #title>
                <div class="d-flex align-center ga-2">
                  <span class="text-body-2 font-weight-medium">{{ source }}</span>
                  <v-icon size="18" color="primary">mdi-arrow-right</v-icon>
                  <span class="text-body-2 text-primary">{{ target }}</span>
                </div>
              </template>
              <template #append>
                <div class="d-flex ga-1">
                  <v-btn
                    size="small"
                    icon
                    variant="text"
                    :disabled="editingSource === source"
                    @click="startEdit(source)"
                  >
                    <v-icon size="18" color="primary">mdi-pencil</v-icon>
                  </v-btn>
                  <v-btn
                    size="small"
                    icon
                    variant="text"
                    @click="removeMapping(source)"
                  >
                    <v-icon size="18" color="error">mdi-delete-outline</v-icon>
                  </v-btn>
                </div>
              </template>
            </v-list-item>
          </v-list>

          <v-row dense>
            <v-col cols="12" md="5">
              <v-text-field
                v-model="draft.source"
                :label="editingSource ? `${sourceLabel}（编辑）` : sourceLabel"
                :placeholder="sourcePlaceholder"
                variant="outlined"
                density="comfortable"
                hide-details="auto"
              />
            </v-col>
            <v-col cols="12" md="5">
              <v-text-field
                v-model="draft.target"
                :label="editingSource ? `${targetLabel}（编辑）` : targetLabel"
                :placeholder="targetPlaceholder"
                variant="outlined"
                density="comfortable"
                hide-details="auto"
                @keyup.enter="upsertMapping"
              />
            </v-col>
            <v-col cols="12" md="2" class="d-flex align-center">
              <v-btn
                color="primary"
                variant="elevated"
                block
                :disabled="!isDraftValid"
                @click="upsertMapping"
              >
                {{ editingSource ? '保存' : '添加' }}
              </v-btn>
            </v-col>
          </v-row>

          <div v-if="editingSource" class="mt-2">
            <v-btn variant="text" color="default" size="small" @click="cancelEdit">
              取消编辑
            </v-btn>
          </div>
        </template>
      </v-card-text>

      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" :disabled="saving" @click="closeDialog">取消</v-btn>
        <v-btn
          color="primary"
          variant="elevated"
          :loading="saving"
          :disabled="loading"
          @click="emitSave"
        >
          保存
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'

interface Props {
  show: boolean
  title: string
  description?: string
  sourceLabel: string
  targetLabel: string
  sourcePlaceholder?: string
  targetPlaceholder?: string
  icon?: string
  mappings: Record<string, string>
  loading?: boolean
  saving?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  sourcePlaceholder: '',
  targetPlaceholder: '',
  icon: 'mdi-swap-horizontal',
  loading: false,
  saving: false
})

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'save', mappings: Record<string, string>): void
}>()

const localMappings = ref<Record<string, string>>({})
const editingSource = ref<string | null>(null)
const draft = reactive({
  source: '',
  target: ''
})

const mappingEntries = computed(() => Object.entries(localMappings.value))

const isDraftValid = computed(() => {
  return draft.source.trim() !== '' && draft.target.trim() !== ''
})

function syncFromProps() {
  localMappings.value = { ...(props.mappings || {}) }
  cancelEdit()
}

watch(() => props.show, visible => {
  if (visible) {
    syncFromProps()
  }
})

watch(() => props.mappings, () => {
  if (props.show && !editingSource.value && draft.source === '' && draft.target === '') {
    syncFromProps()
  }
}, { deep: true })

function onDialogToggle(value: boolean) {
  emit('update:show', value)
}

function closeDialog() {
  emit('update:show', false)
}

function startEdit(source: string) {
  editingSource.value = source
  draft.source = source
  draft.target = localMappings.value[source] || ''
}

function cancelEdit() {
  editingSource.value = null
  draft.source = ''
  draft.target = ''
}

function upsertMapping() {
  const source = draft.source.trim()
  const target = draft.target.trim()
  if (!source || !target) return

  if (editingSource.value && editingSource.value !== source) {
    delete localMappings.value[editingSource.value]
  }

  localMappings.value[source] = target
  cancelEdit()
}

function removeMapping(source: string) {
  if (editingSource.value === source) {
    cancelEdit()
  }
  delete localMappings.value[source]
}

function emitSave() {
  emit('save', { ...localMappings.value })
}
</script>
