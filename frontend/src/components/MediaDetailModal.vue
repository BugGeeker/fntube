<template>
  <a-modal v-model:open="visible" width="1000px" :title="item?.title || '详情'"
    :body-style="{ maxHeight: '80vh', minHeight: '600px', overflow: 'auto' }">
    <a-spin :spinning="loading">
      <template v-if="item">
        <!-- 顶部图片横向布局 -->
        <div style="display: flex; gap: 12px; margin-bottom: 16px;">
          <div style="flex: 1; min-width: 0;">
            <MediaImage :src="proxyImage(item.poster)" ratio="3/2" />
          </div>
          <div style="flex: 1; min-width: 0;">
            <MediaImage :src="proxyImage(item.backdrop)" ratio="3/2" />
          </div>
          <div style="flex: 1; min-width: 0;">
            <MediaImage :src="proxyImage(item.logo)" ratio="3/2" />
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
      </template>
    </a-spin>
    <template #footer>
      <a-space>
        <a-button :loading="scraping" @click="handleScrape(item!)">
          <template #icon>
            <ThunderboltOutlined />
          </template>
          刮削
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
</template>

<script setup lang="ts">
import { useUiStore } from '@/stores/ui'
import { proxyImage } from '@/utils/image'
import { getEpisodes, getItem, getPersons, getPlayURL, getSeasons, type MediaItem, type Person, type Season } from '@/api/trimmedia'
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { rescrapeItem } from '@/api/scrapelog'
import MediaImage from './MediaImage.vue'

import MediaEditModal from './MediaEditModal.vue'

import { EditOutlined, ThunderboltOutlined, PlayCircleOutlined } from '@ant-design/icons-vue'

const uiStore = useUiStore()

const editModalRef = ref<typeof MediaEditModal>()


const visible = ref(false)
const loading = ref(false)
const gettingURL = ref(false)
const editLoading = ref(false)
const scraping = ref(false)
const persons = ref<Person[]>([])
const seasons = ref<Season[]>([])
const episodes = ref<Season[]>([])

const guid = ref('')
const item = ref<MediaItem | null>(null)

function itemYear(item: MediaItem): string {
  const date = item.release_date || item.air_date || ''
  return date.slice(0, 4)
}

// 编辑：触发编辑事件
function handleEdit(item: MediaItem) {
  editModalRef.value?.open(item)
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
  visible.value = true
  loading.value = true
  guid.value = val
  await Promise.all([loadItem(), loadPersons(), loadSeasons()])
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
