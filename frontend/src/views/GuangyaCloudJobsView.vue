<template>
  <a-card title="云添加记录">
    <template #extra>
      <a-button :loading="loading" @click="loadJobs()">
        <template #icon><ReloadOutlined /></template>
      </a-button>
    </template>

    <a-spin :spinning="loading">
      <div class="desktop-table">
        <a-table :data-source="jobs" :columns="columns" row-key="id"
          :pagination="pagination" table-layout="fixed" :scroll="{ x: 1230 }"
          @change="handleTableChange">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag>
            </template>
            <template v-else-if="column.key === 'progress'">
              <a-progress :percent="record.progress" size="small"
                :status="record.status === 'failed' ? 'exception' : record.status === 'completed' ? 'success' : 'active'" />
            </template>
            <template v-else-if="column.key === 'error'">
              <a-tooltip v-if="record.error" :title="record.error">
                <span class="error-text">{{ record.error }}</span>
              </a-tooltip>
              <span v-else>-</span>
            </template>
            <template v-else-if="column.key === 'created_at'">
              {{ formatDate(record.created_at) }}
            </template>
            <template v-else-if="column.key === 'completed_at'">
              {{ record.completed_at ? formatDate(record.completed_at) : '-' }}
            </template>
            <template v-else-if="column.key === 'action'">
              <a-popconfirm title="确认删除这条云添加记录？" ok-text="删除" cancel-text="取消"
                @confirm="handleDelete(record)">
                <a-tooltip title="删除">
                  <a-button danger size="small" :loading="deletingId === record.id" aria-label="删除">
                    <template #icon><DeleteOutlined /></template>
                  </a-button>
                </a-tooltip>
              </a-popconfirm>
            </template>
          </template>
        </a-table>
      </div>

      <div class="mobile-list">
        <div v-for="job in jobs" :key="job.id" class="job-item">
          <div class="job-heading">
            <strong>{{ job.requested_name }}</strong>
            <a-tag :color="statusColor(job.status)">{{ statusLabel(job.status) }}</a-tag>
          </div>
          <div class="job-step">{{ job.current_step || '-' }}</div>
          <a-progress :percent="job.progress" size="small"
            :status="job.status === 'failed' ? 'exception' : job.status === 'completed' ? 'success' : 'active'" />
          <div class="job-meta">创建时间：{{ formatDate(job.created_at) }}</div>
          <div v-if="job.error" class="job-error">{{ job.error }}</div>
          <div class="job-actions">
            <a-popconfirm title="确认删除这条云添加记录？" ok-text="删除" cancel-text="取消"
              @confirm="handleDelete(job)">
              <a-button danger size="small" :loading="deletingId === job.id" aria-label="删除">
                <template #icon><DeleteOutlined /></template>
              </a-button>
            </a-popconfirm>
          </div>
        </div>
        <a-empty v-if="!loading && jobs.length === 0" />
        <a-pagination v-if="total > 0" v-model:current="currentPage"
          v-model:page-size="pageSize" :total="total" show-size-changer
          @change="handleMobilePageChange" />
      </div>
    </a-spin>
  </a-card>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { theme } from 'ant-design-vue'
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import {
  deleteGuangyaCloudAddJob,
  getGuangyaCloudAddJobs,
  guangyaCloudAddEventsURL,
  type GuangyaCloudAddJob,
} from '@/api/guangya'
import { formatDate } from '@/utils/format'

const loading = ref(false)
const jobs = ref<GuangyaCloudAddJob[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const deletingId = ref<number>()
let eventSource: EventSource | undefined
const { token } = theme.useToken()

const pagination = computed(() => ({
  current: currentPage.value,
  pageSize: pageSize.value,
  total: total.value,
  showSizeChanger: true,
  showTotal: (value: number) => `共 ${value} 条`,
}))

const columns = [
  { title: '名称', dataIndex: 'requested_name', key: 'requested_name', width: 180, ellipsis: true },
  { title: '状态', key: 'status', width: 90 },
  { title: '当前步骤', dataIndex: 'current_step', key: 'current_step', width: 190, ellipsis: true },
  { title: '总体进度', key: 'progress', width: 180 },
  { title: '错误信息', key: 'error', width: 180, ellipsis: true },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '完成时间', key: 'completed_at', width: 170 },
  { title: '操作', key: 'action', width: 70, fixed: 'right' },
]

function statusLabel(status: GuangyaCloudAddJob['status']) {
  if (status === 'completed') return '已完成'
  if (status === 'failed') return '失败'
  return '进行中'
}

function statusColor(status: GuangyaCloudAddJob['status']) {
  if (status === 'completed') return 'green'
  if (status === 'failed') return 'red'
  return 'processing'
}

async function loadJobs({ silent = false }: { silent?: boolean } = {}) {
  if (!silent) loading.value = true
  try {
    const { data } = await getGuangyaCloudAddJobs(currentPage.value, pageSize.value)
    jobs.value = data.items || []
    total.value = data.total
  } catch {
    if (!silent) message.error('获取云添加记录失败')
  } finally {
    if (!silent) loading.value = false
  }
}

function handleTableChange(paginationValue: { current?: number; pageSize?: number }) {
  currentPage.value = paginationValue.current || 1
  pageSize.value = paginationValue.pageSize || 20
  loadJobs()
}

function handleMobilePageChange(page: number, size: number) {
  currentPage.value = page
  pageSize.value = size
  loadJobs()
}

function receiveJob(event: MessageEvent<string>) {
  try {
    const updated = JSON.parse(event.data) as GuangyaCloudAddJob
    const index = jobs.value.findIndex((job) => job.id === updated.id)
    if (index >= 0) {
      const next = [...jobs.value]
      next[index] = updated
      jobs.value = next
    } else {
      loadJobs({ silent: true })
    }
  } catch {
    // Ignore malformed events and keep the history stream connected.
  }
}

function applyDeletedJob(id: number, reloadMissing: boolean) {
  const index = jobs.value.findIndex((job) => job.id === id)
  if (index < 0) {
    if (reloadMissing) loadJobs({ silent: true })
    return
  }
  jobs.value = jobs.value.filter((job) => job.id !== id)
  total.value = Math.max(0, total.value - 1)
  if (jobs.value.length === 0 && currentPage.value > 1) {
    currentPage.value--
    loadJobs({ silent: true })
  }
}

function receiveDeleted(event: MessageEvent<string>) {
  try {
    const { id } = JSON.parse(event.data) as { id: number }
    applyDeletedJob(id, true)
  } catch {
    // Ignore malformed events and keep the history stream connected.
  }
}

async function handleDelete(job: GuangyaCloudAddJob) {
  deletingId.value = job.id
  try {
    await deleteGuangyaCloudAddJob(job.id)
    applyDeletedJob(job.id, false)
    message.success('云添加记录已删除')
  } catch {
    message.error('删除云添加记录失败')
  } finally {
    deletingId.value = undefined
  }
}

onMounted(() => {
  loadJobs()
  eventSource = new EventSource(guangyaCloudAddEventsURL)
  eventSource.addEventListener('cloud-add-progress', receiveJob as EventListener)
  eventSource.addEventListener('cloud-add-deleted', receiveDeleted as EventListener)
})

onBeforeUnmount(() => eventSource?.close())
</script>

<style scoped lang="scss">
.mobile-list {
  display: none;
}

.error-text {
  display: block;
  overflow: hidden;
  color: v-bind('token.colorError');
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 639px) {
  .desktop-table {
    display: none;
  }

  .mobile-list {
    display: block;
  }

  .job-item {
    padding: 12px 0;
    border-bottom: 1px solid v-bind('token.colorBorderSecondary');
  }

  .job-heading {
    display: flex;
    min-width: 0;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .job-heading strong {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .job-step,
  .job-meta,
  .job-error {
    margin-top: 6px;
    color: v-bind('token.colorTextSecondary');
    font-size: 12px;
  }

  .job-error {
    color: v-bind('token.colorError');
    overflow-wrap: anywhere;
  }

  .job-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 8px;
  }

  .mobile-list :deep(.ant-pagination) {
    margin-top: 16px;
    text-align: center;
  }
}
</style>
