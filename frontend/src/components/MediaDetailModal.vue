<template>
  <a-modal v-model:open="visible" width="1000px" :title="item?.title || '详情'"
    :body-style="{ maxHeight: '80vh', minHeight: '600px', overflow: 'auto' }">
    <a-spin :spinning="loading" style="min-height: '600px'; width: 100%;">
      <template v-if="item">
        <!-- 顶部图片横向布局 -->
        <div style="display: flex; gap: 12px; margin-bottom: 16px;">
          <div style="flex: 1; min-width: 0;">
            <MediaImage :src="item.poster" ratio="3/2" alt="封面"/>
          </div>
          <div style="flex: 1; min-width: 0;">
            <MediaImage :src="item.backdrop" ratio="3/2" alt="背景"/>
          </div>
          <div style="flex: 1; min-width: 0;">
            <MediaImage :src="item.logo" ratio="3/2" alt="logo"/>
          </div>
        </div>

        <div style="margin-bottom: 16px">
          <p v-if="item.original_title && item.original_title !== item.title" style="color: #999">
            {{ item.original_title }}
          </p>
          <a-space>
            <a-tag>{{ itemYear(item) || '未知年份' }}</a-tag>
            <a-tag v-if="item.type === 'TV'" color="blue">剧集</a-tag>
            <a-tag v-else-if="item.type === 'Movie'" color="orange">电影</a-tag>
          </a-space>
        </div>

        <!-- 简介 -->
        <div v-if="item.overview" style="margin-bottom: 16px">
          <p style="color: #555; line-height: 1.6">{{ item.overview }}</p>
        </div>

        <!-- 演员列表 -->
        <div v-if="persons.length > 0" style="margin-bottom: 16px">
          <h4>演员</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="p in persons" :key="p.person_guid">
              <a-list-item-meta>
                <template #avatar>
                  <a-avatar v-if="p.profile_path" :src="proxyImage(p.profile_path)" :size="40" />
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
        <div v-if="item.type === 'TV' && seasons.length > 0">
          <h4>季列表</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="season in seasons" :key="season.guid" @click="loadEpisodes(season.guid)"
              style="cursor: pointer">
              <a-list-item-meta :title="`第 ${season.season_number} 季 ${season.title ? ` - ${season.title}` : ''}`"
                :description="season.overview" />
            </a-list-item>
          </a-list>
        </div>

        <!-- 集列表 -->
        <div v-if="episodes.length > 0" style="margin-top: 16px">
          <h4>剧集列表</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="ep in episodes" :key="ep.guid">
              <a-list-item-meta :title="`第${ep.episode_number}集${ep.title ? ` - ${ep.title}` : ''}`" />
            </a-list-item>
          </a-list>
        </div>

        <!-- 媒体文件信息 -->
        <div v-if="streamInfo" style="margin-top: 16px">
          <h4>媒体文件</h4>
          <a-table
            :dataSource="streamInfo.files"
            :columns="fileColumns"
            rowKey="guid"
            size="small"
            :pagination="false"
            bordered
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'size'">
                {{ formatFileSize(record.size) }}
              </template>
              <template v-if="column.key === 'path'">
                <a-tooltip :title="record.path">
                  <span style="font-family: monospace; font-size: 12px">{{ record.path }}</span>
                </a-tooltip>
              </template>
              <template v-if="column.key === 'can_play'">
                <a-tag v-if="record.can_play === 1" color="green">可播放</a-tag>
                <a-tag v-else color="red">不可播放</a-tag>
              </template>
            </template>
          </a-table>
        </div>

        <!-- 视频流信息 -->
        <div v-if="streamInfo && streamInfo.video_streams.length > 0" style="margin-top: 16px">
          <h4>视频流</h4>
          <a-table
            :dataSource="streamInfo.video_streams"
            :columns="videoColumns"
            rowKey="guid"
            size="small"
            :pagination="false"
            bordered
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'resolution'">
                {{ record.width }}×{{ record.height }}
              </template>
              <template v-if="column.key === 'resolution_type'">
                <a-tag :color="getResolutionColor(record.resolution_type)">{{ record.resolution_type }}</a-tag>
              </template>
              <template v-if="column.key === 'bps'">
                {{ formatBps(record.bps) }}
              </template>
              <template v-if="column.key === 'duration'">
                {{ formatDuration(record.duration) }}
              </template>
              <template v-if="column.key === 'color_range_type'">
                <a-tag :color="getColorRangeColor(record.color_range_type)">{{ record.color_range_type }}</a-tag>
              </template>
            </template>
          </a-table>
        </div>

        <!-- 音频流信息 -->
        <div v-if="streamInfo && streamInfo.audio_streams.length > 0" style="margin-top: 16px">
          <h4>音频流</h4>
          <a-table
            :dataSource="streamInfo.audio_streams"
            :columns="audioColumns"
            rowKey="guid"
            size="small"
            :pagination="false"
            bordered
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'audio_type'">
                <a-tag :color="getAudioTypeColor(record.audio_type)">{{ record.audio_type }}</a-tag>
              </template>
              <template v-if="column.key === 'bps'">
                {{ formatBps(record.bps) }}
              </template>
              <template v-if="column.key === 'duration'">
                {{ formatDuration(record.duration) }}
              </template>
              <template v-if="column.key === 'is_default'">
                <a-tag v-if="record.is_default === 1" color="blue">默认</a-tag>
                <span v-else>-</span>
              </template>
              <template v-if="column.key === 'channels'">
                {{ getChannelLayout(record.channels) }}
              </template>
            </template>
          </a-table>
        </div>

        <!-- 字幕流信息 -->
        <div v-if="streamInfo && streamInfo.subtitle_streams.length > 0" style="margin-top: 16px">
          <h4>字幕</h4>
          <div style="display: flex; flex-wrap: wrap; gap: 8px;">
            <a-tag v-for="(subtitle, index) in streamInfo.subtitle_streams" :key="index" color="cyan">
              {{ subtitle }}
            </a-tag>
          </div>
        </div>
      </template>
    </a-spin>
    <template #footer>
      <a-space>
        <a-button :loading="scraping" @click="handleScrape(item!)">
          <template #icon>
            <ThunderboltOutlined />
          </template>
          一键刮削
        </a-button>
        <a-button @click="handleSearch">
          <template #icon>
            <SearchOutlined />
          </template>
          搜索
        </a-button>
        <a-button :loading="editLoading" @click="handleEdit(item!)">
          <template #icon>
            <EditOutlined />
          </template>
          编辑
        </a-button>
        <a-button type="primary" :loading="gettingURL" @click="handlePlay">
          <template #icon>
            <PlayCircleOutlined />
          </template>
          播放
        </a-button>
      </a-space>
    </template>
  </a-modal>
  <MediaEditModal ref="editModalRef" @saved="handleSaved" />
  <MetaTubeSearchModal ref="searchModalRef" @select="handleSearchSelect" />
</template>

<script setup lang="ts">
import { proxyImage } from '@/utils/image'
import { getEpisodes, getItem, getPersons, getPlayURL, getSeasons, getStreamList, type MediaItem, type Person, type Season, type StreamListResult } from '@/api/trimmedia'
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { rescrapeItem } from '@/api/scrapelog'
import MediaImage from './MediaImage.vue'

import MediaEditModal from './MediaEditModal.vue'
import MetaTubeSearchModal from './MetaTubeSearchModal.vue'
import { type MovieInfo } from '@/api/metatube'

import { EditOutlined, ThunderboltOutlined, PlayCircleOutlined, SearchOutlined } from '@ant-design/icons-vue'

const editModalRef = ref<typeof MediaEditModal>()
const searchModalRef = ref<typeof MetaTubeSearchModal>()

const visible = ref(false)
const loading = ref(false)
const gettingURL = ref(false)
const editLoading = ref(false)
const scraping = ref(false)
const persons = ref<Person[]>([])
const seasons = ref<Season[]>([])
const episodes = ref<Season[]>([])
const streamInfo = ref<StreamListResult | null>(null)

const guid = ref('')
const item = ref<MediaItem | null>(null)

const fileColumns = [
  { title: '文件名', dataIndex: 'file_name', key: 'file_name', ellipsis: true },
  { title: '大小', key: 'size', width: 100 },
  { title: '路径', dataIndex: 'path', key: 'path', ellipsis: true },
  { title: '状态', key: 'can_play', width: 90 },
]

const videoColumns = [
  { title: '分辨率', key: 'resolution', width: 120 },
  { title: '类型', key: 'resolution_type', width: 80 },
  { title: '编码', dataIndex: 'codec_name', key: 'codec_name', width: 80 },
  { title: 'HDR', key: 'color_range_type', width: 80 },
  { title: '封装', dataIndex: 'wrapper', key: 'wrapper', width: 80 },
  { title: '码率', key: 'bps', width: 100 },
  { title: '帧率', dataIndex: 'r_frame_rate', key: 'r_frame_rate', width: 90 },
  { title: '时长', key: 'duration', width: 90 },
  { title: '位深', dataIndex: 'bit_depth', key: 'bit_depth', width: 70 },
]

const audioColumns = [
  { title: '类型', key: 'audio_type', width: 100 },
  { title: '编码', dataIndex: 'codec_name', key: 'codec_name', width: 80 },
  { title: '语言', dataIndex: 'language', key: 'language', width: 70 },
  { title: '声道', key: 'channels', width: 80 },
  { title: '采样率', dataIndex: 'sample_rate', key: 'sample_rate', width: 90 },
  { title: '码率', key: 'bps', width: 100 },
  { title: '默认', key: 'is_default', width: 70 },
  { title: '时长', key: 'duration', width: 90 },
]

function formatFileSize(bytes: number): string {
  if (!bytes) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function formatBps(bps: number): string {
  if (!bps) return '-'
  if (bps >= 1000000) return `${(bps / 1000000).toFixed(1)} Mbps`
  if (bps >= 1000) return `${(bps / 1000).toFixed(0)} kbps`
  return `${bps} bps`
}

function formatDuration(seconds: number): string {
  if (!seconds) return '-'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

function getChannelLayout(channels: number): string {
  const map: Record<number, string> = {
    1: 'Mono',
    2: 'Stereo',
    6: '5.1',
    8: '7.1',
  }
  return map[channels] || `${channels}ch`
}

function getResolutionColor(type: string): string {
  const map: Record<string, string> = {
    '4k': 'purple',
    '1080p': 'blue',
    '720p': 'green',
    '480p': 'orange',
  }
  return map[type?.toLowerCase()] || 'default'
}

function getColorRangeColor(type: string): string {
  const map: Record<string, string> = {
    'HDR': 'gold',
    'HDR10': 'gold',
    'DolbyVision': 'magenta',
    'SDR': 'default',
  }
  return map[type] || 'default'
}

function getAudioTypeColor(type: string): string {
  const map: Record<string, string> = {
    'DolbyAtmos': 'magenta',
    'DTS': 'gold',
    'TrueHD': 'purple',
  }
  return map[type] || 'blue'
}

function itemYear(item: MediaItem): string {
  const date = item.release_date || item.air_date || ''
  return date.slice(0, 4)
}

// 编辑：触发编辑事件
function handleEdit(item: MediaItem) {
  editModalRef.value?.open(item)
}

async function handleSearch() {
  if (!item.value) return
  const fileName = streamInfo.value?.files[0]?.file_name
  const keyword = fileName?.replace(/\.[^.]+$/, '')
  if (!keyword) {
    message.warning('无法获取媒体文件名')
    return
  }
  searchModalRef.value?.open(keyword)
}

async function handleSearchSelect(info: MovieInfo) {
  if (!item.value) return
  await editModalRef.value?.openWithMovieInfo(item.value, info)
}

// 编辑：刷新详情
function handleSaved() {
  loadItem()
}



const open = async (val: string) => {
  item.value = null
  persons.value = []
  seasons.value = []
  episodes.value = []
  streamInfo.value = null
  visible.value = true
  loading.value = true
  guid.value = val
  await Promise.all([loadItem(), loadPersons(), loadSeasons(), loadStreamInfo()])
  loading.value = false
}

defineExpose({
  open,
})

async function handlePlay() {
  if (!guid.value) return
  gettingURL.value = true
  try {
    const { data } = await getPlayURL(guid.value)
    window.open(data?.url, '_blank')
  } catch {
    message.error('获取播放链接失败')
  } finally {
    gettingURL.value = false
  }
}

async function loadItem() {
  try {
    const { data } = await getItem(guid.value)
    item.value = data
  } catch {
    message.error('获取详情失败')
  }
}

async function loadPersons() {
  try {
    const { data } = await getPersons(guid.value)
    persons.value = data || []
  } catch { }
}

async function loadSeasons() {
  try {
    if (item.value?.type !== 'TV') return
    const { data } = await getSeasons(guid.value)
    seasons.value = data || []
    if (seasons.value.length > 0) {
      await loadEpisodes(seasons.value[0].guid)
    }
  } catch { }
}


async function loadEpisodes(seasonId: string) {
  try {
    const { data } = await getEpisodes(seasonId)
    episodes.value = data || []
  } catch {
    message.error('获取剧集列表失败')
  }
}

async function loadStreamInfo() {
  try {
    const { data } = await getStreamList(guid.value)
    streamInfo.value = data || null
  } catch {
    // 静默失败，不影响详情展示
  }
}

// 刮削：调用后端异步刮削接口
async function handleScrape(item: MediaItem) {
  editLoading.value = true
  scraping.value = true
  try {
    const { data } = await rescrapeItem(item.guid)
    message.success(data.message || '刮削已开始')
  } catch {
    message.error('刮削失败')
  } finally {
    scraping.value = false
  }
}
</script>
