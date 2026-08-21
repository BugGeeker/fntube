<template>
  <div>
    <!-- 媒体库列表 -->
    <a-card title="媒体库">
      <template #extra>
        <a-space>
          <a-input-search
            v-model:value="searchKeyword"
            placeholder="搜索媒体"
            enter-button="搜索"
            style="width: 300px"
            @search="handleSearch"
          />
          <a-button @click="handleRefresh">刷新媒体库</a-button>
        </a-space>
      </template>
      <a-spin :spinning="store.loading">
        <a-row :gutter="[16, 16]">
          <a-col
            v-for="lib in store.libraries"
            :key="lib.id"
            :xs="24" :sm="12" :md="8" :lg="6" :xl="4"
          >
            <a-card hoverable size="small" @click="enterLibrary(lib)">
              <template #cover>
                <img
                  v-if="lib.image_list && lib.image_list.length"
                  :src="proxyImage(lib.image_list[0])"
                  style="height: 160px; object-fit: cover"
                  alt="poster"
                />
                <div v-else style="height: 160px; display: flex; align-items: center; justify-content: center; background: #f0f0f0">
                  <span style="color: #999; font-size: 32px">{{ lib.name.charAt(0) }}</span>
                </div>
              </template>
              <a-card-meta :title="lib.name">
                <template #description>
                  <a-tag :color="typeColor(lib.type)">{{ typeLabel(lib.type) }}</a-tag>
                  <span>{{ lib.item_count }} 项</span>
                </template>
              </a-card-meta>
            </a-card>
          </a-col>
        </a-row>
        <a-empty v-if="!store.loading && store.libraries.length === 0" description="暂无媒体库，请先在配置页设置连接" />
      </a-spin>
    </a-card>

    <!-- 搜索结果 -->
    <a-modal v-model:open="searchVisible" title="搜索结果" width="800" :footer="null">
      <a-spin :spinning="store.loading">
        <a-list :data-source="store.searchResults" bordered>
          <template #renderItem="{ item }">
            <a-list-item style="cursor: pointer">
              <a-list-item-meta :title="item.title" :description="`${formatYear(item)} - ${item.type}`" />
            </a-list-item>
          </template>
          <template #footer>
            <span v-if="store.searchResults.length === 0">无搜索结果</span>
          </template>
        </a-list>
      </a-spin>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { useTrimMediaStore } from '@/stores/trimmedia'
import { proxyImage } from '@/utils/image'
import type { Library, MediaServerItem } from '@/api/trimmedia'

const router = useRouter()
const store = useTrimMediaStore()

const searchVisible = ref(false)
const searchKeyword = ref('')

onMounted(() => {
  store.fetchLibraries().catch(() => {
    message.warning('未连接到飞牛影视，请先配置')
  })
})

function typeLabel(type: string): string {
  const map: Record<string, string> = { movie: '电影', tv: '电视剧', mix: '混合', music: '音乐', other: '其他' }
  return map[type] || type
}

function typeColor(type: string): string {
  const map: Record<string, string> = { movie: 'orange', tv: 'blue', mix: 'purple', music: 'green', other: 'default' }
  return map[type] || 'default'
}

function enterLibrary(lib: Library) {
  router.push(`/media/${lib.id}`)
}

function formatYear(item: MediaServerItem): string {
  const date = item.release_date || item.air_date || ''
  return date.slice(0, 4)
}

async function handleSearch() {
  if (!searchKeyword.value.trim()) return
  searchVisible.value = true
  await store.search(searchKeyword.value).catch(() => {
    message.error('搜索失败')
  })
}

async function handleRefresh() {
  const ok = await store.refresh().catch(() => false)
  if (ok) {
    message.success('已触发刷新媒体库')
  } else {
    message.error('刷新失败，可能需要管理员权限')
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
:deep(.ant-card-meta-title) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
