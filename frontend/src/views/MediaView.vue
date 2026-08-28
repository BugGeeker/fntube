<template>
  <div>
    <!-- 媒体库列表 -->
    <a-card title="媒体库">
      <template #extra>
        <a-space>
          <a-input-search v-model:value="searchKeyword" placeholder="搜索媒体" enter-button="搜索" style="width: 300px"
            @search="handleSearch" />
          <a-button @click="handleRefresh">
            <template #icon>
              <ReloadOutlined />
            </template>
          </a-button>
        </a-space>
      </template>
      <a-spin :spinning="store.loading">
        <a-row :gutter="[16, 16]">
          <a-col v-for="lib in store.libraries" :key="lib.id" :xs="12" :sm="8" :md="6" :lg="4" :xl="3" :xxl="2">
            <a-card hoverable size="small" @click="enterLibrary(lib)">
              <template #cover>
                <MediaImage :src="lib.image_list?.[0]" :alt="lib.name" ratio="2 / 3" />
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
import MediaImage from '@/components/MediaImage.vue'
import type { Library, MediaItem } from '@/api/trimmedia'
import { ReloadOutlined } from '@ant-design/icons-vue'

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

function formatYear(item: MediaItem): string {
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
  store.fetchLibraries().catch(() => {
    message.warning('未连接到飞牛影视，请先配置')
  })
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
