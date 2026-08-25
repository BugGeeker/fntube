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
    <a-modal v-model:open="detailVisible" width="1000px" :title="detailItem?.title || '详情'" :footer="null"
      :body-style="{ maxHeight: '80vh', overflow: 'auto' }">
      <template v-if="detailItem">
        <!-- 顶部图片横向布局 -->
        <div
          v-if="!uiStore.hideImages && (detailItem.backdrop || detailItem.poster || detailItem.logo)"
          style="display: flex; gap: 12px; margin-bottom: 16px;">
          <div style="flex: 1; min-width: 0;">
            <img :src="proxyImage(detailItem.poster)"
              style="width: 100%; aspect-ratio: 3 / 2; object-fit: contain; border-radius: 8px;" alt="poster" />
          </div>
          <div style="flex: 1; min-width: 0;">
            <img :src="proxyImage(detailItem.backdrop)"
              style="width: 100%; aspect-ratio: 3 / 2; object-fit: contain; border-radius: 8px;" alt="backdrop" />
          </div>
          <div style="flex: 1; min-width: 0;">
            <img :src="proxyImage(detailItem.logo)"
              style="width: 100%; aspect-ratio: 3 / 2; object-fit: contain; border-radius: 8px; background: #1a1a2e;"
              alt="logo" />
          </div>
        </div>

        <div style="margin-bottom: 16px">
          <p v-if="detailItem.original_title && detailItem.original_title !== detailItem.title"
            style="color: #999">
            {{ detailItem.original_title }}
          </p>
          <a-space>
            <a-tag>{{ itemYear(detailItem) || '未知年份' }}</a-tag>
            <a-tag v-if="detailItem.type === 'TV'" color="blue">剧集</a-tag>
            <a-tag v-else-if="detailItem.type === 'Movie'" color="orange">电影</a-tag>
          </a-space>
        </div>

        <!-- 简介 -->
        <div v-if="detailItem.overview" style="margin-bottom: 16px">
          <p style="color: #555; line-height: 1.6">{{ detailItem.overview }}</p>
        </div>

        <!-- 演员列表 -->
        <div v-if="persons.length > 0" style="margin-bottom: 16px">
          <h4>演员</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="p in persons" :key="p.person_guid">
              <a-list-item-meta>
                <template #avatar>
                  <a-avatar v-if="!uiStore.hideImages && p.profile_path" :src="proxyImage(p.profile_path)" :size="40" />
                  <a-avatar v-else :size="40">{{ p.name?.charAt(0) || '?' }}</a-avatar>
                </template>
                <template #title>
                  <span>{{ p.name }}</span>
                  <a-tag v-if="p.job" color="blue" style="margin-left: 8px">{{ p.job }}</a-tag>
                </template>
                <template #description>
                  <span v-if="p.role">饰 {{ p.role }}</span>
                  <span v-else-if="p.job">{{ p.job }}</span>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </a-list>
        </div>

        <!-- 剧集季列表 -->
        <div v-if="detailItem.type === 'TV' && seasons.length > 0">
          <h4>季列表</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="season in seasons" :key="season.guid" @click="loadEpisodes(season.guid)"
              style="cursor: pointer">
              <a-list-item-meta :title="season.title || `第 ${season.season_number} 季`" />
            </a-list-item>
          </a-list>
        </div>

        <!-- 集列表 -->
        <div v-if="episodes.length > 0" style="margin-top: 16px">
          <h4>剧集列表</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="ep in episodes" :key="ep.guid">
              <a-list-item-meta :title="`S${ep.season_number}E${ep.episode_number} - ${ep.title || ''}`" />
            </a-list-item>
          </a-list>
        </div>

        <div style="margin-top: 24px">
          <a-space>
            <a-button :loading="rescrapingGuid === detailItem.guid" @click="handleRescrape({ item_guid: detailItem.guid, title: detailItem.title })">
              <template #icon>
                <ThunderboltOutlined />
              </template>
              重新刮削
            </a-button>
          </a-space>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ThunderboltOutlined } from '@ant-design/icons-vue'
import { getScrapeLogs, deleteScrapeLog, rescrapeItem, type ScrapeLog } from '@/api/scrapelog'
import { getItem, getPersons, getSeasons, getEpisodes, type MediaServerItem, type Person, type SeasonItem } from '@/api/trimmedia'
import { useUiStore } from '@/stores/ui'
import { proxyImage } from '@/utils/image'

const uiStore = useUiStore()
const loading = ref(false)
const logs = ref<ScrapeLog[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rescrapingGuid = ref<string | null>(null)

// 详情弹窗
const detailVisible = ref(false)
const detailItem = ref<MediaServerItem | null>(null)
const persons = ref<Person[]>([])
const seasons = ref<SeasonItem[]>([])
const episodes = ref<SeasonItem[]>([])

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

function itemYear(item: MediaServerItem): string {
  const date = item.release_date || item.air_date || ''
  return date.slice(0, 4)
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
  detailVisible.value = true
  detailItem.value = null
  persons.value = []
  seasons.value = []
  episodes.value = []
  try {
    const { data } = await getItem(record.item_guid)
    detailItem.value = data
  } catch {
    message.error('获取详情失败')
    return
  }
  await getPersons(record.item_guid).then(({ data }) => { persons.value = data || [] }).catch(() => {})
  if (detailItem.value?.type === 'TV') {
    await getSeasons(record.item_guid).then(({ data }) => { seasons.value = data || [] }).catch(() => {})
  }
}

async function loadEpisodes(seasonId: string) {
  try {
    const { data } = await getEpisodes(seasonId)
    episodes.value = data || []
  } catch {
    message.error('获取剧集列表失败')
  }
}

async function handleRescrape(record: { item_guid: string; title?: string }) {
  rescrapingGuid.value = record.item_guid
  try {
    const { data } = await rescrapeItem(record.item_guid)
    message.success(data.message || '重新刮削已开始')
    // 延迟刷新
    setTimeout(() => {
      loadLogs()
      // 如果详情弹窗打开，刷新详情
      if (detailVisible.value && detailItem.value?.guid === record.item_guid) {
        showDetail({ item_guid: record.item_guid, title: record.title || '', id: 0, number: '', method: '', created_at: '' })
      }
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
