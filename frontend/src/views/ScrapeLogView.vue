<template>
  <div>
    <a-card title="刮削记录">
      <a-spin :spinning="loading">
        <a-table :dataSource="logs" :columns="columns" rowKey="id" :pagination="pagination" @change="handleTableChange">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'title'">
              <a @click="showDetail(record)">{{ record.title }}</a>
            </template>
            <template v-if="column.key === 'method'">
              <a-tag v-if="record.method === 'auto'" color="green">自动</a-tag>
              <a-tag v-else color="blue">手动</a-tag>
            </template>
            <template v-if="column.key === 'created_at'">
              {{ formatDate(record.created_at) }}
            </template>
            <template v-if="column.key === 'action'">
              <a-space>
                <a-button size="small" :loading="rescrapingGuid === record.item_guid" @click="handleRescrape(record)">重新刮削</a-button>
                <a-button danger size="small" @click="handleDelete(record)">删除</a-button>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-spin>
    </a-card>

    <!-- 媒体详情弹窗 -->
    <MediaDetailModal ref="mediaDetailModelRef">
    </MediaDetailModal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getScrapeLogs, deleteScrapeLog, rescrapeItem, type ScrapeLog } from '@/api/scrapelog'
import MediaDetailModal from '@/components/MediaDetailModal.vue'

const loading = ref(false)
const logs = ref<ScrapeLog[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rescrapingGuid = ref<string | null>(null)
const mediaDetailModelRef = ref<typeof MediaDetailModal>()

const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})

const columns = [
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '番号', dataIndex: 'number', key: 'number', width: 120 },
  { title: '刮削方式', key: 'method', width: 100 },
  { title: '刮削时间', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 180 },
]

function formatDate(date: string): string {
  if (!date) return ''
  const d = new Date(date)
  if (isNaN(d.getTime())) return date
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${m}-${day} ${h}:${min}`
}

async function loadLogs() {
  loading.value = true
  try {
    const { data } = await getScrapeLogs(currentPage.value, pageSize.value)
    logs.value = data.items || []
    total.value = data.total
    pagination.value.current = currentPage.value
    pagination.value.pageSize = pageSize.value
    pagination.value.total = data.total
  } catch {
    message.error('获取刮削记录失败')
  } finally {
    loading.value = false
  }
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
})
</script>
