<template>
  <div>
    <a-card title="刮削记录">
      <template #extra>
        <a-space>
          <a-input-search v-model:value="numberQuery" placeholder="按番号查询" allow-clear :enter-button="true"
            class="header-search" @search="handleSearch" />
          <a-button :loading="loading" @click="loadLogs()">
            <template #icon>
              <ReloadOutlined />
            </template>
          </a-button>
        </a-space>
      </template>
      <a-spin :spinning="loading">
        <div class="desktop-table">
          <a-table :dataSource="logs" :columns="columns" rowKey="id" :pagination="pagination" @change="handleTableChange"
            table-layout="fixed" :scroll="{ x: '100%' }">
            <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'title'">
              <a @click="showDetail(record)">{{ record.title }}</a>
            </template>
            <template v-if="column.key === 'method'">
              <a-tag v-if="record.method === 'auto'" color="green">自动</a-tag>
              <a-tag v-else color="blue">手动</a-tag>
            </template>
            <template v-if="column.key === 'status'">
              <a-tooltip placement="topLeft" v-if="parseSteps(record.steps).length > 0">
                <template #title>
                  <div v-for="(s, i) in parseSteps(record.steps)" :key="i" style="margin-bottom: 2px">
                    <span>{{ stepLabel(s.step) }}:</span>
                    <a-tag v-if="s.status === 'running'" color="processing" style="margin-left: 4px">进行中</a-tag>
                    <a-tag v-else-if="s.status === 'success'" color="green" style="margin-left: 4px">成功</a-tag>
                    <a-tag v-else-if="s.status === 'failed'" color="red" style="margin-left: 4px">失败</a-tag>
                    <span v-if="s.error" style="color: #ff4d4f; margin-left: 4px">{{ s.error }}</span>
                  </div>
                </template>
                <a-badge v-if="record.status === 'in_progress'" :status="stepsBadgeStatus(record.steps)"
                  :text="stepsSummary(record.steps)" />
                <a-tag v-else-if="record.status === 'success'" color="green">成功</a-tag>
                <a-tag v-else-if="record.status === 'failed'" color="red">失败</a-tag>
                <a-tag v-else-if="record.status === 'completed'" color="blue">完成</a-tag>
                <a-tag v-else color="default">{{ record.status }}</a-tag>
              </a-tooltip>
              <span v-else>-</span>
            </template>
            <template v-if="column.key === 'steps'">

            </template>
            <template v-if="column.key === 'error'">
              <a-tooltip v-if="record.error" :title="record.error">
                <span style="color: #ff4d4f; cursor: pointer">{{ truncate(record.error, 30) }}</span>
              </a-tooltip>
              <span v-else>-</span>
            </template>
            <template v-if="column.key === 'created_at'">
              {{ formatDate(record.created_at) }}
            </template>
            <template v-if="column.key === 'action'">
              <a-space>
                <a-button size="small" :loading="rescrapingGuid === record.item_guid"
                  @click="handleRescrape(record)">重新刮削</a-button>
                <a-button danger size="small" @click="handleDelete(record)">删除</a-button>
              </a-space>
            </template>
            </template>
          </a-table>
        </div>
        <div class="mobile-cards">
          <a-card v-for="record in logs" :key="record.id" size="small" class="log-card">
            <div class="card-title" @click="showDetail(record)">{{ record.title || record.number }}</div>
            <div class="card-meta">{{ record.number }} · {{ formatDate(record.created_at) }}</div>
            <div class="card-row">
              <a-tag :color="record.method === 'auto' ? 'green' : 'blue'">{{ record.method === 'auto' ? '自动' : '手动' }}</a-tag>
              <a-tag v-if="record.status === 'success'" color="green">成功</a-tag><a-tag v-else-if="record.status === 'failed'" color="red">失败</a-tag><a-tag v-else-if="record.status === 'completed'" color="blue">完成</a-tag><a-badge v-else-if="record.status === 'in_progress'" :status="stepsBadgeStatus(record.steps)" :text="stepsSummary(record.steps)" /><span v-else>{{ record.status }}</span>
            </div>
            <div v-if="parseSteps(record.steps).length" class="steps-list">
              <a-tag v-for="(s, i) in parseSteps(record.steps)" :key="i"
                :color="s.status === 'success' ? 'green' : s.status === 'failed' ? 'red' : s.status === 'running' ? 'processing' : 'default'">
                {{ stepLabel(s.step) }}：{{ s.status === 'success' ? '成功' : s.status === 'failed' ? '失败' : s.status === 'running' ? '进行中' : s.status }}
              </a-tag>
            </div>
            <a-space class="card-actions"><a-button size="small" :loading="rescrapingGuid === record.item_guid" @click="handleRescrape(record)">重新刮削</a-button><a-button danger size="small" @click="handleDelete(record)">删除</a-button></a-space>
          </a-card>
          <a-pagination v-if="total > 0" v-model:current="pagination.current" v-model:page-size="pagination.pageSize" :total="total" show-size-changer @change="handleTableChange" />
        </div>
      </a-spin>
    </a-card>

    <!-- 媒体详情弹窗 -->
    <MediaDetailModal ref="mediaDetailModelRef">
    </MediaDetailModal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { message } from 'ant-design-vue'
import { getScrapeLogs, deleteScrapeLog, rescrapeItem, type ScrapeLog, type ScrapeStep } from '@/api/scrapelog'
import { formatDate } from '@/utils/format'
import { ReloadOutlined } from '@ant-design/icons-vue'
import MediaDetailModal from '@/components/MediaDetailModal.vue'

const loading = ref(false)
const numberQuery = ref('')
const logs = ref<ScrapeLog[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rescrapingGuid = ref<string | null>(null)
const mediaDetailModelRef = ref<typeof MediaDetailModal>()
let logsRequestPending = false
let autoRefreshTimer: ReturnType<typeof setInterval> | undefined

const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})

const columns = [
  { title: '番号', dataIndex: 'number', key: 'number', width: 120 },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true, width: 200 },
  { title: '刮削方式', key: 'method', width: 100 },
  { title: '状态', key: 'status', width: 80 },
  // { title: '步骤', key: 'steps', width: 180, ellipsis: true },
  // { title: '错误信息', key: 'error', width: 200, ellipsis: true },
  { title: '刮削时间', key: 'created_at', width: 160 },
  { title: '操作', key: 'action', width: 160, fixed: 'right' },
]

function truncate(s: string, n: number): string {
  if (!s) return ''
  return s.length > n ? s.slice(0, n) + '...' : s
}

const stepLabelMap: Record<string, string> = {
  search: '搜索',
  get_detail: '获取详情',
  download_poster: '下载封面',
  download_backdrop: '下载背景图',
  search_actor: '搜索演员',
  translate: '翻译',
}

function stepLabel(step: string): string {
  return stepLabelMap[step] || step
}

function parseSteps(stepsStr: string): ScrapeStep[] {
  if (!stepsStr) return []
  try {
    return JSON.parse(stepsStr)
  } catch {
    return []
  }
}

function stepsSummary(stepsStr: string): string {
  const steps = parseSteps(stepsStr)
  if (steps.length === 0) return '-'
  const total = steps.length
  return `${total}/6`
}

function stepsBadgeStatus(stepsStr: string): string {
  const steps = parseSteps(stepsStr)
  if (steps.some(s => s.status === 'running')) return 'processing'
  if (steps.some(s => s.status === 'failed')) return 'error'
  return 'success'
}

async function loadLogs({ silent = false }: { silent?: boolean } = {}) {
  if (logsRequestPending) return
  logsRequestPending = true
  if (!silent) loading.value = true
  try {
    const { data } = await getScrapeLogs(currentPage.value, pageSize.value, numberQuery.value.trim())
    logs.value = data.items || []
    total.value = data.total
    pagination.value.current = currentPage.value
    pagination.value.pageSize = pageSize.value
    pagination.value.total = data.total
  } catch {
    if (!silent) message.error('获取刮削记录失败')
  } finally {
    if (!silent) loading.value = false
    logsRequestPending = false
  }
}

function handleSearch() {
  currentPage.value = 1
  loadLogs()
}

function handleTableChange(pag: { current: number; pageSize: number }) {
  currentPage.value = pag.current
  pageSize.value = pag.pageSize
  loadLogs()
}

async function handleDelete(record: ScrapeLog) {
  try {
    await deleteScrapeLog(record.id)
    message.success('删除成功')
    loadLogs()
  } catch {
    message.error('删除失败')
  }
}

async function showDetail(record: ScrapeLog) {
  mediaDetailModelRef.value?.open(record.item_guid)
}

async function handleRescrape(record: { item_guid: string; title?: string }) {
  rescrapingGuid.value = record.item_guid
  try {
    const { data } = await rescrapeItem(record.item_guid)
    message.success(data.message || '重新刮削已开始')
    // 延迟刷新
    setTimeout(() => {
      loadLogs()
    }, 5000)
  } catch {
    message.error('重新刮削失败')
  } finally {
    rescrapingGuid.value = null
  }
}

onMounted(() => {
  loadLogs()
  autoRefreshTimer = setInterval(() => {
    loadLogs({ silent: true })
  }, 5000)
})

onUnmounted(() => {
  if (autoRefreshTimer !== undefined) clearInterval(autoRefreshTimer)
})
</script>

<style scoped>
.header-search { width: 240px; }
.mobile-cards { display: none; }
.card-title { font-weight: 600; color: #1677ff; cursor: pointer; }
.card-meta { margin: 4px 0 8px; color: #999; font-size: 12px; }
.card-row { display: flex; justify-content: flex-start; align-items: center; margin: 6px 0; }
.steps-list { display: flex; flex-wrap: wrap; gap: 4px; margin: 8px 0; color: #666; font-size: 12px; }
.card-actions { margin-top: 8px; }
@media (max-width: 639px) {
  .header-search { width: 160px; }
  .desktop-table { display: none; }
  .mobile-cards { display: block; }
  .log-card { margin-bottom: 8px; }
  .mobile-cards :deep(.ant-pagination) { margin: 16px 0 4px; text-align: center; }
}
</style>
