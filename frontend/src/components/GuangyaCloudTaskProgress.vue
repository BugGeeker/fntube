<template>
  <transition-group name="cloud-task" tag="div" class="cloud-task-stack">
    <div v-for="task in visibleTasks" :key="task.key" class="cloud-task-panel">
      <div class="task-header">
        <component :is="task.icon" class="task-icon" :spin="task.state === 'active'" />
        <div class="task-title">
          <strong>{{ task.title }}</strong>
          <span>{{ task.step }}</span>
        </div>
        <a-button type="text" size="small" class="close-button" aria-label="关闭"
          @click="dismiss(task.key)">
          <template #icon><CloseOutlined /></template>
        </a-button>
      </div>
      <a-progress :percent="task.progress" size="small"
        :status="task.state === 'failed' ? 'exception' : task.state === 'completed' ? 'success' : 'active'" />
      <div v-if="task.error" class="task-error">{{ task.error }}</div>
    </div>
  </transition-group>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type Component } from 'vue'
import { CloseOutlined, CloudSyncOutlined, SyncOutlined } from '@ant-design/icons-vue'
import { theme } from 'ant-design-vue'
import {
  guangyaCloudAddEventsURL,
  type GuangyaCloudAddJob,
} from '@/api/guangya'
import {
  scrapeLogEventsURL,
  type ScrapeLog,
} from '@/api/scrapelog'

interface ProgressTask {
  key: string
  title: string
  step: string
  progress: number
  state: 'active' | 'completed' | 'failed'
  error: string
  createdAt: string
  icon: Component
}

const cloudJobs = ref(new Map<number, GuangyaCloudAddJob>())
const scrapeLogs = ref(new Map<number, ScrapeLog>())
const dismissed = new Set<string>()
const removalTimers = new Map<string, ReturnType<typeof setTimeout>>()
let cloudEventSource: EventSource | undefined
let scrapeEventSource: EventSource | undefined
const { token } = theme.useToken()

const visibleTasks = computed<ProgressTask[]>(() =>
  [
    ...Array.from(cloudJobs.value.values()).map(toCloudTask),
    ...Array.from(scrapeLogs.value.values()).map(toScrapeTask),
  ]
    .filter((task) => !dismissed.has(task.key))
    .sort((left, right) => Date.parse(right.createdAt) - Date.parse(left.createdAt))
    .slice(0, 4)
)

function toCloudTask(job: GuangyaCloudAddJob): ProgressTask {
  return {
    key: `cloud-${job.id}`,
    title: job.requested_name,
    step: job.current_step || '处理中',
    progress: job.progress,
    state: job.status === 'failed' ? 'failed' : job.status === 'completed' ? 'completed' : 'active',
    error: job.error,
    createdAt: job.created_at,
    icon: CloudSyncOutlined,
  }
}

function toScrapeTask(log: ScrapeLog): ProgressTask {
  return {
    key: `scrape-${log.id}`,
    title: log.number || log.title || '手动刮削',
    step: log.current_step || '准备刮削',
    progress: log.progress,
    state: log.status === 'failed'
      ? 'failed'
      : ['success', 'completed'].includes(log.status) ? 'completed' : 'active',
    error: log.error,
    createdAt: log.created_at,
    icon: SyncOutlined,
  }
}

function scheduleRemoval(task: ProgressTask) {
  const currentTimer = removalTimers.get(task.key)
  if (currentTimer) clearTimeout(currentTimer)
  if (task.state !== 'active') {
    removalTimers.set(task.key, setTimeout(() => dismiss(task.key), 10000))
  }
}

function receiveCloudJob(event: MessageEvent<string>) {
  try {
    const job = JSON.parse(event.data) as GuangyaCloudAddJob
    const key = `cloud-${job.id}`
    dismissed.delete(key)
    const next = new Map(cloudJobs.value)
    next.set(job.id, job)
    cloudJobs.value = next
    scheduleRemoval(toCloudTask(job))
  } catch {
    // Ignore malformed events and let EventSource continue receiving updates.
  }
}

function receiveScrapeLog(event: MessageEvent<string>) {
  try {
    const log = JSON.parse(event.data) as ScrapeLog
    if (log.method !== 'manual') return
    const key = `scrape-${log.id}`
    dismissed.delete(key)
    const next = new Map(scrapeLogs.value)
    next.set(log.id, log)
    scrapeLogs.value = next
    scheduleRemoval(toScrapeTask(log))
  } catch {
    // Ignore malformed events and let EventSource continue receiving updates.
  }
}

function dismiss(key: string) {
  dismissed.add(key)
  if (key.startsWith('cloud-')) {
    const next = new Map(cloudJobs.value)
    next.delete(Number(key.slice(6)))
    cloudJobs.value = next
  } else {
    const next = new Map(scrapeLogs.value)
    next.delete(Number(key.slice(7)))
    scrapeLogs.value = next
  }
  const timer = removalTimers.get(key)
  if (timer) clearTimeout(timer)
  removalTimers.delete(key)
}

function receiveCloudDeleted(event: MessageEvent<string>) {
  try {
    const { id } = JSON.parse(event.data) as { id: number }
    dismiss(`cloud-${id}`)
  } catch {
    // Ignore malformed events and let EventSource continue receiving updates.
  }
}

function receiveScrapeDeleted(event: MessageEvent<string>) {
  try {
    const { id } = JSON.parse(event.data) as { id: number }
    dismiss(`scrape-${id}`)
  } catch {
    // Ignore malformed events and let EventSource continue receiving updates.
  }
}

onMounted(() => {
  cloudEventSource = new EventSource(guangyaCloudAddEventsURL)
  cloudEventSource.addEventListener('cloud-add-progress', receiveCloudJob as EventListener)
  cloudEventSource.addEventListener('cloud-add-deleted', receiveCloudDeleted as EventListener)
  scrapeEventSource = new EventSource(scrapeLogEventsURL)
  scrapeEventSource.addEventListener('scrape-progress', receiveScrapeLog as EventListener)
  scrapeEventSource.addEventListener('scrape-deleted', receiveScrapeDeleted as EventListener)
})

onBeforeUnmount(() => {
  cloudEventSource?.close()
  scrapeEventSource?.close()
  removalTimers.forEach((timer) => clearTimeout(timer))
})
</script>

<style scoped lang="scss">
.cloud-task-stack {
  position: fixed;
  z-index: 1200;
  top: 20px;
  right: 20px;
  display: flex;
  width: min(360px, calc(100vw - 32px));
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
}

.cloud-task-panel {
  padding: 12px 14px;
  border: 1px solid v-bind('token.colorBorderSecondary');
  border-radius: 8px;
  background: v-bind('token.colorBgElevated');
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.16);
  pointer-events: auto;
}

.task-header {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 10px;
}

.task-icon {
  margin-top: 3px;
  color: v-bind('token.colorPrimary');
  font-size: 18px;
}

.task-title {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.task-title strong,
.task-title span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-title span {
  margin-top: 2px;
  color: v-bind('token.colorTextSecondary');
  font-size: 12px;
}

.close-button {
  flex: 0 0 auto;
  margin: -5px -7px 0 0;
}

.task-error {
  max-height: 42px;
  margin-top: 4px;
  overflow: hidden;
  color: v-bind('token.colorError');
  font-size: 12px;
  line-height: 21px;
}

.cloud-task-enter-active,
.cloud-task-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.cloud-task-enter-from,
.cloud-task-leave-to {
  opacity: 0;
  transform: translateX(24px);
}

@media (max-width: 639px) {
  .cloud-task-stack {
    top: 12px;
    right: 12px;
    width: calc(100vw - 24px);
  }
}
</style>
