<template>
  <div>
    <!-- 媒体条目列表 -->
    <a-card :title="libraryName || '加载中...'">
      <template #extra>
        <a-button @click="backToLibraries">返回媒体库</a-button>
      </template>
      <a-spin :spinning="store.loading">
        <a-row :gutter="[16, 16]">
          <a-col
            v-for="item in store.items"
            :key="item.guid"
            :xs="24" :sm="12" :md="8" :lg="6" :xl="4"
          >
            <a-card hoverable size="small" @click="showDetail(item)" style="overflow: hidden">
              <template #cover>
                <img
                  v-if="item.posters"
                  :src="proxyImage(item.posters)"
                  style="height: 200px; width: 100%; object-fit: cover"
                  alt="poster"
                />
                <div v-else style="height: 200px; display: flex; align-items: center; justify-content: center; background: #f5f5f5">
                  <span style="color: #999; font-size: 24px">{{ item.title?.charAt(0) || '?' }}</span>
                </div>
              </template>
              <a-card-meta>
                <template #title>
                  <div style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis">{{ item.title }}</div>
                </template>
                <template #description>
                  <span>{{ itemYear(item) || '未知年份' }}</span>
                  <a-tag v-if="item.type === 'TV'" color="blue" style="margin-left: 8px">剧集</a-tag>
                  <a-tag v-else-if="item.type === 'Movie'" color="orange" style="margin-left: 8px">电影</a-tag>
                </template>
              </a-card-meta>
            </a-card>
          </a-col>
        </a-row>
        <a-empty v-if="!store.loading && store.items.length === 0" description="该媒体库暂无内容" />
      </a-spin>
    </a-card>

    <!-- 媒体详情抽屉 -->
    <a-drawer
      v-model:open="detailVisible"
      width="500"
      :title="store.currentItem?.title || '详情'"
      placement="right"
    >
      <template v-if="store.currentItem">
        <div style="margin-bottom: 16px">
          <h3>{{ store.currentItem.title }}</h3>
          <p v-if="store.currentItem.original_title && store.currentItem.original_title !== store.currentItem.title" style="color: #999">
            {{ store.currentItem.original_title }}
          </p>
          <a-space>
            <a-tag>{{ itemYear(store.currentItem) || '未知年份' }}</a-tag>
            <a-tag v-if="store.currentItem.type === 'TV'" color="blue">剧集</a-tag>
            <a-tag v-else-if="store.currentItem.type === 'Movie'" color="orange">电影</a-tag>
          </a-space>
        </div>

        <!-- 简介 -->
        <div v-if="store.currentItem.overview" style="margin-bottom: 16px">
          <p style="color: #555; line-height: 1.6">{{ store.currentItem.overview }}</p>
        </div>

        <!-- 演员列表 -->
        <div v-if="store.persons.length > 0" style="margin-bottom: 16px">
          <h4>演员</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="p in store.persons" :key="p.person_guid">
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
        <div v-if="store.currentItem.type === 'TV' && store.seasons.length > 0">
          <h4>季列表</h4>
          <a-list size="small" bordered>
            <a-list-item
              v-for="season in store.seasons"
              :key="season.guid"
              @click="loadEpisodes(season.guid)"
              style="cursor: pointer"
            >
              <a-list-item-meta :title="season.title || `第 ${season.season_number} 季`" />
            </a-list-item>
          </a-list>
        </div>

        <!-- 集列表 -->
        <div v-if="store.episodes.length > 0" style="margin-top: 16px">
          <h4>剧集列表</h4>
          <a-list size="small" bordered>
            <a-list-item v-for="ep in store.episodes" :key="ep.guid">
              <a-list-item-meta
                :title="`S${ep.season_number}E${ep.episode_number} - ${ep.title || ''}`"
              />
            </a-list-item>
          </a-list>
        </div>

        <div style="margin-top: 24px">
          <a-button type="primary" :loading="gettingURL" @click="handlePlay">
            <template #icon><PlayCircleOutlined /></template>
            播放
          </a-button>
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlayCircleOutlined } from '@ant-design/icons-vue'
import { useTrimMediaStore } from '@/stores/trimmedia'
import { proxyImage } from '@/utils/image'
import type { MediaServerItem } from '@/api/trimmedia'

const route = useRoute()
const router = useRouter()
const store = useTrimMediaStore()

const detailVisible = ref(false)
const gettingURL = ref(false)

const libraryId = computed(() => route.params.id as string)
const libraryName = computed(() => {
  const lib = store.libraries.find(l => l.id === libraryId.value)
  return lib?.name || libraryId.value
})

onMounted(async () => {
  // 如果 libraries 还没加载，先加载
  if (store.libraries.length === 0) {
    await store.fetchLibraries().catch(() => {
      message.warning('未连接到飞牛影视，请先配置')
    })
  }
  // 加载该媒体库的条目
  await store.fetchItems(libraryId.value, 0, 50).catch(() => {
    message.error('获取媒体列表失败')
  })
})

function backToLibraries() {
  router.push('/media')
}

function itemYear(item: MediaServerItem): string {
  const date = item.release_date || item.air_date || ''
  return date.slice(0, 4)
}

async function showDetail(item: MediaServerItem) {
  detailVisible.value = true
  await store.fetchItem(item.guid).catch(() => {
    message.error('获取详情失败')
  })
  await store.fetchPersons(item.guid).catch(() => {})
  if (store.currentItem?.type === 'TV') {
    await store.fetchSeasons(item.guid).catch(() => {})
  }
}

async function loadEpisodes(seasonId: string) {
  await store.fetchEpisodes(seasonId).catch(() => {
    message.error('获取剧集列表失败')
  })
}

async function handlePlay() {
  if (!store.currentItem) return
  gettingURL.value = true
  try {
    const url = await store.fetchPlayURL(store.currentItem.guid)
    window.open(url, '_blank')
  } catch {
    message.error('获取播放链接失败')
  } finally {
    gettingURL.value = false
  }
}
</script>

<style scoped>
:deep(.ant-col) {
  min-width: 0;
}
:deep(.ant-card-body) {
  overflow: hidden;
}
:deep(.ant-card-meta) {
  min-width: 0;
}
:deep(.ant-card-meta-detail) {
  min-width: 0;
  overflow: hidden;
}
</style>