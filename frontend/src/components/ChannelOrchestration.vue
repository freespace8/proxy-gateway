<template>
  <v-card elevation="0" rounded="lg" class="channel-orchestration" variant="flat">
    <!-- 调度器统计信息 -->
    <v-card-title class="d-flex align-center justify-space-between py-3 px-0">
      <div class="d-flex align-center">
        <v-icon class="mr-2" color="primary">mdi-swap-vertical-bold</v-icon>
        <span class="text-h6">渠道编排</span>
        <v-chip v-if="isMultiChannelMode" size="small" color="success" variant="tonal" class="ml-3">
          多渠道模式
        </v-chip>
        <v-chip v-else size="small" color="warning" variant="tonal" class="ml-3"> 单渠道模式 </v-chip>
      </div>
      <div class="d-flex align-center ga-2">
        <v-progress-circular v-if="isLoadingMetrics" indeterminate size="16" width="2" color="primary" />
      </div>
    </v-card-title>

    <v-divider />

	    <!-- 故障转移序列 (active + suspended) -->
	    <div class="pt-3 pb-2">
	      <div class="d-flex align-center justify-space-between mb-2">
	        <div class="text-subtitle-2 text-medium-emphasis d-flex align-center">
	          <v-icon size="small" class="mr-1" color="success">mdi-play-circle</v-icon>
	          故障转移序列
          <v-chip size="x-small" class="ml-2">{{ activeChannels.length }}</v-chip>
        </div>
        <div class="d-flex align-center ga-2">
          <span class="text-caption text-medium-emphasis">拖拽调整优先级，自动保存</span>
          <v-progress-circular v-if="isSavingOrder" indeterminate size="16" width="2" color="primary" />
        </div>
      </div>

      <!-- 拖拽列表 -->
      <draggable
        v-model="activeChannels"
        item-key="index"
        handle=".drag-handle"
        ghost-class="ghost"
        class="channel-list"
        @change="onDragChange"
      >
        <template #item="{ element, index }">
          <div class="channel-item-wrapper">
            <div
              class="channel-row"
              :class="{ 'is-suspended': element.status === 'suspended' }"
              @click="toggleChannelStats(element.index)"
            >
            <!-- SVG 活跃度波形柱状图背景 -->
            <svg class="activity-chart-bg" preserveAspectRatio="none" viewBox="0 0 150 100">
              <!-- 渐变定义（为每个柱子单独定义渐变） -->
              <defs>
                <linearGradient
                  v-for="(bar, i) in getActivityBars(element.index)"
                  :id="`gradient-${element.index}-${i}`"
                  :key="`gradient-${element.index}-${i}`"
                  x1="0%"
                  y1="0%"
                  x2="0%"
                  y2="100%"
                >
                  <stop offset="0%" :stop-color="bar.color" stop-opacity="0.8" />
                  <stop offset="100%" :stop-color="bar.color" stop-opacity="0.3" />
                </linearGradient>
              </defs>
              <!-- 波形柱状图 -->
              <g v-for="(bar, i) in getActivityBars(element.index)" :key="i">
                <rect
                  :x="bar.x"
                  :y="bar.y"
                  :width="bar.width"
                  :height="bar.height"
                  :fill="`url(#gradient-${element.index}-${i})`"
                  :rx="bar.radius"
                  :ry="bar.radius"
                  class="activity-bar"
                />
              </g>
            </svg>

            <!-- Grid 内容容器 -->
            <div class="channel-row-content">
            <!-- 拖拽手柄 -->
            <div class="drag-handle" @click.stop>
              <v-icon size="small" color="grey">mdi-drag-vertical</v-icon>
            </div>

            <!-- 优先级序号 -->
            <div class="priority-number" @click.stop>
              <span class="text-caption font-weight-bold">{{ index + 1 }}</span>
            </div>

            <!-- 状态指示器 -->
            <div @click.stop>
	              <ChannelStatusBadge :status="getChannelDisplayStatus(element)" :metrics="getChannelMetrics(element.index)" />
            </div>

            <!-- 渠道名称和描述 -->
            <div class="channel-name">
              <span
                class="font-weight-medium channel-name-link"
                tabindex="0"
                role="button"
                @click.stop="$emit('edit', element)"
                @keydown.enter.stop="$emit('edit', element)"
                @keydown.space.stop="$emit('edit', element)"
              >{{ element.name }}</span>
              <!-- 促销期标识 -->
              <v-chip
                v-if="isInPromotion(element)"
                size="x-small"
                color="info"
                variant="flat"
                class="ml-2"
              >
                <v-icon start size="12">mdi-rocket-launch</v-icon>
                {{ formatPromotionRemaining(element.promotionUntil) }}
              </v-chip>
              <!-- 官网链接按钮 -->
              <v-btn
                :href="getWebsiteUrl(element)"
                target="_blank"
                rel="noopener"
                icon
                size="x-small"
                variant="text"
                color="primary"
                class="ml-1"
                title="打开官网"
                @click.stop
              >
                <v-icon size="14">mdi-open-in-new</v-icon>
              </v-btn>
              <span class="text-caption text-medium-emphasis ml-2">{{ element.serviceType }}</span>
              <span v-if="element.description" class="text-caption text-disabled ml-3 channel-description">{{ element.description }}</span>
              <!-- 展开图标 -->
              <v-icon
                size="x-small"
                class="ml-auto expand-icon"
                :color="statsExpandedChannelIndex === element.index ? 'primary' : 'grey-lighten-1'"
              >{{ statsExpandedChannelIndex === element.index ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
            </div>

            <!-- 指标显示 -->
            <div class="channel-metrics" @click.stop>
              <template v-if="getChannelMetrics(element.index)">
                <v-tooltip location="top" :open-delay="200">
                  <template #activator="{ props: tooltipProps }">
                    <div v-bind="tooltipProps" class="d-flex align-center metrics-display">
                      <!-- 15分钟有请求时显示成功率，否则显示 -- -->
                      <template v-if="get15mStats(element.index)?.requestCount">
                        <v-chip
                          size="x-small"
                          :color="getSuccessRateColor(get15mStats(element.index)?.successRate)"
                          variant="tonal"
                        >
                          {{ get15mStats(element.index)?.successRate?.toFixed(0) }}%
                        </v-chip>
                        <span class="text-caption text-medium-emphasis ml-2 mr-1">
                          {{ get15mStats(element.index)?.requestCount }} 请求
                        </span>
                        <v-chip
                          v-if="shouldShowCacheHitRate(get15mStats(element.index))"
                          size="x-small"
                          :color="getCacheHitRateColor(get15mStats(element.index)?.cacheHitRate)"
                          variant="tonal"
                          class="ml-1"
                        >
                          缓存 {{ get15mStats(element.index)?.cacheHitRate?.toFixed(0) }}%
                        </v-chip>
                      </template>
                      <span v-else class="text-caption text-medium-emphasis">--</span>
                    </div>
                  </template>
                  <div class="metrics-tooltip">
                    <div class="text-caption font-weight-bold mb-1">请求统计</div>
                    <div class="metrics-tooltip-row">
                      <span>15分钟:</span>
                      <span>{{ formatStats(get15mStats(element.index)) }}</span>
                    </div>
                    <div class="metrics-tooltip-row">
                      <span>1小时:</span>
                      <span>{{ formatStats(get1hStats(element.index)) }}</span>
                    </div>
                    <div class="metrics-tooltip-row">
                      <span>6小时:</span>
                      <span>{{ formatStats(get6hStats(element.index)) }}</span>
                    </div>
                    <div class="metrics-tooltip-row">
                      <span>24小时:</span>
                      <span>{{ formatStats(get24hStats(element.index)) }}</span>
                    </div>

                    <div class="text-caption font-weight-bold mt-2 mb-1">缓存统计 (Token)</div>
                    <div class="metrics-tooltip-row">
                      <span>15分钟:</span>
                      <span>{{ formatCacheStats(get15mStats(element.index)) }}</span>
                    </div>
                    <div class="metrics-tooltip-row">
                      <span>1小时:</span>
                      <span>{{ formatCacheStats(get1hStats(element.index)) }}</span>
                    </div>
                    <div class="metrics-tooltip-row">
                      <span>6小时:</span>
                      <span>{{ formatCacheStats(get6hStats(element.index)) }}</span>
                    </div>
                    <div class="metrics-tooltip-row">
                      <span>24小时:</span>
                      <span>{{ formatCacheStats(get24hStats(element.index)) }}</span>
                    </div>
                  </div>
                </v-tooltip>
              </template>
              <span v-else class="text-caption text-medium-emphasis">--</span>
            </div>

            <!-- RPM/TPM 显示 -->
            <div class="channel-rpm-tpm" @click.stop>
              <div class="rpm-tpm-values">
                <span class="rpm-value" :class="{ 'has-data': hasActivityData(element.index) }">{{ formatRPM(element.index) }}</span>
                <span class="rpm-tpm-separator">/</span>
                <span class="tpm-value" :class="{ 'has-data': hasActivityData(element.index) }">{{ formatTPM(element.index) }}</span>
              </div>
              <div class="rpm-tpm-labels">
                <span>RPM</span>
                <span>/</span>
                <span>TPM</span>
              </div>
            </div>

            <!-- 延迟显示 -->
            <div class="channel-latency" @click.stop>
              <v-chip
                v-if="isLatencyValid(element)"
                size="x-small"
                :color="getLatencyColor(element.latency!)"
                variant="tonal"
              >
                {{ element.latency }}ms
              </v-chip>
            </div>

            <!-- API密钥数量 -->
            <div class="channel-keys" @click.stop>
              <v-chip size="x-small" variant="outlined" class="keys-chip" @click="$emit('edit', element)">
                <v-icon start size="x-small">mdi-key</v-icon>
                {{ element.apiKeys?.length || 0 }}
              </v-chip>
            </div>

            <!-- 操作按钮 -->
            <div class="channel-actions" @click.stop>
              <!-- suspended 状态显示恢复按钮 -->
              <v-btn
                v-if="element.status === 'suspended'"
                icon
                size="x-small"
                variant="text"
                color="warning"
                title="恢复"
                @click="resumeChannel(element.index)"
              >
                <v-icon size="small">mdi-refresh</v-icon>
              </v-btn>

              <v-menu>
                <template #activator="{ props: activatorProps }">
                  <v-btn icon size="x-small" variant="text" v-bind="activatorProps">
                    <v-icon size="small">mdi-dots-vertical</v-icon>
                  </v-btn>
                </template>
                <v-list density="compact">
                  <v-list-item @click="$emit('edit', element)">
                    <template #prepend>
                      <v-icon size="small">mdi-pencil</v-icon>
                    </template>
                    <v-list-item-title>编辑</v-list-item-title>
                  </v-list-item>
                  <v-list-item @click="$emit('ping', element.index)">
                    <template #prepend>
                      <v-icon size="small">mdi-speedometer</v-icon>
                    </template>
                    <v-list-item-title>测试延迟</v-list-item-title>
                  </v-list-item>
                  <v-list-item @click="setPromotion(element)">
                    <template #prepend>
                      <v-icon size="small" color="info">mdi-rocket-launch</v-icon>
                    </template>
                    <v-list-item-title>抢优先级 (5分钟)</v-list-item-title>
                  </v-list-item>
                  <v-divider />
                  <v-list-item v-if="element.status === 'suspended'" @click="resumeChannel(element.index)">
                    <template #prepend>
                      <v-icon size="small" color="success">mdi-play-circle</v-icon>
                    </template>
                    <v-list-item-title>恢复 (重置指标)</v-list-item-title>
                  </v-list-item>
                  <v-list-item
                    v-if="element.status !== 'suspended'"
                    @click="setChannelStatus(element.index, 'suspended')"
                  >
                    <template #prepend>
                      <v-icon size="small" color="warning">mdi-pause-circle</v-icon>
                    </template>
                    <v-list-item-title>暂停</v-list-item-title>
                  </v-list-item>
                  <v-list-item @click="setChannelStatus(element.index, 'disabled')">
                    <template #prepend>
                      <v-icon size="small" color="error">mdi-stop-circle</v-icon>
                    </template>
                    <v-list-item-title>移至备用池</v-list-item-title>
                  </v-list-item>
                  <v-list-item :disabled="!canDeleteChannel(element)" @click="handleDeleteChannel(element)">
                    <template #prepend>
                      <v-icon size="small" :color="canDeleteChannel(element) ? 'error' : 'grey'">mdi-delete</v-icon>
                    </template>
                    <v-list-item-title>
                      删除
                      <span v-if="!canDeleteChannel(element)" class="text-caption text-disabled ml-1">
                        (至少保留一个)
                      </span>
                    </v-list-item-title>
                  </v-list-item>
                </v-list>
              </v-menu>
            </div>
          </div><!-- .channel-row-content -->
          </div><!-- .channel-row -->

	          <div class="channel-chart-wrapper">
	              <div class="key-metrics-panel" @click.stop>
		                <div class="d-flex align-center justify-space-between mb-2">
		                  <div class="text-subtitle-2 text-medium-emphasis d-flex align-center">
		                    <v-icon size="small" class="mr-1" color="warning">mdi-key</v-icon>
		                    Key 状态
	                  </div>
	                  <v-btn size="x-small" variant="text" @click.stop="refreshMetrics">刷新</v-btn>
	                </div>
	
	                <v-table density="compact" class="key-metrics-table">
	                  <thead>
	                    <tr>
	                      <th style="width: 64px;">序号</th>
	                      <th>Key</th>
	                      <th style="width: 220px;">描述</th>
	                      <th style="width: 110px;">熔断</th>
                      <th style="width: 120px;">成功率</th>
                      <th style="width: 110px;">请求</th>
                      <th style="width: 140px;">连续失败</th>
                      <th style="width: 130px;">状态</th>
                      <th style="width: 200px;">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="(km, keyIndex) in (getChannelMetrics(element.index)?.keyMetrics || [])"
                      :key="`${element.index}-${keyIndex}`"
                    >
                      <td class="text-caption">{{ keyIndex + 1 }}</td>
                      <td class="text-caption">{{ km.keyMask }}</td>
                      <td class="text-caption">{{ getAPIKeyDescription(element, keyIndex) }}</td>
                      <td>
                        <v-chip
                          v-if="km.circuitBroken"
                          size="x-small"
                          color="warning"
                          variant="tonal"
                        >
                          熔断
                        </v-chip>
                        <v-chip v-else size="x-small" color="success" variant="tonal">正常</v-chip>
                      </td>
	                      <td>{{ km.successRate?.toFixed(0) }}%</td>
	                      <td>{{ km.requestCount }}</td>
	                      <td>{{ km.consecutiveFailures }}</td>
                        <td>
                          <div class="d-flex align-center ga-2">
                            <v-switch
                              :model-value="isAPIKeyEnabled(element, keyIndex)"
                              :disabled="isAPIKeyMetaUpdating(element.index, keyIndex) || !element.apiKeys?.[keyIndex]"
                              hide-details
                              density="compact"
                              color="success"
                              @update:model-value="setAPIKeyEnabled(element, keyIndex, $event)"
                            />
                            <span class="text-caption text-medium-emphasis">{{ isAPIKeyEnabled(element, keyIndex) ? '启用' : '禁用' }}</span>
                          </div>
                        </td>
	                      <td>
	                        <div class="d-flex align-center ga-1">
	                          <v-btn
	                            size="x-small"
	                            variant="text"
	                            @click.stop="openCircuitLog(element.index, keyIndex, km.keyMask)"
	                          >
                            查看失败日志
	                          </v-btn>
	                          <v-btn
	                            size="x-small"
	                            variant="text"
	                            color="warning"
	                            @click.stop="resetKeyStatus(element.index, keyIndex, km.keyMask)"
	                          >
	                            重置状态
	                          </v-btn>
	                          <v-btn
	                            size="x-small"
	                            variant="text"
	                            color="warning"
	                            @click.stop="resetKeyCircuit(element.index, keyIndex, km.keyMask)"
	                          >
	                            重置统计
	                          </v-btn>
	                        </div>
	                      </td>
	                    </tr>
                    <tr v-if="(getChannelMetrics(element.index)?.keyMetrics || []).length === 0">
                      <td colspan="9" class="text-caption text-medium-emphasis py-3">暂无 Key 指标</td>
                    </tr>
                  </tbody>
                </v-table>
	              </div>

		              <v-expand-transition>
		                <div v-if="statsExpandedChannelIndex === element.index">
		                  <v-divider class="my-3" />
		                  <KeyTrendChart
		                    :key="`chart-${channelType}-${element.index}`"
		                    :channel-id="element.index"
		                    :channel-type="channelType"
		                    @close="statsExpandedChannelIndex = null"
		                  />
		                </div>
		              </v-expand-transition>
		          </div>
	          </div>
	        </template>
	      </draggable>

      <!-- 空状态 -->
      <div v-if="activeChannels.length === 0" class="text-center py-6 text-medium-emphasis">
        <v-icon size="48" color="grey-lighten-1">mdi-playlist-remove</v-icon>
        <div class="mt-2">暂无活跃渠道</div>
        <div class="text-caption">从下方备用池启用渠道</div>
      </div>
    </div>

    <v-divider class="my-2" />

    <!-- 备用资源池 (disabled only) -->
    <div class="pt-2 pb-3">
      <div class="inactive-pool-header">
        <div class="text-subtitle-2 text-medium-emphasis d-flex align-center">
          <v-icon size="small" class="mr-1" color="grey">mdi-archive-outline</v-icon>
          备用资源池
          <v-chip size="x-small" class="ml-2">{{ inactiveChannels.length }}</v-chip>
        </div>
        <span class="text-caption text-medium-emphasis">启用后将追加到活跃序列末尾</span>
      </div>

      <div v-if="inactiveChannels.length > 0" class="inactive-pool">
        <div v-for="channel in inactiveChannels" :key="channel.index" class="inactive-channel-row">
          <!-- 渠道信息 -->
          <div class="channel-info">
            <div class="channel-info-main">
              <span
                class="font-weight-medium channel-name-link"
                tabindex="0"
                role="button"
                @click="$emit('edit', channel)"
                @keydown.enter="$emit('edit', channel)"
                @keydown.space.prevent="$emit('edit', channel)"
              >{{ channel.name }}</span>
              <span class="text-caption text-disabled ml-2">{{ channel.serviceType }}</span>
            </div>
            <div v-if="channel.description" class="channel-info-desc text-caption text-disabled">
              {{ channel.description }}
            </div>
          </div>

          <!-- API密钥数量 -->
          <div class="channel-keys">
            <v-chip size="x-small" variant="outlined" color="grey" class="keys-chip" @click="$emit('edit', channel)">
              <v-icon start size="x-small">mdi-key</v-icon>
              {{ channel.apiKeys?.length || 0 }}
            </v-chip>
          </div>

          <!-- 操作按钮 -->
          <div class="channel-actions">
            <v-btn size="small" color="success" variant="tonal" @click="enableChannel(channel.index)">
              <v-icon start size="small">mdi-play-circle</v-icon>
              启用
            </v-btn>

            <v-menu>
              <template #activator="{ props: activatorProps }">
                <v-btn icon size="x-small" variant="text" v-bind="activatorProps">
                  <v-icon size="small">mdi-dots-vertical</v-icon>
                </v-btn>
              </template>
              <v-list density="compact">
                <v-list-item @click="$emit('edit', channel)">
                  <template #prepend>
                    <v-icon size="small">mdi-pencil</v-icon>
                  </template>
                  <v-list-item-title>编辑</v-list-item-title>
                </v-list-item>
                <v-divider />
                <v-list-item @click="enableChannel(channel.index)">
                  <template #prepend>
                    <v-icon size="small" color="success">mdi-play-circle</v-icon>
                  </template>
                  <v-list-item-title>启用</v-list-item-title>
                </v-list-item>
                <v-list-item @click="$emit('delete', channel.index)">
                  <template #prepend>
                    <v-icon size="small" color="error">mdi-delete</v-icon>
                  </template>
                  <v-list-item-title>删除</v-list-item-title>
                </v-list-item>
              </v-list>
            </v-menu>
          </div>
        </div>
      </div>

      <div v-else class="text-center py-4 text-medium-emphasis text-caption">所有渠道都处于活跃状态</div>
    </div>
  </v-card>

  <CircuitLogModal
    v-model="isCircuitLogOpen"
    :title="circuitLogTitle"
    :log="circuitLog"
    :loading="isCircuitLogLoading"
    :error="circuitLogError"
  />
</template>

<script setup lang="ts">
	import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
	import draggable from 'vuedraggable'
	import { api, type Channel, type ChannelMetrics, type ChannelStatus, type TimeWindowStats, type ChannelRecentActivity } from '../services/api'
	import ChannelStatusBadge from './ChannelStatusBadge.vue'
	import KeyTrendChart from './KeyTrendChart.vue'
	import CircuitLogModal from './CircuitLogModal.vue'

const props = defineProps<{
  channels: Channel[]
  currentChannelIndex: number
  channelType: 'messages' | 'responses' | 'gemini'
  // 可选：从父组件传入的 metrics 和 stats（使用 dashboard 接口时）
  dashboardMetrics?: ChannelMetrics[]
  dashboardStats?: {
    multiChannelMode: boolean
    activeChannelCount: number
    traceAffinityCount: number
    traceAffinityTTL: string
    failureThreshold: number
    windowSize: number
    circuitRecoveryTime?: string
  }
  // 可选：从父组件传入的实时活跃度数据
  dashboardRecentActivity?: ChannelRecentActivity[]
}>()

const emit = defineEmits<{
  (_e: 'edit', _channel: Channel): void
  (_e: 'delete', _channelId: number): void
  (_e: 'ping', _channelId: number): void
  (_e: 'refresh'): void
  (_e: 'error', _message: string): void
  (_e: 'success', _message: string): void
}>()

// 状态
const metrics = ref<ChannelMetrics[]>([])
const recentActivity = ref<ChannelRecentActivity[]>([])
const schedulerStats = ref<{
  multiChannelMode: boolean
  activeChannelCount: number
  traceAffinityCount: number
  traceAffinityTTL: string
  failureThreshold: number
  windowSize: number
} | null>(null)
const isLoadingMetrics = ref(false)
	const isSavingOrder = ref(false)

// 熔断日志弹窗
const isCircuitLogOpen = ref(false)
const isCircuitLogLoading = ref(false)
const circuitLogTitle = ref('')
const circuitLog = ref('')
const circuitLogError = ref('')

// 延迟测试结果有效期（5 分钟）
const LATENCY_VALID_DURATION = 5 * 60 * 1000
// 用于触发响应式更新的时间戳
const currentTime = ref(Date.now())
let latencyCheckTimer: ReturnType<typeof setInterval> | null = null

// 用于触发活跃度视图更新的时间戳（每 2 秒更新）
const activityUpdateTick = ref(0)
let activityUpdateTimer: ReturnType<typeof setInterval> | null = null

// Key 元信息开关（启用/禁用）请求中
const apiKeyMetaUpdating = ref<Record<string, boolean>>({})

const isAPIKeyMetaUpdating = (channelId: number, keyIndex: number): boolean => {
  return apiKeyMetaUpdating.value[`${channelId}-${keyIndex}`] === true
}

const getAPIKeyDescription = (channel: Channel, keyIndex: number): string => {
  const apiKey = channel.apiKeys?.[keyIndex]
  if (!apiKey) return ''
  return channel.apiKeyMeta?.[apiKey]?.description || ''
}

const isAPIKeyEnabled = (channel: Channel, keyIndex: number): boolean => {
  const apiKey = channel.apiKeys?.[keyIndex]
  if (!apiKey) return false
  return channel.apiKeyMeta?.[apiKey]?.disabled !== true
}

const setAPIKeyEnabled = async (channel: Channel, keyIndex: number, enabled: boolean | null) => {
  const apiKey = channel.apiKeys?.[keyIndex]
  if (!apiKey) return
  if (enabled === null) return

  const key = `${channel.index}-${keyIndex}`
  if (apiKeyMetaUpdating.value[key]) return

  const prevMeta = channel.apiKeyMeta?.[apiKey] ? { ...channel.apiKeyMeta[apiKey] } : undefined
  apiKeyMetaUpdating.value[key] = true

  try {
    // optimistic update
    if (!channel.apiKeyMeta) channel.apiKeyMeta = {}

    if (!enabled) {
      channel.apiKeyMeta[apiKey] = { ...(channel.apiKeyMeta[apiKey] || {}), disabled: true }
    } else if (channel.apiKeyMeta[apiKey]) {
      const next = { ...channel.apiKeyMeta[apiKey] }
      delete next.disabled
      const desc = (next.description || '').trim()
      if (desc === '') delete next.description

      if (Object.keys(next).length === 0) {
        delete channel.apiKeyMeta[apiKey]
      } else {
        channel.apiKeyMeta[apiKey] = next
      }
    }

    if (channel.apiKeyMeta && Object.keys(channel.apiKeyMeta).length === 0) {
      channel.apiKeyMeta = undefined
    }

    await api.setAPIKeyDisabled(props.channelType, channel.index, keyIndex, !enabled)
  } catch (error) {
    // revert
    if (prevMeta) {
      if (!channel.apiKeyMeta) channel.apiKeyMeta = {}
      channel.apiKeyMeta[apiKey] = prevMeta
    } else if (channel.apiKeyMeta) {
      delete channel.apiKeyMeta[apiKey]
      if (Object.keys(channel.apiKeyMeta).length === 0) {
        channel.apiKeyMeta = undefined
      }
    }

    const errorMessage = error instanceof Error ? error.message : '未知错误'
    emit('error', `更新 Key 状态失败: ${errorMessage}`)
  } finally {
    apiKeyMetaUpdating.value[key] = false
  }
}

	// 统计图表展开状态（Key 状态始终显示；折叠仅作用于统计图表）
	const statsExpandedChannelIndex = ref<number | null>(null)

	const toggleChannelStats = (channelIndex: number) => {
	  statsExpandedChannelIndex.value = statsExpandedChannelIndex.value === channelIndex ? null : channelIndex
	}

		const getChannelDisplayStatus = (channel: Channel): ChannelStatus | 'healthy' | 'error' | 'unknown' => {
		  if (channel.status === 'disabled' || channel.status === 'suspended') return channel.status
		  const baseStatus: ChannelStatus = channel.status || 'active'
		  return channel.health ?? baseStatus
		}

// 活跃渠道（可拖拽排序）- 包含 active 和 suspended 状态
const activeChannels = ref<Channel[]>([])

// 计算属性：非活跃渠道 - 仅 disabled 状态
const inactiveChannels = computed(() => {
  return props.channels.filter(ch => ch.status === 'disabled')
})

// 计算属性：是否为多渠道模式
// 多渠道模式判断逻辑：
// 1. 只有一个启用的渠道 → 单渠道模式
// 2. 有一个 active + 几个 suspended → 单渠道模式
// 3. 有多个 active 渠道 → 多渠道模式
const isMultiChannelMode = computed(() => {
  const activeCount = props.channels.filter(
    ch => ch.status === 'active' || ch.status === undefined || ch.status === ''
  ).length
  return activeCount > 1
})

// 初始化活跃渠道列表 - active + suspended 都参与故障转移序列
// 优化：只在结构变化时更新，避免频繁重建导致子组件销毁
	const initActiveChannels = () => {
  const newActive = props.channels
    .filter(ch => ch.status !== 'disabled')
    .sort((a, b) => (a.priority ?? a.index) - (b.priority ?? b.index))

  // 检查是否需要更新：比较 index 列表是否变化
  const currentIndexes = activeChannels.value.map(ch => ch.index).join(',')
  const newIndexes = newActive.map(ch => ch.index).join(',')

	  if (currentIndexes !== newIndexes) {
	    // 结构变化（新增/删除/重排），需要重建数组
	    activeChannels.value = [...newActive]
	  } else {
    // 结构未变，只更新现有对象的属性（保持引用不变）
	    activeChannels.value.forEach((ch, i) => {
	      Object.assign(ch, newActive[i])
	    })
	  }

	  // Key 状态始终显示；不做默认展开统计图表
	}

// 监听 channels 变化
watch(() => props.channels, initActiveChannels, { immediate: true, deep: true })

// 监听 dashboard props 变化（从父组件传入的合并数据）
watch(() => props.dashboardMetrics, (newMetrics) => {
  if (newMetrics) {
    metrics.value = newMetrics
  }
}, { immediate: true })

watch(() => props.dashboardStats, (newStats) => {
  if (newStats) {
    schedulerStats.value = newStats
  }
}, { immediate: true })

// 监听 recentActivity props 变化
watch(() => props.dashboardRecentActivity, (newActivity) => {
  recentActivity.value = newActivity ?? []
}, { immediate: true })

		// 监听 channelType 变化 - 切换时刷新指标并收起图表
		watch(() => props.channelType, () => {
		  statsExpandedChannelIndex.value = null
		  // 如果没有使用 dashboard props，则自己刷新
		  if (!props.dashboardMetrics) {
	    refreshMetrics()
	  }
	})

// 获取渠道指标
const getChannelMetrics = (channelIndex: number): ChannelMetrics | undefined => {
  return metrics.value.find(m => m.channelIndex === channelIndex)
}

// 获取分时段统计的辅助方法
const get15mStats = (channelIndex: number) => {
  return getChannelMetrics(channelIndex)?.timeWindows?.['15m']
}

const get1hStats = (channelIndex: number) => {
  return getChannelMetrics(channelIndex)?.timeWindows?.['1h']
}

const get6hStats = (channelIndex: number) => {
  return getChannelMetrics(channelIndex)?.timeWindows?.['6h']
}

const get24hStats = (channelIndex: number) => {
  return getChannelMetrics(channelIndex)?.timeWindows?.['24h']
}

// 获取成功率颜色
const getSuccessRateColor = (rate?: number): string => {
  if (rate === undefined) return 'grey'
  if (rate >= 90) return 'success'
  if (rate >= 70) return 'warning'
  return 'error'
}

const getCacheHitRateColor = (rate?: number): string => {
  if (rate === undefined) return 'grey'
  if (rate >= 50) return 'success'
  if (rate >= 20) return 'info'
  if (rate >= 5) return 'warning'
  return 'orange'
}

const shouldShowCacheHitRate = (stats?: TimeWindowStats): boolean => {
  if (!stats || !stats.requestCount) return false
  const inputTokens = stats.inputTokens ?? 0
  const cacheReadTokens = stats.cacheReadTokens ?? 0
  return (inputTokens + cacheReadTokens) > 0
}

// 获取延迟颜色
const getLatencyColor = (latency: number): string => {
  if (latency < 500) return 'success'
  if (latency < 1000) return 'warning'
  return 'error'
}

// 判断延迟测试结果是否仍然有效（5 分钟内）
const isLatencyValid = (channel: Channel): boolean => {
  // 没有延迟值，不显示
  if (channel.latency === undefined || channel.latency === null) return false
  // 没有测试时间戳（兼容旧数据），不显示
  if (!channel.latencyTestTime) return false
  // 检查是否在有效期内（使用 currentTime.value 触发响应式更新）
  return (currentTime.value - channel.latencyTestTime) < LATENCY_VALID_DURATION
}

// 判断渠道是否处于促销期
const isInPromotion = (channel: Channel): boolean => {
  if (!channel.promotionUntil) return false
  return new Date(channel.promotionUntil) > new Date()
}

// 格式化促销期剩余时间
const formatPromotionRemaining = (until?: string): string => {
  if (!until) return ''
  const remaining = Math.max(0, new Date(until).getTime() - Date.now())
  const minutes = Math.ceil(remaining / 60000)
  if (minutes <= 0) return '即将结束'
  return `${minutes}分钟`
}

// 格式化统计数据：有请求显示"N 请求 (X%)"，无请求显示"--"
const formatStats = (stats?: TimeWindowStats): string => {
  if (!stats || !stats.requestCount) return '--'
  return `${stats.requestCount} 请求 (${stats.successRate?.toFixed(0)}%)`
}

const formatTokens = (num?: number): string => {
  const value = num ?? 0
  if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return Math.round(value).toString()
}

const formatCacheStats = (stats?: TimeWindowStats): string => {
  if (!stats || !stats.requestCount) return '--'

  const inputTokens = stats.inputTokens ?? 0
  const cacheReadTokens = stats.cacheReadTokens ?? 0
  const cacheCreationTokens = stats.cacheCreationTokens ?? 0
  const denom = inputTokens + cacheReadTokens

  if (denom <= 0) return '--'

  const hitRate = stats.cacheHitRate ?? (cacheReadTokens / denom * 100)
  return `命中 ${hitRate.toFixed(0)}% · 读 ${formatTokens(cacheReadTokens)} · 写 ${formatTokens(cacheCreationTokens)}`
}

// 获取官网 URL（优先使用 website，否则从 baseUrl 提取域名）
const getWebsiteUrl = (channel: Channel): string => {
  if (channel.website) return channel.website
  try {
    const url = new URL(channel.baseUrl)
    return `${url.protocol}//${url.host}`
  } catch {
    return channel.baseUrl
  }
}

// ============== 渠道实时活跃度相关函数 ==============

// 活跃度数据 Map 缓存（避免线性查找）
const activityMap = computed(() => {
  const map = new Map<number, ChannelRecentActivity>()
  for (const a of recentActivity.value) {
    map.set(a.channelIndex, a)
  }
  return map
})

// 每个渠道的历史最大请求数（用于固定柱状图高度比例）
const maxRequestsHistory = ref(new Map<number, number>())

// 更新历史最大值
watch(activityMap, (newMap) => {
  for (const [channelIndex, activity] of newMap.entries()) {
    if (!activity.segments || activity.segments.length === 0) continue

    const currentMax = Math.max(...activity.segments.map(s => s.requestCount), 0)
    const historicalMax = maxRequestsHistory.value.get(channelIndex) ?? 0

    // 只在当前最大值更大时更新（保持历史峰值）
    if (currentMax > historicalMax) {
      maxRequestsHistory.value.set(channelIndex, currentMax)
    }
  }
})

// 获取渠道的活跃度数据
const getChannelActivity = (channelIndex: number): ChannelRecentActivity | undefined => {
  return activityMap.value.get(channelIndex)
}

// 缓存所有渠道的柱状图数据（避免在模板中重复计算）
const activityBarsCache = computed(() => {
  const cache = new Map<number, Array<{ x: number; y: number; width: number; height: number; radius: number; color: string }>>()

  // 使用 activityUpdateTick 触发响应式更新
  const _ = activityUpdateTick.value

  for (const [channelIndex, activity] of activityMap.value.entries()) {
    if (!activity || !activity.segments || activity.segments.length === 0) {
      cache.set(channelIndex, [])
      continue
    }

    const segments = activity.segments
    const numSegments = segments.length // 150（后端已聚合为每 6 秒一段）

    // 每个段一个柱子
    const barWidth = 150 / numSegments
    const barGap = barWidth * 0.2 // 20% 间隙
    const actualBarWidth = barWidth - barGap

    // 使用历史最大值作为归一化基准（避免高流量段离开后柱子突然变高）
    const maxRequests = maxRequestsHistory.value.get(channelIndex) ?? Math.max(...segments.map(s => s.requestCount), 1)

    const bars: Array<{ x: number; y: number; width: number; height: number; radius: number; color: string }> = []

    for (let i = 0; i < numSegments; i++) {
      const segment = segments[i]
      const requests = segment.requestCount

      // 计算柱子高度（最小高度 2，避免完全消失）
      const heightPercent = requests / maxRequests
      const height = Math.max(heightPercent * 85, requests > 0 ? 2 : 0)
      const y = 100 - height

      // 根据该 6 秒段的成功率计算颜色（7 档分级：极端档位 + 整数档位）
      let color = 'rgb(74, 222, 128)' // 默认绿色（无请求或 100% 成功）

      if (requests > 0) {
        const successCount = requests - segment.failureCount
        const successRate = (successCount / requests) * 100

        if (successRate < 5) {
          color = 'rgb(220, 38, 38)' // 0-5%：深红色（极端故障）
        } else if (successRate < 20) {
          color = 'rgb(239, 68, 68)' // 5-20%：红色（严重失败）
        } else if (successRate < 40) {
          color = 'rgb(249, 115, 22)' // 20-40%：深橙色（高失败率）
        } else if (successRate < 60) {
          color = 'rgb(251, 146, 60)' // 40-60%：橙色（中等失败率）
        } else if (successRate < 80) {
          color = 'rgb(250, 204, 21)' // 60-80%：黄色（轻微失败）
        } else if (successRate < 95) {
          color = 'rgb(132, 204, 22)' // 80-95%：黄绿色（良好）
        } else {
          color = 'rgb(74, 222, 128)' // 95-100%：绿色（优秀）
        }
      }

      const x = i * barWidth + barGap / 2
      const radius = actualBarWidth * 0.4

      bars.push({
        x,
        y,
        width: actualBarWidth,
        height,
        radius,
        color
      })
    }

    cache.set(channelIndex, bars)
  }

  return cache
})

const getActivityBars = (channelIndex: number): Array<{ x: number; y: number; width: number; height: number; radius: number; color: string }> => {
  return activityBarsCache.value.get(channelIndex) ?? []
}

const hasActivityData = (channelIndex: number): boolean => {
  const activity = getChannelActivity(channelIndex)
  return !!activity && !!activity.segments && activity.segments.some(s => s.requestCount > 0)
}

const getActivityTotals = (channelIndex: number): { rpm: number; tpm: number } => {
  const activity = getChannelActivity(channelIndex)
  if (!activity) return { rpm: 0, tpm: 0 }
  return { rpm: activity.rpm ?? 0, tpm: activity.tpm ?? 0 }
}

const formatRPM = (channelIndex: number): string => {
  const { rpm } = getActivityTotals(channelIndex)
  if (!rpm || rpm <= 0) return '--'
  if (rpm >= 1000) return `${(rpm / 1000).toFixed(1)}k`
  return Math.round(rpm).toString()
}

const formatTPM = (channelIndex: number): string => {
  const { tpm } = getActivityTotals(channelIndex)
  if (!tpm || tpm <= 0) return '--'
  if (tpm >= 1000000) return `${(tpm / 1000000).toFixed(1)}M`
  if (tpm >= 1000) return `${(tpm / 1000).toFixed(1)}K`
  return Math.round(tpm).toString()
}

// 刷新指标
const refreshMetrics = async () => {
  isLoadingMetrics.value = true
  try {
    const [metricsData, statsData] = await Promise.all([
      props.channelType === 'gemini'
        ? api.getGeminiChannelMetrics()
        : props.channelType === 'responses'
          ? api.getResponsesChannelMetrics()
          : api.getChannelMetrics(),
      api.getSchedulerStats(props.channelType)
    ])
    metrics.value = metricsData
    schedulerStats.value = statsData
  } catch (error) {
    console.error('Failed to load metrics:', error)
  } finally {
    isLoadingMetrics.value = false
  }
}

const openCircuitLog = async (channelId: number, keyIndex: number, keyMask: string) => {
  isCircuitLogOpen.value = true
  isCircuitLogLoading.value = true
  circuitLogTitle.value = `渠道 ${channelId + 1} · ${keyMask} · 最后失败日志`
  circuitLog.value = ''
  circuitLogError.value = ''

  try {
    const resp = await api.getKeyCircuitLog(props.channelType, channelId, keyIndex)
    circuitLog.value = resp.log
  } catch (e: any) {
    circuitLogError.value = e?.message || '获取熔断日志失败'
  } finally {
    isCircuitLogLoading.value = false
  }
}

	const resetKeyCircuit = async (channelId: number, keyIndex: number, keyMask: string) => {
	  if (!confirm(`确认重置 ${keyMask} 的统计数据？此操作会清空该 Key 的请求/成功/失败等累计统计。`)) return
	  try {
	    await api.resetKeyCircuit(props.channelType, channelId, keyIndex)
	    emit('success', `已重置 ${keyMask}`)
	    await refreshMetrics()
	  } catch (e: any) {
	    emit('error', e?.message || '重置失败')
	  }
	}

	const resetKeyStatus = async (channelId: number, keyIndex: number, keyMask: string) => {
	  if (!confirm(`确认重置 ${keyMask} 的状态？此操作会清除熔断/冷却状态，但保留累计统计。`)) return
	  try {
	    await api.resetKeyStatus(props.channelType, channelId, keyIndex)
	    emit('success', `已重置状态 ${keyMask}`)
	    await refreshMetrics()
	  } catch (e: any) {
	    emit('error', e?.message || '重置状态失败')
	  }
	}

// 拖拽变更事件 - 自动保存顺序
const onDragChange = () => {
  // 拖拽后自动保存顺序到后端
  saveOrder()
}

// 保存顺序
const saveOrder = async () => {
  isSavingOrder.value = true
  try {
    const order = activeChannels.value.map(ch => ch.index)
    if (props.channelType === 'gemini') {
      await api.reorderGeminiChannels(order)
    } else if (props.channelType === 'responses') {
      await api.reorderResponsesChannels(order)
    } else {
      await api.reorderChannels(order)
    }
    // 不调用 emit('refresh')，避免触发父组件刷新导致列表闪烁
  } catch (error) {
    console.error('Failed to save order:', error)
    const errorMessage = error instanceof Error ? error.message : '未知错误'
    emit('error', `保存渠道顺序失败: ${errorMessage}`)
    // 保存失败时重新初始化列表，恢复原始顺序
    initActiveChannels()
  } finally {
    isSavingOrder.value = false
  }
}

// 设置渠道状态
const setChannelStatus = async (channelId: number, status: ChannelStatus) => {
  try {
    if (props.channelType === 'gemini') {
      await api.setGeminiChannelStatus(channelId, status)
    } else if (props.channelType === 'responses') {
      await api.setResponsesChannelStatus(channelId, status)
    } else {
      await api.setChannelStatus(channelId, status)
    }
    emit('refresh')
  } catch (error) {
    console.error('Failed to set channel status:', error)
    const errorMessage = error instanceof Error ? error.message : '未知错误'
    emit('error', `设置渠道状态失败: ${errorMessage}`)
  }
}

// 启用渠道（从备用池移到活跃序列）
const enableChannel = async (channelId: number) => {
  await setChannelStatus(channelId, 'active')
}

// 恢复渠道（重置指标并设为 active）
const resumeChannel = async (channelId: number) => {
  try {
    if (props.channelType === 'gemini') {
      await api.resumeGeminiChannel(channelId)
    } else if (props.channelType === 'responses') {
      await api.resumeResponsesChannel(channelId)
    } else {
      await api.resumeChannel(channelId)
    }
    await setChannelStatus(channelId, 'active')
  } catch (error) {
    console.error('Failed to resume channel:', error)
  }
}

// 设置渠道促销期（抢优先级）
const setPromotion = async (channel: Channel) => {
  try {
    const PROMOTION_DURATION = 300 // 5分钟

    // 如果渠道是熔断状态，先恢复它
    if (channel.status === 'suspended') {
      if (props.channelType === 'gemini') {
        await api.resumeGeminiChannel(channel.index)
      } else if (props.channelType === 'responses') {
        await api.resumeResponsesChannel(channel.index)
      } else {
        await api.resumeChannel(channel.index)
      }
      await setChannelStatus(channel.index, 'active')
    }

    if (props.channelType === 'gemini') {
      await api.setGeminiChannelPromotion(channel.index, PROMOTION_DURATION)
    } else if (props.channelType === 'responses') {
      await api.setResponsesChannelPromotion(channel.index, PROMOTION_DURATION)
    } else {
      await api.setChannelPromotion(channel.index, PROMOTION_DURATION)
    }
    emit('refresh')
    // 通知用户
    emit('success', `渠道 ${channel.name} 已设为最高优先级，5分钟内优先使用`)
  } catch (error) {
    console.error('Failed to set promotion:', error)
    const errorMessage = error instanceof Error ? error.message : '未知错误'
    emit('error', `设置优先级失败: ${errorMessage}`)
  }
}

// 判断渠道是否可以删除
// 规则：故障转移序列中至少要保留一个 active 状态的渠道
const canDeleteChannel = (channel: Channel): boolean => {
  // 统计当前 active 状态的渠道数量
  const activeCount = activeChannels.value.filter(
    ch => ch.status === 'active' || ch.status === undefined || ch.status === ''
  ).length

  // 如果要删除的是 active 渠道，且只剩一个 active，则不允许删除
  const isActive = channel.status === 'active' || channel.status === undefined || channel.status === ''
  if (isActive && activeCount <= 1) {
    return false
  }

  return true
}

// 处理删除渠道
const handleDeleteChannel = (channel: Channel) => {
  if (!canDeleteChannel(channel)) {
    emit('error', '无法删除：故障转移序列中至少需要保留一个活跃渠道')
    return
  }
  emit('delete', channel.index)
}

// 组件挂载时加载指标并启动延迟过期检查定时器
onMounted(() => {
  if (!props.dashboardMetrics) {
    refreshMetrics()
  }
  // 每 30 秒更新一次 currentTime，触发延迟显示的响应式更新
  latencyCheckTimer = setInterval(() => {
    currentTime.value = Date.now()
  }, 30000)
  // 每 2 秒更新一次 activityUpdateTick，触发活跃度视图更新
  activityUpdateTimer = setInterval(() => {
    activityUpdateTick.value++
  }, 2000)
})

// 组件卸载时清理定时器
onUnmounted(() => {
  if (latencyCheckTimer) {
    clearInterval(latencyCheckTimer)
    latencyCheckTimer = null
  }
  if (activityUpdateTimer) {
    clearInterval(activityUpdateTimer)
    activityUpdateTimer = null
  }
})

// 暴露方法给父组件
defineExpose({
  refreshMetrics
})
</script>

<style scoped>
/* =====================================================
   🎮 渠道编排 - 复古像素主题样式
   Neo-Brutalism: 直角、粗黑边框、硬阴影
   ===================================================== */

.channel-orchestration {
  overflow: hidden;
  background: transparent;
  border: none;
}

.channel-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.channel-item-wrapper {
  display: flex;
  flex-direction: column;
}

.channel-row {
  position: relative;
  padding: 10px 12px;
  margin: 2px;
  background: rgb(var(--v-theme-surface));
  border: 2px solid rgb(var(--v-theme-on-surface));
  box-shadow: 4px 4px 0 0 rgb(var(--v-theme-on-surface));
  min-height: 52px;
  transition: all 0.1s ease;
  cursor: pointer;
  overflow: hidden;
}

/* Grid 内容容器 */
.channel-row-content {
  display: grid;
  grid-template-columns: 28px 28px 90px minmax(120px, 1fr) auto 50px 50px 50px auto;
  align-items: center;
  gap: 6px;
  position: relative;
  z-index: 1;
}

/* SVG 活跃度波形柱状图背景 */
.activity-chart-bg {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}

/* 柱状图无动画：避免数据更新时的缩小-增长抖动效果 */
.activity-bar {
  transition: none;
}

/* 图表展开区域 */
.channel-chart-wrapper {
  margin: 0 2px 8px 2px;
}

.channel-row:hover {
  background: rgba(var(--v-theme-primary), 0.08);
  transform: translate(-2px, -2px);
  box-shadow: 6px 6px 0 0 rgb(var(--v-theme-on-surface));
  border: 2px solid rgb(var(--v-theme-on-surface));
}

.channel-row:active {
  transform: translate(2px, 2px);
  box-shadow: none;
}

.v-theme--dark .channel-row {
  background: rgb(var(--v-theme-surface));
  border-color: rgba(255, 255, 255, 0.7);
  box-shadow: 4px 4px 0 0 rgba(255, 255, 255, 0.7);
}
.v-theme--dark .channel-row:hover {
  background: rgba(var(--v-theme-primary), 0.12);
  box-shadow: 6px 6px 0 0 rgba(255, 255, 255, 0.7);
  border-color: rgba(255, 255, 255, 0.7);
}

/* suspended 状态的视觉区分 */
.channel-row.is-suspended {
  background: rgba(var(--v-theme-warning), 0.1);
  border-color: rgb(var(--v-theme-warning));
  box-shadow: 4px 4px 0 0 rgb(var(--v-theme-on-surface));
}
.channel-row.is-suspended:hover {
  background: rgba(var(--v-theme-warning), 0.15);
  box-shadow: 6px 6px 0 0 rgb(var(--v-theme-on-surface));
}

.v-theme--dark .channel-row.is-suspended {
  box-shadow: 4px 4px 0 0 rgba(255, 255, 255, 0.7);
}

.v-theme--dark .channel-row.is-suspended:hover {
  box-shadow: 6px 6px 0 0 rgba(255, 255, 255, 0.7);
}

.channel-row.ghost {
  opacity: 0.6;
  background: rgba(var(--v-theme-primary), 0.15);
  border: 2px dashed rgb(var(--v-theme-primary));
  box-shadow: none;
}

.drag-handle {
  cursor: grab;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  transition: all 0.1s ease;
}

.drag-handle:hover {
  background: rgba(var(--v-theme-on-surface), 0.1);
}

.drag-handle:active {
  cursor: grabbing;
  background: rgba(var(--v-theme-primary), 0.2);
}

.priority-number {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: rgb(var(--v-theme-primary));
  color: white;
  font-size: 12px;
  font-weight: 700;
  border: 2px solid rgb(var(--v-theme-on-surface));
  text-transform: uppercase;
}

.v-theme--dark .priority-number {
  border-color: rgba(255, 255, 255, 0.6);
}

.channel-name {
  display: flex;
  align-items: center;
  overflow: hidden;
}

.channel-name .expand-icon {
  flex-shrink: 0;
}

.channel-name .font-weight-medium {
  font-size: 0.95rem;
  flex-shrink: 0;
}

/* 描述文本限制最多两行 */
.channel-description {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
  max-height: calc(1.4em * 2);
  word-break: break-word;
}

.channel-name-link {
  cursor: pointer;
  transition: all 0.15s ease;
}

.channel-name-link:hover,
.channel-name-link:focus {
  color: rgb(var(--v-theme-primary));
  text-decoration: underline;
  outline: none;
}

.channel-name-link:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
  border-radius: 2px;
}

.channel-metrics {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: nowrap;
  white-space: nowrap;
}

.channel-latency {
  display: flex;
  align-items: center;
  min-width: 60px;
}

/* RPM/TPM 显示样式 */
.channel-rpm-tpm {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 60px;
  margin-left: 8px;
}

.rpm-tpm-values {
  display: flex;
  align-items: baseline;
  gap: 2px;
  font-size: 13px;
  font-weight: 600;
  color: rgba(var(--v-theme-on-surface), 0.6);
}

.rpm-tpm-values .rpm-value.has-data,
.rpm-tpm-values .tpm-value.has-data {
  color: rgb(var(--v-theme-primary));
}

.rpm-tpm-separator {
  color: rgba(var(--v-theme-on-surface), 0.3);
  font-weight: 400;
}

.rpm-tpm-labels {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: 9px;
  color: rgba(var(--v-theme-on-surface), 0.5);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.channel-keys {
  display: flex;
  align-items: center;
}

.channel-keys .keys-chip {
  cursor: pointer;
  transition: all 0.1s ease;
}

.channel-keys .keys-chip:hover {
  background: rgba(var(--v-theme-primary), 0.1);
  border-color: rgb(var(--v-theme-primary));
  color: rgb(var(--v-theme-primary));
}

.channel-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  justify-content: flex-end;
  min-width: 50px;
}

/* 备用资源池样式 */
.inactive-pool-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.inactive-pool {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
  background: rgb(var(--v-theme-surface));
  padding: 16px;
  border: 2px dashed rgb(var(--v-theme-on-surface));
}

.v-theme--dark .inactive-pool {
  background: rgb(var(--v-theme-surface));
  border-color: rgba(255, 255, 255, 0.5);
}

.inactive-channel-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  background: rgb(var(--v-theme-surface));
  border: 2px solid rgb(var(--v-theme-on-surface));
  box-shadow: 3px 3px 0 0 rgb(var(--v-theme-on-surface));
  transition: all 0.1s ease;
}

.inactive-channel-row:hover {
  background: rgba(var(--v-theme-primary), 0.08);
  transform: translate(-1px, -1px);
  box-shadow: 4px 4px 0 0 rgb(var(--v-theme-on-surface));
}

.inactive-channel-row:active {
  transform: translate(2px, 2px);
  box-shadow: none;
}

.v-theme--dark .inactive-channel-row {
  background: rgb(var(--v-theme-surface));
  border-color: rgba(255, 255, 255, 0.6);
  box-shadow: 3px 3px 0 0 rgba(255, 255, 255, 0.6);
}

.v-theme--dark .inactive-channel-row:hover {
  background: rgba(var(--v-theme-primary), 0.12);
  box-shadow: 4px 4px 0 0 rgba(255, 255, 255, 0.6);
}

.inactive-channel-row .channel-info {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.inactive-channel-row .channel-info-main {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inactive-channel-row .channel-info-desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.3;
  max-width: 100%;
}

.inactive-channel-row .channel-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 响应式调整 */
@media (max-width: 1400px) {
  .channel-row-content {
    grid-template-columns: 28px 28px 85px minmax(100px, 1fr) auto 45px 45px 45px auto;
    gap: 5px;
  }
  .channel-row {
    padding: 10px 10px;
  }
}

@media (max-width: 1200px) {
  .channel-row-content {
    grid-template-columns: 26px 26px 80px minmax(80px, 1fr) auto 40px 40px 40px auto;
    gap: 4px;
  }
  .channel-row {
    padding: 8px 8px;
  }

  .rpm-tpm-values {
    font-size: 11px;
  }

  .rpm-tpm-labels {
    font-size: 8px;
  }
}

@media (max-width: 960px) {
  .channel-row-content {
    grid-template-columns: 26px 26px 75px minmax(60px, 1fr) auto 38px 38px 38px auto;
    gap: 4px;
  }
  .channel-row {
    padding: 8px 6px;
  }
}

@media (max-width: 600px) {
  .channel-row-content {
    grid-template-columns: 28px 1fr 60px;
    gap: 8px;
  }
  .channel-row {
    padding: 10px;
    box-shadow: 3px 3px 0 0 rgb(var(--v-theme-on-surface));
  }

  .channel-metrics,
  .channel-latency,
  .channel-keys,
  .channel-rpm-tpm {
    display: none;
  }

  .v-theme--dark .channel-row {
    box-shadow: 3px 3px 0 0 rgba(255, 255, 255, 0.6);
  }

  .priority-number,
  .drag-handle {
    display: none;
  }
}

/* 指标显示样式 */
.metrics-display {
  cursor: help;
}

/* 指标 tooltip 样式 */
.metrics-tooltip {
  font-size: 12px;
  line-height: 1.5;
  color: rgb(var(--v-theme-on-surface));
}

.metrics-tooltip-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 2px 0;
}

.metrics-tooltip-row span:first-child {
  color: rgba(var(--v-theme-on-surface), 0.7);
}

.metrics-tooltip-row span:last-child {
  font-weight: 500;
  color: rgb(var(--v-theme-on-surface));
}

.key-metrics-panel {
  background: rgba(var(--v-theme-surface-variant), 0.35);
  border-radius: 12px;
  padding: 12px;
}

.key-metrics-table :deep(th),
.key-metrics-table :deep(td) {
  font-size: 14px;
}
</style>
